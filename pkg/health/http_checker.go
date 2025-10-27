package health

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPChecker performs HTTP/HTTPS-based health checks
type HTTPChecker struct {
	config *CheckConfig
	client *http.Client
}

// NewHTTPChecker creates a new HTTP-based health checker
func NewHTTPChecker(config *CheckConfig) *HTTPChecker {
	// Create custom HTTP client with timeout and interface binding
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// If we have a source address, bind to it
			var localAddr net.Addr
			if config.PingSourceAddr != "" {
				localAddr = &net.TCPAddr{
					IP: net.ParseIP(config.PingSourceAddr),
				}
			}

			dialer := &net.Dialer{
				LocalAddr: localAddr,
				Timeout:   config.Timeout,
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.HTTPInsecureTLS,
		},
		DisableKeepAlives: true, // Each check is independent
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	if !config.HTTPFollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &HTTPChecker{
		config: config,
		client: client,
	}
}

// Check performs an HTTP-based health check
func (c *HTTPChecker) Check(target string) (*CheckResult, error) {
	result := &CheckResult{
		WANID:     c.config.WANID,
		Timestamp: time.Now(),
		Method:    c.config.Method,
		Target:    target,
		Metadata:  make(map[string]interface{}),
	}

	// Build URL
	url := target
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// Determine protocol based on config
		if c.config.Method == CheckMethodHTTPS {
			url = "https://" + url
		} else {
			url = "http://" + url
		}
	}

	// Add path if specified
	if c.config.HTTPPath != "" && c.config.HTTPPath != "/" {
		url += c.config.HTTPPath
	}

	// Create request
	req, err := http.NewRequest(c.config.HTTPMethod, url, nil)
	if err != nil {
		result.Error = fmt.Errorf("failed to create HTTP request: %w", err)
		result.Status = WANStatusDown
		return result, result.Error
	}

	// Add custom headers
	for key, value := range c.config.HTTPHeaders {
		req.Header.Set(key, value)
	}

	// Set User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "MultiWANBond-HealthChecker/1.0")
	}

	// Perform request and measure latency
	start := time.Now()
	resp, err := c.client.Do(req)
	latency := time.Since(start)
	result.Latency = latency

	if err != nil {
		result.Error = fmt.Errorf("HTTP request failed: %w", err)
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}
	defer resp.Body.Close()

	// Read response body (with limit)
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024)) // Max 10KB
	if err != nil {
		result.Error = fmt.Errorf("failed to read response body: %w", err)
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}

	// Record status code
	result.HTTPStatusCode = resp.StatusCode

	// Check status code
	expectedStatus := c.config.HTTPExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	if resp.StatusCode != expectedStatus {
		result.Error = fmt.Errorf("unexpected status code: got %d, expected %d", resp.StatusCode, expectedStatus)
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}

	// Check expected body content if configured
	if c.config.HTTPExpectedBody != "" {
		bodyStr := string(body)
		if !strings.Contains(bodyStr, c.config.HTTPExpectedBody) {
			result.Error = fmt.Errorf("expected body content not found")
			result.Success = false
			result.Status = WANStatusDown
			return result, result.Error
		}
	}

	// Success
	result.Success = true

	// Determine status based on latency
	if latency > c.config.DegradedLatency {
		result.Status = WANStatusDegraded
	} else {
		result.Status = WANStatusUp
	}

	result.Metadata["status_code"] = resp.StatusCode
	result.Metadata["body_size"] = len(body)
	result.Metadata["content_type"] = resp.Header.Get("Content-Type")

	return result, nil
}

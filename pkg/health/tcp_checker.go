package health

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// TCPChecker performs TCP connection-based health checks
type TCPChecker struct {
	config *CheckConfig
}

// NewTCPChecker creates a new TCP-based health checker
func NewTCPChecker(config *CheckConfig) *TCPChecker {
	return &TCPChecker{
		config: config,
	}
}

// Check performs a TCP-based health check
func (c *TCPChecker) Check(target string) (*CheckResult, error) {
	result := &CheckResult{
		WANID:     c.config.WANID,
		Timestamp: time.Now(),
		Method:    CheckMethodTCP,
		Target:    target,
		Metadata:  make(map[string]interface{}),
	}

	// Build address with port
	address := target
	if !strings.Contains(address, ":") {
		// Add default port if not specified
		port := c.config.TCPPort
		if port == 0 {
			port = 80 // Default to HTTP port
		}
		address = fmt.Sprintf("%s:%d", address, port)
	}

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: c.config.Timeout,
	}

	// Bind to source address if specified
	if c.config.PingSourceAddr != "" {
		dialer.LocalAddr = &net.TCPAddr{
			IP: net.ParseIP(c.config.PingSourceAddr),
		}
	}

	// Attempt TCP connection
	start := time.Now()
	conn, err := dialer.Dial("tcp", address)
	connectTime := time.Since(start)
	result.Latency = connectTime
	result.TCPConnectTime = connectTime

	if err != nil {
		result.Error = fmt.Errorf("TCP connection failed: %w", err)
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}
	defer conn.Close()

	// Set deadline for send/receive operations
	conn.SetDeadline(time.Now().Add(c.config.Timeout))

	// Send data if configured
	if c.config.TCPSend != "" {
		_, err = conn.Write([]byte(c.config.TCPSend))
		if err != nil {
			result.Error = fmt.Errorf("TCP send failed: %w", err)
			result.Success = false
			result.Status = WANStatusDown
			return result, result.Error
		}

		// If we expect a response, read it
		if c.config.TCPExpect != "" {
			reader := bufio.NewReader(conn)
			response, err := reader.ReadString('\n')
			if err != nil {
				result.Error = fmt.Errorf("TCP receive failed: %w", err)
				result.Success = false
				result.Status = WANStatusDown
				return result, result.Error
			}

			// Check if response matches expected
			if !strings.Contains(response, c.config.TCPExpect) {
				result.Error = fmt.Errorf("unexpected TCP response: got %q, expected %q", response, c.config.TCPExpect)
				result.Success = false
				result.Status = WANStatusDown
				return result, result.Error
			}

			result.Metadata["response"] = strings.TrimSpace(response)
		}
	}

	// Success
	result.Success = true

	// Determine status based on latency
	if connectTime > c.config.DegradedLatency {
		result.Status = WANStatusDegraded
	} else {
		result.Status = WANStatusUp
	}

	result.Metadata["connect_time_ms"] = connectTime.Milliseconds()
	result.Metadata["address"] = address

	return result, nil
}

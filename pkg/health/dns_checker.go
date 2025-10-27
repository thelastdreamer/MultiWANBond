package health

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
)

// DNSChecker performs DNS-based health checks
type DNSChecker struct {
	config *CheckConfig
}

// NewDNSChecker creates a new DNS-based health checker
func NewDNSChecker(config *CheckConfig) *DNSChecker {
	return &DNSChecker{
		config: config,
	}
}

// Check performs a DNS-based health check
func (c *DNSChecker) Check(target string) (*CheckResult, error) {
	result := &CheckResult{
		WANID:     c.config.WANID,
		Timestamp: time.Now(),
		Method:    CheckMethodDNS,
		Target:    target,
		Metadata:  make(map[string]interface{}),
	}

	// Determine domain to query
	domain := c.config.DNSQueryDomain
	if domain == "" {
		domain = "google.com"
	}

	// Create custom resolver if DNS server is specified
	var resolver *net.Resolver
	if target != "" {
		// Use the target as DNS server
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: c.config.Timeout,
				}
				// Use target as DNS server
				dnsServer := target
				if !strings.Contains(dnsServer, ":") {
					dnsServer += ":53"
				}
				return d.DialContext(ctx, "udp", dnsServer)
			},
		}
	} else {
		// Use system resolver
		resolver = net.DefaultResolver
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
	defer cancel()

	// Perform DNS lookup based on query type
	start := time.Now()
	var answers []string
	var err error

	switch strings.ToUpper(c.config.DNSQueryType) {
	case "A", "":
		// IPv4 addresses
		ips, lookupErr := resolver.LookupIP(ctx, "ip4", domain)
		err = lookupErr
		for _, ip := range ips {
			answers = append(answers, ip.String())
		}

	case "AAAA":
		// IPv6 addresses
		ips, lookupErr := resolver.LookupIP(ctx, "ip6", domain)
		err = lookupErr
		for _, ip := range ips {
			answers = append(answers, ip.String())
		}

	case "CNAME":
		// Canonical name
		cname, lookupErr := resolver.LookupCNAME(ctx, domain)
		err = lookupErr
		if cname != "" {
			answers = append(answers, cname)
		}

	case "MX":
		// Mail exchange records
		mxRecords, lookupErr := resolver.LookupMX(ctx, domain)
		err = lookupErr
		for _, mx := range mxRecords {
			answers = append(answers, fmt.Sprintf("%s (priority %d)", mx.Host, mx.Pref))
		}

	case "TXT":
		// Text records
		txtRecords, lookupErr := resolver.LookupTXT(ctx, domain)
		err = lookupErr
		answers = txtRecords

	case "NS":
		// Name server records
		nsRecords, lookupErr := resolver.LookupNS(ctx, domain)
		err = lookupErr
		for _, ns := range nsRecords {
			answers = append(answers, ns.Host)
		}

	default:
		result.Error = fmt.Errorf("unsupported DNS query type: %s", c.config.DNSQueryType)
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}

	latency := time.Since(start)
	result.Latency = latency
	result.DNSResolveTime = latency
	result.DNSAnswers = answers

	if err != nil {
		result.Error = fmt.Errorf("DNS lookup failed: %w", err)
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}

	// Check if we got any answers
	if len(answers) == 0 {
		result.Error = fmt.Errorf("no DNS answers received")
		result.Success = false
		result.Status = WANStatusDown
		return result, result.Error
	}

	// Check expected IP if configured
	if c.config.DNSExpectedIP != "" {
		found := false
		for _, answer := range answers {
			if answer == c.config.DNSExpectedIP {
				found = true
				break
			}
		}
		if !found {
			result.Error = fmt.Errorf("expected IP %s not found in DNS answers", c.config.DNSExpectedIP)
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

	result.Metadata["query_type"] = c.config.DNSQueryType
	result.Metadata["domain"] = domain
	result.Metadata["answer_count"] = len(answers)

	return result, nil
}

package health

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// PingChecker performs ICMP ping-based health checks
type PingChecker struct {
	config *CheckConfig
}

// NewPingChecker creates a new ping-based health checker
func NewPingChecker(config *CheckConfig) *PingChecker {
	return &PingChecker{
		config: config,
	}
}

// Check performs a ping-based health check
func (c *PingChecker) Check(target string) (*CheckResult, error) {
	result := &CheckResult{
		WANID:     c.config.WANID,
		Timestamp: time.Now(),
		Method:    CheckMethodPing,
		Target:    target,
		Metadata:  make(map[string]interface{}),
	}

	// Parse target IP
	targetIP := net.ParseIP(target)
	if targetIP == nil {
		// Try to resolve if it's a hostname
		addrs, err := net.LookupIP(target)
		if err != nil || len(addrs) == 0 {
			result.Error = fmt.Errorf("invalid target or unable to resolve: %s", target)
			return result, result.Error
		}
		targetIP = addrs[0]
	}

	// Perform multiple pings as configured
	var totalLatency time.Duration
	var latencies []time.Duration
	successCount := 0

	for i := 0; i < c.config.PingCount; i++ {
		latency, err := c.singlePing(targetIP)
		if err == nil {
			successCount++
			totalLatency += latency
			latencies = append(latencies, latency)
		}

		// Small delay between pings
		if i < c.config.PingCount-1 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	// Calculate results
	if successCount > 0 {
		result.Latency = totalLatency / time.Duration(successCount)
		result.PacketLoss = float64(c.config.PingCount-successCount) / float64(c.config.PingCount)

		// Calculate jitter (variance in latency)
		if len(latencies) > 1 {
			result.Jitter = calculateJitter(latencies)
		}

		// Consider success if at least 50% of pings succeeded
		result.Success = successCount >= (c.config.PingCount / 2)

		// Determine status based on thresholds
		if result.Success {
			if result.Latency > c.config.DegradedLatency || result.PacketLoss > c.config.DegradedPacketLoss {
				result.Status = WANStatusDegraded
			} else {
				result.Status = WANStatusUp
			}
		} else {
			result.Status = WANStatusDown
		}
	} else {
		result.Success = false
		result.Status = WANStatusDown
		result.PacketLoss = 1.0
		result.Error = fmt.Errorf("all pings failed")
	}

	result.Metadata["ping_count"] = c.config.PingCount
	result.Metadata["success_count"] = successCount
	result.Metadata["latencies"] = latencies

	return result, nil
}

// singlePing performs a single ICMP ping
func (c *PingChecker) singlePing(target net.IP) (time.Duration, error) {
	// Determine protocol (IPv4 or IPv6)
	var network string
	var protocol int
	if target.To4() != nil {
		network = "ip4:icmp"
		protocol = 1 // ICMP for IPv4
	} else {
		network = "ip6:ipv6-icmp"
		protocol = 58 // ICMPv6
	}

	// Create ICMP connection
	conn, err := icmp.ListenPacket(network, "")
	if err != nil {
		return 0, fmt.Errorf("failed to create ICMP connection: %w", err)
	}
	defer conn.Close()

	// Set deadline
	conn.SetDeadline(time.Now().Add(c.config.Timeout))

	// Create ICMP echo request
	msg := &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(c.config.WANID),
			Seq:  1,
			Data: make([]byte, c.config.PingSize),
		},
	}

	if target.To4() == nil {
		msg.Type = ipv6.ICMPTypeEchoRequest
	}

	// Marshal message
	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal ICMP message: %w", err)
	}

	// Send ping
	start := time.Now()
	_, err = conn.WriteTo(msgBytes, &net.IPAddr{IP: target})
	if err != nil {
		return 0, fmt.Errorf("failed to send ICMP echo: %w", err)
	}

	// Wait for reply
	reply := make([]byte, 1500)
	n, _, err := conn.ReadFrom(reply)
	if err != nil {
		return 0, fmt.Errorf("failed to receive ICMP reply: %w", err)
	}
	latency := time.Since(start)

	// Parse reply
	if target.To4() != nil {
		replyMsg, err := icmp.ParseMessage(protocol, reply[:n])
		if err != nil {
			return 0, fmt.Errorf("failed to parse ICMP reply: %w", err)
		}

		if replyMsg.Type != ipv4.ICMPTypeEchoReply {
			return 0, fmt.Errorf("unexpected ICMP message type: %v", replyMsg.Type)
		}
	} else {
		replyMsg, err := icmp.ParseMessage(protocol, reply[:n])
		if err != nil {
			return 0, fmt.Errorf("failed to parse ICMPv6 reply: %w", err)
		}

		if replyMsg.Type != ipv6.ICMPTypeEchoReply {
			return 0, fmt.Errorf("unexpected ICMPv6 message type: %v", replyMsg.Type)
		}
	}

	return latency, nil
}

// calculateJitter calculates the variance in latencies
func calculateJitter(latencies []time.Duration) time.Duration {
	if len(latencies) < 2 {
		return 0
	}

	// Calculate average
	var sum time.Duration
	for _, lat := range latencies {
		sum += lat
	}
	avg := sum / time.Duration(len(latencies))

	// Calculate variance
	var variance float64
	for _, lat := range latencies {
		diff := float64(lat - avg)
		variance += diff * diff
	}
	variance /= float64(len(latencies))

	// Standard deviation (jitter)
	return time.Duration(variance)
}

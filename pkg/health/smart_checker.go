package health

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// SmartChecker is an adaptive health checker that automatically selects the best method
type SmartChecker struct {
	config        *CheckConfig
	wanHealth     *WANHealth
	mu            sync.RWMutex
	currentTarget int
	lastCheck     time.Time

	// Method-specific checkers
	pingChecker *PingChecker
	httpChecker *HTTPChecker
	dnsChecker  *DNSChecker
	tcpChecker  *TCPChecker

	// For adaptive interval adjustment
	currentInterval time.Duration
	lastMethodEval  time.Time
}

// NewSmartChecker creates a new adaptive health checker
func NewSmartChecker(config *CheckConfig) *SmartChecker {
	sc := &SmartChecker{
		config:          config,
		wanHealth:       &WANHealth{
			WANID:             config.WANID,
			InterfaceName:     config.InterfaceName,
			Status:            WANStatusUnknown,
			MethodPerformance: make(map[CheckMethod]*MethodStats),
			LastResults:       make([]CheckResult, 0, 10),
			StateChanges:      make([]StateChange, 0, 10),
		},
		currentInterval: config.Interval,
		lastMethodEval:  time.Now(),
	}

	// Initialize method-specific checkers
	sc.pingChecker = NewPingChecker(config)
	sc.httpChecker = NewHTTPChecker(config)
	sc.dnsChecker = NewDNSChecker(config)
	sc.tcpChecker = NewTCPChecker(config)

	// Initialize method performance tracking
	methods := []CheckMethod{CheckMethodPing, CheckMethodHTTP, CheckMethodHTTPS, CheckMethodDNS, CheckMethodTCP}
	for _, method := range methods {
		sc.wanHealth.MethodPerformance[method] = &MethodStats{
			Method:      method,
			Reliability: 0.5, // Start with neutral reliability
		}
	}

	return sc
}

// Check performs a health check using the currently selected method
func (sc *SmartChecker) Check() (*CheckResult, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Update WAN health status to testing
	if sc.wanHealth.Status == WANStatusUnknown {
		sc.wanHealth.Status = WANStatusTesting
	}

	// Select method and target
	method := sc.selectMethod()
	target := sc.selectTarget()

	// Perform check based on method
	var result *CheckResult
	var err error

	switch method {
	case CheckMethodPing:
		result, err = sc.pingChecker.Check(target)
	case CheckMethodHTTP, CheckMethodHTTPS:
		result, err = sc.httpChecker.Check(target)
	case CheckMethodDNS:
		result, err = sc.dnsChecker.Check(target)
	case CheckMethodTCP:
		result, err = sc.tcpChecker.Check(target)
	default:
		// Default to ping
		result, err = sc.pingChecker.Check(target)
	}

	// Update statistics and health
	if result != nil {
		sc.updateStatistics(result)
		sc.updateWANHealth(result)
	}

	// Adjust interval if adaptive mode is enabled
	if sc.config.AdaptiveInterval {
		sc.adjustInterval(result)
	}

	// Reevaluate method selection periodically
	if sc.config.AutoMethodSelection && time.Since(sc.lastMethodEval) > sc.config.MethodSwitchInterval {
		sc.evaluateMethods()
		sc.lastMethodEval = time.Now()
	}

	sc.lastCheck = time.Now()

	return result, err
}

// selectMethod chooses the best health check method
func (sc *SmartChecker) selectMethod() CheckMethod {
	// If auto method selection is disabled, use configured method
	if !sc.config.AutoMethodSelection || sc.config.Method != CheckMethodAuto {
		return sc.config.Method
	}

	// Find method with highest reliability
	var bestMethod CheckMethod
	var bestReliability float64 = -1

	for method, stats := range sc.wanHealth.MethodPerformance {
		// Only consider methods that have been tried
		if stats.UsageCount > 0 && stats.Reliability > bestReliability {
			bestReliability = stats.Reliability
			bestMethod = method
		}
	}

	// If no method has been tried yet, start with ping
	if bestMethod == "" {
		// Try each method in round-robin fashion initially
		if sc.wanHealth.TotalChecks%5 == 0 {
			return CheckMethodPing
		} else if sc.wanHealth.TotalChecks%5 == 1 {
			return CheckMethodDNS
		} else if sc.wanHealth.TotalChecks%5 == 2 {
			return CheckMethodTCP
		} else if sc.wanHealth.TotalChecks%5 == 3 {
			return CheckMethodHTTP
		} else {
			return CheckMethodHTTPS
		}
	}

	// 10% exploration: try a random method occasionally
	if rand.Float64() < 0.1 {
		methods := []CheckMethod{CheckMethodPing, CheckMethodDNS, CheckMethodTCP, CheckMethodHTTP, CheckMethodHTTPS}
		return methods[rand.Intn(len(methods))]
	}

	return bestMethod
}

// selectTarget chooses the next target to check
func (sc *SmartChecker) selectTarget() string {
	if len(sc.config.Targets) == 0 {
		return "8.8.8.8" // Default fallback
	}

	if sc.config.TargetRotation {
		// Rotate through targets
		target := sc.config.Targets[sc.currentTarget]
		sc.currentTarget = (sc.currentTarget + 1) % len(sc.config.Targets)
		return target
	}

	// Use first target
	return sc.config.Targets[0]
}

// updateStatistics updates method performance statistics
func (sc *SmartChecker) updateStatistics(result *CheckResult) {
	stats := sc.wanHealth.MethodPerformance[result.Method]
	if stats == nil {
		stats = &MethodStats{Method: result.Method}
		sc.wanHealth.MethodPerformance[result.Method] = stats
	}

	stats.UsageCount++
	stats.LastUsed = result.Timestamp

	if result.Success {
		stats.SuccessCount++

		// Update average latency (exponential moving average)
		if stats.AvgLatency == 0 {
			stats.AvgLatency = result.Latency
		} else {
			// Weight: 80% old, 20% new
			stats.AvgLatency = time.Duration(float64(stats.AvgLatency)*0.8 + float64(result.Latency)*0.2)
		}
	} else {
		stats.FailureCount++
	}

	// Calculate success rate
	stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.UsageCount)

	// Calculate reliability (combines success rate and latency)
	// Lower latency = higher reliability
	latencyFactor := 1.0
	if stats.AvgLatency > 0 {
		// Normalize latency (1.0 = excellent <50ms, 0.0 = poor >500ms)
		latencyMs := float64(stats.AvgLatency.Milliseconds())
		latencyFactor = 1.0 - (latencyMs / 500.0)
		if latencyFactor < 0 {
			latencyFactor = 0
		}
	}

	// Reliability = 70% success rate + 30% latency factor
	stats.Reliability = stats.SuccessRate*0.7 + latencyFactor*0.3
}

// updateWANHealth updates the overall WAN health status
func (sc *SmartChecker) updateWANHealth(result *CheckResult) {
	wh := sc.wanHealth

	// Update counters
	wh.TotalChecks++
	wh.LastCheck = result.Timestamp

	if result.Success {
		wh.TotalSuccesses++
		wh.ConsecutiveSuccesses++
		wh.ConsecutiveFailures = 0
		wh.LastSuccess = result.Timestamp

		// Update latency statistics
		if wh.AvgLatency == 0 {
			wh.AvgLatency = result.Latency
		} else {
			wh.AvgLatency = time.Duration(float64(wh.AvgLatency)*0.9 + float64(result.Latency)*0.1)
		}

		if wh.MinLatency == 0 || result.Latency < wh.MinLatency {
			wh.MinLatency = result.Latency
		}

		if result.Latency > wh.MaxLatency {
			wh.MaxLatency = result.Latency
		}

		// Update jitter
		if result.Jitter > 0 {
			if wh.AvgJitter == 0 {
				wh.AvgJitter = result.Jitter
			} else {
				wh.AvgJitter = time.Duration(float64(wh.AvgJitter)*0.9 + float64(result.Jitter)*0.1)
			}
		}

		// Update packet loss
		if wh.AvgPacketLoss == 0 {
			wh.AvgPacketLoss = result.PacketLoss
		} else {
			wh.AvgPacketLoss = wh.AvgPacketLoss*0.9 + result.PacketLoss*0.1
		}
	} else {
		wh.TotalFailures++
		wh.ConsecutiveFailures++
		wh.ConsecutiveSuccesses = 0
		wh.LastFailure = result.Timestamp
	}

	// Calculate uptime
	if wh.TotalChecks > 0 {
		wh.Uptime = float64(wh.TotalSuccesses) / float64(wh.TotalChecks)
	}

	// Update current method
	wh.CurrentMethod = result.Method

	// Determine status change
	oldStatus := wh.Status
	newStatus := sc.determineStatus()

	if oldStatus != newStatus {
		// Record state change
		change := StateChange{
			Timestamp:   result.Timestamp,
			FromStatus:  oldStatus,
			ToStatus:    newStatus,
			Reason:      fmt.Sprintf("Consecutive successes: %d, failures: %d", wh.ConsecutiveSuccesses, wh.ConsecutiveFailures),
			CheckResult: result,
		}
		wh.StateChanges = append(wh.StateChanges, change)

		// Keep only last 100 state changes
		if len(wh.StateChanges) > 100 {
			wh.StateChanges = wh.StateChanges[1:]
		}

		wh.Status = newStatus
	}

	// Add to recent results (keep last 10)
	wh.LastResults = append(wh.LastResults, *result)
	if len(wh.LastResults) > 10 {
		wh.LastResults = wh.LastResults[1:]
	}
}

// determineStatus determines the WAN status based on check results
func (sc *SmartChecker) determineStatus() WANStatus {
	wh := sc.wanHealth

	// Need minimum checks before making determination
	if wh.TotalChecks < uint64(sc.config.RetryCount) {
		return WANStatusTesting
	}

	// Down if consecutive failures exceed threshold
	if wh.ConsecutiveFailures >= sc.config.RetryCount {
		return WANStatusDown
	}

	// Up if consecutive successes exceed threshold
	if wh.ConsecutiveSuccesses >= sc.config.RetryCount {
		// Check if degraded based on performance
		if wh.AvgLatency > sc.config.DegradedLatency || wh.AvgPacketLoss > sc.config.DegradedPacketLoss {
			return WANStatusDegraded
		}
		return WANStatusUp
	}

	// In transition - use recent success rate
	recentSuccesses := 0
	recentCount := len(wh.LastResults)
	for _, result := range wh.LastResults {
		if result.Success {
			recentSuccesses++
		}
	}

	if recentCount > 0 {
		successRate := float64(recentSuccesses) / float64(recentCount)
		if successRate >= 0.7 {
			return WANStatusUp
		} else if successRate >= 0.3 {
			return WANStatusDegraded
		}
	}

	return WANStatusDown
}

// adjustInterval adjusts the check interval based on results
func (sc *SmartChecker) adjustInterval(result *CheckResult) {
	if result.Success {
		// On success, gradually increase interval (less frequent checks)
		sc.currentInterval = time.Duration(float64(sc.currentInterval) * sc.config.SuccessSpeedup)
		if sc.currentInterval > sc.config.MaxInterval {
			sc.currentInterval = sc.config.MaxInterval
		}
	} else {
		// On failure, decrease interval (more frequent checks)
		sc.currentInterval = time.Duration(float64(sc.currentInterval) * sc.config.FailureBackoff)
		if sc.currentInterval < sc.config.MinInterval {
			sc.currentInterval = sc.config.MinInterval
		}
	}
}

// evaluateMethods periodically evaluates and potentially switches methods
func (sc *SmartChecker) evaluateMethods() {
	// This is where ML-based method selection would go
	// For now, we use simple heuristics based on reliability scores

	// Find the most reliable method
	var bestMethod CheckMethod
	var bestReliability float64 = -1

	for method, stats := range sc.wanHealth.MethodPerformance {
		if stats.UsageCount >= 5 && stats.Reliability > bestReliability {
			bestReliability = stats.Reliability
			bestMethod = method
		}
	}

	// If current method is significantly worse, consider switching
	currentStats := sc.wanHealth.MethodPerformance[sc.wanHealth.CurrentMethod]
	if currentStats != nil && bestReliability > currentStats.Reliability*1.2 {
		// Switch to better method
		sc.config.Method = bestMethod
	}
}

// GetHealth returns the current WAN health status
func (sc *SmartChecker) GetHealth() *WANHealth {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Return a copy to avoid race conditions
	healthCopy := *sc.wanHealth
	return &healthCopy
}

// GetCurrentInterval returns the current check interval
func (sc *SmartChecker) GetCurrentInterval() time.Duration {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.currentInterval
}

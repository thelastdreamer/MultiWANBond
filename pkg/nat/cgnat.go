package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// CGNATDetector detects and handles CGNAT scenarios
type CGNATDetector struct {
	config *CGNATConfig
	mu     sync.RWMutex

	// Detection results
	detected      bool
	detectedTime  time.Time
	publicIP      net.IP
	localIP       net.IP

	// CGNAT characteristics
	portAllocation string // "sequential", "random", "block"
	portLifetime   time.Duration
	sharedIP       bool

	// Stats
	detectionAttempts uint64
	detectionCount    uint64
}

// NewCGNATDetector creates a new CGNAT detector
func NewCGNATDetector(config *CGNATConfig) *CGNATDetector {
	if config == nil {
		config = DefaultCGNATConfig()
	}

	return &CGNATDetector{
		config: config,
	}
}

// DetectCGNAT detects if we're behind CGNAT
func (cd *CGNATDetector) DetectCGNAT(localAddr, publicAddr *net.UDPAddr) (*CGNATInfo, error) {
	cd.mu.Lock()
	defer cd.mu.Unlock()

	cd.detectionAttempts++

	if localAddr == nil || publicAddr == nil {
		return nil, fmt.Errorf("invalid addresses")
	}

	cd.localIP = localAddr.IP
	cd.publicIP = publicAddr.IP

	info := &CGNATInfo{
		LocalAddr:  localAddr,
		PublicAddr: publicAddr,
		Detected:   false,
	}

	// Check 1: Is local IP in CGNAT range (RFC 6598: 100.64.0.0/10)?
	if cd.config.IsCGNATAddress(localAddr.IP) {
		cd.detected = true
		cd.detectedTime = time.Now()
		cd.detectionCount++

		info.Detected = true
		info.Type = CGNATTypeRFC6598
		info.Confidence = 1.0 // 100% confidence
		return info, nil
	}

	// Check 2: Is local IP in private range but public IP shared?
	if IsPrivateAddress(localAddr.IP) {
		// Check for CGNAT indicators
		indicators := cd.detectCGNATIndicators(localAddr, publicAddr)

		if indicators.SharedIPLikely {
			cd.detected = true
			cd.detectedTime = time.Now()
			cd.detectionCount++
			cd.sharedIP = true

			info.Detected = true
			info.Type = CGNATTypeInferred
			info.Confidence = indicators.Confidence
			info.Indicators = indicators
			return info, nil
		}
	}

	return info, nil
}

// detectCGNATIndicators detects various CGNAT indicators
func (cd *CGNATDetector) detectCGNATIndicators(localAddr, publicAddr *net.UDPAddr) *CGNATIndicators {
	indicators := &CGNATIndicators{
		SharedIPLikely: false,
		Confidence:     0.0,
	}

	score := 0.0

	// Indicator 1: High port number mapping (CGNAT typically uses high ports)
	if publicAddr.Port > 40000 {
		indicators.HighPortMapping = true
		score += 0.3
	}

	// Indicator 2: Port not close to local port (CGNAT often changes ports significantly)
	portDiff := publicAddr.Port - localAddr.Port
	if portDiff < 0 {
		portDiff = -portDiff
	}
	if portDiff > 10000 {
		indicators.PortTranslation = true
		score += 0.2
	}

	// Indicator 3: Check if public IP is in known ISP CGNAT ranges
	// Common ISP CGNAT public IPs are often in certain ranges
	if cd.isKnownCGNATPublicIP(publicAddr.IP) {
		indicators.KnownCGNATIP = true
		score += 0.4
	}

	// Indicator 4: Multiple mappings to same public IP (would need history)
	// This is a heuristic and would need tracking over time
	// For now, we'll mark it as potential if other indicators are present
	if score > 0.3 {
		indicators.SharedIPLikely = true
	}

	indicators.Confidence = score
	if indicators.Confidence > 1.0 {
		indicators.Confidence = 1.0
	}

	return indicators
}

// isKnownCGNATPublicIP checks if IP is in known CGNAT public ranges
func (cd *CGNATDetector) isKnownCGNATPublicIP(ip net.IP) bool {
	// This is a heuristic - in practice you'd maintain a database
	// of known ISP CGNAT public IP ranges

	// Some mobile carriers use specific ranges
	// Example: Check if in certain /16 or /20 blocks
	// This would be populated from real-world data

	return false // Placeholder
}

// DetectPortAllocationPattern detects how CGNAT allocates ports
func (cd *CGNATDetector) DetectPortAllocationPattern(measurements []PortMeasurement) string {
	if len(measurements) < 3 {
		return "unknown"
	}

	// Check if sequential
	sequential := true
	for i := 1; i < len(measurements); i++ {
		diff := measurements[i].Port - measurements[i-1].Port
		if diff < 0 {
			diff = -diff
		}
		if diff > 10 { // Allow small gaps
			sequential = false
			break
		}
	}

	if sequential {
		cd.mu.Lock()
		cd.portAllocation = "sequential"
		cd.mu.Unlock()
		return "sequential"
	}

	// Check if block allocation (ports in groups)
	blockSize := 64 // Common CGNAT block size
	inSameBlock := true
	firstBlock := measurements[0].Port / blockSize
	for i := 1; i < len(measurements); i++ {
		block := measurements[i].Port / blockSize
		if block != firstBlock {
			inSameBlock = false
			break
		}
	}

	if inSameBlock {
		cd.mu.Lock()
		cd.portAllocation = "block"
		cd.mu.Unlock()
		return "block"
	}

	cd.mu.Lock()
	cd.portAllocation = "random"
	cd.mu.Unlock()
	return "random"
}

// EstimatePortLifetime estimates NAT mapping lifetime behind CGNAT
func (cd *CGNATDetector) EstimatePortLifetime(keepAliveResults []KeepAliveResult) time.Duration {
	if len(keepAliveResults) == 0 {
		return 30 * time.Second // Default conservative estimate
	}

	// Find the maximum successful interval
	maxLifetime := time.Duration(0)
	for _, result := range keepAliveResults {
		if result.Success {
			if result.Interval > maxLifetime {
				maxLifetime = result.Interval
			}
		}
	}

	// Add 20% safety margin
	lifetime := time.Duration(float64(maxLifetime) * 0.8)

	cd.mu.Lock()
	cd.portLifetime = lifetime
	cd.mu.Unlock()

	return lifetime
}

// GetRecommendedStrategy returns recommended NAT traversal strategy for CGNAT
func (cd *CGNATDetector) GetRecommendedStrategy() TraversalStrategy {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	if !cd.detected {
		// Standard strategy for non-CGNAT
		return TraversalStrategy{
			PreferRelay:       false,
			AggressivePunch:   false,
			KeepAliveInterval: 30 * time.Second,
		}
	}

	strategy := TraversalStrategy{
		PreferRelay:     cd.config.ForceRelay,
		AggressivePunch: cd.config.AggressivePunch,
		KeepAliveInterval: 15 * time.Second, // More frequent for CGNAT
	}

	// Adjust based on CGNAT characteristics
	switch cd.portAllocation {
	case "sequential":
		strategy.PredictPorts = true
		strategy.PortPredictionRange = 20
	case "block":
		strategy.PredictPorts = true
		strategy.PortPredictionRange = 64
	case "random":
		strategy.PreferRelay = true // Hard to predict
	}

	// Adjust keep-alive based on port lifetime
	if cd.portLifetime > 0 {
		// Keep alive at 50% of lifetime
		strategy.KeepAliveInterval = cd.portLifetime / 2
		if strategy.KeepAliveInterval < 5*time.Second {
			strategy.KeepAliveInterval = 5 * time.Second
		}
	}

	return strategy
}

// IsCGNATDetected returns whether CGNAT was detected
func (cd *CGNATDetector) IsCGNATDetected() bool {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.detected
}

// GetCGNATInfo returns detailed CGNAT information
func (cd *CGNATDetector) GetCGNATInfo() *CGNATInfo {
	cd.mu.RLock()
	defer cd.mu.RUnlock()

	if !cd.detected {
		return nil
	}

	info := &CGNATInfo{
		Detected:       true,
		LocalAddr:      &net.UDPAddr{IP: cd.localIP},
		PublicAddr:     &net.UDPAddr{IP: cd.publicIP},
		DetectedAt:     cd.detectedTime,
		PortAllocation: cd.portAllocation,
		PortLifetime:   cd.portLifetime,
		SharedIP:       cd.sharedIP,
	}

	if cd.config.IsCGNATAddress(cd.localIP) {
		info.Type = CGNATTypeRFC6598
		info.Confidence = 1.0
	} else {
		info.Type = CGNATTypeInferred
	}

	return info
}

// GetStats returns CGNAT detector statistics
func (cd *CGNATDetector) GetStats() (attempts, detected uint64) {
	cd.mu.RLock()
	defer cd.mu.RUnlock()
	return cd.detectionAttempts, cd.detectionCount
}

// CGNATInfo contains information about detected CGNAT
type CGNATInfo struct {
	Detected       bool
	Type           CGNATType
	Confidence     float64
	LocalAddr      *net.UDPAddr
	PublicAddr     *net.UDPAddr
	DetectedAt     time.Time
	PortAllocation string
	PortLifetime   time.Duration
	SharedIP       bool
	Indicators     *CGNATIndicators
}

// CGNATType represents the type of CGNAT detection
type CGNATType int

const (
	CGNATTypeNone CGNATType = iota
	CGNATTypeRFC6598   // Detected via RFC 6598 IP range
	CGNATTypeInferred  // Inferred from behavior
)

// String returns string representation
func (t CGNATType) String() string {
	switch t {
	case CGNATTypeNone:
		return "None"
	case CGNATTypeRFC6598:
		return "RFC6598 (100.64.0.0/10)"
	case CGNATTypeInferred:
		return "Inferred"
	default:
		return "Unknown"
	}
}

// CGNATIndicators contains indicators of CGNAT
type CGNATIndicators struct {
	HighPortMapping bool
	PortTranslation bool
	KnownCGNATIP    bool
	SharedIPLikely  bool
	Confidence      float64
}

// PortMeasurement represents a port mapping measurement
type PortMeasurement struct {
	Port      int
	Timestamp time.Time
}

// KeepAliveResult represents result of a keep-alive test
type KeepAliveResult struct {
	Interval time.Duration
	Success  bool
}

// TraversalStrategy contains recommended NAT traversal strategy
type TraversalStrategy struct {
	PreferRelay          bool
	AggressivePunch      bool
	PredictPorts         bool
	PortPredictionRange  int
	KeepAliveInterval    time.Duration
}

// AdaptiveKeepAlive manages adaptive keep-alive for CGNAT
type AdaptiveKeepAlive struct {
	mu                sync.RWMutex
	currentInterval   time.Duration
	minInterval       time.Duration
	maxInterval       time.Duration
	successfulTests   int
	failedTests       int
	lastAdjustment    time.Time
}

// NewAdaptiveKeepAlive creates adaptive keep-alive manager
func NewAdaptiveKeepAlive(initialInterval time.Duration) *AdaptiveKeepAlive {
	return &AdaptiveKeepAlive{
		currentInterval: initialInterval,
		minInterval:     5 * time.Second,
		maxInterval:     60 * time.Second,
		lastAdjustment:  time.Now(),
	}
}

// RecordSuccess records a successful keep-alive
func (aka *AdaptiveKeepAlive) RecordSuccess() {
	aka.mu.Lock()
	defer aka.mu.Unlock()

	aka.successfulTests++

	// After 10 successful tests, try increasing interval
	if aka.successfulTests >= 10 && time.Since(aka.lastAdjustment) > 5*time.Minute {
		newInterval := time.Duration(float64(aka.currentInterval) * 1.5)
		if newInterval <= aka.maxInterval {
			aka.currentInterval = newInterval
			aka.successfulTests = 0
			aka.lastAdjustment = time.Now()
		}
	}
}

// RecordFailure records a failed keep-alive
func (aka *AdaptiveKeepAlive) RecordFailure() {
	aka.mu.Lock()
	defer aka.mu.Unlock()

	aka.failedTests++

	// Immediately decrease interval on failure
	newInterval := time.Duration(float64(aka.currentInterval) * 0.5)
	if newInterval >= aka.minInterval {
		aka.currentInterval = newInterval
		aka.failedTests = 0
		aka.successfulTests = 0
		aka.lastAdjustment = time.Now()
	}
}

// GetInterval returns current keep-alive interval
func (aka *AdaptiveKeepAlive) GetInterval() time.Duration {
	aka.mu.RLock()
	defer aka.mu.RUnlock()
	return aka.currentInterval
}

// Reset resets the adaptive keep-alive state
func (aka *AdaptiveKeepAlive) Reset() {
	aka.mu.Lock()
	defer aka.mu.Unlock()

	aka.successfulTests = 0
	aka.failedTests = 0
	aka.lastAdjustment = time.Now()
}

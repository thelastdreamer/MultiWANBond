package health

import (
	"time"
)

// CheckMethod defines the health check method
type CheckMethod string

const (
	// CheckMethodPing uses ICMP ping
	CheckMethodPing CheckMethod = "ping"

	// CheckMethodHTTP uses HTTP GET request
	CheckMethodHTTP CheckMethod = "http"

	// CheckMethodHTTPS uses HTTPS GET request
	CheckMethodHTTPS CheckMethod = "https"

	// CheckMethodDNS uses DNS query
	CheckMethodDNS CheckMethod = "dns"

	// CheckMethodTCP uses TCP connection
	CheckMethodTCP CheckMethod = "tcp"

	// CheckMethodAuto automatically selects the best method
	CheckMethodAuto CheckMethod = "auto"
)

// WANStatus defines the status of a WAN interface
type WANStatus string

const (
	// WANStatusUp WAN is healthy and operational
	WANStatusUp WANStatus = "up"

	// WANStatusDown WAN is down or unreachable
	WANStatusDown WANStatus = "down"

	// WANStatusDegraded WAN is up but experiencing issues
	WANStatusDegraded WANStatus = "degraded"

	// WANStatusUnknown WAN status is unknown (not yet checked)
	WANStatusUnknown WANStatus = "unknown"

	// WANStatusTesting WAN is being tested
	WANStatusTesting WANStatus = "testing"
)

// CheckConfig represents health check configuration for a WAN
type CheckConfig struct {
	// WANID identifies the WAN interface
	WANID uint8

	// InterfaceName of the WAN
	InterfaceName string

	// Method to use for health checking
	Method CheckMethod

	// Interval between health checks
	Interval time.Duration

	// Timeout for each health check
	Timeout time.Duration

	// RetryCount number of retries before marking as down
	RetryCount int

	// RetryInterval time between retries
	RetryInterval time.Duration

	// Targets is a list of targets to check (IPs, URLs, domains)
	Targets []string

	// TargetRotation rotates through targets on each check
	TargetRotation bool

	// Ping-specific settings
	PingCount      int  // Number of pings per check
	PingSize       int  // Size of ping packets in bytes
	PingDontFrag   bool // Don't fragment flag for MTU detection
	PingSourceAddr string

	// HTTP-specific settings
	HTTPMethod          string            // GET, POST, etc.
	HTTPPath            string            // Path to request
	HTTPHeaders         map[string]string // Custom headers
	HTTPExpectedStatus  int               // Expected HTTP status code
	HTTPExpectedBody    string            // Expected body substring
	HTTPFollowRedirects bool
	HTTPInsecureTLS     bool // Skip TLS verification

	// DNS-specific settings
	DNSQueryType   string // A, AAAA, MX, TXT, etc.
	DNSQueryDomain string // Domain to query
	DNSServer      string // DNS server to query (if empty, uses system default)
	DNSExpectedIP  string // Expected IP in response

	// TCP-specific settings
	TCPPort int    // Port to connect to
	TCPSend string // Data to send after connection
	TCPExpect string // Expected response

	// Thresholds
	LatencyThreshold      time.Duration // Max acceptable latency
	JitterThreshold       time.Duration // Max acceptable jitter
	PacketLossThreshold   float64       // Max acceptable packet loss (0.0-1.0)
	DegradedLatency       time.Duration // Latency for degraded status
	DegradedPacketLoss    float64       // Packet loss for degraded status

	// Adaptive settings
	AdaptiveInterval     bool          // Automatically adjust check interval
	MinInterval          time.Duration // Minimum check interval
	MaxInterval          time.Duration // Maximum check interval
	FailureBackoff       float64       // Backoff multiplier on failure
	SuccessSpeedup       float64       // Speedup multiplier on success

	// ML-based method selection
	AutoMethodSelection  bool // Enable automatic method selection
	MethodSwitchInterval time.Duration // How often to reevaluate method

	// Notification settings
	NotifyOnUp        bool // Notify when WAN comes up
	NotifyOnDown      bool // Notify when WAN goes down
	NotifyOnDegraded  bool // Notify when WAN becomes degraded
}

// CheckResult represents the result of a single health check
type CheckResult struct {
	// WANID that was checked
	WANID uint8

	// Timestamp of the check
	Timestamp time.Time

	// Success indicates if the check passed
	Success bool

	// Status of the WAN after this check
	Status WANStatus

	// Method used for this check
	Method CheckMethod

	// Target that was checked
	Target string

	// Latency measured (RTT)
	Latency time.Duration

	// PacketLoss percentage (0.0-1.0)
	PacketLoss float64

	// Jitter measured
	Jitter time.Duration

	// Error if check failed
	Error error

	// HTTPStatusCode for HTTP checks
	HTTPStatusCode int

	// DNSResolveTime for DNS checks
	DNSResolveTime time.Duration

	// DNSAnswers for DNS checks
	DNSAnswers []string

	// TCPConnectTime for TCP checks
	TCPConnectTime time.Duration

	// Metadata for additional information
	Metadata map[string]interface{}
}

// WANHealth represents the health status of a WAN interface
type WANHealth struct {
	// WANID identifies the WAN
	WANID uint8

	// InterfaceName
	InterfaceName string

	// Status current status
	Status WANStatus

	// LastCheck timestamp
	LastCheck time.Time

	// LastSuccess timestamp
	LastSuccess time.Time

	// LastFailure timestamp
	LastFailure time.Time

	// ConsecutiveSuccesses count
	ConsecutiveSuccesses int

	// ConsecutiveFailures count
	ConsecutiveFailures int

	// TotalChecks performed
	TotalChecks uint64

	// TotalSuccesses count
	TotalSuccesses uint64

	// TotalFailures count
	TotalFailures uint64

	// AvgLatency average latency over recent checks
	AvgLatency time.Duration

	// MinLatency minimum latency observed
	MinLatency time.Duration

	// MaxLatency maximum latency observed
	MaxLatency time.Duration

	// AvgJitter average jitter
	AvgJitter time.Duration

	// AvgPacketLoss average packet loss
	AvgPacketLoss float64

	// Uptime percentage (0.0-1.0)
	Uptime float64

	// CurrentMethod being used
	CurrentMethod CheckMethod

	// MethodPerformance tracks performance of each method
	MethodPerformance map[CheckMethod]*MethodStats

	// LastResults recent check results (sliding window)
	LastResults []CheckResult

	// StateChanges history of status changes
	StateChanges []StateChange
}

// MethodStats represents performance statistics for a check method
type MethodStats struct {
	// Method name
	Method CheckMethod

	// UsageCount how many times this method was used
	UsageCount uint64

	// SuccessCount
	SuccessCount uint64

	// FailureCount
	FailureCount uint64

	// AvgLatency for this method
	AvgLatency time.Duration

	// SuccessRate (0.0-1.0)
	SuccessRate float64

	// Reliability score (0.0-1.0)
	Reliability float64

	// LastUsed timestamp
	LastUsed time.Time
}

// StateChange represents a change in WAN status
type StateChange struct {
	// Timestamp of the change
	Timestamp time.Time

	// FromStatus previous status
	FromStatus WANStatus

	// ToStatus new status
	ToStatus WANStatus

	// Reason for the change
	Reason string

	// CheckResult that triggered the change
	CheckResult *CheckResult
}

// DefaultCheckConfig returns a default health check configuration
func DefaultCheckConfig(wanID uint8, interfaceName string) *CheckConfig {
	return &CheckConfig{
		WANID:                wanID,
		InterfaceName:        interfaceName,
		Method:               CheckMethodAuto,
		Interval:             200 * time.Millisecond, // Sub-second checking
		Timeout:              5 * time.Second,
		RetryCount:           3,
		RetryInterval:        100 * time.Millisecond,
		Targets:              []string{"8.8.8.8", "1.1.1.1", "8.8.4.4"},
		TargetRotation:       true,
		PingCount:            3,
		PingSize:             56,
		PingDontFrag:         false,
		HTTPMethod:           "GET",
		HTTPPath:             "/",
		HTTPExpectedStatus:   200,
		HTTPFollowRedirects:  true,
		HTTPInsecureTLS:      false,
		DNSQueryType:         "A",
		DNSQueryDomain:       "google.com",
		TCPPort:              443,
		LatencyThreshold:     500 * time.Millisecond,
		JitterThreshold:      100 * time.Millisecond,
		PacketLossThreshold:  0.2, // 20%
		DegradedLatency:      200 * time.Millisecond,
		DegradedPacketLoss:   0.05, // 5%
		AdaptiveInterval:     true,
		MinInterval:          100 * time.Millisecond,
		MaxInterval:          5 * time.Second,
		FailureBackoff:       1.5,
		SuccessSpeedup:       0.9,
		AutoMethodSelection:  true,
		MethodSwitchInterval: 60 * time.Second,
		NotifyOnUp:           true,
		NotifyOnDown:         true,
		NotifyOnDegraded:     true,
	}
}

// PingCheckConfig returns a configuration optimized for ping-based checking
func PingCheckConfig(wanID uint8, interfaceName string) *CheckConfig {
	config := DefaultCheckConfig(wanID, interfaceName)
	config.Method = CheckMethodPing
	config.PingCount = 5
	config.PingSize = 56
	config.Interval = 200 * time.Millisecond
	return config
}

// HTTPCheckConfig returns a configuration optimized for HTTP-based checking
func HTTPCheckConfig(wanID uint8, interfaceName string, url string) *CheckConfig {
	config := DefaultCheckConfig(wanID, interfaceName)
	config.Method = CheckMethodHTTPS
	config.Targets = []string{url}
	config.HTTPExpectedStatus = 200
	config.Interval = 1 * time.Second
	return config
}

// DNSCheckConfig returns a configuration optimized for DNS-based checking
func DNSCheckConfig(wanID uint8, interfaceName string) *CheckConfig {
	config := DefaultCheckConfig(wanID, interfaceName)
	config.Method = CheckMethodDNS
	config.DNSQueryType = "A"
	config.DNSQueryDomain = "google.com"
	config.Targets = []string{"8.8.8.8", "1.1.1.1"}
	config.Interval = 500 * time.Millisecond
	return config
}

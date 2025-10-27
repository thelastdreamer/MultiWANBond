package server

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ClientSession represents a connected client session
type ClientSession struct {
	ID              string                     // Unique session ID
	ClientID        string                     // Client identifier (username, device ID, etc.)
	RemoteAddr      *net.UDPAddr               // Client's remote address
	PublicIP        net.IP                     // Assigned public IP (from NAT pool)
	PrivateIP       net.IP                     // Client's private IP
	WANInterfaces   map[uint8]*ClientWANState  // Per-WAN state for this client
	NATMappings     *NATTable                  // NAT port mappings for this client
	BandwidthQuota  *BandwidthQuota            // Bandwidth limits and usage
	StartTime       time.Time                  // Session start time
	LastSeen        time.Time                  // Last packet from client
	BytesSent       uint64                     // Total bytes sent to client
	BytesReceived   uint64                     // Total bytes received from client
	PacketsSent     uint64                     // Total packets sent
	PacketsReceived uint64                     // Total packets received
	Config          *ClientConfig              // Client-specific configuration
	State           ClientState                // Current session state
	Metadata        map[string]interface{}     // Custom metadata
	mu              sync.RWMutex               // Protects session data
}

// ClientWANState tracks per-WAN state for a client
type ClientWANState struct {
	WANID           uint8
	Active          bool
	BytesSent       uint64
	BytesReceived   uint64
	PacketsSent     uint64
	PacketsReceived uint64
	LastUsed        time.Time
}

// ClientState represents the state of a client session
type ClientState string

const (
	ClientStateConnecting   ClientState = "connecting"   // Initial connection
	ClientStateAuthenticated ClientState = "authenticated" // Auth completed
	ClientStateActive       ClientState = "active"       // Fully active
	ClientStateIdle         ClientState = "idle"         // No traffic for a while
	ClientStateSuspended    ClientState = "suspended"    // Temporarily suspended
	ClientStateDisconnected ClientState = "disconnected" // Disconnected
)

// ClientConfig contains per-client configuration
type ClientConfig struct {
	// Bandwidth limits
	MaxUploadBandwidth   uint64 // Bytes per second
	MaxDownloadBandwidth uint64 // Bytes per second

	// Quotas
	DailyDataQuota   uint64 // Total daily data limit (bytes)
	MonthlyDataQuota uint64 // Total monthly data limit (bytes)

	// Priority
	Priority int // Higher = more priority in congestion

	// Features
	AllowedWANs         []uint8 // Which WANs this client can use (empty = all)
	EnableInterClient   bool    // Can communicate with other clients
	EnablePortForwarding bool   // Can create port forwards

	// Timeouts
	IdleTimeout       time.Duration // Disconnect after idle
	SessionTimeout    time.Duration // Max session duration

	// NAT settings
	NATPoolStart net.IP // Start of NAT IP range for this client
	NATPoolEnd   net.IP // End of NAT IP range
}

// BandwidthQuota tracks bandwidth usage and limits
type BandwidthQuota struct {
	mu                sync.RWMutex
	MaxUpload         uint64 // Bytes per second
	MaxDownload       uint64 // Bytes per second
	CurrentUpload     uint64 // Current upload rate
	CurrentDownload   uint64 // Current download rate
	TotalUploaded     uint64 // Total uploaded this session
	TotalDownloaded   uint64 // Total downloaded this session
	DailyUploaded     uint64 // Uploaded today
	DailyDownloaded   uint64 // Downloaded today
	MonthlyUploaded   uint64 // Uploaded this month
	MonthlyDownloaded uint64 // Downloaded this month
	LastReset         time.Time
	QuotaExceeded     bool
}

// NATTable manages NAT port mappings for a client
type NATTable struct {
	mu       sync.RWMutex
	mappings map[string]*NATMapping // Key: "srcIP:srcPort:protocol"
}

// NATMapping represents a single NAT translation
type NATMapping struct {
	SourceIP      net.IP
	SourcePort    uint16
	PublicIP      net.IP
	PublicPort    uint16
	DestIP        net.IP
	DestPort      uint16
	Protocol      uint8 // TCP=6, UDP=17
	Created       time.Time
	LastUsed      time.Time
	BytesForward  uint64
	BytesReverse  uint64
	PacketsForward uint64
	PacketsReverse uint64
}

// ServerConfig contains server-wide configuration
type ServerConfig struct {
	// Listen settings
	ListenAddr string // Address to listen on
	ListenPort int    // Port to listen on

	// Connection limits
	MaxClients           int           // Maximum simultaneous clients
	MaxClientsPerIP      int           // Max clients from same IP
	MaxSessionsPerClient int           // Max sessions per client ID

	// NAT pool
	NATPoolStart net.IP // Start of NAT IP pool
	NATPoolSize  int    // Number of IPs in pool

	// Timeouts
	ClientIdleTimeout    time.Duration
	SessionTimeout       time.Duration
	HandshakeTimeout     time.Duration

	// Bandwidth
	TotalUploadBandwidth   uint64 // Total server upload capacity
	TotalDownloadBandwidth uint64 // Total server download capacity

	// Default client config
	DefaultClientConfig *ClientConfig

	// Inter-client communication
	AllowInterClient bool // Allow clients to communicate with each other
	InterClientSubnet string // Subnet for inter-client communication

	// Security
	RequireAuthentication bool
	AllowedCIDRs          []string // Allowed client IP ranges
	BlockedIPs            []string // Blocked IPs

	// Performance
	WorkerThreads     int
	BufferSize        int
	SendQueueSize     int
	ReceiveQueueSize  int
}

// ServerStats contains server statistics
type ServerStats struct {
	StartTime           time.Time
	TotalClients        uint64
	ActiveClients       int
	TotalSessions       uint64
	ActiveSessions      int
	TotalBytesForwarded uint64
	TotalPacketsForwarded uint64
	BytesPerSecond      uint64
	PacketsPerSecond    uint64
	WANUtilization      map[uint8]float64 // WANID -> utilization %
}

// SessionEvent represents a session lifecycle event
type SessionEvent struct {
	Type      SessionEventType
	SessionID string
	ClientID  string
	Timestamp time.Time
	Details   string
}

// SessionEventType defines types of session events
type SessionEventType string

const (
	EventSessionCreated      SessionEventType = "session_created"
	EventSessionAuthenticated SessionEventType = "session_authenticated"
	EventSessionActive       SessionEventType = "session_active"
	EventSessionIdle         SessionEventType = "session_idle"
	EventSessionSuspended    SessionEventType = "session_suspended"
	EventSessionDisconnected SessionEventType = "session_disconnected"
	EventQuotaExceeded       SessionEventType = "quota_exceeded"
	EventBandwidthLimitHit   SessionEventType = "bandwidth_limit"
)

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ListenAddr:               "0.0.0.0",
		ListenPort:               8888,
		MaxClients:               1000,
		MaxClientsPerIP:          10,
		MaxSessionsPerClient:     5,
		NATPoolStart:             net.ParseIP("10.100.0.1"),
		NATPoolSize:              254,
		ClientIdleTimeout:        5 * time.Minute,
		SessionTimeout:           24 * time.Hour,
		HandshakeTimeout:         30 * time.Second,
		TotalUploadBandwidth:     1000 * 1024 * 1024, // 1 Gbps
		TotalDownloadBandwidth:   1000 * 1024 * 1024, // 1 Gbps
		AllowInterClient:         true,
		InterClientSubnet:        "10.100.0.0/16",
		RequireAuthentication:    true,
		WorkerThreads:            4,
		BufferSize:               65536,
		SendQueueSize:            1000,
		ReceiveQueueSize:         1000,
		DefaultClientConfig:      DefaultClientConfig(),
	}
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		MaxUploadBandwidth:    100 * 1024 * 1024, // 100 Mbps
		MaxDownloadBandwidth:  100 * 1024 * 1024, // 100 Mbps
		DailyDataQuota:        10 * 1024 * 1024 * 1024, // 10 GB
		MonthlyDataQuota:      100 * 1024 * 1024 * 1024, // 100 GB
		Priority:              50, // Medium priority
		AllowedWANs:           []uint8{}, // All WANs allowed
		EnableInterClient:     true,
		EnablePortForwarding:  true,
		IdleTimeout:           5 * time.Minute,
		SessionTimeout:        24 * time.Hour,
	}
}

// NewNATTable creates a new NAT table
func NewNATTable() *NATTable {
	return &NATTable{
		mappings: make(map[string]*NATMapping),
	}
}

// AddMapping adds a NAT mapping
func (nt *NATTable) AddMapping(mapping *NATMapping) {
	nt.mu.Lock()
	defer nt.mu.Unlock()

	key := makeNATKey(mapping.SourceIP, mapping.SourcePort, mapping.Protocol)
	nt.mappings[key] = mapping
}

// GetMapping retrieves a NAT mapping
func (nt *NATTable) GetMapping(srcIP net.IP, srcPort uint16, protocol uint8) *NATMapping {
	nt.mu.RLock()
	defer nt.mu.RUnlock()

	key := makeNATKey(srcIP, srcPort, protocol)
	return nt.mappings[key]
}

// DeleteMapping removes a NAT mapping
func (nt *NATTable) DeleteMapping(srcIP net.IP, srcPort uint16, protocol uint8) {
	nt.mu.Lock()
	defer nt.mu.Unlock()

	key := makeNATKey(srcIP, srcPort, protocol)
	delete(nt.mappings, key)
}

// makeNATKey creates a unique key for NAT mappings
func makeNATKey(ip net.IP, port uint16, protocol uint8) string {
	return fmt.Sprintf("%s:%d/%d", ip.String(), port, protocol)
}

// UpdateBandwidth updates bandwidth usage
func (bq *BandwidthQuota) UpdateBandwidth(uploaded, downloaded uint64) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	bq.TotalUploaded += uploaded
	bq.TotalDownloaded += downloaded
	bq.DailyUploaded += uploaded
	bq.DailyDownloaded += downloaded
	bq.MonthlyUploaded += uploaded
	bq.MonthlyDownloaded += downloaded
}

// CheckQuota checks if quota is exceeded
func (bq *BandwidthQuota) CheckQuota(dailyQuota, monthlyQuota uint64) bool {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	if dailyQuota > 0 && (bq.DailyUploaded+bq.DailyDownloaded) > dailyQuota {
		return true // Daily quota exceeded
	}

	if monthlyQuota > 0 && (bq.MonthlyUploaded+bq.MonthlyDownloaded) > monthlyQuota {
		return true // Monthly quota exceeded
	}

	return false
}

// ResetDaily resets daily counters
func (bq *BandwidthQuota) ResetDaily() {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	bq.DailyUploaded = 0
	bq.DailyDownloaded = 0
}

// ResetMonthly resets monthly counters
func (bq *BandwidthQuota) ResetMonthly() {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	bq.MonthlyUploaded = 0
	bq.MonthlyDownloaded = 0
}

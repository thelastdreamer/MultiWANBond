// Package nat provides NAT traversal capabilities including STUN, hole punching, and relay fallback
package nat

import (
	"net"
	"time"
)

// NATType represents the type of NAT detected
type NATType int

const (
	// NATTypeUnknown means NAT type hasn't been determined yet
	NATTypeUnknown NATType = iota

	// NATTypeOpen means no NAT (direct internet connection)
	NATTypeOpen

	// NATTypeFullCone means full cone NAT - easiest to traverse
	// External IP:port is mapped to internal IP:port for all external sources
	NATTypeFullCone

	// NATTypeRestrictedCone means restricted cone NAT
	// External IP:port is mapped, but only accepts from IPs we've sent to
	NATTypeRestrictedCone

	// NATTypePortRestrictedCone means port-restricted cone NAT
	// External IP:port is mapped, but only accepts from IP:ports we've sent to
	NATTypePortRestrictedCone

	// NATTypeSymmetric means symmetric NAT - hardest to traverse
	// Different external IP:port for each destination
	NATTypeSymmetric

	// NATTypeBlocked means UDP is completely blocked
	NATTypeBlocked
)

// String returns string representation of NAT type
func (t NATType) String() string {
	switch t {
	case NATTypeUnknown:
		return "Unknown"
	case NATTypeOpen:
		return "Open (No NAT)"
	case NATTypeFullCone:
		return "Full Cone NAT"
	case NATTypeRestrictedCone:
		return "Restricted Cone NAT"
	case NATTypePortRestrictedCone:
		return "Port-Restricted Cone NAT"
	case NATTypeSymmetric:
		return "Symmetric NAT"
	case NATTypeBlocked:
		return "Blocked (UDP Filtered)"
	default:
		return "Unknown"
	}
}

// CanDirectConnect returns true if direct P2P connection is possible
func (t NATType) CanDirectConnect() bool {
	return t == NATTypeOpen || t == NATTypeFullCone ||
		   t == NATTypeRestrictedCone || t == NATTypePortRestrictedCone
}

// NeedsRelay returns true if relay server is required
func (t NATType) NeedsRelay() bool {
	return t == NATTypeSymmetric || t == NATTypeBlocked
}

// NATMapping represents a discovered NAT mapping
type NATMapping struct {
	// LocalAddr is the local address (internal IP:port)
	LocalAddr *net.UDPAddr

	// PublicAddr is the public address (external IP:port) as seen by STUN server
	PublicAddr *net.UDPAddr

	// MappingType is the type of NAT detected
	MappingType NATType

	// Discovered is when this mapping was discovered
	Discovered time.Time

	// LastRefresh is when this mapping was last refreshed
	LastRefresh time.Time

	// TTL is the estimated lifetime of this mapping
	TTL time.Duration

	// STUNServer is the STUN server used for discovery
	STUNServer string
}

// IsExpired returns true if the mapping is likely expired
func (m *NATMapping) IsExpired() bool {
	if m.TTL == 0 {
		// Default NAT timeout is usually 30-300 seconds
		// Use conservative 30 seconds
		return time.Since(m.LastRefresh) > 30*time.Second
	}
	return time.Since(m.LastRefresh) > m.TTL
}

// STUNConfig contains configuration for STUN client
type STUNConfig struct {
	// PrimaryServer is the primary STUN server address
	PrimaryServer string

	// SecondaryServer is the secondary STUN server (for NAT type detection)
	SecondaryServer string

	// Timeout for STUN requests
	Timeout time.Duration

	// RetryCount is number of retries for failed requests
	RetryCount int

	// RefreshInterval is how often to refresh NAT mappings
	RefreshInterval time.Duration

	// LocalPort is the local port to bind (0 for random)
	LocalPort int
}

// DefaultSTUNConfig returns default STUN configuration
func DefaultSTUNConfig() *STUNConfig {
	return &STUNConfig{
		PrimaryServer:   "stun.l.google.com:19302",
		SecondaryServer: "stun1.l.google.com:19302",
		Timeout:         5 * time.Second,
		RetryCount:      3,
		RefreshInterval: 25 * time.Second, // Refresh before 30s NAT timeout
		LocalPort:       0,
	}
}

// HolePunchConfig contains configuration for UDP hole punching
type HolePunchConfig struct {
	// Timeout for hole punching attempts
	Timeout time.Duration

	// MaxAttempts is maximum number of punch attempts
	MaxAttempts int

	// RetryInterval is delay between punch attempts
	RetryInterval time.Duration

	// KeepAliveInterval for maintaining hole
	KeepAliveInterval time.Duration
}

// DefaultHolePunchConfig returns default hole punching configuration
func DefaultHolePunchConfig() *HolePunchConfig {
	return &HolePunchConfig{
		Timeout:           10 * time.Second,
		MaxAttempts:       10,
		RetryInterval:     500 * time.Millisecond,
		KeepAliveInterval: 15 * time.Second, // Keep hole open
	}
}

// RelayConfig contains configuration for relay fallback
type RelayConfig struct {
	// RelayServers is list of relay server addresses
	RelayServers []string

	// Timeout for relay operations
	Timeout time.Duration

	// MaxBandwidth for relay connections (bytes/sec)
	MaxBandwidth uint64

	// EnableRelay enables relay fallback
	EnableRelay bool

	// PreferDirect tries direct connection first before relay
	PreferDirect bool
}

// DefaultRelayConfig returns default relay configuration
func DefaultRelayConfig() *RelayConfig {
	return &RelayConfig{
		RelayServers: []string{},
		Timeout:      10 * time.Second,
		MaxBandwidth: 10 * 1024 * 1024, // 10 MB/s
		EnableRelay:  true,
		PreferDirect: true,
	}
}

// PeerInfo contains information about a peer for NAT traversal
type PeerInfo struct {
	// PeerID is unique identifier for the peer
	PeerID string

	// LocalAddr is peer's local address (may be private)
	LocalAddr *net.UDPAddr

	// PublicAddr is peer's public address (from STUN)
	PublicAddr *net.UDPAddr

	// NATType is the peer's NAT type
	NATType NATType

	// LastSeen is when we last heard from this peer
	LastSeen time.Time
}

// TraversalMethod represents the method used to establish connection
type TraversalMethod int

const (
	// TraversalMethodDirect means direct connection (no NAT)
	TraversalMethodDirect TraversalMethod = iota

	// TraversalMethodSTUN means STUN-assisted connection
	TraversalMethodSTUN

	// TraversalMethodHolePunch means UDP hole punching
	TraversalMethodHolePunch

	// TraversalMethodRelay means relay server (TURN-like)
	TraversalMethodRelay
)

// String returns string representation of traversal method
func (m TraversalMethod) String() string {
	switch m {
	case TraversalMethodDirect:
		return "Direct"
	case TraversalMethodSTUN:
		return "STUN"
	case TraversalMethodHolePunch:
		return "Hole Punch"
	case TraversalMethodRelay:
		return "Relay"
	default:
		return "Unknown"
	}
}

// ConnectionInfo contains information about an established connection
type ConnectionInfo struct {
	// PeerID is the peer identifier
	PeerID string

	// LocalAddr is our local address
	LocalAddr *net.UDPAddr

	// RemoteAddr is the peer's address we're connected to
	RemoteAddr *net.UDPAddr

	// Method is the traversal method used
	Method TraversalMethod

	// Established is when connection was established
	Established time.Time

	// LastActivity is last send/receive time
	LastActivity time.Time

	// BytesSent is total bytes sent
	BytesSent uint64

	// BytesReceived is total bytes received
	BytesReceived uint64

	// RTT is round-trip time
	RTT time.Duration
}

// CGNATConfig contains configuration for CGNAT detection and handling
type CGNATConfig struct {
	// EnableCGNATDetection enables automatic CGNAT detection
	EnableCGNATDetection bool

	// CGNATRanges are known CGNAT IP ranges (RFC 6598: 100.64.0.0/10)
	CGNATRanges []*net.IPNet

	// ForceRelay forces relay usage when behind CGNAT
	ForceRelay bool

	// AggressivePunch uses more aggressive hole punching for CGNAT
	AggressivePunch bool
}

// DefaultCGNATConfig returns default CGNAT configuration
func DefaultCGNATConfig() *CGNATConfig {
	// RFC 6598 CGNAT range: 100.64.0.0/10
	_, cgnatNet, _ := net.ParseCIDR("100.64.0.0/10")

	return &CGNATConfig{
		EnableCGNATDetection: true,
		CGNATRanges:         []*net.IPNet{cgnatNet},
		ForceRelay:          false,
		AggressivePunch:     true,
	}
}

// IsCGNATAddress checks if an IP is in CGNAT range
func (c *CGNATConfig) IsCGNATAddress(ip net.IP) bool {
	for _, ipNet := range c.CGNATRanges {
		if ipNet.Contains(ip) {
			return true
		}
	}
	return false
}

// IsPrivateAddress checks if an IP is in private range (RFC 1918)
func IsPrivateAddress(ip net.IP) bool {
	// 10.0.0.0/8
	if ip[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}
	return false
}

// NATTraversalConfig is the complete configuration for NAT traversal
type NATTraversalConfig struct {
	STUN      *STUNConfig
	HolePunch *HolePunchConfig
	Relay     *RelayConfig
	CGNAT     *CGNATConfig
}

// DefaultNATTraversalConfig returns default NAT traversal configuration
func DefaultNATTraversalConfig() *NATTraversalConfig {
	return &NATTraversalConfig{
		STUN:      DefaultSTUNConfig(),
		HolePunch: DefaultHolePunchConfig(),
		Relay:     DefaultRelayConfig(),
		CGNAT:     DefaultCGNATConfig(),
	}
}

// TraversalStats contains statistics about NAT traversal
type TraversalStats struct {
	// STUNRequests is total STUN requests sent
	STUNRequests uint64

	// STUNSuccesses is successful STUN requests
	STUNSuccesses uint64

	// STUNFailures is failed STUN requests
	STUNFailures uint64

	// HolePunchAttempts is total hole punch attempts
	HolePunchAttempts uint64

	// HolePunchSuccesses is successful hole punches
	HolePunchSuccesses uint64

	// RelayConnections is total relay connections established
	RelayConnections uint64

	// DirectConnections is total direct connections
	DirectConnections uint64

	// ActiveConnections is currently active connections
	ActiveConnections uint64

	// CGNATDetected is number of times CGNAT was detected
	CGNATDetected uint64
}

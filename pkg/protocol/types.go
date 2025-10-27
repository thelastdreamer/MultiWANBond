package protocol

import (
	"fmt"
	"net"
	"time"
)

// Protocol version for future compatibility
const (
	ProtocolVersion = 1
	MaxPacketSize   = 65535
)

// PacketType defines different packet types in the protocol
type PacketType uint8

const (
	PacketTypeData PacketType = iota
	PacketTypeAck
	PacketTypeHeartbeat
	PacketTypeControl
	PacketTypeMulticast
	PacketTypeFEC // Forward Error Correction packet
)

// Packet represents the base protocol packet
type Packet struct {
	Version    uint8      // Protocol version
	Type       PacketType // Packet type
	Flags      uint16     // Flags for various options
	SessionID  uint64     // Session identifier
	SequenceID uint64     // Sequence number for ordering
	Timestamp  int64      // Timestamp in nanoseconds
	WANID      uint8      // WAN interface ID that sent this packet
	Priority   uint8      // Packet priority (0-255)
	DataLen    uint32     // Length of data payload
	Data       []byte     // Actual data payload
	Checksum   uint32     // Packet checksum
}

// PacketFlags
const (
	FlagDuplicate  uint16 = 1 << 0 // This is a duplicate packet
	FlagFEC        uint16 = 1 << 1 // Packet contains FEC data
	FlagCompressed uint16 = 1 << 2 // Packet is compressed
	FlagEncrypted  uint16 = 1 << 3 // Packet is encrypted
	FlagFragment   uint16 = 1 << 4 // Packet is fragmented
	FlagLastFrag   uint16 = 1 << 5 // Last fragment
)

// WANInterface represents a single WAN connection
type WANInterface struct {
	ID          uint8         // Unique ID for this interface
	Name        string        // Interface name (e.g., "eth0", "wlan0")
	Type        WANType       // Connection type
	LocalAddr   net.IP        // Local IP address
	RemoteAddr  *net.UDPAddr  // Remote endpoint address
	Conn        *net.UDPConn  // UDP connection
	Metrics     *WANMetrics   // Current metrics
	State       WANState      // Current state
	Config      WANConfig     // Configuration
	LastSeen    time.Time     // Last successful packet
}

// WANType identifies the type of WAN connection
type WANType uint8

const (
	WANTypeUnknown WANType = iota
	WANTypeADSL
	WANTypeVDSL
	WANTypeFiber
	WANTypeStarlink
	WANTypeSatellite
	WANTypeLTE
	WANType5G
	WANTypeCable
)

func (t WANType) String() string {
	switch t {
	case WANTypeADSL:
		return "ADSL"
	case WANTypeVDSL:
		return "VDSL"
	case WANTypeFiber:
		return "Fiber"
	case WANTypeStarlink:
		return "Starlink"
	case WANTypeSatellite:
		return "Satellite"
	case WANTypeLTE:
		return "LTE"
	case WANType5G:
		return "5G"
	case WANTypeCable:
		return "Cable"
	default:
		return "Unknown"
	}
}

// WANState represents the current state of a WAN connection
type WANState uint8

const (
	WANStateDown WANState = iota
	WANStateStarting
	WANStateUp
	WANStateDegraded
	WANStateRecovering
)

// WANMetrics contains real-time metrics for a WAN interface
type WANMetrics struct {
	Latency       time.Duration // Current RTT
	Jitter        time.Duration // Jitter (variance in latency)
	PacketLoss    float64       // Packet loss percentage (0-100)
	Bandwidth     uint64        // Available bandwidth in bytes/sec
	BytesSent     uint64        // Total bytes sent
	BytesReceived uint64        // Total bytes received
	PacketsSent   uint64        // Total packets sent
	PacketsRecv   uint64        // Total packets received
	PacketsLost   uint64        // Total packets lost
	LastUpdate    time.Time     // Last metrics update

	// Moving averages for stability
	AvgLatency    time.Duration
	AvgJitter     time.Duration
	AvgPacketLoss float64
}

// WANConfig contains configuration for a WAN interface
type WANConfig struct {
	MaxBandwidth    uint64        // Maximum bandwidth limit (bytes/sec)
	MaxLatency      time.Duration // Maximum acceptable latency
	MaxJitter       time.Duration // Maximum acceptable jitter
	MaxPacketLoss   float64       // Maximum acceptable packet loss %
	HealthCheckInterval time.Duration // How often to check health
	FailureThreshold    int       // Consecutive failures before marking down
	Weight          int           // Weight for load balancing (higher = more traffic)
	Priority        int           // Priority for failover (0 = highest/primary, higher = backup)
	Enabled         bool          // Whether this interface is enabled
}

// Session represents a bonded connection session
type Session struct {
	ID            uint64                  // Unique session ID
	LocalEndpoint  string                  // Local endpoint address
	RemoteEndpoint string                  // Remote endpoint address
	WANInterfaces  map[uint8]*WANInterface // Active WAN interfaces
	StartTime     time.Time               // Session start time
	Config        *SessionConfig          // Session configuration
}

// SessionConfig contains session-level configuration
type SessionConfig struct {
	// Redundancy settings
	DuplicatePackets bool          // Send duplicates on multiple paths
	DuplicateFilter  DuplicateMode // How to handle duplicates

	// FEC settings
	FECEnabled       bool    // Enable Forward Error Correction
	FECRedundancy    float64 // FEC redundancy ratio (e.g., 0.2 = 20% redundant)

	// Packet ordering
	ReorderBuffer    int           // Size of reorder buffer
	ReorderTimeout   time.Duration // Max time to wait for out-of-order packets

	// Load balancing
	LoadBalanceMode  LoadBalanceMode // Load balancing strategy

	// Multicast
	MulticastEnabled bool
	MulticastGroups  []string

	// Performance
	MaxInflightPackets int // Maximum packets in flight
	SendBufferSize     int // Send buffer size
	RecvBufferSize     int // Receive buffer size
}

// DuplicateMode determines how duplicate packets are handled
type DuplicateMode uint8

const (
	DuplicateKeepFirst  DuplicateMode = iota // Keep first received
	DuplicateKeepFastest                      // Keep fastest received
	DuplicateKeepBest                         // Keep from best connection
)

// LoadBalanceMode determines packet distribution strategy
type LoadBalanceMode uint8

const (
	LoadBalanceRoundRobin LoadBalanceMode = iota // Simple round-robin
	LoadBalanceWeighted                           // Weighted by bandwidth/latency
	LoadBalanceLeastUsed                          // Send on least utilized
	LoadBalanceLeastLatency                       // Send on lowest latency
	LoadBalancePerFlow                            // Consistent per-flow routing
	LoadBalanceAdaptive                           // Adaptive based on conditions
	LoadBalanceFailover                           // Failover mode (primary/backup with sub-second switching)
)

// RoutingDecision contains information about where to send a packet
type RoutingDecision struct {
	PrimaryWAN   uint8   // Primary WAN to use
	BackupWANs   []uint8 // Backup WANs for redundancy
	UseFEC       bool    // Whether to use FEC for this packet
	Priority     uint8   // Packet priority
}

// FlowKey identifies a unique flow (for per-flow routing)
type FlowKey struct {
	SrcIP   net.IP
	DstIP   net.IP
	SrcPort uint16
	DstPort uint16
	Protocol uint8
}

// String converts FlowKey to a string for use as a map key
func (f FlowKey) String() string {
	return fmt.Sprintf("%s:%d->%s:%d/%d", f.SrcIP, f.SrcPort, f.DstIP, f.DstPort, f.Protocol)
}

package protocol

import (
	"context"
	"io"
)

// Bonder is the main interface for the multi-WAN bonding system
type Bonder interface {
	// Start begins the bonding service
	Start(ctx context.Context) error

	// Stop gracefully stops the bonding service
	Stop() error

	// AddWAN adds a new WAN interface to the bond
	AddWAN(wan *WANInterface) error

	// RemoveWAN removes a WAN interface from the bond
	RemoveWAN(wanID uint8) error

	// GetWANs returns all active WAN interfaces
	GetWANs() map[uint8]*WANInterface

	// GetMetrics returns current metrics for all WANs
	GetMetrics() map[uint8]*WANMetrics

	// Send sends data through the bonded connection
	Send(data []byte) error

	// Receive returns a channel for receiving data
	Receive() <-chan []byte

	// GetSession returns the current session
	GetSession() *Session

	// UpdateConfig updates the session configuration
	UpdateConfig(config *SessionConfig) error
}

// HealthChecker monitors the health of WAN connections
type HealthChecker interface {
	// Start begins health monitoring
	Start(ctx context.Context) error

	// Stop stops health monitoring
	Stop() error

	// CheckWAN performs a health check on a specific WAN
	CheckWAN(wan *WANInterface) (*WANMetrics, error)

	// GetMetrics returns current metrics for a WAN
	GetMetrics(wanID uint8) (*WANMetrics, error)

	// Subscribe to health events
	Subscribe() <-chan HealthEvent
}

// HealthEvent represents a health status change
type HealthEvent struct {
	WANID     uint8
	OldState  WANState
	NewState  WANState
	Metrics   *WANMetrics
	Timestamp int64
}

// Router determines which WAN(s) to use for sending packets
type Router interface {
	// Route determines routing for a packet
	Route(packet *Packet, flowKey *FlowKey) (*RoutingDecision, error)

	// UpdateMetrics updates routing decisions based on new metrics
	UpdateMetrics(wanID uint8, metrics *WANMetrics)

	// SetMode changes the load balancing mode
	SetMode(mode LoadBalanceMode)
}

// PacketProcessor handles packet encoding/decoding and ordering
type PacketProcessor interface {
	// Encode encodes a packet for transmission
	Encode(packet *Packet) ([]byte, error)

	// Decode decodes a received packet
	Decode(data []byte) (*Packet, error)

	// Reorder handles packet reordering
	Reorder(packet *Packet) ([]byte, bool, error)

	// Reset resets the reorder buffer
	Reset()
}

// FECEncoder handles Forward Error Correction encoding
type FECEncoder interface {
	// Encode adds FEC redundancy to data
	Encode(data []byte, redundancy float64) ([][]byte, error)

	// Decode recovers data from FEC packets (may have missing packets)
	Decode(packets [][]byte, missing []int) ([]byte, error)

	// CanRecover checks if data can be recovered given packet loss
	CanRecover(totalPackets, receivedPackets int) bool
}

// PacketSender sends packets over WAN interfaces
type PacketSender interface {
	// Send sends a packet on a specific WAN
	Send(wan *WANInterface, packet *Packet) error

	// SendMultiple sends a packet on multiple WANs
	SendMultiple(wans []*WANInterface, packet *Packet) error

	// Flush flushes any buffered packets
	Flush() error
}

// PacketReceiver receives packets from WAN interfaces
type PacketReceiver interface {
	// Receive returns a channel for received packets
	Receive() <-chan *ReceivedPacket

	// Start starts receiving on a WAN interface
	Start(ctx context.Context, wan *WANInterface) error

	// Stop stops receiving
	Stop() error
}

// ReceivedPacket wraps a received packet with metadata
type ReceivedPacket struct {
	Packet       *Packet
	WANID        uint8
	ReceivedAt   int64
	SourceAddr   string
}

// MulticastManager handles multicast packet distribution
type MulticastManager interface {
	// Join joins a multicast group
	Join(group string) error

	// Leave leaves a multicast group
	Leave(group string) error

	// Send sends a multicast packet
	Send(group string, data []byte) error

	// Receive returns a channel for multicast packets
	Receive() <-chan *MulticastPacket
}

// MulticastPacket represents a multicast packet
type MulticastPacket struct {
	Group string
	Data  []byte
	From  string
}

// Plugin is the interface that all plugins must implement
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Init initializes the plugin
	Init(config map[string]interface{}) error

	// Start starts the plugin
	Start(ctx context.Context) error

	// Stop stops the plugin
	Stop() error
}

// PacketFilter allows plugins to filter/modify packets
type PacketFilter interface {
	Plugin

	// FilterOutgoing filters outgoing packets (return nil to drop)
	FilterOutgoing(packet *Packet) (*Packet, error)

	// FilterIncoming filters incoming packets (return nil to drop)
	FilterIncoming(packet *Packet) (*Packet, error)

	// Priority returns filter priority (lower = runs first)
	Priority() int
}

// MetricsCollector collects and reports metrics
type MetricsCollector interface {
	Plugin

	// RecordPacket records packet metrics
	RecordPacket(wanID uint8, packet *Packet, sent bool)

	// RecordMetrics records WAN metrics
	RecordMetrics(wanID uint8, metrics *WANMetrics)

	// GetReport returns a metrics report
	GetReport() (map[string]interface{}, error)

	// Export exports metrics to an external system
	Export(writer io.Writer) error
}

// AlertManager handles alerts and notifications
type AlertManager interface {
	Plugin

	// Alert sends an alert
	Alert(level AlertLevel, message string, details map[string]interface{}) error

	// Subscribe returns a channel for alerts
	Subscribe() <-chan Alert
}

// Alert represents an alert/notification
type Alert struct {
	Level     AlertLevel
	Message   string
	Details   map[string]interface{}
	Timestamp int64
}

// AlertLevel defines alert severity
type AlertLevel uint8

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelError
	AlertLevelCritical
)

// ConfigProvider provides configuration to the system
type ConfigProvider interface {
	// Get returns a configuration value
	Get(key string) (interface{}, error)

	// Set sets a configuration value
	Set(key string, value interface{}) error

	// Watch watches for configuration changes
	Watch(key string) (<-chan interface{}, error)

	// Load loads configuration from source
	Load() error

	// Save saves configuration to source
	Save() error
}

// Package webui provides web-based management interface
package webui

import (
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/dpi"
	"github.com/thelastdreamer/MultiWANBond/pkg/health"
	"github.com/thelastdreamer/MultiWANBond/pkg/nat"
	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
	"github.com/thelastdreamer/MultiWANBond/pkg/routing"
)

// Config contains web UI configuration
type Config struct {
	// ListenAddr is the address to listen on
	ListenAddr string

	// ListenPort is the port to listen on
	ListenPort int

	// EnableTLS enables HTTPS
	EnableTLS bool

	// CertFile is the TLS certificate file
	CertFile string

	// KeyFile is the TLS key file
	KeyFile string

	// EnableAuth enables authentication
	EnableAuth bool

	// Username for basic auth
	Username string

	// Password for basic auth
	Password string

	// EnableCORS enables CORS headers
	EnableCORS bool

	// AllowedOrigins for CORS
	AllowedOrigins []string

	// StaticDir is the directory for static files
	StaticDir string

	// EnableMetrics enables Prometheus metrics endpoint
	EnableMetrics bool

	// MetricsPath is the path for metrics endpoint
	MetricsPath string
}

// DefaultConfig returns default web UI configuration
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:     "0.0.0.0",
		ListenPort:     8080,
		EnableTLS:      false,
		EnableAuth:     false,
		EnableCORS:     true,
		AllowedOrigins: []string{"*"},
		StaticDir:      "./webui",
		EnableMetrics:  true,
		MetricsPath:    "/metrics",
	}
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// DashboardStats contains overall system statistics
type DashboardStats struct {
	// System info
	Uptime        time.Duration `json:"uptime"`
	Version       string        `json:"version"`
	Platform      string        `json:"platform"`

	// WAN stats
	ActiveWANs    int     `json:"active_wans"`
	TotalWANs     int     `json:"total_wans"`
	HealthyWANs   int     `json:"healthy_wans"`
	DegradedWANs  int     `json:"degraded_wans"`
	DownWANs      int     `json:"down_wans"`

	// Traffic stats
	TotalPackets  uint64  `json:"total_packets"`
	TotalBytes    uint64  `json:"total_bytes"`
	CurrentPPS    uint64  `json:"current_pps"`
	CurrentBPS    uint64  `json:"current_bps"`

	// Connection stats
	ActiveFlows   int     `json:"active_flows"`
	TotalSessions int     `json:"total_sessions"`

	// NAT stats
	NATType       string  `json:"nat_type"`
	PublicIP      string  `json:"public_ip"`
	CGNATDetected bool    `json:"cgnat_detected"`

	// Timestamp
	Timestamp     time.Time `json:"timestamp"`
}

// WANStatus represents the status of a WAN interface
type WANStatus struct {
	ID               uint8                  `json:"id"`
	Name             string                 `json:"name"`
	Interface        string                 `json:"interface"`
	Status           string                 `json:"status"` // "up", "degraded", "down"
	Health           float64                `json:"health"` // 0-100
	Latency          int64                  `json:"latency_ms"`
	Jitter           int64                  `json:"jitter_ms"`
	PacketLoss       float64                `json:"packet_loss"`
	Bandwidth        uint64                 `json:"bandwidth_bps"`
	BytesSent        uint64                 `json:"bytes_sent"`
	BytesReceived    uint64                 `json:"bytes_received"`
	PacketsSent      uint64                 `json:"packets_sent"`
	PacketsReceived  uint64                 `json:"packets_received"`
	Uptime           time.Duration          `json:"uptime"`
	Priority         int                    `json:"priority"`
	Weight           int                    `json:"weight"`
	Config           *protocol.WANConfig    `json:"config,omitempty"`
}

// FlowInfo represents information about a network flow
type FlowInfo struct {
	SrcIP       string    `json:"src_ip"`
	DstIP       string    `json:"dst_ip"`
	SrcPort     uint16    `json:"src_port"`
	DstPort     uint16    `json:"dst_port"`
	Protocol    string    `json:"protocol"`
	Application string    `json:"application"`
	Category    string    `json:"category"`
	WANID       uint8     `json:"wan_id"`
	Packets     uint64    `json:"packets"`
	Bytes       uint64    `json:"bytes"`
	Duration    int64     `json:"duration_ms"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
}

// TrafficStats contains traffic statistics
type TrafficStats struct {
	Timestamp     time.Time          `json:"timestamp"`
	TotalBytes    uint64             `json:"total_bytes"`
	TotalPackets  uint64             `json:"total_packets"`
	BytesPerWAN   map[uint8]uint64   `json:"bytes_per_wan"`
	PacketsPerWAN map[uint8]uint64   `json:"packets_per_wan"`
	TopProtocols  []ProtocolStat     `json:"top_protocols"`
	TopFlows      []FlowInfo         `json:"top_flows"`
}

// ProtocolStat contains statistics for a protocol
type ProtocolStat struct {
	Protocol string `json:"protocol"`
	Category string `json:"category"`
	Flows    uint64 `json:"flows"`
	Bytes    uint64 `json:"bytes"`
	Packets  uint64 `json:"packets"`
}

// WANConfig contains WAN configuration for API
type WANConfig struct {
	ID              uint8  `json:"id"`
	Name            string `json:"name"`
	Interface       string `json:"interface"`
	Priority        int    `json:"priority"`
	Weight          int    `json:"weight"`
	MaxBandwidth    uint64 `json:"max_bandwidth"`
	MaxLatency      int64  `json:"max_latency_ms"`
	MaxJitter       int64  `json:"max_jitter_ms"`
	MaxPacketLoss   float64 `json:"max_packet_loss"`
	HealthCheckURL  string `json:"health_check_url"`
	HealthCheckInterval int64 `json:"health_check_interval_ms"`
	Enabled         bool   `json:"enabled"`
}

// RoutingPolicy contains routing policy for API
type RoutingPolicy struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "source", "destination", "application"
	Match       string `json:"match"` // IP, CIDR, domain, or app name based on Type
	TargetWAN   uint8  `json:"target_wan"` // WAN ID to use
	Priority    int    `json:"priority"`
	Enabled     bool   `json:"enabled"`
}

// SystemConfig contains system configuration
type SystemConfig struct {
	LoadBalanceMode string         `json:"load_balance_mode"`
	EnableFEC       bool           `json:"enable_fec"`
	FECDataShards   int            `json:"fec_data_shards"`
	FECParityShards int            `json:"fec_parity_shards"`
	EnableDPI       bool           `json:"enable_dpi"`
	EnableQoS       bool           `json:"enable_qos"`
	EnableNATT      bool           `json:"enable_nat_traversal"`
	RoutingPolicies []RoutingPolicy `json:"routing_policies"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// EventType represents the type of event
type EventType string

const (
	EventWANStatusChange   EventType = "wan_status_change"
	EventWANHealthUpdate   EventType = "wan_health_update"
	EventFlowCreated       EventType = "flow_created"
	EventFlowClosed        EventType = "flow_closed"
	EventFailover          EventType = "failover"
	EventTrafficUpdate     EventType = "traffic_update"
	EventSystemAlert       EventType = "system_alert"
	EventConfigChange      EventType = "config_change"
)

// Event represents a system event
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Severity  string      `json:"severity"` // "info", "warning", "error"
}

// WANHealthUpdate represents a WAN health update event
type WANHealthUpdate struct {
	WANID      uint8   `json:"wan_id"`
	Status     string  `json:"status"`
	Latency    int64   `json:"latency_ms"`
	Jitter     int64   `json:"jitter_ms"`
	PacketLoss float64 `json:"packet_loss"`
}

// FailoverEvent represents a failover event
type FailoverEvent struct {
	FromWAN uint8  `json:"from_wan"`
	ToWAN   uint8  `json:"to_wan"`
	Reason  string `json:"reason"`
}

// Alert represents a system alert
type Alert struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Resolved  bool      `json:"resolved"`
}

// ChartData represents time-series chart data
type ChartData struct {
	Labels []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

// Dataset represents a chart dataset
type Dataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
	Color string    `json:"color,omitempty"`
}

// NATInfo contains NAT traversal information for UI
type NATInfo struct {
	LocalAddr     string `json:"local_addr"`
	PublicAddr    string `json:"public_addr"`
	NATType       string `json:"nat_type"`
	CGNATDetected bool   `json:"cgnat_detected"`
	CanDirect     bool   `json:"can_direct_connect"`
	NeedsRelay    bool   `json:"needs_relay"`
	RelayAvailable bool  `json:"relay_available"`
}

// HealthCheckInfo contains health check information
type HealthCheckInfo struct {
	WANID         uint8   `json:"wan_id"`
	Method        string  `json:"method"`
	Target        string  `json:"target"`
	Interval      int64   `json:"interval_ms"`
	LastCheck     time.Time `json:"last_check"`
	Status        string  `json:"status"`
	Latency       int64   `json:"latency_ms"`
	Jitter        int64   `json:"jitter_ms"`
	PacketLoss    float64 `json:"packet_loss"`
	Successes     int     `json:"successes"`
	Failures      int     `json:"failures"`
}

// ClientInfo contains client session information
type ClientInfo struct {
	SessionID     string    `json:"session_id"`
	ClientID      string    `json:"client_id"`
	RemoteAddr    string    `json:"remote_addr"`
	PublicIP      string    `json:"public_ip"`
	Connected     time.Time `json:"connected"`
	LastSeen      time.Time `json:"last_seen"`
	BytesSent     uint64    `json:"bytes_sent"`
	BytesReceived uint64    `json:"bytes_received"`
	ActiveFlows   int       `json:"active_flows"`
}

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Goroutines  int     `json:"goroutines"`
	Uptime      int64   `json:"uptime_seconds"`
}

// LogEntry represents a log entry for UI
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Component string    `json:"component"`
	Message   string    `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ToWANStatus converts internal types to API types
func ToWANStatus(wan *protocol.WANInterface, wanHealth *health.WANHealth) *WANStatus {
	status := "unknown"
	if wanHealth != nil {
		switch wanHealth.Status {
		case health.WANStatusUp:
			status = "up"
		case health.WANStatusDegraded:
			status = "degraded"
		case health.WANStatusDown:
			status = "down"
		}
	}

	healthPercent := 0.0
	if wanHealth != nil {
		healthPercent = wanHealth.Uptime * 100.0
	}

	latency := int64(0)
	jitter := int64(0)
	packetLoss := 0.0
	if wanHealth != nil {
		latency = wanHealth.AvgLatency.Milliseconds()
		jitter = wanHealth.AvgJitter.Milliseconds()
		packetLoss = wanHealth.AvgPacketLoss
	}

	return &WANStatus{
		ID:          wan.ID,
		Name:        wan.Name,
		Interface:   wan.Name,
		Status:      status,
		Health:      healthPercent,
		Latency:     latency,
		Jitter:      jitter,
		PacketLoss:  packetLoss,
		Priority:    wan.Config.Priority,
		Weight:      wan.Config.Weight,
		Config:      &wan.Config,
	}
}

// ToNATInfo converts NAT capabilities to API type
func ToNATInfo(caps *nat.TraversalCapabilities) *NATInfo {
	localAddr := ""
	publicAddr := ""
	if caps.LocalAddr != nil {
		localAddr = caps.LocalAddr.String()
	}
	if caps.PublicAddr != nil {
		publicAddr = caps.PublicAddr.String()
	}

	return &NATInfo{
		LocalAddr:      localAddr,
		PublicAddr:     publicAddr,
		NATType:        caps.NATType.String(),
		CGNATDetected:  caps.CGNATDetected,
		CanDirect:      caps.CanDirectConnect,
		NeedsRelay:     caps.NeedsRelay,
		RelayAvailable: caps.RelayAvailable,
	}
}

// ToFlowInfo converts DPI flow to API type
func ToFlowInfo(flow *dpi.Flow, wanID uint8) *FlowInfo {
	duration := flow.LastSeen.Sub(flow.FirstSeen)

	return &FlowInfo{
		SrcIP:       flow.SrcIP.String(),
		DstIP:       flow.DstIP.String(),
		SrcPort:     flow.SrcPort,
		DstPort:     flow.DstPort,
		Protocol:    flow.Protocol.String(),
		Application: flow.Protocol.String(),
		Category:    flow.Category.String(),
		WANID:       wanID,
		Packets:     flow.Packets,
		Bytes:       flow.Bytes,
		Duration:    duration.Milliseconds(),
		FirstSeen:   flow.FirstSeen,
		LastSeen:    flow.LastSeen,
	}
}

// ToRoutingPolicyAPI converts internal routing policy to API type
func ToRoutingPolicyAPI(policy *routing.RoutingPolicy) []RoutingPolicy {
	policies := make([]RoutingPolicy, 0, len(policy.Rules))

	for i, rule := range policy.Rules {
		policyType := "custom"
		match := ""

		if rule.SourceNetwork != nil {
			policyType = "source"
			match = rule.SourceNetwork.String()
		}
		if rule.DestNetwork != nil {
			if policyType == "source" {
				policyType = "both"
			} else {
				policyType = "destination"
			}
			match = rule.DestNetwork.String()
		}

		policies = append(policies, RoutingPolicy{
			ID:          i + 1,
			Name:        policy.Name,
			Description: policy.Description,
			Type:        policyType,
			Match:       match,
			TargetWAN:   0, // TODO: Get from rule if available
			Priority:    rule.Priority,
			Enabled:     rule.Enabled,
		})
	}

	return policies
}

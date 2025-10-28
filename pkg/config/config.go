package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// Config represents the main configuration structure
type Config struct {
	mu        sync.RWMutex
	filePath  string
	data      map[string]interface{}
	watchers  map[string][]chan interface{}
	lastMod   time.Time
}

// BondConfig represents the configuration for the bonding system
type BondConfig struct {
	// Session configuration
	Session SessionConfig `json:"session"`

	// WAN interfaces
	WANs []WANInterfaceConfig `json:"wans"`

	// Routing configuration
	Routing RoutingConfig `json:"routing"`

	// FEC configuration
	FEC FECConfig `json:"fec"`

	// Monitoring configuration
	Monitoring MonitoringConfig `json:"monitoring"`

	// Plugins configuration
	Plugins []PluginConfig `json:"plugins"`

	// Web UI configuration
	WebUI *WebUIConfig `json:"webui,omitempty"`
}

// SessionConfig contains session-level configuration
type SessionConfig struct {
	LocalEndpoint    string `json:"local_endpoint"`
	RemoteEndpoint   string `json:"remote_endpoint"`
	DuplicatePackets bool   `json:"duplicate_packets"`
	DuplicateMode    string `json:"duplicate_mode"` // "first", "fastest", "best"
	ReorderBuffer    int    `json:"reorder_buffer"`
	ReorderTimeout   string `json:"reorder_timeout"` // e.g., "500ms"
	MulticastEnabled bool   `json:"multicast_enabled"`
	MulticastGroups  []string `json:"multicast_groups"`
}

// WANInterfaceConfig contains configuration for a WAN interface
type WANInterfaceConfig struct {
	ID                  uint8  `json:"id"`
	Name                string `json:"name"`
	Type                string `json:"type"` // "adsl", "vdsl", "fiber", etc.
	LocalAddr           string `json:"local_addr"`
	RemoteAddr          string `json:"remote_addr"`
	MaxBandwidth        uint64 `json:"max_bandwidth"` // bytes/sec
	MaxLatency          string `json:"max_latency"` // e.g., "100ms"
	MaxJitter           string `json:"max_jitter"` // e.g., "50ms"
	MaxPacketLoss       float64 `json:"max_packet_loss"` // percentage
	HealthCheckInterval string `json:"health_check_interval"` // e.g., "200ms"
	FailureThreshold    int    `json:"failure_threshold"`
	Weight              int    `json:"weight"` // for weighted routing
	Enabled             bool   `json:"enabled"`
}

// RoutingConfig contains routing configuration
type RoutingConfig struct {
	Mode                string          `json:"mode"` // "round_robin", "weighted", "least_used", etc.
	BandwidthResetInterval string       `json:"bandwidth_reset_interval"` // e.g., "1m"
	Policies            []RoutingPolicy `json:"policies,omitempty"` // Routing policies
}

// RoutingPolicy defines a routing policy rule
type RoutingPolicy struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "source", "destination", "application"
	Match       string `json:"match"` // IP, CIDR, domain, or app name
	TargetWAN   uint8  `json:"target_wan"` // WAN ID to use
	Priority    int    `json:"priority"` // Lower = higher priority
	Enabled     bool   `json:"enabled"`
}

// FECConfig contains FEC configuration
type FECConfig struct {
	Enabled    bool    `json:"enabled"`
	Redundancy float64 `json:"redundancy"` // e.g., 0.2 for 20%
	DataShards int     `json:"data_shards"`
	ParityShards int   `json:"parity_shards"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	Enabled         bool   `json:"enabled"`
	MetricsInterval string `json:"metrics_interval"` // e.g., "10s"
	AlertsEnabled   bool   `json:"alerts_enabled"`
}

// PluginConfig contains plugin configuration
type PluginConfig struct {
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// WebUIConfig contains Web UI authentication settings
type WebUIConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// NewConfig creates a new configuration instance
func NewConfig(filePath string) *Config {
	return &Config{
		filePath: filePath,
		data:     make(map[string]interface{}),
		watchers: make(map[string][]chan interface{}),
	}
}

// Load loads configuration from file
func (c *Config) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Read file
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var configData map[string]interface{}
	if err := json.Unmarshal(data, &configData); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	c.data = configData

	// Update last modified time
	if info, err := os.Stat(c.filePath); err == nil {
		c.lastMod = info.ModTime()
	}

	return nil
}

// Save saves configuration to file
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns a configuration value
func (c *Config) Get(key string) (interface{}, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, exists := c.data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}

	return value, nil
}

// Set sets a configuration value
func (c *Config) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldValue := c.data[key]
	c.data[key] = value

	// Notify watchers
	if watchers, exists := c.watchers[key]; exists {
		for _, ch := range watchers {
			select {
			case ch <- value:
			default:
				// Channel full, skip
			}
		}
	}

	// If value changed, consider it as modification
	if oldValue != value {
		c.lastMod = time.Now()
	}

	return nil
}

// Watch watches for configuration changes
func (c *Config) Watch(key string) (<-chan interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan interface{}, 10)
	c.watchers[key] = append(c.watchers[key], ch)

	return ch, nil
}

// CheckForUpdates checks if the config file has been modified
func (c *Config) CheckForUpdates() (bool, error) {
	info, err := os.Stat(c.filePath)
	if err != nil {
		return false, fmt.Errorf("failed to stat config file: %w", err)
	}

	c.mu.RLock()
	lastMod := c.lastMod
	c.mu.RUnlock()

	return info.ModTime().After(lastMod), nil
}

// Reload reloads configuration from file if it has been modified
func (c *Config) Reload() (bool, error) {
	updated, err := c.CheckForUpdates()
	if err != nil {
		return false, err
	}

	if !updated {
		return false, nil
	}

	if err := c.Load(); err != nil {
		return false, err
	}

	return true, nil
}

// LoadBondConfig loads and parses the full bond configuration
func LoadBondConfig(filePath string) (*BondConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config BondConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveBondConfig saves the bond configuration to file
func SaveBondConfig(filePath string, config *BondConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ToSessionConfig converts config to protocol.SessionConfig
func (sc *SessionConfig) ToSessionConfig() (*protocol.SessionConfig, error) {
	// Parse durations
	reorderTimeout, err := time.ParseDuration(sc.ReorderTimeout)
	if err != nil {
		reorderTimeout = 500 * time.Millisecond
	}

	// Parse duplicate mode
	var duplicateMode protocol.DuplicateMode
	switch sc.DuplicateMode {
	case "first":
		duplicateMode = protocol.DuplicateKeepFirst
	case "fastest":
		duplicateMode = protocol.DuplicateKeepFastest
	case "best":
		duplicateMode = protocol.DuplicateKeepBest
	default:
		duplicateMode = protocol.DuplicateKeepFirst
	}

	return &protocol.SessionConfig{
		DuplicatePackets: sc.DuplicatePackets,
		DuplicateFilter:  duplicateMode,
		ReorderBuffer:    sc.ReorderBuffer,
		ReorderTimeout:   reorderTimeout,
		MulticastEnabled: sc.MulticastEnabled,
		MulticastGroups:  sc.MulticastGroups,
	}, nil
}

// ToWANConfig converts config to protocol.WANConfig
func (wc *WANInterfaceConfig) ToWANConfig() (*protocol.WANConfig, error) {
	// Parse durations
	maxLatency, err := time.ParseDuration(wc.MaxLatency)
	if err != nil {
		maxLatency = 100 * time.Millisecond
	}

	maxJitter, err := time.ParseDuration(wc.MaxJitter)
	if err != nil {
		maxJitter = 50 * time.Millisecond
	}

	healthCheckInterval, err := time.ParseDuration(wc.HealthCheckInterval)
	if err != nil {
		healthCheckInterval = 200 * time.Millisecond
	}

	return &protocol.WANConfig{
		MaxBandwidth:        wc.MaxBandwidth,
		MaxLatency:          maxLatency,
		MaxJitter:           maxJitter,
		MaxPacketLoss:       wc.MaxPacketLoss,
		HealthCheckInterval: healthCheckInterval,
		FailureThreshold:    wc.FailureThreshold,
		Weight:              wc.Weight,
		Enabled:             wc.Enabled,
	}, nil
}

// ParseWANType converts string to WANType
func ParseWANType(typeStr string) protocol.WANType {
	switch typeStr {
	case "adsl":
		return protocol.WANTypeADSL
	case "vdsl":
		return protocol.WANTypeVDSL
	case "fiber":
		return protocol.WANTypeFiber
	case "starlink":
		return protocol.WANTypeStarlink
	case "satellite":
		return protocol.WANTypeSatellite
	case "lte":
		return protocol.WANTypeLTE
	case "5g":
		return protocol.WANType5G
	case "cable":
		return protocol.WANTypeCable
	default:
		return protocol.WANTypeUnknown
	}
}

// ParseLoadBalanceMode converts string to LoadBalanceMode
func ParseLoadBalanceMode(mode string) protocol.LoadBalanceMode {
	switch mode {
	case "round_robin":
		return protocol.LoadBalanceRoundRobin
	case "weighted":
		return protocol.LoadBalanceWeighted
	case "least_used":
		return protocol.LoadBalanceLeastUsed
	case "least_latency":
		return protocol.LoadBalanceLeastLatency
	case "per_flow":
		return protocol.LoadBalancePerFlow
	case "adaptive":
		return protocol.LoadBalanceAdaptive
	default:
		return protocol.LoadBalanceAdaptive
	}
}

// DefaultConfig returns a default configuration
func DefaultConfig() *BondConfig {
	return &BondConfig{
		Session: SessionConfig{
			LocalEndpoint:    "0.0.0.0:9000",
			RemoteEndpoint:   "",
			DuplicatePackets: false,
			DuplicateMode:    "fastest",
			ReorderBuffer:    1000,
			ReorderTimeout:   "500ms",
			MulticastEnabled: false,
			MulticastGroups:  []string{},
		},
		WANs: []WANInterfaceConfig{},
		Routing: RoutingConfig{
			Mode:                   "adaptive",
			BandwidthResetInterval: "1m",
		},
		FEC: FECConfig{
			Enabled:      false,
			Redundancy:   0.2,
			DataShards:   4,
			ParityShards: 2,
		},
		Monitoring: MonitoringConfig{
			Enabled:         true,
			MetricsInterval: "10s",
			AlertsEnabled:   true,
		},
		Plugins: []PluginConfig{},
	}
}

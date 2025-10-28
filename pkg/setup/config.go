package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Mode represents the operation mode
type Mode string

const (
	ModeStandalone Mode = "standalone"
	ModeClient     Mode = "client"
	ModeServer     Mode = "server"
)

// Config represents the complete application configuration
type Config struct {
	Version  string          `json:"version"`
	Mode     Mode            `json:"mode"`
	WANs     []*WANConfig    `json:"wans"`
	Server   *ServerConfig   `json:"server,omitempty"`
	Security *SecurityConfig `json:"security,omitempty"`
	Health   *HealthConfig   `json:"health,omitempty"`
	Routing  *RoutingConfig  `json:"routing,omitempty"`
	WebUI    *WebUIConfig    `json:"webui,omitempty"`
}

// WANConfig represents a WAN interface configuration
type WANConfig struct {
	ID        uint8  `json:"id"`
	Name      string `json:"name"`
	Interface string `json:"interface"`
	Enabled   bool   `json:"enabled"`
	Weight    int    `json:"weight"`
}

// ServerConfig represents server/client settings
type ServerConfig struct {
	ListenAddress string `json:"listen_address,omitempty"`
	ListenPort    int    `json:"listen_port,omitempty"`
	RemoteAddress string `json:"remote_address,omitempty"`
}

// SecurityConfig represents security settings
type SecurityConfig struct {
	EncryptionEnabled bool   `json:"encryption_enabled"`
	EncryptionType    string `json:"encryption_type,omitempty"`
	PreSharedKey      string `json:"pre_shared_key,omitempty"`
}

// HealthConfig represents health check settings
type HealthConfig struct {
	CheckIntervalMs int      `json:"check_interval_ms"`
	TimeoutMs       int      `json:"timeout_ms"`
	RetryCount      int      `json:"retry_count"`
	CheckHosts      []string `json:"check_hosts"`
}

// RoutingConfig represents routing settings
type RoutingConfig struct {
	Mode string `json:"mode"`
}

// WebUIConfig represents Web UI authentication settings
type WebUIConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  bool   `json:"enabled"`
}

// SaveToFile saves the configuration to a JSON file
func (c *Config) SaveToFile(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to pretty JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// LoadFromFile loads configuration from a JSON file
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SetDefaults sets default values for optional fields
func (c *Config) SetDefaults() {
	if c.Health == nil {
		c.Health = &HealthConfig{
			CheckIntervalMs: 5000,
			TimeoutMs:       3000,
			RetryCount:      3,
			CheckHosts:      []string{"8.8.8.8", "1.1.1.1"},
		}
	}

	if c.Routing == nil {
		c.Routing = &RoutingConfig{
			Mode: "adaptive",
		}
	}

	if c.Security == nil {
		c.Security = &SecurityConfig{
			EncryptionEnabled: true,
			EncryptionType:    "chacha20poly1305",
		}
	}

	if c.WebUI == nil {
		c.WebUI = &WebUIConfig{
			Username: "admin",
			Password: "", // Will be generated during setup
			Enabled:  true,
		}
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.WANs) == 0 {
		return fmt.Errorf("at least one WAN interface must be configured")
	}

	// Validate WAN IDs are unique
	ids := make(map[uint8]bool)
	for _, wan := range c.WANs {
		if ids[wan.ID] {
			return fmt.Errorf("duplicate WAN ID: %d", wan.ID)
		}
		ids[wan.ID] = true

		if wan.Name == "" {
			return fmt.Errorf("WAN %d has no name", wan.ID)
		}

		if wan.Interface == "" {
			return fmt.Errorf("WAN %d (%s) has no interface", wan.ID, wan.Name)
		}
	}

	// Validate server config
	if c.Mode == ModeServer {
		if c.Server == nil || c.Server.ListenPort == 0 {
			return fmt.Errorf("server mode requires listen port")
		}
	}

	if c.Mode == ModeClient {
		if c.Server == nil || c.Server.RemoteAddress == "" {
			// Allow empty remote address for initial setup
			// User can configure it later
		}
	}

	return nil
}

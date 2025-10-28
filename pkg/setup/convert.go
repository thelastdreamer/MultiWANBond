package setup

import (
	"fmt"

	"github.com/thelastdreamer/MultiWANBond/pkg/config"
	"github.com/thelastdreamer/MultiWANBond/pkg/network"
)

// ToBondConfig converts setup.Config to config.BondConfig
func (c *Config) ToBondConfig(detector *network.UniversalDetector) (*config.BondConfig, error) {
	bondConfig := config.DefaultConfig()

	// Configure session based on mode
	if c.Mode == ModeServer {
		if c.Server != nil {
			bondConfig.Session.LocalEndpoint = fmt.Sprintf("%s:%d",
				c.Server.ListenAddress, c.Server.ListenPort)
			bondConfig.Session.RemoteEndpoint = ""
		}
	} else if c.Mode == ModeClient {
		if c.Server != nil {
			bondConfig.Session.LocalEndpoint = "0.0.0.0:0" // Auto-assign port
			bondConfig.Session.RemoteEndpoint = c.Server.RemoteAddress
		}
	} else {
		// Standalone mode - no remote endpoint
		bondConfig.Session.LocalEndpoint = "0.0.0.0:9000"
		bondConfig.Session.RemoteEndpoint = ""
	}

	// Convert WANs
	bondConfig.WANs = make([]config.WANInterfaceConfig, 0, len(c.WANs))

	for _, wan := range c.WANs {
		if !wan.Enabled {
			continue
		}

		// Get interface details
		iface, err := detector.GetInterfaceByName(wan.Interface)
		if err != nil {
			return nil, fmt.Errorf("failed to get interface %s: %w", wan.Interface, err)
		}

		// Get primary IPv4 address
		localAddr := ""
		if len(iface.IPv4Addresses) > 0 {
			localAddr = iface.IPv4Addresses[0].String()
		} else {
			return nil, fmt.Errorf("interface %s has no IPv4 address", wan.Interface)
		}

		// Determine WAN type - default to ethernet
		// User can modify this in the config file later
		wanType := "ethernet"

		// Create WAN config
		wanConfig := config.WANInterfaceConfig{
			ID:                  wan.ID,
			Name:                wan.Name,
			Type:                wanType,
			LocalAddr:           localAddr,
			RemoteAddr:          bondConfig.Session.RemoteEndpoint,
			MaxBandwidth:        0, // Auto-detect
			MaxLatency:          "100ms",
			MaxJitter:           "50ms",
			MaxPacketLoss:       5.0,
			HealthCheckInterval: "200ms",
			FailureThreshold:    3,
			Weight:              wan.Weight,
			Enabled:             wan.Enabled,
		}

		bondConfig.WANs = append(bondConfig.WANs, wanConfig)
	}

	// Configure security
	if c.Security != nil {
		// Security is handled separately by the bonder
		// Just ensure encryption is configured
		if c.Security.EncryptionEnabled {
			// The encryption will be handled by the bonder's security package
		}
	}

	// Configure health checks
	if c.Health != nil {
		// Health check settings are embedded in WAN configs
		// Update interval and timeout for all WANs
		interval := fmt.Sprintf("%dms", c.Health.CheckIntervalMs)
		for i := range bondConfig.WANs {
			bondConfig.WANs[i].HealthCheckInterval = interval
			bondConfig.WANs[i].FailureThreshold = c.Health.RetryCount
		}
	}

	// Configure routing
	if c.Routing != nil {
		bondConfig.Routing.Mode = c.Routing.Mode
	}

	// Configure Web UI
	if c.WebUI != nil {
		bondConfig.WebUI = &config.WebUIConfig{
			Username: c.WebUI.Username,
			Password: c.WebUI.Password,
			Enabled:  c.WebUI.Enabled,
		}
	}

	return bondConfig, nil
}

// SaveAsBondConfig converts and saves the config as a BondConfig
func (c *Config) SaveAsBondConfig(path string, detector *network.UniversalDetector) error {
	bondConfig, err := c.ToBondConfig(detector)
	if err != nil {
		return fmt.Errorf("failed to convert config: %w", err)
	}

	if err := config.SaveBondConfig(path, bondConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

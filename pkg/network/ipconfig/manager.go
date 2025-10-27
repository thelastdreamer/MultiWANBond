package ipconfig

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Manager manages IP configuration for network interfaces
type Manager interface {
	// Apply applies IP configuration to an interface
	Apply(config *IPConfig) error

	// Remove removes IP configuration from an interface
	Remove(interfaceName string) error

	// Get gets current IP configuration for an interface
	Get(interfaceName string) (*InterfaceState, error)

	// List lists all configured interfaces
	List() (map[string]*InterfaceState, error)

	// AddRoute adds a static route
	AddRoute(route *RouteConfig) error

	// RemoveRoute removes a static route
	RemoveRoute(route *RouteConfig) error

	// ListRoutes lists all routes
	ListRoutes() ([]*RouteConfig, error)

	// SetDNS sets DNS servers for an interface
	SetDNS(interfaceName string, servers []string) error

	// GetDNS gets DNS servers for an interface
	GetDNS(interfaceName string) ([]string, error)

	// RenewDHCP renews DHCP lease for an interface
	RenewDHCP(interfaceName string) error

	// ReleaseDHCP releases DHCP lease for an interface
	ReleaseDHCP(interfaceName string) error
}

// UniversalManager provides cross-platform IP configuration management
type UniversalManager struct {
	mu           sync.RWMutex
	platformImpl Manager
	configs      map[string]*IPConfig
}

// newPlatformManager is implemented by platform-specific files
// (manager_init_linux.go, manager_init_windows.go, manager_init_darwin.go)

// NewManager creates a new IP configuration manager
func NewManager() (*UniversalManager, error) {
	impl, err := newPlatformManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create platform manager: %w", err)
	}

	return &UniversalManager{
		platformImpl: impl,
		configs:      make(map[string]*IPConfig),
	}, nil
}

// Apply applies IP configuration to an interface
func (m *UniversalManager) Apply(config *IPConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply using platform implementation
	if err := m.platformImpl.Apply(config); err != nil {
		return err
	}

	// Store in cache
	config.Applied = true
	config.AppliedAt = time.Now()
	m.configs[config.InterfaceName] = config

	return nil
}

// Remove removes IP configuration from an interface
func (m *UniversalManager) Remove(interfaceName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove using platform implementation
	if err := m.platformImpl.Remove(interfaceName); err != nil {
		return err
	}

	// Remove from cache
	delete(m.configs, interfaceName)

	return nil
}

// Get gets current IP configuration for an interface
func (m *UniversalManager) Get(interfaceName string) (*InterfaceState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.platformImpl.Get(interfaceName)
}

// List lists all configured interfaces
func (m *UniversalManager) List() (map[string]*InterfaceState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.platformImpl.List()
}

// AddRoute adds a static route
func (m *UniversalManager) AddRoute(route *RouteConfig) error {
	if route == nil {
		return fmt.Errorf("route cannot be nil")
	}

	// Validate route
	if err := m.validateRoute(route); err != nil {
		return fmt.Errorf("invalid route: %w", err)
	}

	return m.platformImpl.AddRoute(route)
}

// RemoveRoute removes a static route
func (m *UniversalManager) RemoveRoute(route *RouteConfig) error {
	if route == nil {
		return fmt.Errorf("route cannot be nil")
	}

	return m.platformImpl.RemoveRoute(route)
}

// ListRoutes lists all routes
func (m *UniversalManager) ListRoutes() ([]*RouteConfig, error) {
	return m.platformImpl.ListRoutes()
}

// SetDNS sets DNS servers for an interface
func (m *UniversalManager) SetDNS(interfaceName string, servers []string) error {
	// Validate DNS servers
	for _, server := range servers {
		if net.ParseIP(server) == nil {
			return fmt.Errorf("%w: %s", ErrInvalidDNS, server)
		}
	}

	return m.platformImpl.SetDNS(interfaceName, servers)
}

// GetDNS gets DNS servers for an interface
func (m *UniversalManager) GetDNS(interfaceName string) ([]string, error) {
	return m.platformImpl.GetDNS(interfaceName)
}

// RenewDHCP renews DHCP lease for an interface
func (m *UniversalManager) RenewDHCP(interfaceName string) error {
	return m.platformImpl.RenewDHCP(interfaceName)
}

// ReleaseDHCP releases DHCP lease for an interface
func (m *UniversalManager) ReleaseDHCP(interfaceName string) error {
	return m.platformImpl.ReleaseDHCP(interfaceName)
}

// validateConfig validates IP configuration
func (m *UniversalManager) validateConfig(config *IPConfig) error {
	// Interface name is required
	if config.InterfaceName == "" {
		return ErrInterfaceNotFound
	}

	// Validate IPv4 configuration
	if config.IPv4Method == ConfigMethodStatic {
		// Validate IPv4 address
		if err := ValidateIPv4(config.IPv4Address); err != nil {
			return err
		}

		// Validate netmask/CIDR
		if config.IPv4Netmask != "" {
			cidr, err := ParseNetmask(config.IPv4Netmask)
			if err != nil {
				return err
			}
			config.IPv4CIDR = cidr
		} else if config.IPv4CIDR > 0 {
			if err := ValidateCIDR(config.IPv4CIDR, false); err != nil {
				return err
			}
		} else {
			return ErrInvalidNetmask
		}
	}

	// Validate IPv6 configuration
	if config.IPv6Method == ConfigMethodStatic {
		// Validate IPv6 address
		if err := ValidateIPv6(config.IPv6Address); err != nil {
			return err
		}

		// Validate CIDR
		if config.IPv6CIDR == 0 {
			return ErrInvalidCIDR
		}
		if err := ValidateCIDR(config.IPv6CIDR, true); err != nil {
			return err
		}
	}

	// Validate gateway
	if config.GatewayMethod == GatewayMethodStatic || config.GatewayMethod == GatewayMethodMetric {
		if config.IPv4Gateway != "" {
			if err := ValidateIPv4(config.IPv4Gateway); err != nil {
				return ErrInvalidGateway
			}
		}
		if config.IPv6Gateway != "" {
			if err := ValidateIPv6(config.IPv6Gateway); err != nil {
				return ErrInvalidGateway
			}
		}
	}

	// Validate DNS servers
	if config.DNSMethod == DNSMethodStatic {
		if len(config.DNSServers) == 0 {
			return ErrInvalidDNS
		}
		for _, server := range config.DNSServers {
			if net.ParseIP(server) == nil {
				return fmt.Errorf("%w: %s", ErrInvalidDNS, server)
			}
		}
	}

	// Validate MTU
	if config.MTU < 0 || config.MTU > 9000 {
		return fmt.Errorf("invalid MTU: must be between 0 and 9000")
	}

	// Set default DHCP timeout
	if config.DHCPTimeout == 0 {
		config.DHCPTimeout = 30 * time.Second
	}

	return nil
}

// validateRoute validates a route configuration
func (m *UniversalManager) validateRoute(route *RouteConfig) error {
	// Validate destination
	if route.Destination == "" {
		return fmt.Errorf("destination is required")
	}

	_, _, err := net.ParseCIDR(route.Destination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	// Validate gateway
	if route.Gateway != "" {
		if net.ParseIP(route.Gateway) == nil {
			return ErrInvalidGateway
		}
	}

	// Interface or gateway is required
	if route.Interface == "" && route.Gateway == "" {
		return fmt.Errorf("either interface or gateway is required")
	}

	return nil
}

// GetCachedConfig gets cached configuration for an interface
func (m *UniversalManager) GetCachedConfig(interfaceName string) (*IPConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[interfaceName]
	return config, exists
}

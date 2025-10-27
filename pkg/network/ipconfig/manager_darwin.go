//go:build darwin
// +build darwin

package ipconfig

import (
	"fmt"
	"os"
)

// DarwinManager implements IP configuration management for macOS
type DarwinManager struct{}

// newDarwinManager creates a new macOS IP configuration manager
func newDarwinManager() (*DarwinManager, error) {
	// Check if running as root
	if os.Geteuid() != 0 {
		return nil, ErrPermissionDenied
	}

	return &DarwinManager{}, nil
}

// Apply applies IP configuration to an interface
func (m *DarwinManager) Apply(config *IPConfig) error {
	return ErrNotSupported
}

// Remove removes IP configuration from an interface
func (m *DarwinManager) Remove(interfaceName string) error {
	return ErrNotSupported
}

// Get gets current IP configuration for an interface
func (m *DarwinManager) Get(interfaceName string) (*InterfaceState, error) {
	return nil, ErrNotSupported
}

// List lists all configured interfaces
func (m *DarwinManager) List() (map[string]*InterfaceState, error) {
	return nil, ErrNotSupported
}

// AddRoute adds a static route
func (m *DarwinManager) AddRoute(route *RouteConfig) error {
	return ErrNotSupported
}

// RemoveRoute removes a static route
func (m *DarwinManager) RemoveRoute(route *RouteConfig) error {
	return ErrNotSupported
}

// ListRoutes lists all routes
func (m *DarwinManager) ListRoutes() ([]*RouteConfig, error) {
	return nil, ErrNotSupported
}

// SetDNS sets DNS servers for an interface
func (m *DarwinManager) SetDNS(interfaceName string, servers []string) error {
	return ErrNotSupported
}

// GetDNS gets DNS servers for an interface
func (m *DarwinManager) GetDNS(interfaceName string) ([]string, error) {
	return nil, ErrNotSupported
}

// RenewDHCP renews DHCP lease for an interface
func (m *DarwinManager) RenewDHCP(interfaceName string) error {
	return ErrNotSupported
}

// ReleaseDHCP releases DHCP lease for an interface
func (m *DarwinManager) ReleaseDHCP(interfaceName string) error {
	return fmt.Errorf("not implemented for macOS")
}

package vlan

import (
	"fmt"
	"sync"
)

// Manager manages VLAN interfaces
type Manager interface {
	// Create creates a new VLAN interface
	Create(config *Config) (*Interface, error)

	// Delete deletes a VLAN interface
	Delete(name string) error

	// Get gets a VLAN interface by name
	Get(name string) (*Interface, error)

	// List lists all managed VLAN interfaces
	List() ([]*Interface, error)

	// Update updates a VLAN interface configuration
	Update(name string, config *Config) error

	// Exists checks if a VLAN interface exists
	Exists(name string) (bool, error)
}

// UniversalManager provides cross-platform VLAN management
type UniversalManager struct {
	mu           sync.RWMutex
	platformImpl Manager
	vlans        map[string]*Interface
}

// newPlatformManager is implemented by platform-specific files
// (manager_init_linux.go, manager_init_windows.go, manager_init_darwin.go)

// NewManager creates a new VLAN manager
func NewManager() (*UniversalManager, error) {
	impl, err := newPlatformManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create platform manager: %w", err)
	}

	return &UniversalManager{
		platformImpl: impl,
		vlans:        make(map[string]*Interface),
	}, nil
}

// Create creates a new VLAN interface
func (m *UniversalManager) Create(config *Config) (*Interface, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if _, exists := m.vlans[config.Name]; exists {
		return nil, ErrVLANExists
	}

	// Create using platform implementation
	iface, err := m.platformImpl.Create(config)
	if err != nil {
		return nil, err
	}

	// Store in cache
	m.vlans[config.Name] = iface

	return iface, nil
}

// Delete deletes a VLAN interface
func (m *UniversalManager) Delete(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if exists
	if _, exists := m.vlans[name]; !exists {
		return ErrVLANNotFound
	}

	// Delete using platform implementation
	if err := m.platformImpl.Delete(name); err != nil {
		return err
	}

	// Remove from cache
	delete(m.vlans, name)

	return nil
}

// Get gets a VLAN interface by name
func (m *UniversalManager) Get(name string) (*Interface, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if iface, exists := m.vlans[name]; exists {
		return iface, nil
	}

	// Try to get from platform
	return m.platformImpl.Get(name)
}

// List lists all managed VLAN interfaces
func (m *UniversalManager) List() ([]*Interface, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Interface, 0, len(m.vlans))
	for _, iface := range m.vlans {
		result = append(result, iface)
	}

	return result, nil
}

// Update updates a VLAN interface configuration
func (m *UniversalManager) Update(name string, config *Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate configuration
	if err := m.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Check if exists
	if _, exists := m.vlans[name]; !exists {
		return ErrVLANNotFound
	}

	// Update using platform implementation
	return m.platformImpl.Update(name, config)
}

// Exists checks if a VLAN interface exists
func (m *UniversalManager) Exists(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.vlans[name]; exists {
		return true, nil
	}

	return m.platformImpl.Exists(name)
}

// validateConfig validates VLAN configuration
func (m *UniversalManager) validateConfig(config *Config) error {
	// Validate VLAN ID
	if err := ValidateID(config.ID); err != nil {
		return err
	}

	// Validate priority
	if err := ValidatePriority(config.Priority); err != nil {
		return err
	}

	// Validate parent interface
	if config.ParentInterface == "" {
		return ErrInvalidParent
	}

	// Validate name
	if config.Name == "" {
		config.Name = GenerateName(config.ParentInterface, config.ID)
	}

	// Validate MTU
	if config.MTU < 0 || config.MTU > 9000 {
		return fmt.Errorf("invalid MTU: must be between 0 and 9000")
	}

	return nil
}

// Refresh refreshes the list of VLAN interfaces from the system
func (m *UniversalManager) Refresh() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get fresh list from platform
	vlans, err := m.platformImpl.List()
	if err != nil {
		return err
	}

	// Update cache
	m.vlans = make(map[string]*Interface)
	for _, vlan := range vlans {
		m.vlans[vlan.SystemName] = vlan
	}

	return nil
}

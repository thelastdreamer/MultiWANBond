// +build darwin

package bonding

import (
	"fmt"
)

// DarwinManager implements bonding management for macOS
// Note: macOS doesn't have native bonding like Linux, but supports link aggregation
type DarwinManager struct{}

// newDarwinManager creates a new macOS bonding manager
func newDarwinManager() (Manager, error) {
	return &DarwinManager{}, nil
}

// Create creates a new bonding interface
func (m *DarwinManager) Create(config *BondConfig) error {
	return &BondError{
		Op:   "Create",
		Bond: config.Name,
		Err:  fmt.Errorf("%w: macOS bonding requires ifconfig bond commands", ErrNotSupported),
	}
}

// Delete removes a bonding interface
func (m *DarwinManager) Delete(name string) error {
	return &BondError{
		Op:   "Delete",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// Get retrieves information about a bonding interface
func (m *DarwinManager) Get(name string) (*BondInfo, error) {
	return nil, &BondError{
		Op:   "Get",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// List returns all bonding interfaces
func (m *DarwinManager) List() ([]*BondInfo, error) {
	return []*BondInfo{}, nil
}

// Exists checks if a bonding interface exists
func (m *DarwinManager) Exists(name string) (bool, error) {
	return false, nil
}

// Update updates the configuration of an existing bond
func (m *DarwinManager) Update(config *BondConfig) error {
	return &BondError{
		Op:   "Update",
		Bond: config.Name,
		Err:  ErrNotSupported,
	}
}

// AddSlave adds a slave interface to a bond
func (m *DarwinManager) AddSlave(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "AddSlave",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// RemoveSlave removes a slave interface from a bond
func (m *DarwinManager) RemoveSlave(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "RemoveSlave",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// GetSlaves returns all slaves of a bond
func (m *DarwinManager) GetSlaves(bondName string) ([]SlaveInfo, error) {
	return nil, &BondError{
		Op:   "GetSlaves",
		Bond: bondName,
		Err:  ErrNotSupported,
	}
}

// SetPrimary sets the primary slave for active-backup mode
func (m *DarwinManager) SetPrimary(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "SetPrimary",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// SetActive manually sets the active slave
func (m *DarwinManager) SetActive(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "SetActive",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// GetStats retrieves statistics for a bond
func (m *DarwinManager) GetStats(bondName string) (*BondStats, error) {
	return nil, &BondError{
		Op:   "GetStats",
		Bond: bondName,
		Err:  ErrNotSupported,
	}
}

// Enable brings a bond interface up
func (m *DarwinManager) Enable(name string) error {
	return &BondError{
		Op:   "Enable",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// Disable brings a bond interface down
func (m *DarwinManager) Disable(name string) error {
	return &BondError{
		Op:   "Disable",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// SetMTU sets the MTU for a bond interface
func (m *DarwinManager) SetMTU(name string, mtu int) error {
	return &BondError{
		Op:   "SetMTU",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// SetMACAddress sets the MAC address for a bond interface
func (m *DarwinManager) SetMACAddress(name, mac string) error {
	return &BondError{
		Op:   "SetMACAddress",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

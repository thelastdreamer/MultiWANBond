// +build windows

package bonding

import (
	"fmt"
)

// WindowsManager implements bonding management for Windows
// Note: Windows bonding (NIC Teaming) requires PowerShell and is limited
type WindowsManager struct{}

// newWindowsManager creates a new Windows bonding manager
func newWindowsManager() (Manager, error) {
	// Windows NIC Teaming is only available on Windows Server 2012+
	// and Windows 8+ (with limitations)
	return &WindowsManager{}, nil
}

// Create creates a new bonding interface using PowerShell NIC Teaming
func (m *WindowsManager) Create(config *BondConfig) error {
	return &BondError{
		Op:   "Create",
		Bond: config.Name,
		Err:  fmt.Errorf("%w: Windows bonding requires PowerShell NIC Teaming cmdlets (New-NetLbfoTeam)", ErrNotSupported),
	}
}

// Delete removes a bonding interface
func (m *WindowsManager) Delete(name string) error {
	return &BondError{
		Op:   "Delete",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// Get retrieves information about a bonding interface
func (m *WindowsManager) Get(name string) (*BondInfo, error) {
	return nil, &BondError{
		Op:   "Get",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// List returns all bonding interfaces
func (m *WindowsManager) List() ([]*BondInfo, error) {
	return []*BondInfo{}, nil
}

// Exists checks if a bonding interface exists
func (m *WindowsManager) Exists(name string) (bool, error) {
	return false, nil
}

// Update updates the configuration of an existing bond
func (m *WindowsManager) Update(config *BondConfig) error {
	return &BondError{
		Op:   "Update",
		Bond: config.Name,
		Err:  ErrNotSupported,
	}
}

// AddSlave adds a slave interface to a bond
func (m *WindowsManager) AddSlave(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "AddSlave",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// RemoveSlave removes a slave interface from a bond
func (m *WindowsManager) RemoveSlave(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "RemoveSlave",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// GetSlaves returns all slaves of a bond
func (m *WindowsManager) GetSlaves(bondName string) ([]SlaveInfo, error) {
	return nil, &BondError{
		Op:   "GetSlaves",
		Bond: bondName,
		Err:  ErrNotSupported,
	}
}

// SetPrimary sets the primary slave for active-backup mode
func (m *WindowsManager) SetPrimary(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "SetPrimary",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// SetActive manually sets the active slave
func (m *WindowsManager) SetActive(bondName, slaveName string) error {
	return &SlaveError{
		Op:    "SetActive",
		Bond:  bondName,
		Slave: slaveName,
		Err:   ErrNotSupported,
	}
}

// GetStats retrieves statistics for a bond
func (m *WindowsManager) GetStats(bondName string) (*BondStats, error) {
	return nil, &BondError{
		Op:   "GetStats",
		Bond: bondName,
		Err:  ErrNotSupported,
	}
}

// Enable brings a bond interface up
func (m *WindowsManager) Enable(name string) error {
	return &BondError{
		Op:   "Enable",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// Disable brings a bond interface down
func (m *WindowsManager) Disable(name string) error {
	return &BondError{
		Op:   "Disable",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// SetMTU sets the MTU for a bond interface
func (m *WindowsManager) SetMTU(name string, mtu int) error {
	return &BondError{
		Op:   "SetMTU",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

// SetMACAddress sets the MAC address for a bond interface
func (m *WindowsManager) SetMACAddress(name, mac string) error {
	return &BondError{
		Op:   "SetMACAddress",
		Bond: name,
		Err:  ErrNotSupported,
	}
}

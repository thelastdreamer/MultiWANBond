// +build windows

package bridge

import ()

// WindowsManager implements bridge management for Windows
type WindowsManager struct{}

// newWindowsManager creates a new Windows bridge manager
func newWindowsManager() (Manager, error) {
	return &WindowsManager{}, nil
}

// Create creates a new bridge
func (m *WindowsManager) Create(config *BridgeConfig) error {
	return &BridgeError{Op: "Create", Bridge: config.Name, Err: ErrNotSupported}
}

// Delete removes a bridge
func (m *WindowsManager) Delete(name string) error {
	return &BridgeError{Op: "Delete", Bridge: name, Err: ErrNotSupported}
}

// Get retrieves bridge information
func (m *WindowsManager) Get(name string) (*BridgeInfo, error) {
	return nil, &BridgeError{Op: "Get", Bridge: name, Err: ErrNotSupported}
}

// List returns all bridges
func (m *WindowsManager) List() ([]*BridgeInfo, error) {
	return []*BridgeInfo{}, nil
}

// Exists checks if a bridge exists
func (m *WindowsManager) Exists(name string) (bool, error) {
	return false, nil
}

// Update updates bridge configuration
func (m *WindowsManager) Update(config *BridgeConfig) error {
	return &BridgeError{Op: "Update", Bridge: config.Name, Err: ErrNotSupported}
}

// AddPort adds a port to a bridge
func (m *WindowsManager) AddPort(bridgeName, portName string) error {
	return &PortError{Op: "AddPort", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// RemovePort removes a port from a bridge
func (m *WindowsManager) RemovePort(bridgeName, portName string) error {
	return &PortError{Op: "RemovePort", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// GetPorts returns all ports of a bridge
func (m *WindowsManager) GetPorts(bridgeName string) ([]PortInfo, error) {
	return nil, &BridgeError{Op: "GetPorts", Bridge: bridgeName, Err: ErrNotSupported}
}

// SetPortPriority sets port STP priority
func (m *WindowsManager) SetPortPriority(bridgeName, portName string, priority int) error {
	return &PortError{Op: "SetPortPriority", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// SetPortPathCost sets port STP path cost
func (m *WindowsManager) SetPortPathCost(bridgeName, portName string, cost int) error {
	return &PortError{Op: "SetPortPathCost", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// EnableSTP enables Spanning Tree Protocol
func (m *WindowsManager) EnableSTP(bridgeName string) error {
	return &BridgeError{Op: "EnableSTP", Bridge: bridgeName, Err: ErrNotSupported}
}

// DisableSTP disables Spanning Tree Protocol
func (m *WindowsManager) DisableSTP(bridgeName string) error {
	return &BridgeError{Op: "DisableSTP", Bridge: bridgeName, Err: ErrNotSupported}
}

// SetSTPPriority sets bridge STP priority
func (m *WindowsManager) SetSTPPriority(bridgeName string, priority int) error {
	return &BridgeError{Op: "SetSTPPriority", Bridge: bridgeName, Err: ErrNotSupported}
}

// GetFDB retrieves forwarding database
func (m *WindowsManager) GetFDB(bridgeName string) ([]FDBEntry, error) {
	return nil, &BridgeError{Op: "GetFDB", Bridge: bridgeName, Err: ErrNotSupported}
}

// AddFDBEntry adds a static FDB entry
func (m *WindowsManager) AddFDBEntry(bridgeName, mac, port string, vlanID int) error {
	return &BridgeError{Op: "AddFDBEntry", Bridge: bridgeName, Err: ErrNotSupported}
}

// DeleteFDBEntry deletes an FDB entry
func (m *WindowsManager) DeleteFDBEntry(bridgeName, mac string, vlanID int) error {
	return &BridgeError{Op: "DeleteFDBEntry", Bridge: bridgeName, Err: ErrNotSupported}
}

// FlushFDB flushes forwarding database
func (m *WindowsManager) FlushFDB(bridgeName string) error {
	return &BridgeError{Op: "FlushFDB", Bridge: bridgeName, Err: ErrNotSupported}
}

// GetStats retrieves bridge statistics
func (m *WindowsManager) GetStats(bridgeName string) (*BridgeStats, error) {
	return nil, &BridgeError{Op: "GetStats", Bridge: bridgeName, Err: ErrNotSupported}
}

// Enable brings bridge up
func (m *WindowsManager) Enable(name string) error {
	return &BridgeError{Op: "Enable", Bridge: name, Err: ErrNotSupported}
}

// Disable brings bridge down
func (m *WindowsManager) Disable(name string) error {
	return &BridgeError{Op: "Disable", Bridge: name, Err: ErrNotSupported}
}

// SetMTU sets bridge MTU
func (m *WindowsManager) SetMTU(name string, mtu int) error {
	return &BridgeError{Op: "SetMTU", Bridge: name, Err: ErrNotSupported}
}

// SetMACAddress sets bridge MAC address
func (m *WindowsManager) SetMACAddress(name, mac string) error {
	return &BridgeError{Op: "SetMACAddress", Bridge: name, Err: ErrNotSupported}
}

// SetAgeingTime sets MAC ageing time
func (m *WindowsManager) SetAgeingTime(name string, seconds int) error {
	return &BridgeError{Op: "SetAgeingTime", Bridge: name, Err: ErrNotSupported}
}

// SetVLANFiltering enables/disables VLAN filtering
func (m *WindowsManager) SetVLANFiltering(name string, enabled bool) error {
	return &BridgeError{Op: "SetVLANFiltering", Bridge: name, Err: ErrNotSupported}
}

// SetMulticastSnooping enables/disables multicast snooping
func (m *WindowsManager) SetMulticastSnooping(name string, enabled bool) error {
	return &BridgeError{Op: "SetMulticastSnooping", Bridge: name, Err: ErrNotSupported}
}

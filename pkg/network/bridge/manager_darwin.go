// +build darwin

package bridge

import ()

// DarwinManager implements bridge management for macOS
type DarwinManager struct{}

// newDarwinManager creates a new macOS bridge manager
func newDarwinManager() (Manager, error) {
	return &DarwinManager{}, nil
}

// Create creates a new bridge
func (m *DarwinManager) Create(config *BridgeConfig) error {
	return &BridgeError{Op: "Create", Bridge: config.Name, Err: ErrNotSupported}
}

// Delete removes a bridge
func (m *DarwinManager) Delete(name string) error {
	return &BridgeError{Op: "Delete", Bridge: name, Err: ErrNotSupported}
}

// Get retrieves bridge information
func (m *DarwinManager) Get(name string) (*BridgeInfo, error) {
	return nil, &BridgeError{Op: "Get", Bridge: name, Err: ErrNotSupported}
}

// List returns all bridges
func (m *DarwinManager) List() ([]*BridgeInfo, error) {
	return []*BridgeInfo{}, nil
}

// Exists checks if a bridge exists
func (m *DarwinManager) Exists(name string) (bool, error) {
	return false, nil
}

// Update updates bridge configuration
func (m *DarwinManager) Update(config *BridgeConfig) error {
	return &BridgeError{Op: "Update", Bridge: config.Name, Err: ErrNotSupported}
}

// AddPort adds a port to a bridge
func (m *DarwinManager) AddPort(bridgeName, portName string) error {
	return &PortError{Op: "AddPort", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// RemovePort removes a port from a bridge
func (m *DarwinManager) RemovePort(bridgeName, portName string) error {
	return &PortError{Op: "RemovePort", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// GetPorts returns all ports of a bridge
func (m *DarwinManager) GetPorts(bridgeName string) ([]PortInfo, error) {
	return nil, &BridgeError{Op: "GetPorts", Bridge: bridgeName, Err: ErrNotSupported}
}

// SetPortPriority sets port STP priority
func (m *DarwinManager) SetPortPriority(bridgeName, portName string, priority int) error {
	return &PortError{Op: "SetPortPriority", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// SetPortPathCost sets port STP path cost
func (m *DarwinManager) SetPortPathCost(bridgeName, portName string, cost int) error {
	return &PortError{Op: "SetPortPathCost", Bridge: bridgeName, Port: portName, Err: ErrNotSupported}
}

// EnableSTP enables Spanning Tree Protocol
func (m *DarwinManager) EnableSTP(bridgeName string) error {
	return &BridgeError{Op: "EnableSTP", Bridge: bridgeName, Err: ErrNotSupported}
}

// DisableSTP disables Spanning Tree Protocol
func (m *DarwinManager) DisableSTP(bridgeName string) error {
	return &BridgeError{Op: "DisableSTP", Bridge: bridgeName, Err: ErrNotSupported}
}

// SetSTPPriority sets bridge STP priority
func (m *DarwinManager) SetSTPPriority(bridgeName string, priority int) error {
	return &BridgeError{Op: "SetSTPPriority", Bridge: bridgeName, Err: ErrNotSupported}
}

// GetFDB retrieves forwarding database
func (m *DarwinManager) GetFDB(bridgeName string) ([]FDBEntry, error) {
	return nil, &BridgeError{Op: "GetFDB", Bridge: bridgeName, Err: ErrNotSupported}
}

// AddFDBEntry adds a static FDB entry
func (m *DarwinManager) AddFDBEntry(bridgeName, mac, port string, vlanID int) error {
	return &BridgeError{Op: "AddFDBEntry", Bridge: bridgeName, Err: ErrNotSupported}
}

// DeleteFDBEntry deletes an FDB entry
func (m *DarwinManager) DeleteFDBEntry(bridgeName, mac string, vlanID int) error {
	return &BridgeError{Op: "DeleteFDBEntry", Bridge: bridgeName, Err: ErrNotSupported}
}

// FlushFDB flushes forwarding database
func (m *DarwinManager) FlushFDB(bridgeName string) error {
	return &BridgeError{Op: "FlushFDB", Bridge: bridgeName, Err: ErrNotSupported}
}

// GetStats retrieves bridge statistics
func (m *DarwinManager) GetStats(bridgeName string) (*BridgeStats, error) {
	return nil, &BridgeError{Op: "GetStats", Bridge: bridgeName, Err: ErrNotSupported}
}

// Enable brings bridge up
func (m *DarwinManager) Enable(name string) error {
	return &BridgeError{Op: "Enable", Bridge: name, Err: ErrNotSupported}
}

// Disable brings bridge down
func (m *DarwinManager) Disable(name string) error {
	return &BridgeError{Op: "Disable", Bridge: name, Err: ErrNotSupported}
}

// SetMTU sets bridge MTU
func (m *DarwinManager) SetMTU(name string, mtu int) error {
	return &BridgeError{Op: "SetMTU", Bridge: name, Err: ErrNotSupported}
}

// SetMACAddress sets bridge MAC address
func (m *DarwinManager) SetMACAddress(name, mac string) error {
	return &BridgeError{Op: "SetMACAddress", Bridge: name, Err: ErrNotSupported}
}

// SetAgeingTime sets MAC ageing time
func (m *DarwinManager) SetAgeingTime(name string, seconds int) error {
	return &BridgeError{Op: "SetAgeingTime", Bridge: name, Err: ErrNotSupported}
}

// SetVLANFiltering enables/disables VLAN filtering
func (m *DarwinManager) SetVLANFiltering(name string, enabled bool) error {
	return &BridgeError{Op: "SetVLANFiltering", Bridge: name, Err: ErrNotSupported}
}

// SetMulticastSnooping enables/disables multicast snooping
func (m *DarwinManager) SetMulticastSnooping(name string, enabled bool) error {
	return &BridgeError{Op: "SetMulticastSnooping", Bridge: name, Err: ErrNotSupported}
}

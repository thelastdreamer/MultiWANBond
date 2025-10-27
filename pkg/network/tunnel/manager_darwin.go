// +build darwin

package tunnel

import ()

// DarwinManager implements tunnel management for macOS
type DarwinManager struct{}

// newDarwinManager creates a new macOS tunnel manager
func newDarwinManager() (Manager, error) {
	return &DarwinManager{}, nil
}

// Create creates a new tunnel
func (m *DarwinManager) Create(config *TunnelConfig) error {
	return &TunnelError{Op: "Create", Tunnel: config.Name, Type: config.Type, Err: ErrNotSupported}
}

// Delete removes a tunnel
func (m *DarwinManager) Delete(name string) error {
	return &TunnelError{Op: "Delete", Tunnel: name, Err: ErrNotSupported}
}

// Get retrieves tunnel information
func (m *DarwinManager) Get(name string) (*TunnelInfo, error) {
	return nil, &TunnelError{Op: "Get", Tunnel: name, Err: ErrNotSupported}
}

// List returns all tunnels
func (m *DarwinManager) List() ([]*TunnelInfo, error) {
	return []*TunnelInfo{}, nil
}

// Exists checks if a tunnel exists
func (m *DarwinManager) Exists(name string) (bool, error) {
	return false, nil
}

// Update updates tunnel configuration
func (m *DarwinManager) Update(config *TunnelConfig) error {
	return &TunnelError{Op: "Update", Tunnel: config.Name, Type: config.Type, Err: ErrNotSupported}
}

// Enable brings tunnel up
func (m *DarwinManager) Enable(name string) error {
	return &TunnelError{Op: "Enable", Tunnel: name, Err: ErrNotSupported}
}

// Disable brings tunnel down
func (m *DarwinManager) Disable(name string) error {
	return &TunnelError{Op: "Disable", Tunnel: name, Err: ErrNotSupported}
}

// SetMTU sets tunnel MTU
func (m *DarwinManager) SetMTU(name string, mtu int) error {
	return &TunnelError{Op: "SetMTU", Tunnel: name, Err: ErrNotSupported}
}

// SetLocalAddress sets local endpoint
func (m *DarwinManager) SetLocalAddress(name, address string) error {
	return &TunnelError{Op: "SetLocalAddress", Tunnel: name, Err: ErrNotSupported}
}

// SetRemoteAddress sets remote endpoint
func (m *DarwinManager) SetRemoteAddress(name, address string) error {
	return &TunnelError{Op: "SetRemoteAddress", Tunnel: name, Err: ErrNotSupported}
}

// GetStats retrieves tunnel statistics
func (m *DarwinManager) GetStats(name string) (*TunnelStats, error) {
	return nil, &TunnelError{Op: "GetStats", Tunnel: name, Err: ErrNotSupported}
}

// AddWireGuardPeer adds a WireGuard peer
func (m *DarwinManager) AddWireGuardPeer(tunnelName string, peer *WireGuardPeer) error {
	return &TunnelError{Op: "AddWireGuardPeer", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// RemoveWireGuardPeer removes a WireGuard peer
func (m *DarwinManager) RemoveWireGuardPeer(tunnelName, publicKey string) error {
	return &TunnelError{Op: "RemoveWireGuardPeer", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// GetWireGuardPeers retrieves all WireGuard peers
func (m *DarwinManager) GetWireGuardPeers(tunnelName string) ([]WireGuardPeerInfo, error) {
	return nil, &TunnelError{Op: "GetWireGuardPeers", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// SetWireGuardPrivateKey sets WireGuard private key
func (m *DarwinManager) SetWireGuardPrivateKey(tunnelName, privateKey string) error {
	return &TunnelError{Op: "SetWireGuardPrivateKey", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// SetWireGuardListenPort sets WireGuard listen port
func (m *DarwinManager) SetWireGuardListenPort(tunnelName string, port int) error {
	return &TunnelError{Op: "SetWireGuardListenPort", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// SetGREKey sets GRE key
func (m *DarwinManager) SetGREKey(tunnelName string, key uint32) error {
	return &TunnelError{Op: "SetGREKey", Tunnel: tunnelName, Type: TunnelTypeGRE, Err: ErrNotSupported}
}

// SetGREChecksum enables/disables GRE checksum
func (m *DarwinManager) SetGREChecksum(tunnelName string, enabled bool) error {
	return &TunnelError{Op: "SetGREChecksum", Tunnel: tunnelName, Type: TunnelTypeGRE, Err: ErrNotSupported}
}

// SetGRESeq enables/disables GRE sequence numbers
func (m *DarwinManager) SetGRESeq(tunnelName string, enabled bool) error {
	return &TunnelError{Op: "SetGRESeq", Tunnel: tunnelName, Type: TunnelTypeGRE, Err: ErrNotSupported}
}

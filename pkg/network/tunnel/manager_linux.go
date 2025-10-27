// +build linux

package tunnel

import (
	"fmt"
)

// LinuxManager implements tunnel management for Linux using netlink
// Full implementation would use github.com/vishvananda/netlink
type LinuxManager struct{}

// newLinuxManager creates a new Linux tunnel manager
func newLinuxManager() (Manager, error) {
	return &LinuxManager{}, nil
}

// Create creates a new tunnel
func (m *LinuxManager) Create(config *TunnelConfig) error {
	return &TunnelError{Op: "Create", Tunnel: config.Name, Type: config.Type, Err: fmt.Errorf("%w: full implementation pending", ErrNotSupported)}
}

// Delete removes a tunnel
func (m *LinuxManager) Delete(name string) error {
	return &TunnelError{Op: "Delete", Tunnel: name, Err: ErrNotSupported}
}

// Get retrieves tunnel information
func (m *LinuxManager) Get(name string) (*TunnelInfo, error) {
	return nil, &TunnelError{Op: "Get", Tunnel: name, Err: ErrNotSupported}
}

// List returns all tunnels
func (m *LinuxManager) List() ([]*TunnelInfo, error) {
	return []*TunnelInfo{}, nil
}

// Exists checks if a tunnel exists
func (m *LinuxManager) Exists(name string) (bool, error) {
	return false, nil
}

// Update updates tunnel configuration
func (m *LinuxManager) Update(config *TunnelConfig) error {
	return &TunnelError{Op: "Update", Tunnel: config.Name, Type: config.Type, Err: ErrNotSupported}
}

// Enable brings tunnel up
func (m *LinuxManager) Enable(name string) error {
	return &TunnelError{Op: "Enable", Tunnel: name, Err: ErrNotSupported}
}

// Disable brings tunnel down
func (m *LinuxManager) Disable(name string) error {
	return &TunnelError{Op: "Disable", Tunnel: name, Err: ErrNotSupported}
}

// SetMTU sets tunnel MTU
func (m *LinuxManager) SetMTU(name string, mtu int) error {
	return &TunnelError{Op: "SetMTU", Tunnel: name, Err: ErrNotSupported}
}

// SetLocalAddress sets local endpoint
func (m *LinuxManager) SetLocalAddress(name, address string) error {
	return &TunnelError{Op: "SetLocalAddress", Tunnel: name, Err: ErrNotSupported}
}

// SetRemoteAddress sets remote endpoint
func (m *LinuxManager) SetRemoteAddress(name, address string) error {
	return &TunnelError{Op: "SetRemoteAddress", Tunnel: name, Err: ErrNotSupported}
}

// GetStats retrieves tunnel statistics
func (m *LinuxManager) GetStats(name string) (*TunnelStats, error) {
	return nil, &TunnelError{Op: "GetStats", Tunnel: name, Err: ErrNotSupported}
}

// AddWireGuardPeer adds a WireGuard peer
func (m *LinuxManager) AddWireGuardPeer(tunnelName string, peer *WireGuardPeer) error {
	return &TunnelError{Op: "AddWireGuardPeer", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// RemoveWireGuardPeer removes a WireGuard peer
func (m *LinuxManager) RemoveWireGuardPeer(tunnelName, publicKey string) error {
	return &TunnelError{Op: "RemoveWireGuardPeer", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// GetWireGuardPeers retrieves all WireGuard peers
func (m *LinuxManager) GetWireGuardPeers(tunnelName string) ([]WireGuardPeerInfo, error) {
	return nil, &TunnelError{Op: "GetWireGuardPeers", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// SetWireGuardPrivateKey sets WireGuard private key
func (m *LinuxManager) SetWireGuardPrivateKey(tunnelName, privateKey string) error {
	return &TunnelError{Op: "SetWireGuardPrivateKey", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// SetWireGuardListenPort sets WireGuard listen port
func (m *LinuxManager) SetWireGuardListenPort(tunnelName string, port int) error {
	return &TunnelError{Op: "SetWireGuardListenPort", Tunnel: tunnelName, Type: TunnelTypeWireGuard, Err: ErrNotSupported}
}

// SetGREKey sets GRE key
func (m *LinuxManager) SetGREKey(tunnelName string, key uint32) error {
	return &TunnelError{Op: "SetGREKey", Tunnel: tunnelName, Type: TunnelTypeGRE, Err: ErrNotSupported}
}

// SetGREChecksum enables/disables GRE checksum
func (m *LinuxManager) SetGREChecksum(tunnelName string, enabled bool) error {
	return &TunnelError{Op: "SetGREChecksum", Tunnel: tunnelName, Type: TunnelTypeGRE, Err: ErrNotSupported}
}

// SetGRESeq enables/disables GRE sequence numbers
func (m *LinuxManager) SetGRESeq(tunnelName string, enabled bool) error {
	return &TunnelError{Op: "SetGRESeq", Tunnel: tunnelName, Type: TunnelTypeGRE, Err: ErrNotSupported}
}

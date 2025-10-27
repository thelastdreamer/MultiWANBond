package tunnel

// Manager provides cross-platform tunnel interface management
type Manager interface {
	// Create creates a new tunnel with the given configuration
	Create(config *TunnelConfig) error

	// Delete removes a tunnel
	Delete(name string) error

	// Get retrieves information about a specific tunnel
	Get(name string) (*TunnelInfo, error)

	// List returns all tunnels
	List() ([]*TunnelInfo, error)

	// Exists checks if a tunnel exists
	Exists(name string) (bool, error)

	// Update updates the configuration of an existing tunnel
	Update(config *TunnelConfig) error

	// Enable brings a tunnel interface up
	Enable(name string) error

	// Disable brings a tunnel interface down
	Disable(name string) error

	// SetMTU sets the MTU for a tunnel
	SetMTU(name string, mtu int) error

	// SetLocalAddress sets the local endpoint address
	SetLocalAddress(name, address string) error

	// SetRemoteAddress sets the remote endpoint address
	SetRemoteAddress(name, address string) error

	// GetStats retrieves statistics for a tunnel
	GetStats(name string) (*TunnelStats, error)

	// WireGuard-specific operations

	// AddWireGuardPeer adds a peer to a WireGuard tunnel
	AddWireGuardPeer(tunnelName string, peer *WireGuardPeer) error

	// RemoveWireGuardPeer removes a peer from a WireGuard tunnel
	RemoveWireGuardPeer(tunnelName, publicKey string) error

	// GetWireGuardPeers retrieves all peers of a WireGuard tunnel
	GetWireGuardPeers(tunnelName string) ([]WireGuardPeerInfo, error)

	// SetWireGuardPrivateKey sets the private key for a WireGuard tunnel
	SetWireGuardPrivateKey(tunnelName, privateKey string) error

	// SetWireGuardListenPort sets the listen port for a WireGuard tunnel
	SetWireGuardListenPort(tunnelName string, port int) error

	// GRE-specific operations

	// SetGREKey sets the GRE key
	SetGREKey(tunnelName string, key uint32) error

	// SetGREChecksum enables or disables GRE checksum
	SetGREChecksum(tunnelName string, enabled bool) error

	// SetGRESeq enables or disables GRE sequence numbers
	SetGRESeq(tunnelName string, enabled bool) error
}

// NewManager creates a platform-specific tunnel manager
func NewManager() (Manager, error) {
	return newPlatformManager()
}

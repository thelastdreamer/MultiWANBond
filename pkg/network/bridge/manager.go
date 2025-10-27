package bridge

// Manager provides cross-platform bridge interface management
type Manager interface {
	// Create creates a new bridge with the given configuration
	Create(config *BridgeConfig) error

	// Delete removes a bridge
	Delete(name string) error

	// Get retrieves information about a specific bridge
	Get(name string) (*BridgeInfo, error)

	// List returns all bridges
	List() ([]*BridgeInfo, error)

	// Exists checks if a bridge exists
	Exists(name string) (bool, error)

	// Update updates the configuration of an existing bridge
	Update(config *BridgeConfig) error

	// AddPort adds a port to a bridge
	AddPort(bridgeName, portName string) error

	// RemovePort removes a port from a bridge
	RemovePort(bridgeName, portName string) error

	// GetPorts returns all ports of a bridge
	GetPorts(bridgeName string) ([]PortInfo, error)

	// SetPortPriority sets the STP priority for a port
	SetPortPriority(bridgeName, portName string, priority int) error

	// SetPortPathCost sets the STP path cost for a port
	SetPortPathCost(bridgeName, portName string, cost int) error

	// EnableSTP enables Spanning Tree Protocol
	EnableSTP(bridgeName string) error

	// DisableSTP disables Spanning Tree Protocol
	DisableSTP(bridgeName string) error

	// SetSTPPriority sets the bridge priority for STP
	SetSTPPriority(bridgeName string, priority int) error

	// GetFDB retrieves the forwarding database
	GetFDB(bridgeName string) ([]FDBEntry, error)

	// AddFDBEntry adds a static FDB entry
	AddFDBEntry(bridgeName, mac, port string, vlanID int) error

	// DeleteFDBEntry deletes an FDB entry
	DeleteFDBEntry(bridgeName, mac string, vlanID int) error

	// FlushFDB flushes the forwarding database
	FlushFDB(bridgeName string) error

	// GetStats retrieves statistics for a bridge
	GetStats(bridgeName string) (*BridgeStats, error)

	// Enable brings a bridge interface up
	Enable(name string) error

	// Disable brings a bridge interface down
	Disable(name string) error

	// SetMTU sets the MTU for a bridge
	SetMTU(name string, mtu int) error

	// SetMACAddress sets the MAC address for a bridge
	SetMACAddress(name, mac string) error

	// SetAgeingTime sets the MAC address ageing time
	SetAgeingTime(name string, seconds int) error

	// SetVLANFiltering enables or disables VLAN filtering
	SetVLANFiltering(name string, enabled bool) error

	// SetMulticastSnooping enables or disables multicast snooping
	SetMulticastSnooping(name string, enabled bool) error
}

// NewManager creates a platform-specific bridge manager
func NewManager() (Manager, error) {
	return newPlatformManager()
}

package bonding

// Manager provides cross-platform bonding interface management
type Manager interface {
	// Create creates a new bonding interface with the given configuration
	Create(config *BondConfig) error

	// Delete removes a bonding interface
	Delete(name string) error

	// Get retrieves information about a specific bonding interface
	Get(name string) (*BondInfo, error)

	// List returns all bonding interfaces
	List() ([]*BondInfo, error)

	// Exists checks if a bonding interface exists
	Exists(name string) (bool, error)

	// Update updates the configuration of an existing bond
	Update(config *BondConfig) error

	// AddSlave adds a slave interface to a bond
	AddSlave(bondName, slaveName string) error

	// RemoveSlave removes a slave interface from a bond
	RemoveSlave(bondName, slaveName string) error

	// GetSlaves returns all slaves of a bond
	GetSlaves(bondName string) ([]SlaveInfo, error)

	// SetPrimary sets the primary slave for active-backup mode
	SetPrimary(bondName, slaveName string) error

	// SetActive manually sets the active slave (for testing/debugging)
	SetActive(bondName, slaveName string) error

	// GetStats retrieves statistics for a bond
	GetStats(bondName string) (*BondStats, error)

	// Enable brings a bond interface up
	Enable(name string) error

	// Disable brings a bond interface down
	Disable(name string) error

	// SetMTU sets the MTU for a bond interface
	SetMTU(name string, mtu int) error

	// SetMACAddress sets the MAC address for a bond interface
	SetMACAddress(name, mac string) error
}

// NewManager creates a platform-specific bonding manager
func NewManager() (Manager, error) {
	return newPlatformManager()
}

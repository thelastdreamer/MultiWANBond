package bonding

import "fmt"

// Error types for bonding operations
var (
	// ErrBondNotFound is returned when a bond interface doesn't exist
	ErrBondNotFound = fmt.Errorf("bond interface not found")

	// ErrBondExists is returned when trying to create a bond that already exists
	ErrBondExists = fmt.Errorf("bond interface already exists")

	// ErrInvalidBondName is returned when bond name is invalid
	ErrInvalidBondName = fmt.Errorf("invalid bond interface name")

	// ErrInvalidMode is returned when bonding mode is invalid
	ErrInvalidMode = fmt.Errorf("invalid bonding mode")

	// ErrNoSlaves is returned when trying to create a bond with no slaves
	ErrNoSlaves = fmt.Errorf("no slave interfaces specified")

	// ErrSlaveNotFound is returned when a slave interface doesn't exist
	ErrSlaveNotFound = fmt.Errorf("slave interface not found")

	// ErrSlaveAlreadyInBond is returned when trying to add a slave that's already bonded
	ErrSlaveAlreadyInBond = fmt.Errorf("slave interface already in a bond")

	// ErrSlaveHasIP is returned when trying to bond an interface with IP addresses
	ErrSlaveHasIP = fmt.Errorf("slave interface has IP addresses assigned")

	// ErrInvalidMIIMonInterval is returned when MII monitoring interval is invalid
	ErrInvalidMIIMonInterval = fmt.Errorf("invalid MII monitoring interval")

	// ErrInvalidDelay is returned when up/down delay is not a multiple of MII interval
	ErrInvalidDelay = fmt.Errorf("delay must be a multiple of MII monitoring interval")

	// ErrInvalidARPInterval is returned when ARP monitoring interval is invalid
	ErrInvalidARPInterval = fmt.Errorf("invalid ARP monitoring interval")

	// ErrNoARPTargets is returned when ARP monitoring is enabled but no targets specified
	ErrNoARPTargets = fmt.Errorf("ARP monitoring enabled but no targets specified")

	// ErrMIIAndARPBothEnabled is returned when both MII and ARP monitoring are enabled
	ErrMIIAndARPBothEnabled = fmt.Errorf("cannot enable both MII and ARP monitoring")

	// ErrInvalidXmitHashPolicy is returned when transmit hash policy is invalid
	ErrInvalidXmitHashPolicy = fmt.Errorf("invalid transmit hash policy")

	// ErrInvalidLACPRate is returned when LACP rate is invalid
	ErrInvalidLACPRate = fmt.Errorf("invalid LACP rate")

	// ErrInvalidADSelect is returned when AD select policy is invalid
	ErrInvalidADSelect = fmt.Errorf("invalid AD select policy")

	// ErrLACPRequires8023AD is returned when LACP settings used without 802.3ad mode
	ErrLACPRequires8023AD = fmt.Errorf("LACP settings require 802.3ad bonding mode")

	// ErrPrimaryNotInSlaves is returned when primary slave is not in slaves list
	ErrPrimaryNotInSlaves = fmt.Errorf("primary slave not in slaves list")

	// ErrInvalidMTU is returned when MTU is invalid
	ErrInvalidMTU = fmt.Errorf("invalid MTU value")

	// ErrInvalidMACAddress is returned when MAC address format is invalid
	ErrInvalidMACAddress = fmt.Errorf("invalid MAC address format")

	// ErrPermissionDenied is returned when operation requires root/admin privileges
	ErrPermissionDenied = fmt.Errorf("operation requires administrator privileges")

	// ErrNotSupported is returned when operation is not supported on this platform
	ErrNotSupported = fmt.Errorf("bonding not supported on this platform")

	// ErrKernelModuleNotLoaded is returned when bonding kernel module is not loaded
	ErrKernelModuleNotLoaded = fmt.Errorf("bonding kernel module not loaded")

	// ErrSlaveInUse is returned when trying to remove a slave that's the only active link
	ErrSlaveInUse = fmt.Errorf("cannot remove slave: it's the only active link")

	// ErrBondHasSlaves is returned when trying to delete a bond that still has slaves
	ErrBondHasSlaves = fmt.Errorf("cannot delete bond: remove all slaves first")
)

// BondError wraps an error with additional bond-specific context
type BondError struct {
	Op   string // Operation that failed (e.g., "Create", "AddSlave")
	Bond string // Bond interface name
	Err  error  // Underlying error
}

func (e *BondError) Error() string {
	if e.Bond != "" {
		return fmt.Sprintf("bond %s: %s: %v", e.Bond, e.Op, e.Err)
	}
	return fmt.Sprintf("bond: %s: %v", e.Op, e.Err)
}

func (e *BondError) Unwrap() error {
	return e.Err
}

// SlaveError wraps an error with slave-specific context
type SlaveError struct {
	Op    string // Operation that failed
	Bond  string // Bond interface name
	Slave string // Slave interface name
	Err   error  // Underlying error
}

func (e *SlaveError) Error() string {
	return fmt.Sprintf("bond %s: slave %s: %s: %v", e.Bond, e.Slave, e.Op, e.Err)
}

func (e *SlaveError) Unwrap() error {
	return e.Err
}

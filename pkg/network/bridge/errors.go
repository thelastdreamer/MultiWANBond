package bridge

import "fmt"

// Error types for bridge operations
var (
	// ErrBridgeNotFound is returned when a bridge doesn't exist
	ErrBridgeNotFound = fmt.Errorf("bridge not found")

	// ErrBridgeExists is returned when trying to create a bridge that already exists
	ErrBridgeExists = fmt.Errorf("bridge already exists")

	// ErrInvalidBridgeName is returned when bridge name is invalid
	ErrInvalidBridgeName = fmt.Errorf("invalid bridge name")

	// ErrNoPorts is returned when trying to create a bridge with no ports
	ErrNoPorts = fmt.Errorf("no ports specified")

	// ErrPortNotFound is returned when a port interface doesn't exist
	ErrPortNotFound = fmt.Errorf("port interface not found")

	// ErrPortAlreadyInBridge is returned when trying to add a port that's already in a bridge
	ErrPortAlreadyInBridge = fmt.Errorf("port already in a bridge")

	// ErrPortHasIP is returned when trying to bridge an interface with IP addresses
	ErrPortHasIP = fmt.Errorf("port interface has IP addresses assigned")

	// ErrInvalidSTPPriority is returned when STP priority is invalid
	ErrInvalidSTPPriority = fmt.Errorf("invalid STP priority (must be 0-65535)")

	// ErrInvalidSTPForwardDelay is returned when forward delay is invalid
	ErrInvalidSTPForwardDelay = fmt.Errorf("invalid STP forward delay (must be 4-30 seconds)")

	// ErrInvalidSTPHelloTime is returned when hello time is invalid
	ErrInvalidSTPHelloTime = fmt.Errorf("invalid STP hello time (must be 1-10 seconds)")

	// ErrInvalidSTPMaxAge is returned when max age is invalid
	ErrInvalidSTPMaxAge = fmt.Errorf("invalid STP max age (must be 6-40 seconds)")

	// ErrInvalidAgeingTime is returned when ageing time is invalid
	ErrInvalidAgeingTime = fmt.Errorf("invalid ageing time")

	// ErrInvalidVLANID is returned when VLAN ID is invalid
	ErrInvalidVLANID = fmt.Errorf("invalid VLAN ID (must be 1-4094)")

	// ErrInvalidMTU is returned when MTU is invalid
	ErrInvalidMTU = fmt.Errorf("invalid MTU value")

	// ErrInvalidMACAddress is returned when MAC address format is invalid
	ErrInvalidMACAddress = fmt.Errorf("invalid MAC address format")

	// ErrPermissionDenied is returned when operation requires root/admin privileges
	ErrPermissionDenied = fmt.Errorf("operation requires administrator privileges")

	// ErrNotSupported is returned when operation is not supported on this platform
	ErrNotSupported = fmt.Errorf("bridging not supported on this platform")

	// ErrBridgeHasPorts is returned when trying to delete a bridge that still has ports
	ErrBridgeHasPorts = fmt.Errorf("cannot delete bridge: remove all ports first")

	// ErrFDBEntryNotFound is returned when an FDB entry doesn't exist
	ErrFDBEntryNotFound = fmt.Errorf("FDB entry not found")
)

// BridgeError wraps an error with additional bridge-specific context
type BridgeError struct {
	Op     string // Operation that failed
	Bridge string // Bridge name
	Err    error  // Underlying error
}

func (e *BridgeError) Error() string {
	if e.Bridge != "" {
		return fmt.Sprintf("bridge %s: %s: %v", e.Bridge, e.Op, e.Err)
	}
	return fmt.Sprintf("bridge: %s: %v", e.Op, e.Err)
}

func (e *BridgeError) Unwrap() error {
	return e.Err
}

// PortError wraps an error with port-specific context
type PortError struct {
	Op     string // Operation that failed
	Bridge string // Bridge name
	Port   string // Port name
	Err    error  // Underlying error
}

func (e *PortError) Error() string {
	return fmt.Sprintf("bridge %s: port %s: %s: %v", e.Bridge, e.Port, e.Op, e.Err)
}

func (e *PortError) Unwrap() error {
	return e.Err
}

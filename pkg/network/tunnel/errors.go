package tunnel

import "fmt"

// Error types for tunnel operations
var (
	// ErrTunnelNotFound is returned when a tunnel doesn't exist
	ErrTunnelNotFound = fmt.Errorf("tunnel not found")

	// ErrTunnelExists is returned when trying to create a tunnel that already exists
	ErrTunnelExists = fmt.Errorf("tunnel already exists")

	// ErrInvalidTunnelName is returned when tunnel name is invalid
	ErrInvalidTunnelName = fmt.Errorf("invalid tunnel name")

	// ErrInvalidTunnelType is returned when tunnel type is invalid
	ErrInvalidTunnelType = fmt.Errorf("invalid tunnel type")

	// ErrInvalidAddress is returned when address is invalid
	ErrInvalidAddress = fmt.Errorf("invalid address")

	// ErrLocalAddressRequired is returned when local address is required but not provided
	ErrLocalAddressRequired = fmt.Errorf("local address is required")

	// ErrRemoteAddressRequired is returned when remote address is required but not provided
	ErrRemoteAddressRequired = fmt.Errorf("remote address is required")

	// ErrInvalidTTL is returned when TTL is invalid
	ErrInvalidTTL = fmt.Errorf("invalid TTL (must be 0-255)")

	// ErrInvalidMTU is returned when MTU is invalid
	ErrInvalidMTU = fmt.Errorf("invalid MTU value")

	// ErrInvalidKey is returned when GRE key is invalid
	ErrInvalidKey = fmt.Errorf("invalid GRE key")

	// ErrInvalidPort is returned when port is invalid
	ErrInvalidPort = fmt.Errorf("invalid port number")

	// ErrInvalidVNI is returned when VXLAN network identifier is invalid
	ErrInvalidVNI = fmt.Errorf("invalid VNI (must be 1-16777215)")

	// ErrWireGuardKeyInvalid is returned when WireGuard key is invalid
	ErrWireGuardKeyInvalid = fmt.Errorf("invalid WireGuard key")

	// ErrWireGuardNoPeers is returned when WireGuard tunnel has no peers
	ErrWireGuardNoPeers = fmt.Errorf("WireGuard tunnel requires at least one peer")

	// ErrPermissionDenied is returned when operation requires root/admin privileges
	ErrPermissionDenied = fmt.Errorf("operation requires administrator privileges")

	// ErrNotSupported is returned when operation is not supported on this platform
	ErrNotSupported = fmt.Errorf("tunnel type not supported on this platform")

	// ErrInvalidMACAddress is returned when MAC address format is invalid
	ErrInvalidMACAddress = fmt.Errorf("invalid MAC address format")
)

// TunnelError wraps an error with additional tunnel-specific context
type TunnelError struct {
	Op     string // Operation that failed
	Tunnel string // Tunnel name
	Type   TunnelType
	Err    error // Underlying error
}

func (e *TunnelError) Error() string {
	if e.Tunnel != "" {
		return fmt.Sprintf("tunnel %s (%s): %s: %v", e.Tunnel, e.Type, e.Op, e.Err)
	}
	return fmt.Sprintf("tunnel (%s): %s: %v", e.Type, e.Op, e.Err)
}

func (e *TunnelError) Unwrap() error {
	return e.Err
}

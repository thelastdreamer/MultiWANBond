package vlan

import "errors"

var (
	// ErrInvalidVLANID is returned when VLAN ID is out of range
	ErrInvalidVLANID = errors.New("VLAN ID must be between 1 and 4094")

	// ErrReservedVLANID is returned when trying to use a reserved VLAN ID
	ErrReservedVLANID = errors.New("VLAN ID is reserved")

	// ErrInvalidPriority is returned when priority is out of range
	ErrInvalidPriority = errors.New("802.1p priority must be between 0 and 7")

	// ErrParentNotFound is returned when parent interface doesn't exist
	ErrParentNotFound = errors.New("parent interface not found")

	// ErrVLANExists is returned when VLAN already exists
	ErrVLANExists = errors.New("VLAN interface already exists")

	// ErrVLANNotFound is returned when VLAN doesn't exist
	ErrVLANNotFound = errors.New("VLAN interface not found")

	// ErrInvalidParent is returned when parent interface is invalid
	ErrInvalidParent = errors.New("invalid parent interface")

	// ErrNotSupported is returned when operation is not supported on this platform
	ErrNotSupported = errors.New("operation not supported on this platform")

	// ErrPermissionDenied is returned when lacking permissions
	ErrPermissionDenied = errors.New("permission denied (try running as root/administrator)")

	// ErrVLANInUse is returned when trying to delete a VLAN that's in use
	ErrVLANInUse = errors.New("VLAN interface is in use")
)

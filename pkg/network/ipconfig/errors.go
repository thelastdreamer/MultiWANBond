package ipconfig

import "errors"

var (
	// ErrInvalidIPAddress is returned when IP address is invalid
	ErrInvalidIPAddress = errors.New("invalid IP address")

	// ErrInvalidNetmask is returned when netmask is invalid
	ErrInvalidNetmask = errors.New("invalid netmask")

	// ErrInvalidCIDR is returned when CIDR notation is invalid
	ErrInvalidCIDR = errors.New("invalid CIDR notation")

	// ErrInvalidGateway is returned when gateway address is invalid
	ErrInvalidGateway = errors.New("invalid gateway address")

	// ErrInvalidDNS is returned when DNS server address is invalid
	ErrInvalidDNS = errors.New("invalid DNS server address")

	// ErrInterfaceNotFound is returned when interface doesn't exist
	ErrInterfaceNotFound = errors.New("network interface not found")

	// ErrInterfaceDown is returned when interface is down
	ErrInterfaceDown = errors.New("network interface is down")

	// ErrNoCarrier is returned when interface has no carrier
	ErrNoCarrier = errors.New("network interface has no carrier")

	// ErrDHCPTimeout is returned when DHCP request times out
	ErrDHCPTimeout = errors.New("DHCP request timed out")

	// ErrDHCPFailed is returned when DHCP configuration fails
	ErrDHCPFailed = errors.New("DHCP configuration failed")

	// ErrAddressInUse is returned when IP address is already in use
	ErrAddressInUse = errors.New("IP address already in use")

	// ErrNotSupported is returned when operation is not supported
	ErrNotSupported = errors.New("operation not supported on this platform")

	// ErrPermissionDenied is returned when lacking permissions
	ErrPermissionDenied = errors.New("permission denied (try running as root/administrator)")

	// ErrConfigNotApplied is returned when configuration hasn't been applied
	ErrConfigNotApplied = errors.New("IP configuration has not been applied")

	// ErrInvalidMethod is returned when configuration method is invalid
	ErrInvalidMethod = errors.New("invalid configuration method")

	// ErrIPv6Disabled is returned when IPv6 is disabled on the system
	ErrIPv6Disabled = errors.New("IPv6 is disabled on this system")

	// ErrRouteExists is returned when route already exists
	ErrRouteExists = errors.New("route already exists")

	// ErrRouteNotFound is returned when route doesn't exist
	ErrRouteNotFound = errors.New("route not found")
)

package ipconfig

import (
	"net"
	"time"
)

// ConfigMethod represents the IP configuration method
type ConfigMethod string

const (
	ConfigMethodDHCP   ConfigMethod = "dhcp"   // DHCP (automatic)
	ConfigMethodStatic ConfigMethod = "static" // Static IP
	ConfigMethodNone   ConfigMethod = "none"   // No IP configuration
)

// DNSMethod represents the DNS configuration method
type DNSMethod string

const (
	DNSMethodAuto   DNSMethod = "auto"   // Automatic (from DHCP or system)
	DNSMethodStatic DNSMethod = "static" // Static DNS servers
	DNSMethodNone   DNSMethod = "none"   // No DNS configuration
)

// GatewayMethod represents the gateway configuration method
type GatewayMethod string

const (
	GatewayMethodAuto    GatewayMethod = "auto"    // Automatic (from DHCP)
	GatewayMethodStatic  GatewayMethod = "static"  // Static gateway
	GatewayMethodMetric  GatewayMethod = "metric"  // Static with custom metric
	GatewayMethodDisable GatewayMethod = "disable" // No default gateway
)

// IPConfig represents IP configuration for a network interface
type IPConfig struct {
	// Interface
	InterfaceName string // e.g., "eth0", "vlan100"

	// IPv4 Configuration
	IPv4Method  ConfigMethod // dhcp, static, none
	IPv4Address string       // e.g., "192.168.1.100"
	IPv4Netmask string       // e.g., "255.255.255.0" or "24" (CIDR)
	IPv4CIDR    int          // CIDR notation (e.g., 24 for /24)

	// IPv6 Configuration
	IPv6Method  ConfigMethod // dhcp, static, none
	IPv6Address string       // e.g., "2001:db8::1"
	IPv6CIDR    int          // CIDR notation (e.g., 64 for /64)

	// Gateway Configuration
	GatewayMethod GatewayMethod // auto, static, metric, disable
	IPv4Gateway   string        // e.g., "192.168.1.1"
	IPv6Gateway   string        // e.g., "2001:db8::1"
	GatewayMetric int           // Route metric (priority)

	// DNS Configuration
	DNSMethod      DNSMethod // auto, static, none
	DNSServers     []string  // e.g., ["8.8.8.8", "8.8.4.4"]
	DNSSearch      []string  // DNS search domains
	DNSCaching     bool      // Enable local DNS caching
	DNSForwarding  bool      // Enable DNS forwarding

	// DHCP Options (when using DHCP)
	DHCPHostname string        // Hostname to send to DHCP server
	DHCPClientID string        // DHCP client identifier
	DHCPTimeout  time.Duration // DHCP timeout (default: 30s)

	// Advanced Options
	MTU            int  // Maximum Transmission Unit
	AcceptRA       bool // Accept Router Advertisements (IPv6)
	IgnoreCarrier  bool // Configure even if no carrier
	RequireCarrier bool // Wait for carrier before configuring

	// State
	Applied   bool      // Configuration has been applied
	AppliedAt time.Time // When configuration was applied
}

// InterfaceState represents the current state of an interface
type InterfaceState struct {
	InterfaceName string

	// Current IPv4 Configuration
	IPv4Addresses []net.IPNet // Current IPv4 addresses
	IPv4Gateway   net.IP      // Current IPv4 gateway

	// Current IPv6 Configuration
	IPv6Addresses []net.IPNet // Current IPv6 addresses
	IPv6Gateway   net.IP      // Current IPv6 gateway

	// Current DNS Configuration
	DNSServers []net.IP // Current DNS servers

	// DHCP State
	DHCPLeaseExpiry time.Time // When DHCP lease expires
	DHCPServer      net.IP    // DHCP server IP

	// Interface State
	IsUp          bool
	HasCarrier    bool
	LastCheckedAt time.Time
}

// DHCPLease represents a DHCP lease
type DHCPLease struct {
	InterfaceName string
	IPAddress     net.IP
	Netmask       net.IPMask
	Gateway       net.IP
	DNSServers    []net.IP
	LeaseTime     time.Duration
	RenewTime     time.Duration
	RebindTime    time.Duration
	Server        net.IP
	AcquiredAt    time.Time
	ExpiresAt     time.Time
}

// RouteConfig represents a static route configuration
type RouteConfig struct {
	Destination string // e.g., "10.0.0.0/8", "0.0.0.0/0" (default route)
	Gateway     string // e.g., "192.168.1.1"
	Interface   string // Interface name
	Metric      int    // Route metric
	Table       int    // Routing table ID (Linux)
}

// ValidateIPv4 validates an IPv4 address
func ValidateIPv4(ip string) error {
	if ip == "" {
		return ErrInvalidIPAddress
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return ErrInvalidIPAddress
	}
	return nil
}

// ValidateIPv6 validates an IPv6 address
func ValidateIPv6(ip string) error {
	if ip == "" {
		return ErrInvalidIPAddress
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() != nil {
		return ErrInvalidIPAddress
	}
	return nil
}

// ValidateCIDR validates a CIDR notation
func ValidateCIDR(cidr int, ipv6 bool) error {
	if ipv6 {
		if cidr < 0 || cidr > 128 {
			return ErrInvalidCIDR
		}
	} else {
		if cidr < 0 || cidr > 32 {
			return ErrInvalidCIDR
		}
	}
	return nil
}

// ParseNetmask parses a netmask string to CIDR notation
func ParseNetmask(netmask string) (int, error) {
	// Try as CIDR first
	if len(netmask) <= 3 {
		// Might be CIDR notation (e.g., "24")
		cidr := 0
		for _, c := range netmask {
			if c < '0' || c > '9' {
				goto tryIPv4
			}
			cidr = cidr*10 + int(c-'0')
		}
		if cidr >= 0 && cidr <= 32 {
			return cidr, nil
		}
	}

tryIPv4:
	// Try as IPv4 netmask (e.g., "255.255.255.0")
	ip := net.ParseIP(netmask)
	if ip == nil {
		return 0, ErrInvalidNetmask
	}

	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0, ErrInvalidNetmask
	}

	// Convert to CIDR
	mask := net.IPMask(ipv4)
	ones, _ := mask.Size()
	return ones, nil
}

// CIDRToNetmask converts CIDR notation to netmask string
func CIDRToNetmask(cidr int) string {
	if cidr < 0 || cidr > 32 {
		return ""
	}
	mask := net.CIDRMask(cidr, 32)
	return net.IP(mask).String()
}

//go:build linux
// +build linux

package ipconfig

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"
)

// LinuxManager implements IP configuration management for Linux
type LinuxManager struct{}

// newLinuxManager creates a new Linux IP configuration manager
func newLinuxManager() (*LinuxManager, error) {
	// Check if we have necessary permissions
	if os.Geteuid() != 0 {
		return nil, ErrPermissionDenied
	}

	return &LinuxManager{}, nil
}

// Apply applies IP configuration to an interface
func (m *LinuxManager) Apply(config *IPConfig) error {
	// Get the link
	link, err := netlink.LinkByName(config.InterfaceName)
	if err != nil {
		return ErrInterfaceNotFound
	}

	// Check carrier if required
	if config.RequireCarrier {
		attrs := link.Attrs()
		if attrs.OperState != netlink.OperUp {
			return ErrNoCarrier
		}
	}

	// Set MTU if specified
	if config.MTU > 0 {
		if err := netlink.LinkSetMTU(link, config.MTU); err != nil {
			return fmt.Errorf("failed to set MTU: %w", err)
		}
	}

	// Bring interface up
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to bring interface up: %w", err)
	}

	// Apply IPv4 configuration
	if config.IPv4Method == ConfigMethodStatic {
		if err := m.applyIPv4Static(link, config); err != nil {
			return fmt.Errorf("failed to apply IPv4 configuration: %w", err)
		}
	} else if config.IPv4Method == ConfigMethodDHCP {
		if err := m.applyIPv4DHCP(config.InterfaceName, config); err != nil {
			return fmt.Errorf("failed to apply DHCP configuration: %w", err)
		}
	}

	// Apply IPv6 configuration
	if config.IPv6Method == ConfigMethodStatic {
		if err := m.applyIPv6Static(link, config); err != nil {
			return fmt.Errorf("failed to apply IPv6 configuration: %w", err)
		}
	} else if config.IPv6Method == ConfigMethodDHCP {
		// IPv6 DHCP (DHCPv6) - typically handled by dhclient
		if err := m.applyIPv6DHCP(config.InterfaceName, config); err != nil {
			return fmt.Errorf("failed to apply DHCPv6 configuration: %w", err)
		}
	}

	// Apply gateway configuration
	if config.GatewayMethod == GatewayMethodStatic || config.GatewayMethod == GatewayMethodMetric {
		if err := m.applyGateway(link, config); err != nil {
			return fmt.Errorf("failed to apply gateway: %w", err)
		}
	}

	// Apply DNS configuration
	if config.DNSMethod == DNSMethodStatic {
		if err := m.applyDNS(config); err != nil {
			return fmt.Errorf("failed to apply DNS: %w", err)
		}
	}

	return nil
}

// applyIPv4Static applies static IPv4 configuration
func (m *LinuxManager) applyIPv4Static(link netlink.Link, config *IPConfig) error {
	// Parse IP address
	ip := net.ParseIP(config.IPv4Address)
	if ip == nil {
		return ErrInvalidIPAddress
	}

	// Create IP address with CIDR
	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(config.IPv4CIDR, 32),
		},
	}

	// Remove existing addresses (optional - could keep multiple)
	existingAddrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err == nil {
		for _, existing := range existingAddrs {
			netlink.AddrDel(link, &existing)
		}
	}

	// Add the new address
	if err := netlink.AddrAdd(link, addr); err != nil {
		// Check if address already exists
		if err == syscall.EEXIST {
			return ErrAddressInUse
		}
		return fmt.Errorf("failed to add IPv4 address: %w", err)
	}

	return nil
}

// applyIPv4DHCP applies DHCP configuration
func (m *LinuxManager) applyIPv4DHCP(interfaceName string, config *IPConfig) error {
	// Try systemd-networkd first
	if m.hasSystemdNetworkd() {
		return m.applyDHCPSystemd(interfaceName, config)
	}

	// Fall back to dhclient
	return m.applyDHCPClient(interfaceName, config)
}

// applyDHCPSystemd uses systemd-networkd for DHCP
func (m *LinuxManager) applyDHCPSystemd(interfaceName string, config *IPConfig) error {
	// Create systemd-networkd configuration
	networkdPath := fmt.Sprintf("/etc/systemd/network/50-%s.network", interfaceName)

	content := fmt.Sprintf(`[Match]
Name=%s

[Network]
DHCP=ipv4

[DHCP]
UseDNS=true
UseRoutes=true
`, interfaceName)

	if config.DHCPHostname != "" {
		content += fmt.Sprintf("Hostname=%s\n", config.DHCPHostname)
	}

	// Write configuration file
	if err := os.WriteFile(networkdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write systemd-networkd config: %w", err)
	}

	// Restart systemd-networkd
	cmd := exec.Command("systemctl", "restart", "systemd-networkd")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to restart systemd-networkd: %w", err)
	}

	return nil
}

// applyDHCPClient uses dhclient for DHCP
func (m *LinuxManager) applyDHCPClient(interfaceName string, config *IPConfig) error {
	// Kill existing dhclient for this interface
	exec.Command("killall", "-q", fmt.Sprintf("dhclient-%s", interfaceName)).Run()

	// Build dhclient command
	args := []string{"-v"}

	if config.DHCPHostname != "" {
		args = append(args, "-H", config.DHCPHostname)
	}

	args = append(args, interfaceName)

	// Start dhclient
	cmd := exec.Command("dhclient", args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start dhclient: %w", err)
	}

	// Wait for DHCP lease (with timeout)
	timeout := config.DHCPTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer timer.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return ErrDHCPTimeout
		case <-ticker.C:
			// Check if we got an IP
			link, err := netlink.LinkByName(interfaceName)
			if err != nil {
				continue
			}

			addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
			if err != nil {
				continue
			}

			for _, addr := range addrs {
				// Ignore localhost
				if !addr.IP.IsLoopback() && addr.IP.To4() != nil {
					return nil // Success!
				}
			}
		}
	}
}

// applyIPv6Static applies static IPv6 configuration
func (m *LinuxManager) applyIPv6Static(link netlink.Link, config *IPConfig) error {
	// Parse IP address
	ip := net.ParseIP(config.IPv6Address)
	if ip == nil {
		return ErrInvalidIPAddress
	}

	// Create IP address with CIDR
	addr := &netlink.Addr{
		IPNet: &net.IPNet{
			IP:   ip,
			Mask: net.CIDRMask(config.IPv6CIDR, 128),
		},
	}

	// Add the address
	if err := netlink.AddrAdd(link, addr); err != nil {
		if err == syscall.EEXIST {
			return ErrAddressInUse
		}
		return fmt.Errorf("failed to add IPv6 address: %w", err)
	}

	return nil
}

// applyIPv6DHCP applies DHCPv6 configuration
func (m *LinuxManager) applyIPv6DHCP(interfaceName string, config *IPConfig) error {
	// DHCPv6 typically uses dhclient with -6 flag
	cmd := exec.Command("dhclient", "-6", interfaceName)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start DHCPv6 client: %w", err)
	}

	return nil
}

// applyGateway applies gateway configuration
func (m *LinuxManager) applyGateway(link netlink.Link, config *IPConfig) error {
	// Add IPv4 gateway
	if config.IPv4Gateway != "" {
		gw := net.ParseIP(config.IPv4Gateway)
		if gw == nil {
			return ErrInvalidGateway
		}

		route := &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       nil, // Default route
			Gw:        gw,
			Priority:  config.GatewayMetric,
		}

		// Remove existing default route through this interface
		routes, err := netlink.RouteList(link, netlink.FAMILY_V4)
		if err == nil {
			for _, r := range routes {
				if r.Dst == nil { // Default route
					netlink.RouteDel(&r)
				}
			}
		}

		// Add new route
		if err := netlink.RouteAdd(route); err != nil {
			if err != syscall.EEXIST {
				return fmt.Errorf("failed to add gateway: %w", err)
			}
		}
	}

	// Add IPv6 gateway
	if config.IPv6Gateway != "" {
		gw := net.ParseIP(config.IPv6Gateway)
		if gw == nil {
			return ErrInvalidGateway
		}

		route := &netlink.Route{
			LinkIndex: link.Attrs().Index,
			Dst:       nil, // Default route
			Gw:        gw,
			Priority:  config.GatewayMetric,
		}

		if err := netlink.RouteAdd(route); err != nil {
			if err != syscall.EEXIST {
				return fmt.Errorf("failed to add IPv6 gateway: %w", err)
			}
		}
	}

	return nil
}

// applyDNS applies DNS configuration
func (m *LinuxManager) applyDNS(config *IPConfig) error {
	// Read existing resolv.conf
	content := "# Generated by MultiWANBond\n"

	// Add search domains
	if len(config.DNSSearch) > 0 {
		content += "search " + strings.Join(config.DNSSearch, " ") + "\n"
	}

	// Add nameservers
	for _, server := range config.DNSServers {
		content += fmt.Sprintf("nameserver %s\n", server)
	}

	// Write to resolv.conf
	if err := os.WriteFile("/etc/resolv.conf", []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write resolv.conf: %w", err)
	}

	return nil
}

// Remove removes IP configuration from an interface
func (m *LinuxManager) Remove(interfaceName string) error {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return ErrInterfaceNotFound
	}

	// Remove all IP addresses
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to list addresses: %w", err)
	}

	for _, addr := range addrs {
		if err := netlink.AddrDel(link, &addr); err != nil {
			// Continue even if some fail
		}
	}

	// Remove all routes through this interface
	routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
	if err == nil {
		for _, route := range routes {
			netlink.RouteDel(&route)
		}
	}

	// Kill dhclient if running
	exec.Command("killall", "-q", fmt.Sprintf("dhclient-%s", interfaceName)).Run()

	// Bring interface down
	if err := netlink.LinkSetDown(link); err != nil {
		return fmt.Errorf("failed to bring interface down: %w", err)
	}

	return nil
}

// Get gets current IP configuration for an interface
func (m *LinuxManager) Get(interfaceName string) (*InterfaceState, error) {
	link, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return nil, ErrInterfaceNotFound
	}

	state := &InterfaceState{
		InterfaceName: interfaceName,
		LastCheckedAt: time.Now(),
	}

	// Get link state
	attrs := link.Attrs()
	state.IsUp = attrs.Flags&net.FlagUp != 0
	state.HasCarrier = attrs.OperState == netlink.OperUp

	// Get IPv4 addresses
	addrsV4, err := netlink.AddrList(link, netlink.FAMILY_V4)
	if err == nil {
		for _, addr := range addrsV4 {
			if !addr.IP.IsLoopback() {
				state.IPv4Addresses = append(state.IPv4Addresses, *addr.IPNet)
			}
		}
	}

	// Get IPv6 addresses
	addrsV6, err := netlink.AddrList(link, netlink.FAMILY_V6)
	if err == nil {
		for _, addr := range addrsV6 {
			if !addr.IP.IsLoopback() {
				state.IPv6Addresses = append(state.IPv6Addresses, *addr.IPNet)
			}
		}
	}

	// Get routes (gateways)
	routes, err := netlink.RouteList(link, netlink.FAMILY_ALL)
	if err == nil {
		for _, route := range routes {
			if route.Dst == nil && route.Gw != nil {
				// Default route
				if route.Gw.To4() != nil {
					state.IPv4Gateway = route.Gw
				} else {
					state.IPv6Gateway = route.Gw
				}
			}
		}
	}

	// Get DNS servers from resolv.conf
	dnsServers, _ := m.readResolvConf()
	state.DNSServers = dnsServers

	return state, nil
}

// List lists all configured interfaces
func (m *LinuxManager) List() (map[string]*InterfaceState, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	states := make(map[string]*InterfaceState)

	for _, link := range links {
		name := link.Attrs().Name
		state, err := m.Get(name)
		if err == nil {
			states[name] = state
		}
	}

	return states, nil
}

// AddRoute adds a static route
func (m *LinuxManager) AddRoute(route *RouteConfig) error {
	// Parse destination
	_, dst, err := net.ParseCIDR(route.Destination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	netlinkRoute := &netlink.Route{
		Dst:      dst,
		Priority: route.Metric,
	}

	// Set gateway if specified
	if route.Gateway != "" {
		gw := net.ParseIP(route.Gateway)
		if gw == nil {
			return ErrInvalidGateway
		}
		netlinkRoute.Gw = gw
	}

	// Set interface if specified
	if route.Interface != "" {
		link, err := netlink.LinkByName(route.Interface)
		if err != nil {
			return ErrInterfaceNotFound
		}
		netlinkRoute.LinkIndex = link.Attrs().Index
	}

	// Set table if specified
	if route.Table > 0 {
		netlinkRoute.Table = route.Table
	}

	// Add route
	if err := netlink.RouteAdd(netlinkRoute); err != nil {
		if err == syscall.EEXIST {
			return ErrRouteExists
		}
		return fmt.Errorf("failed to add route: %w", err)
	}

	return nil
}

// RemoveRoute removes a static route
func (m *LinuxManager) RemoveRoute(route *RouteConfig) error {
	_, dst, err := net.ParseCIDR(route.Destination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	netlinkRoute := &netlink.Route{
		Dst: dst,
	}

	if route.Gateway != "" {
		netlinkRoute.Gw = net.ParseIP(route.Gateway)
	}

	if err := netlink.RouteDel(netlinkRoute); err != nil {
		return ErrRouteNotFound
	}

	return nil
}

// ListRoutes lists all routes
func (m *LinuxManager) ListRoutes() ([]*RouteConfig, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	var configs []*RouteConfig

	for _, route := range routes {
		config := &RouteConfig{
			Metric: route.Priority,
			Table:  route.Table,
		}

		if route.Dst != nil {
			config.Destination = route.Dst.String()
		} else {
			// Default route
			if route.Gw.To4() != nil {
				config.Destination = "0.0.0.0/0"
			} else {
				config.Destination = "::/0"
			}
		}

		if route.Gw != nil {
			config.Gateway = route.Gw.String()
		}

		if route.LinkIndex > 0 {
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err == nil {
				config.Interface = link.Attrs().Name
			}
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// SetDNS sets DNS servers for an interface
func (m *LinuxManager) SetDNS(interfaceName string, servers []string) error {
	// For now, just update resolv.conf
	// In production, you might want to use systemd-resolved or resolvconf
	content := "# Generated by MultiWANBond\n"

	for _, server := range servers {
		content += fmt.Sprintf("nameserver %s\n", server)
	}

	return os.WriteFile("/etc/resolv.conf", []byte(content), 0644)
}

// GetDNS gets DNS servers for an interface
func (m *LinuxManager) GetDNS(interfaceName string) ([]string, error) {
	return m.readResolvConf()
}

// RenewDHCP renews DHCP lease for an interface
func (m *LinuxManager) RenewDHCP(interfaceName string) error {
	// Send SIGUSR1 to dhclient to renew lease
	cmd := exec.Command("killall", "-USR1", fmt.Sprintf("dhclient-%s", interfaceName))
	if err := cmd.Run(); err != nil {
		return ErrDHCPFailed
	}
	return nil
}

// ReleaseDHCP releases DHCP lease for an interface
func (m *LinuxManager) ReleaseDHCP(interfaceName string) error {
	cmd := exec.Command("dhclient", "-r", interfaceName)
	if err := cmd.Run(); err != nil {
		return ErrDHCPFailed
	}
	return nil
}

// hasSystemdNetworkd checks if systemd-networkd is available
func (m *LinuxManager) hasSystemdNetworkd() bool {
	cmd := exec.Command("systemctl", "is-active", "systemd-networkd")
	err := cmd.Run()
	return err == nil
}

// readResolvConf reads DNS servers from /etc/resolv.conf
func (m *LinuxManager) readResolvConf() ([]net.IP, error) {
	file, err := os.Open("/etc/resolv.conf")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []net.IP
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				ip := net.ParseIP(parts[1])
				if ip != nil {
					servers = append(servers, ip)
				}
			}
		}
	}

	return servers, scanner.Err()
}

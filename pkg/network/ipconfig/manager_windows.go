//go:build windows
// +build windows

package ipconfig

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// WindowsManager implements IP configuration management for Windows
type WindowsManager struct{}

// newWindowsManager creates a new Windows IP configuration manager
func newWindowsManager() (*WindowsManager, error) {
	// Check if running as administrator
	cmd := exec.Command("net", "session")
	if err := cmd.Run(); err != nil {
		return nil, ErrPermissionDenied
	}

	return &WindowsManager{}, nil
}

// Apply applies IP configuration to an interface
func (m *WindowsManager) Apply(config *IPConfig) error {
	// For static IP
	if config.IPv4Method == ConfigMethodStatic {
		return m.applyIPv4Static(config)
	}

	// For DHCP
	if config.IPv4Method == ConfigMethodDHCP {
		return m.applyIPv4DHCP(config)
	}

	return nil
}

// applyIPv4Static applies static IPv4 configuration using netsh
func (m *WindowsManager) applyIPv4Static(config *IPConfig) error {
	// netsh interface ip set address name="Interface Name" static IP MASK GATEWAY
	args := []string{
		"interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", config.InterfaceName),
		"static",
		config.IPv4Address,
		CIDRToNetmask(config.IPv4CIDR),
	}

	if config.IPv4Gateway != "" {
		args = append(args, config.IPv4Gateway)
		if config.GatewayMetric > 0 {
			args = append(args, fmt.Sprintf("%d", config.GatewayMetric))
		}
	}

	cmd := exec.Command("netsh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set static IP: %s: %w", string(output), err)
	}

	return nil
}

// applyIPv4DHCP applies DHCP configuration using netsh
func (m *WindowsManager) applyIPv4DHCP(config *IPConfig) error {
	// netsh interface ip set address name="Interface Name" dhcp
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", config.InterfaceName), "dhcp")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set DHCP: %s: %w", string(output), err)
	}

	return nil
}

// Remove removes IP configuration from an interface
func (m *WindowsManager) Remove(interfaceName string) error {
	// Set to DHCP (which effectively clears static config)
	cmd := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf("name=%s", interfaceName), "dhcp")
	return cmd.Run()
}

// Get gets current IP configuration for an interface
func (m *WindowsManager) Get(interfaceName string) (*InterfaceState, error) {
	state := &InterfaceState{
		InterfaceName: interfaceName,
		LastCheckedAt: time.Now(),
	}

	// Use PowerShell to get interface info
	psCmd := fmt.Sprintf(`
		$adapter = Get-NetAdapter -Name "%s" -ErrorAction SilentlyContinue
		if ($adapter) {
			$ip = Get-NetIPAddress -InterfaceAlias "%s" -ErrorAction SilentlyContinue
			$route = Get-NetRoute -InterfaceAlias "%s" -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue
			$dns = Get-DnsClientServerAddress -InterfaceAlias "%s" -ErrorAction SilentlyContinue

			Write-Output "Status:$($adapter.Status)"
			if ($ip) {
				foreach ($addr in $ip) {
					Write-Output "IP:$($addr.IPAddress)/$($addr.PrefixLength)"
				}
			}
			if ($route) {
				Write-Output "Gateway:$($route.NextHop)"
			}
			if ($dns) {
				foreach ($server in $dns.ServerAddresses) {
					Write-Output "DNS:$server"
				}
			}
		}
	`, interfaceName, interfaceName, interfaceName, interfaceName)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, ErrInterfaceNotFound
	}

	// Parse output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			status := strings.TrimPrefix(line, "Status:")
			state.IsUp = status == "Up"
			state.HasCarrier = status == "Up"
		} else if strings.HasPrefix(line, "IP:") {
			ipStr := strings.TrimPrefix(line, "IP:")
			if _, ipnet, err := net.ParseCIDR(ipStr); err == nil {
				if ipnet.IP.To4() != nil {
					state.IPv4Addresses = append(state.IPv4Addresses, *ipnet)
				} else {
					state.IPv6Addresses = append(state.IPv6Addresses, *ipnet)
				}
			}
		} else if strings.HasPrefix(line, "Gateway:") {
			gwStr := strings.TrimPrefix(line, "Gateway:")
			if gw := net.ParseIP(gwStr); gw != nil {
				if gw.To4() != nil {
					state.IPv4Gateway = gw
				} else {
					state.IPv6Gateway = gw
				}
			}
		} else if strings.HasPrefix(line, "DNS:") {
			dnsStr := strings.TrimPrefix(line, "DNS:")
			if dns := net.ParseIP(dnsStr); dns != nil {
				state.DNSServers = append(state.DNSServers, dns)
			}
		}
	}

	return state, nil
}

// List lists all configured interfaces
func (m *WindowsManager) List() (map[string]*InterfaceState, error) {
	// Get all adapter names
	cmd := exec.Command("powershell", "-Command", "Get-NetAdapter | Select-Object -ExpandProperty Name")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list adapters: %w", err)
	}

	states := make(map[string]*InterfaceState)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}

		state, err := m.Get(name)
		if err == nil {
			states[name] = state
		}
	}

	return states, nil
}

// AddRoute adds a static route
func (m *WindowsManager) AddRoute(route *RouteConfig) error {
	// route add DESTINATION mask NETMASK GATEWAY metric METRIC if INTERFACE
	// For Windows, we need to use: route add -p for persistent routes
	args := []string{"add"}

	// Parse destination to get network and mask
	_, ipnet, err := net.ParseCIDR(route.Destination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	args = append(args, ipnet.IP.String())
	args = append(args, "mask", net.IP(ipnet.Mask).String())

	if route.Gateway != "" {
		args = append(args, route.Gateway)
	}

	if route.Metric > 0 {
		args = append(args, "metric", fmt.Sprintf("%d", route.Metric))
	}

	cmd := exec.Command("route", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add route: %s: %w", string(output), err)
	}

	return nil
}

// RemoveRoute removes a static route
func (m *WindowsManager) RemoveRoute(route *RouteConfig) error {
	_, ipnet, err := net.ParseCIDR(route.Destination)
	if err != nil {
		return fmt.Errorf("invalid destination: %w", err)
	}

	cmd := exec.Command("route", "delete", ipnet.IP.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete route: %s: %w", string(output), err)
	}

	return nil
}

// ListRoutes lists all routes
func (m *WindowsManager) ListRoutes() ([]*RouteConfig, error) {
	cmd := exec.Command("route", "print")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	var routes []*RouteConfig
	lines := strings.Split(string(output), "\n")

	// Parse route print output (format varies)
	routeRe := regexp.MustCompile(`\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\d+)`)

	for _, line := range lines {
		matches := routeRe.FindStringSubmatch(line)
		if len(matches) >= 6 {
			destination := matches[1]
			netmask := matches[2]
			gateway := matches[3]
			metric := matches[5]

			// Convert netmask to CIDR
			mask := net.ParseIP(netmask)
			if mask == nil {
				continue
			}

			ones, _ := net.IPMask(mask.To4()).Size()

			route := &RouteConfig{
				Destination: fmt.Sprintf("%s/%d", destination, ones),
				Gateway:     gateway,
			}

			fmt.Sscanf(metric, "%d", &route.Metric)

			routes = append(routes, route)
		}
	}

	return routes, nil
}

// SetDNS sets DNS servers for an interface
func (m *WindowsManager) SetDNS(interfaceName string, servers []string) error {
	// netsh interface ip set dns name="Interface" static DNS_SERVER
	if len(servers) == 0 {
		return nil
	}

	// Set primary DNS
	cmd := exec.Command("netsh", "interface", "ip", "set", "dns",
		fmt.Sprintf("name=%s", interfaceName), "static", servers[0])
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set primary DNS: %w", err)
	}

	// Add secondary DNS servers
	for i := 1; i < len(servers); i++ {
		cmd = exec.Command("netsh", "interface", "ip", "add", "dns",
			fmt.Sprintf("name=%s", interfaceName), servers[i], fmt.Sprintf("index=%d", i+1))
		if err := cmd.Run(); err != nil {
			// Continue even if this fails
		}
	}

	return nil
}

// GetDNS gets DNS servers for an interface
func (m *WindowsManager) GetDNS(interfaceName string) ([]string, error) {
	psCmd := fmt.Sprintf(`
		$dns = Get-DnsClientServerAddress -InterfaceAlias "%s" -ErrorAction SilentlyContinue
		if ($dns) {
			$dns.ServerAddresses
		}
	`, interfaceName)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS: %w", err)
	}

	var servers []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			servers = append(servers, line)
		}
	}

	return servers, nil
}

// RenewDHCP renews DHCP lease for an interface
func (m *WindowsManager) RenewDHCP(interfaceName string) error {
	cmd := exec.Command("ipconfig", "/renew", interfaceName)
	return cmd.Run()
}

// ReleaseDHCP releases DHCP lease for an interface
func (m *WindowsManager) ReleaseDHCP(interfaceName string) error {
	cmd := exec.Command("ipconfig", "/release", interfaceName)
	return cmd.Run()
}

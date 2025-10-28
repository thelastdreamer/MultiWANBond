// +build linux

package network

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// LinuxDetector implements Detector for Linux
type LinuxDetector struct {
}

// newLinuxDetector creates a new Linux detector
func newLinuxDetector() (*LinuxDetector, error) {
	return &LinuxDetector{}, nil
}

// DetectAll detects all network interfaces on Linux
func (d *LinuxDetector) DetectAll() ([]*NetworkInterface, error) {
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list links: %w", err)
	}

	result := make([]*NetworkInterface, 0, len(links))

	for _, link := range links {
		iface, err := d.convertLink(link)
		if err != nil {
			// Log error but continue with other interfaces
			continue
		}

		// Enrich with additional information
		d.enrichInterfaceInfo(iface)

		result = append(result, iface)
	}

	return result, nil
}

// DetectByName detects a specific interface by name
func (d *LinuxDetector) DetectByName(name string) (*NetworkInterface, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get link %s: %w", name, err)
	}

	iface, err := d.convertLink(link)
	if err != nil {
		return nil, err
	}

	d.enrichInterfaceInfo(iface)

	return iface, nil
}

// convertLink converts a netlink.Link to NetworkInterface
func (d *LinuxDetector) convertLink(link netlink.Link) (*NetworkInterface, error) {
	attrs := link.Attrs()

	iface := &NetworkInterface{
		SystemName: attrs.Name,
		Index:      attrs.Index,
		MACAddress: attrs.HardwareAddr.String(),
		MTU:        attrs.MTU,
		DetectedAt: time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Determine interface type
	iface.Type = d.determineInterfaceType(link)

	// Set state
	iface.AdminState = "down"
	iface.OperState = "down"
	if attrs.Flags&net.FlagUp != 0 {
		iface.AdminState = "up"
	}
	if attrs.OperState == netlink.OperUp {
		iface.OperState = "up"
		iface.HasCarrier = true
	} else if attrs.OperState == netlink.OperDown {
		iface.OperState = "down"
		iface.HasCarrier = false
	}

	// Get flags
	iface.Flags = d.getFlagsFromLink(attrs.Flags)

	// Get statistics
	if stats := attrs.Statistics; stats != nil {
		iface.RxBytes = stats.RxBytes
		iface.TxBytes = stats.TxBytes
		iface.RxPackets = stats.RxPackets
		iface.TxPackets = stats.TxPackets
		iface.RxErrors = stats.RxErrors
		iface.TxErrors = stats.TxErrors
		iface.RxDropped = stats.RxDropped
		iface.TxDropped = stats.TxDropped
	}

	// Handle VLAN interfaces
	if vlan, ok := link.(*netlink.Vlan); ok {
		parent, _ := netlink.LinkByIndex(vlan.ParentIndex)
		iface.VLANInfo = &VLANInfo{
			ID:       vlan.VlanId,
			Priority: uint8(vlan.VlanProtocol),
		}
		if parent != nil {
			iface.VLANInfo.Parent = parent.Attrs().Name
		}
	}

	// Handle bond interfaces
	if bond, ok := link.(*netlink.Bond); ok {
		slaves, _ := d.getBondSlaves(attrs.Name)
		iface.BondInfo = &BondInfo{
			Mode:   d.getBondMode(bond.Mode),
			Slaves: slaves,
		}
	}

	// Handle bridge interfaces
	if _, ok := link.(*netlink.Bridge); ok {
		members, _ := d.getBridgeMembers(attrs.Name)
		iface.BridgeInfo = &BridgeInfo{
			Members: members,
		}
	}

	return iface, nil
}

// enrichInterfaceInfo adds additional information to the interface
func (d *LinuxDetector) enrichInterfaceInfo(iface *NetworkInterface) {
	// Get IP addresses
	iface.IPv4Addresses, iface.IPv6Addresses = d.getIPAddresses(iface.SystemName)
	iface.HasIP = len(iface.IPv4Addresses) > 0 || len(iface.IPv6Addresses) > 0

	// Get speed and duplex from ethtool
	speed, duplex := d.getSpeedAndDuplex(iface.SystemName)
	iface.Speed = speed
	iface.Duplex = duplex

	// Get driver info
	iface.Driver, iface.Vendor, iface.Model = d.getDriverInfo(iface.SystemName)

	// Test internet connectivity if interface is up
	if IsInterfaceUp(iface) && iface.HasIP {
		test, _ := d.TestConnectivity(iface.SystemName, "8.8.8.8", "ping")
		if test != nil && test.Success {
			iface.HasInternet = true
			iface.TestLatency = test.Latency
		}
	}
}

// determineInterfaceType determines the type of interface
func (d *LinuxDetector) determineInterfaceType(link netlink.Link) InterfaceType {
	attrs := link.Attrs()

	// Check by type name first (works with all versions)
	typeName := link.Type()
	switch typeName {
	case "vlan":
		return InterfaceVLAN
	case "bond":
		return InterfaceBond
	case "bridge":
		return InterfaceBridge
	case "veth":
		return InterfaceVirtual
	case "dummy":
		return InterfaceVirtual
	case "gre", "gretap", "ip6tnl", "ipip", "sit", "ip6gre":
		return InterfaceTunnel
	case "tun", "tap":
		return InterfaceTunnel
	}

	// Fallback to type assertion for common types
	switch link.(type) {
	case *netlink.Vlan:
		return InterfaceVLAN
	case *netlink.Bond:
		return InterfaceBond
	case *netlink.Bridge:
		return InterfaceBridge
	case *netlink.Veth:
		return InterfaceVirtual
	case *netlink.Dummy:
		return InterfaceVirtual
	}

	// Check by name patterns
	if attrs.Flags&net.FlagLoopback != 0 {
		return InterfaceLoopback
	}

	// Check if it's a wireless interface
	if strings.HasPrefix(attrs.Name, "wl") || strings.HasPrefix(attrs.Name, "wlan") {
		return InterfacePhysical
	}

	// Check if it's virtual by name
	if strings.HasPrefix(attrs.Name, "veth") || strings.HasPrefix(attrs.Name, "tap") || strings.HasPrefix(attrs.Name, "tun") {
		return InterfaceVirtual
	}

	// Check if it's a tunnel by name
	if strings.HasPrefix(attrs.Name, "gre") || strings.HasPrefix(attrs.Name, "tun") || strings.HasPrefix(attrs.Name, "ip6") {
		return InterfaceTunnel
	}

	// Default to physical
	return InterfacePhysical
}

// getIPAddresses gets IPv4 and IPv6 addresses for an interface
func (d *LinuxDetector) getIPAddresses(name string) ([]net.IP, []net.IP) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, nil
	}

	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return nil, nil
	}

	var ipv4, ipv6 []net.IP

	for _, addr := range addrs {
		if addr.IP.To4() != nil {
			ipv4 = append(ipv4, addr.IP)
		} else {
			ipv6 = append(ipv6, addr.IP)
		}
	}

	return ipv4, ipv6
}

// getSpeedAndDuplex gets speed and duplex using ethtool
func (d *LinuxDetector) getSpeedAndDuplex(name string) (uint64, DuplexMode) {
	// Try to read from sysfs first (faster)
	speedPath := fmt.Sprintf("/sys/class/net/%s/speed", name)
	if data, err := os.ReadFile(speedPath); err == nil {
		if speedMbps, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil && speedMbps > 0 {
			speed := uint64(speedMbps) * 1000000 // Convert Mbps to bps

			// Try to get duplex
			duplexPath := fmt.Sprintf("/sys/class/net/%s/duplex", name)
			duplex := DuplexUnknown
			if duplexData, err := os.ReadFile(duplexPath); err == nil {
				duplexStr := strings.TrimSpace(string(duplexData))
				if duplexStr == "full" {
					duplex = DuplexFull
				} else if duplexStr == "half" {
					duplex = DuplexHalf
				}
			}

			return speed, duplex
		}
	}

	// Fallback to ethtool command
	cmd := exec.Command("ethtool", name)
	output, err := cmd.Output()
	if err != nil {
		return 0, DuplexUnknown
	}

	lines := strings.Split(string(output), "\n")
	var speed uint64
	duplex := DuplexUnknown

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Speed:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				speedStr := strings.TrimSuffix(parts[1], "Mb/s")
				if speedMbps, err := strconv.ParseUint(speedStr, 10, 64); err == nil {
					speed = speedMbps * 1000000 // Convert to bps
				}
			}
		} else if strings.HasPrefix(line, "Duplex:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				if strings.ToLower(parts[1]) == "full" {
					duplex = DuplexFull
				} else if strings.ToLower(parts[1]) == "half" {
					duplex = DuplexHalf
				}
			}
		}
	}

	return speed, duplex
}

// getDriverInfo gets driver information
func (d *LinuxDetector) getDriverInfo(name string) (driver, vendor, model string) {
	// Try to read driver from sysfs
	driverPath := fmt.Sprintf("/sys/class/net/%s/device/driver", name)
	if target, err := os.Readlink(driverPath); err == nil {
		driver = strings.TrimPrefix(target, "../../../")
		parts := strings.Split(driver, "/")
		if len(parts) > 0 {
			driver = parts[len(parts)-1]
		}
	}

	// Try ethtool for more info
	cmd := exec.Command("ethtool", "-i", name)
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "driver:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					driver = parts[1]
				}
			}
		}
	}

	return driver, vendor, model
}

// getFlagsFromLink converts netlink flags to string slice
func (d *LinuxDetector) getFlagsFromLink(flags net.Flags) []string {
	var result []string
	if flags&net.FlagUp != 0 {
		result = append(result, "UP")
	}
	if flags&net.FlagBroadcast != 0 {
		result = append(result, "BROADCAST")
	}
	if flags&net.FlagLoopback != 0 {
		result = append(result, "LOOPBACK")
	}
	if flags&net.FlagPointToPoint != 0 {
		result = append(result, "POINTTOPOINT")
	}
	if flags&net.FlagMulticast != 0 {
		result = append(result, "MULTICAST")
	}
	return result
}

// getBondSlaves gets slave interfaces of a bond
func (d *LinuxDetector) getBondSlaves(bondName string) ([]string, error) {
	slavesPath := fmt.Sprintf("/sys/class/net/%s/bonding/slaves", bondName)
	data, err := os.ReadFile(slavesPath)
	if err != nil {
		return nil, err
	}

	slavesStr := strings.TrimSpace(string(data))
	if slavesStr == "" {
		return []string{}, nil
	}

	return strings.Fields(slavesStr), nil
}

// getBondMode converts bond mode integer to string
func (d *LinuxDetector) getBondMode(mode netlink.BondMode) string {
	modeMap := map[netlink.BondMode]string{
		0: "balance-rr",
		1: "active-backup",
		2: "balance-xor",
		3: "broadcast",
		4: "802.3ad",
		5: "balance-tlb",
		6: "balance-alb",
	}

	if modeStr, ok := modeMap[mode]; ok {
		return modeStr
	}
	return "unknown"
}

// getBridgeMembers gets member interfaces of a bridge
func (d *LinuxDetector) getBridgeMembers(bridgeName string) ([]string, error) {
	memberPath := fmt.Sprintf("/sys/class/net/%s/brif", bridgeName)
	entries, err := os.ReadDir(memberPath)
	if err != nil {
		return nil, err
	}

	members := make([]string, 0, len(entries))
	for _, entry := range entries {
		members = append(members, entry.Name())
	}

	return members, nil
}

// GetCapabilities returns interface capabilities
func (d *LinuxDetector) GetCapabilities(name string) (*InterfaceCapabilities, error) {
	caps := &InterfaceCapabilities{
		SupportsVLAN:    true, // Most Linux interfaces support VLANs
		SupportsBonding: true,
		SupportsBridge:  true,
	}

	// Try to get offload features from ethtool
	cmd := exec.Command("ethtool", "-k", name)
	output, err := cmd.Output()
	if err == nil {
		outputStr := string(output)
		caps.SupportsTSO = strings.Contains(outputStr, "tcp-segmentation-offload: on")
		caps.SupportsGSO = strings.Contains(outputStr, "generic-segmentation-offload: on")
		caps.SupportsGRO = strings.Contains(outputStr, "generic-receive-offload: on")
		caps.SupportsLRO = strings.Contains(outputStr, "large-receive-offload: on")
	}

	return caps, nil
}

// TestConnectivity tests internet connectivity using ping
func (d *LinuxDetector) TestConnectivity(ifaceName string, target string, method string) (*ConnectivityTest, error) {
	test := &ConnectivityTest{
		Interface: ifaceName,
		Target:    target,
		Method:    method,
		TestedAt:  time.Now(),
	}

	switch method {
	case "ping":
		return d.testPing(ifaceName, target, test)
	case "http":
		return d.testHTTP(ifaceName, target, test)
	case "dns":
		return d.testDNS(ifaceName, target, test)
	default:
		test.Error = "unsupported method"
		return test, fmt.Errorf("unsupported connectivity test method: %s", method)
	}
}

// testPing tests connectivity using ping
func (d *LinuxDetector) testPing(ifaceName string, target string, test *ConnectivityTest) (*ConnectivityTest, error) {
	start := time.Now()

	// Use ping with specific interface
	cmd := exec.Command("ping", "-c", "1", "-W", "2", "-I", ifaceName, target)
	output, err := cmd.Output()

	test.Latency = time.Since(start)

	if err != nil {
		test.Success = false
		test.Error = err.Error()
		return test, nil
	}

	// Parse ping output for more accurate latency
	outputStr := string(output)
	if strings.Contains(outputStr, "time=") {
		// Extract time value
		parts := strings.Split(outputStr, "time=")
		if len(parts) > 1 {
			timeStr := strings.Fields(parts[1])[0]
			if latencyMs, err := strconv.ParseFloat(timeStr, 64); err == nil {
				test.Latency = time.Duration(latencyMs * float64(time.Millisecond))
			}
		}
	}

	test.Success = true
	return test, nil
}

// testHTTP tests connectivity using HTTP request
func (d *LinuxDetector) testHTTP(ifaceName string, target string, test *ConnectivityTest) (*ConnectivityTest, error) {
	// For HTTP testing, we'd need to bind to specific interface
	// This is more complex and would require custom HTTP client configuration
	// For now, return ping-based test as fallback
	return d.testPing(ifaceName, target, test)
}

// testDNS tests connectivity using DNS query
func (d *LinuxDetector) testDNS(ifaceName string, target string, test *ConnectivityTest) (*ConnectivityTest, error) {
	// Similar to HTTP, DNS testing with interface binding is complex
	// Use ping as fallback
	return d.testPing(ifaceName, target, test)
}

// Monitor monitors for interface changes
func (d *LinuxDetector) Monitor(ctx context.Context) (<-chan *InterfaceChange, error) {
	changeCh := make(chan *InterfaceChange, 100)

	// Subscribe to netlink events
	updateCh := make(chan netlink.LinkUpdate)
	done := make(chan struct{})

	if err := netlink.LinkSubscribe(updateCh, done); err != nil {
		return nil, fmt.Errorf("failed to subscribe to link updates: %w", err)
	}

	go func() {
		defer close(changeCh)
		defer close(done)

		for {
			select {
			case <-ctx.Done():
				return

			case update := <-updateCh:
				change := &InterfaceChange{
					InterfaceName: update.Attrs().Name,
					Timestamp:     time.Now(),
				}

				// Determine change type based on update
				if update.Header.Type == unix.RTM_NEWLINK {
					change.ChangeType = ChangeAdded
				} else if update.Header.Type == unix.RTM_DELLINK {
					change.ChangeType = ChangeRemoved
				} else {
					change.ChangeType = ChangeAdminState
				}

				// Send change notification
				select {
				case changeCh <- change:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return changeCh, nil
}

// +build windows

package network

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// WindowsDetector implements Detector for Windows
type WindowsDetector struct {
}

// newWindowsDetector creates a new Windows detector
func newWindowsDetector() (*WindowsDetector, error) {
	return &WindowsDetector{}, nil
}

// DetectAll detects all network interfaces on Windows
func (d *WindowsDetector) DetectAll() ([]*NetworkInterface, error) {
	// Use net.Interfaces() as base, then enhance with Windows-specific info
	systemIfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	result := make([]*NetworkInterface, 0, len(systemIfaces))

	for _, sysIface := range systemIfaces {
		iface := &NetworkInterface{
			SystemName: sysIface.Name,
			Index:      sysIface.Index,
			MACAddress: sysIface.HardwareAddr.String(),
			MTU:        sysIface.MTU,
			DetectedAt: time.Now(),
			UpdatedAt:  time.Now(),
		}

		// Set state based on flags
		if sysIface.Flags&net.FlagUp != 0 {
			iface.AdminState = "up"
			iface.OperState = "up"
			iface.HasCarrier = true
		} else {
			iface.AdminState = "down"
			iface.OperState = "down"
			iface.HasCarrier = false
		}

		// Determine type
		if sysIface.Flags&net.FlagLoopback != 0 {
			iface.Type = InterfaceLoopback
		} else {
			iface.Type = InterfacePhysical
		}

		// Get IP addresses
		addrs, err := sysIface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if ipNet, ok := addr.(*net.IPNet); ok {
					if ipNet.IP.To4() != nil {
						iface.IPv4Addresses = append(iface.IPv4Addresses, ipNet.IP)
					} else {
						iface.IPv6Addresses = append(iface.IPv6Addresses, ipNet.IP)
					}
				}
			}
		}

		iface.HasIP = len(iface.IPv4Addresses) > 0 || len(iface.IPv6Addresses) > 0

		// Try to get additional info using PowerShell/netsh
		d.enrichWithWindowsInfo(iface)

		result = append(result, iface)
	}

	return result, nil
}

// enrichWithWindowsInfo enriches interface info using Windows commands
func (d *WindowsDetector) enrichWithWindowsInfo(iface *NetworkInterface) {
	// Try to get speed and description using PowerShell Get-NetAdapter
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-NetAdapter -Name '%s' | Select-Object LinkSpeed, DriverDescription, Status | ConvertTo-Json", iface.SystemName))

	output, err := cmd.Output()
	if err == nil {
		// Parse JSON output to get speed and driver info
		// Simplified parsing for now
		outputStr := string(output)
		if strings.Contains(outputStr, "LinkSpeed") {
			// Extract link speed
			// This is a simplified version - full implementation would parse JSON properly
		}
	}
}

// DetectByName detects a specific interface by name
func (d *WindowsDetector) DetectByName(name string) (*NetworkInterface, error) {
	all, err := d.DetectAll()
	if err != nil {
		return nil, err
	}

	for _, iface := range all {
		if iface.SystemName == name {
			return iface, nil
		}
	}

	return nil, fmt.Errorf("interface %s not found", name)
}

// GetCapabilities returns interface capabilities
func (d *WindowsDetector) GetCapabilities(name string) (*InterfaceCapabilities, error) {
	return &InterfaceCapabilities{
		SupportsVLAN:    true,
		SupportsBonding: true,
		SupportsBridge:  false, // Windows bridging is different
	}, nil
}

// TestConnectivity tests internet connectivity
func (d *WindowsDetector) TestConnectivity(ifaceName string, target string, method string) (*ConnectivityTest, error) {
	test := &ConnectivityTest{
		Interface: ifaceName,
		Target:    target,
		Method:    method,
		TestedAt:  time.Now(),
	}

	// Use ping command
	start := time.Now()
	cmd := exec.Command("ping", "-n", "1", "-w", "2000", target)
	output, err := cmd.Output()
	test.Latency = time.Since(start)

	if err != nil {
		test.Success = false
		test.Error = err.Error()
		return test, nil
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "Reply from") {
		test.Success = true
		// Parse latency from output if possible
		if strings.Contains(outputStr, "time=") {
			// Extract time value
		}
	} else {
		test.Success = false
		test.Error = "no reply"
	}

	return test, nil
}

// Monitor monitors for interface changes
func (d *WindowsDetector) Monitor(ctx context.Context) (<-chan *InterfaceChange, error) {
	changeCh := make(chan *InterfaceChange, 100)

	// Windows doesn't have a direct equivalent to netlink
	// We'd need to poll or use WMI events
	// For now, implement simple polling

	go func() {
		defer close(changeCh)

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		var lastState map[string]string

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Poll current state and compare
				ifaces, err := d.DetectAll()
				if err != nil {
					continue
				}

				currentState := make(map[string]string)
				for _, iface := range ifaces {
					currentState[iface.SystemName] = iface.OperState
				}

				if lastState != nil {
					// Check for changes
					for name, state := range currentState {
						if oldState, ok := lastState[name]; ok && oldState != state {
							change := &InterfaceChange{
								InterfaceName: name,
								ChangeType:    ChangeOperState,
								OldState:      oldState,
								NewState:      state,
								Timestamp:     time.Now(),
							}

							select {
							case changeCh <- change:
							case <-ctx.Done():
								return
							}
						}
					}
				}

				lastState = currentState
			}
		}
	}()

	return changeCh, nil
}

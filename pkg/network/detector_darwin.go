// +build darwin

package network

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// DarwinDetector implements Detector for macOS
type DarwinDetector struct {
}

// newDarwinDetector creates a new macOS detector
func newDarwinDetector() (*DarwinDetector, error) {
	return &DarwinDetector{}, nil
}

// DetectAll detects all network interfaces on macOS
func (d *DarwinDetector) DetectAll() ([]*NetworkInterface, error) {
	// Use net.Interfaces() as base
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
		} else if strings.HasPrefix(sysIface.Name, "en") {
			iface.Type = InterfacePhysical
		} else if strings.HasPrefix(sysIface.Name, "bridge") {
			iface.Type = InterfaceBridge
		} else if strings.HasPrefix(sysIface.Name, "vlan") {
			iface.Type = InterfaceVLAN
		} else if strings.HasPrefix(sysIface.Name, "utun") || strings.HasPrefix(sysIface.Name, "tun") {
			iface.Type = InterfaceTunnel
		} else {
			iface.Type = InterfaceVirtual
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

		// Try to get additional info using networksetup
		d.enrichWithDarwinInfo(iface)

		result = append(result, iface)
	}

	return result, nil
}

// enrichWithDarwinInfo enriches interface info using macOS commands
func (d *DarwinDetector) enrichWithDarwinInfo(iface *NetworkInterface) {
	// Try to get hardware info using networksetup
	cmd := exec.Command("networksetup", "-getinfo", iface.SystemName)
	output, err := cmd.Output()
	if err == nil {
		// Parse output for configuration info
		outputStr := string(output)
		if strings.Contains(outputStr, "DHCP") {
			// Interface uses DHCP
		}
	}

	// Try to get media info using ifconfig
	cmd = exec.Command("ifconfig", iface.SystemName)
	output, err = cmd.Output()
	if err == nil {
		outputStr := string(output)
		// Parse ifconfig output for speed, status, etc.
		if strings.Contains(outputStr, "status: active") {
			iface.OperState = "up"
			iface.HasCarrier = true
		}

		// Try to extract media type and speed
		if strings.Contains(outputStr, "media:") {
			// Parse media line for speed info
		}
	}
}

// DetectByName detects a specific interface by name
func (d *DarwinDetector) DetectByName(name string) (*NetworkInterface, error) {
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
func (d *DarwinDetector) GetCapabilities(name string) (*InterfaceCapabilities, error) {
	return &InterfaceCapabilities{
		SupportsVLAN:    true,
		SupportsBonding: true,
		SupportsBridge:  true,
	}, nil
}

// TestConnectivity tests internet connectivity
func (d *DarwinDetector) TestConnectivity(ifaceName string, target string, method string) (*ConnectivityTest, error) {
	test := &ConnectivityTest{
		Interface: ifaceName,
		Target:    target,
		Method:    method,
		TestedAt:  time.Now(),
	}

	// Use ping command with interface binding
	start := time.Now()
	cmd := exec.Command("ping", "-c", "1", "-W", "2000", "-b", ifaceName, target)
	output, err := cmd.Output()
	test.Latency = time.Since(start)

	if err != nil {
		test.Success = false
		test.Error = err.Error()
		return test, nil
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "bytes from") {
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
func (d *DarwinDetector) Monitor(ctx context.Context) (<-chan *InterfaceChange, error) {
	changeCh := make(chan *InterfaceChange, 100)

	// macOS doesn't have a direct equivalent to netlink
	// Implement simple polling

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

					// Check for added/removed interfaces
					for name := range currentState {
						if _, ok := lastState[name]; !ok {
							change := &InterfaceChange{
								InterfaceName: name,
								ChangeType:    ChangeAdded,
								NewState:      currentState[name],
								Timestamp:     time.Now(),
							}

							select {
							case changeCh <- change:
							case <-ctx.Done():
								return
							}
						}
					}

					for name := range lastState {
						if _, ok := currentState[name]; !ok {
							change := &InterfaceChange{
								InterfaceName: name,
								ChangeType:    ChangeRemoved,
								OldState:      lastState[name],
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

//go:build windows
// +build windows

package vlan

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// WindowsManager implements VLAN management for Windows
type WindowsManager struct{}

// newWindowsManager creates a new Windows VLAN manager
func newWindowsManager() (*WindowsManager, error) {
	// Check if running as administrator by attempting a privileged operation test
	cmd := exec.Command("net", "session")
	if err := cmd.Run(); err != nil {
		return nil, ErrPermissionDenied
	}

	return &WindowsManager{}, nil
}

// Create creates a new VLAN interface on Windows
func (m *WindowsManager) Create(config *Config) (*Interface, error) {
	// Windows VLAN support is typically through network adapter drivers
	// This implementation uses netsh to configure VLAN tagging

	// Generate name if not provided
	name := config.Name
	if name == "" {
		name = GenerateName(config.ParentInterface, config.ID)
	}

	// Check if VLAN already exists
	exists, err := m.Exists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrVLANExists
	}

	// Windows doesn't have native VLAN interface creation like Linux
	// VLANs are typically configured through the network adapter's advanced properties
	// or using vendor-specific tools

	// For now, we'll use PowerShell to add VLAN via the adapter properties
	// This requires the adapter to support 802.1Q VLAN tagging

	psCmd := fmt.Sprintf(`
		$adapter = Get-NetAdapter -Name "%s" -ErrorAction SilentlyContinue
		if ($adapter) {
			# Add VLAN using Add-NetLbfoTeamNic or Set-NetAdapterAdvancedProperty
			# Note: This is a simplified implementation
			# Real implementation depends on adapter driver support
			Write-Output "VLAN configuration requires adapter-specific commands"
			exit 1
		} else {
			Write-Output "Parent adapter not found"
			exit 1
		}
	`, config.ParentInterface)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create VLAN: %s: %w", string(output), err)
	}

	// Build and return Interface
	iface := &Interface{
		Config:      config,
		SystemName:  name,
		ParentIndex: 0, // Not easily available on Windows
		State:       StateActive,
	}

	return iface, ErrNotSupported // Returning not supported for now
}

// Delete deletes a VLAN interface
func (m *WindowsManager) Delete(name string) error {
	// Windows VLAN deletion
	return ErrNotSupported
}

// Get retrieves a VLAN interface by name
func (m *WindowsManager) Get(name string) (*Interface, error) {
	// Try to get the interface via PowerShell
	psCmd := fmt.Sprintf(`
		$adapter = Get-NetAdapter -Name "%s" -ErrorAction SilentlyContinue
		if ($adapter) {
			# Check if it's a VLAN interface
			$vlanId = Get-NetAdapterAdvancedProperty -Name "%s" -RegistryKeyword "VlanID" -ErrorAction SilentlyContinue
			if ($vlanId) {
				Write-Output "$($adapter.Name)|$($vlanId.RegistryValue)|$($adapter.Status)|$($adapter.LinkSpeed)"
			}
		}
	`, name, name)

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, ErrVLANNotFound
	}

	// Parse output
	parts := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(parts) < 4 {
		return nil, ErrVLANNotFound
	}

	vlanID, _ := strconv.Atoi(parts[1])
	state := StateNone
	if parts[2] == "Up" {
		state = StateActive
	}

	config := &Config{
		ID:              vlanID,
		ParentInterface: "", // Not easily available
		Name:            name,
		Enabled:         state == StateActive,
	}

	iface := &Interface{
		Config:     config,
		SystemName: name,
		State:      state,
	}

	return iface, nil
}

// List lists all VLAN interfaces
func (m *WindowsManager) List() ([]*Interface, error) {
	// Get all network adapters with VLAN configuration
	psCmd := `
		Get-NetAdapter | ForEach-Object {
			$adapter = $_
			$vlanId = Get-NetAdapterAdvancedProperty -Name $adapter.Name -RegistryKeyword "VlanID" -ErrorAction SilentlyContinue
			if ($vlanId -and $vlanId.RegistryValue -ne "0") {
				Write-Output "$($adapter.Name)|$($vlanId.RegistryValue)|$($adapter.Status)"
			}
		}
	`

	cmd := exec.Command("powershell", "-Command", psCmd)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list VLANs: %w", err)
	}

	var vlans []*Interface
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		vlanID, _ := strconv.Atoi(parts[1])
		state := StateNone
		if parts[2] == "Up" {
			state = StateActive
		}

		config := &Config{
			ID:      vlanID,
			Name:    parts[0],
			Enabled: state == StateActive,
		}

		iface := &Interface{
			Config:     config,
			SystemName: parts[0],
			State:      state,
		}

		vlans = append(vlans, iface)
	}

	return vlans, nil
}

// Update updates a VLAN interface configuration
func (m *WindowsManager) Update(name string, config *Config) error {
	// Windows VLAN update
	// Enable/disable adapter
	if config.Enabled {
		cmd := exec.Command("netsh", "interface", "set", "interface", name, "admin=enabled")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to enable interface: %w", err)
		}
	} else {
		cmd := exec.Command("netsh", "interface", "set", "interface", name, "admin=disabled")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to disable interface: %w", err)
		}
	}

	return nil
}

// Exists checks if a VLAN interface exists
func (m *WindowsManager) Exists(name string) (bool, error) {
	psCmd := fmt.Sprintf(`
		$adapter = Get-NetAdapter -Name "%s" -ErrorAction SilentlyContinue
		if ($adapter) {
			exit 0
		} else {
			exit 1
		}
	`, name)

	cmd := exec.Command("powershell", "-Command", psCmd)
	err := cmd.Run()
	return err == nil, nil
}

// parseWindowsSpeed converts Windows link speed string to bps
func parseWindowsSpeed(speed string) uint64 {
	// Speed format: "1 Gbps", "100 Mbps", etc.
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([MGT]?bps)`)
	matches := re.FindStringSubmatch(speed)
	if len(matches) < 3 {
		return 0
	}

	value, _ := strconv.ParseFloat(matches[1], 64)
	unit := matches[2]

	switch unit {
	case "Gbps":
		return uint64(value * 1_000_000_000)
	case "Mbps":
		return uint64(value * 1_000_000)
	case "Kbps":
		return uint64(value * 1_000)
	default:
		return uint64(value)
	}
}

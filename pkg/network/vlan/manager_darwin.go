//go:build darwin
// +build darwin

package vlan

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DarwinManager implements VLAN management for macOS
type DarwinManager struct{}

// newDarwinManager creates a new macOS VLAN manager
func newDarwinManager() (*DarwinManager, error) {
	// Check if running as root
	if os.Geteuid() != 0 {
		return nil, ErrPermissionDenied
	}

	return &DarwinManager{}, nil
}

// Create creates a new VLAN interface on macOS
func (m *DarwinManager) Create(config *Config) (*Interface, error) {
	// Generate name if not provided
	name := config.Name
	if name == "" {
		// macOS uses vlanX naming convention
		name = fmt.Sprintf("vlan%d", config.ID)
	}

	// Check if VLAN already exists
	exists, err := m.Exists(name)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrVLANExists
	}

	// Create VLAN interface using ifconfig
	// ifconfig vlanX create
	cmd := exec.Command("ifconfig", name, "create")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create VLAN interface: %w", err)
	}

	// Configure VLAN
	// ifconfig vlanX vlan vlan-tag vlandev parent-interface
	cmd = exec.Command("ifconfig", name, "vlan", strconv.Itoa(config.ID), "vlandev", config.ParentInterface)
	if err := cmd.Run(); err != nil {
		// Cleanup on failure
		exec.Command("ifconfig", name, "destroy").Run()
		return nil, fmt.Errorf("failed to configure VLAN: %w", err)
	}

	// Set MTU if specified
	if config.MTU > 0 {
		cmd = exec.Command("ifconfig", name, "mtu", strconv.Itoa(config.MTU))
		if err := cmd.Run(); err != nil {
			// Non-fatal, continue
		}
	}

	// Bring interface up if enabled
	if config.Enabled {
		cmd = exec.Command("ifconfig", name, "up")
		if err := cmd.Run(); err != nil {
			// Cleanup on failure
			exec.Command("ifconfig", name, "destroy").Run()
			return nil, fmt.Errorf("failed to bring VLAN up: %w", err)
		}
	}

	// Build and return Interface
	iface := &Interface{
		Config:      config,
		SystemName:  name,
		ParentIndex: 0, // Not easily available on macOS
		State:       StateActive,
	}

	return iface, nil
}

// Delete deletes a VLAN interface
func (m *DarwinManager) Delete(name string) error {
	// Check if exists
	exists, err := m.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrVLANNotFound
	}

	// Destroy VLAN interface
	cmd := exec.Command("ifconfig", name, "destroy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete VLAN: %w", err)
	}

	return nil
}

// Get retrieves a VLAN interface by name
func (m *DarwinManager) Get(name string) (*Interface, error) {
	// Get interface info using ifconfig
	cmd := exec.Command("ifconfig", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, ErrVLANNotFound
	}

	// Parse ifconfig output
	info := string(output)

	// Check if it's a VLAN interface
	if !strings.Contains(info, "vlan:") {
		return nil, fmt.Errorf("interface %s is not a VLAN", name)
	}

	// Extract VLAN ID and parent
	vlanID := 0
	parentInterface := ""

	// Parse "vlan: 100 parent interface: en0"
	vlanRe := regexp.MustCompile(`vlan:\s*(\d+)\s+parent interface:\s*(\w+)`)
	if matches := vlanRe.FindStringSubmatch(info); len(matches) >= 3 {
		vlanID, _ = strconv.Atoi(matches[1])
		parentInterface = matches[2]
	}

	// Extract MTU
	mtu := 1500
	mtuRe := regexp.MustCompile(`mtu\s+(\d+)`)
	if matches := mtuRe.FindStringSubmatch(info); len(matches) >= 2 {
		mtu, _ = strconv.Atoi(matches[1])
	}

	// Determine state
	state := StateNone
	if strings.Contains(info, "<UP,") || strings.Contains(info, ",UP,") || strings.Contains(info, ",UP>") {
		state = StateActive
	}

	// Build config
	config := &Config{
		ID:              vlanID,
		ParentInterface: parentInterface,
		Name:            name,
		MTU:             mtu,
		Enabled:         state == StateActive,
	}

	// Build interface
	iface := &Interface{
		Config:     config,
		SystemName: name,
		State:      state,
	}

	return iface, nil
}

// List lists all VLAN interfaces
func (m *DarwinManager) List() ([]*Interface, error) {
	// Get all interfaces
	cmd := exec.Command("ifconfig", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var vlans []*Interface

	// Parse ifconfig output
	// Split by interface (lines starting with non-whitespace)
	lines := strings.Split(string(output), "\n")
	var currentInterface string
	var currentBlock []string

	for _, line := range lines {
		// New interface starts with non-whitespace
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			// Process previous block
			if currentInterface != "" && len(currentBlock) > 0 {
				iface, err := m.parseInterfaceBlock(currentInterface, currentBlock)
				if err == nil && iface != nil {
					vlans = append(vlans, iface)
				}
			}

			// Start new block
			parts := strings.SplitN(line, ":", 2)
			if len(parts) >= 1 {
				currentInterface = parts[0]
				currentBlock = []string{line}
			}
		} else {
			currentBlock = append(currentBlock, line)
		}
	}

	// Process last block
	if currentInterface != "" && len(currentBlock) > 0 {
		iface, err := m.parseInterfaceBlock(currentInterface, currentBlock)
		if err == nil && iface != nil {
			vlans = append(vlans, iface)
		}
	}

	return vlans, nil
}

// parseInterfaceBlock parses an ifconfig block for a single interface
func (m *DarwinManager) parseInterfaceBlock(name string, lines []string) (*Interface, error) {
	info := strings.Join(lines, "\n")

	// Check if it's a VLAN interface
	if !strings.Contains(info, "vlan:") {
		return nil, nil // Not a VLAN, skip
	}

	// Extract VLAN ID and parent
	vlanID := 0
	parentInterface := ""

	vlanRe := regexp.MustCompile(`vlan:\s*(\d+)\s+parent interface:\s*(\w+)`)
	if matches := vlanRe.FindStringSubmatch(info); len(matches) >= 3 {
		vlanID, _ = strconv.Atoi(matches[1])
		parentInterface = matches[2]
	}

	// Extract MTU
	mtu := 1500
	mtuRe := regexp.MustCompile(`mtu\s+(\d+)`)
	if matches := mtuRe.FindStringSubmatch(info); len(matches) >= 2 {
		mtu, _ = strconv.Atoi(matches[1])
	}

	// Determine state
	state := StateNone
	if strings.Contains(info, "<UP,") || strings.Contains(info, ",UP,") || strings.Contains(info, ",UP>") {
		state = StateActive
	}

	// Build config
	config := &Config{
		ID:              vlanID,
		ParentInterface: parentInterface,
		Name:            name,
		MTU:             mtu,
		Enabled:         state == StateActive,
	}

	// Build interface
	iface := &Interface{
		Config:     config,
		SystemName: name,
		State:      state,
	}

	return iface, nil
}

// Update updates a VLAN interface configuration
func (m *DarwinManager) Update(name string, config *Config) error {
	// Check if exists
	exists, err := m.Exists(name)
	if err != nil {
		return err
	}
	if !exists {
		return ErrVLANNotFound
	}

	// Update MTU if specified
	if config.MTU > 0 {
		cmd := exec.Command("ifconfig", name, "mtu", strconv.Itoa(config.MTU))
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to update MTU: %w", err)
		}
	}

	// Update link state
	if config.Enabled {
		cmd := exec.Command("ifconfig", name, "up")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to bring interface up: %w", err)
		}
	} else {
		cmd := exec.Command("ifconfig", name, "down")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to bring interface down: %w", err)
		}
	}

	// Note: VLAN ID and parent cannot be changed after creation
	// Would need to delete and recreate

	return nil
}

// Exists checks if a VLAN interface exists
func (m *DarwinManager) Exists(name string) (bool, error) {
	cmd := exec.Command("ifconfig", name)
	err := cmd.Run()
	if err != nil {
		// Interface doesn't exist
		return false, nil
	}
	return true, nil
}

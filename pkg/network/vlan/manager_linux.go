//go:build linux
// +build linux

package vlan

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/vishvananda/netlink"
)

// LinuxManager implements VLAN management for Linux using netlink
type LinuxManager struct{}

// newLinuxManager creates a new Linux VLAN manager
func newLinuxManager() (*LinuxManager, error) {
	// Check if we have necessary permissions
	if os.Geteuid() != 0 {
		return nil, ErrPermissionDenied
	}

	return &LinuxManager{}, nil
}

// Create creates a new VLAN interface on Linux
func (m *LinuxManager) Create(config *Config) (*Interface, error) {
	// Get parent link
	parentLink, err := netlink.LinkByName(config.ParentInterface)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParentNotFound, err)
	}

	// Generate name if not provided
	name := config.Name
	if name == "" {
		name = GenerateName(config.ParentInterface, config.ID)
	}

	// Check if VLAN already exists
	if _, err := netlink.LinkByName(name); err == nil {
		return nil, ErrVLANExists
	}

	// Create VLAN link attributes
	vlanLink := &netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        name,
			ParentIndex: parentLink.Attrs().Index,
			MTU:         config.MTU,
		},
		VlanId:       config.ID,
		VlanProtocol: netlink.VLAN_PROTOCOL_8021Q,
	}

	// Create the VLAN
	if err := netlink.LinkAdd(vlanLink); err != nil {
		return nil, fmt.Errorf("failed to create VLAN interface: %w", err)
	}

	// Set link up if enabled
	if config.Enabled {
		link, err := netlink.LinkByName(name)
		if err != nil {
			// Cleanup on failure
			netlink.LinkDel(vlanLink)
			return nil, fmt.Errorf("failed to get created VLAN: %w", err)
		}

		if err := netlink.LinkSetUp(link); err != nil {
			// Cleanup on failure
			netlink.LinkDel(vlanLink)
			return nil, fmt.Errorf("failed to bring VLAN up: %w", err)
		}
	}

	// Build and return Interface
	iface := &Interface{
		Config:      config,
		SystemName:  name,
		ParentIndex: parentLink.Attrs().Index,
		State:       StateActive,
	}

	return iface, nil
}

// Delete deletes a VLAN interface
func (m *LinuxManager) Delete(name string) error {
	// Get the link
	link, err := netlink.LinkByName(name)
	if err != nil {
		return ErrVLANNotFound
	}

	// Verify it's actually a VLAN
	if _, ok := link.(*netlink.Vlan); !ok {
		return fmt.Errorf("interface %s is not a VLAN", name)
	}

	// Check if interface is in use (has assigned IPs)
	addrs, err := netlink.AddrList(link, netlink.FAMILY_ALL)
	if err == nil && len(addrs) > 0 {
		// Interface has IP addresses, might be in use
		// This is a soft check - we'll still allow deletion
	}

	// Delete the VLAN
	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("failed to delete VLAN: %w", err)
	}

	return nil
}

// Get retrieves a VLAN interface by name
func (m *LinuxManager) Get(name string) (*Interface, error) {
	// Get the link
	link, err := netlink.LinkByName(name)
	if err != nil {
		return nil, ErrVLANNotFound
	}

	// Verify it's a VLAN
	vlanLink, ok := link.(*netlink.Vlan)
	if !ok {
		return nil, fmt.Errorf("interface %s is not a VLAN", name)
	}

	// Get parent link
	parentLink, err := netlink.LinkByIndex(vlanLink.Attrs().ParentIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent interface: %w", err)
	}

	// Determine state
	state := StateNone
	attrs := vlanLink.Attrs()
	if attrs.Flags&net.FlagUp != 0 {
		state = StateActive
	}

	// Build config
	config := &Config{
		ID:              vlanLink.VlanId,
		ParentInterface: parentLink.Attrs().Name,
		Name:            name,
		Priority:        0, // Not directly available from netlink
		MTU:             attrs.MTU,
		Enabled:         state == StateActive,
		CreatedAt:       time.Time{}, // Not available
		UpdatedAt:       time.Time{}, // Not available
	}

	// Build interface
	iface := &Interface{
		Config:      config,
		SystemName:  name,
		ParentIndex: attrs.ParentIndex,
		State:       state,
	}

	return iface, nil
}

// List lists all VLAN interfaces
func (m *LinuxManager) List() ([]*Interface, error) {
	// Get all links
	links, err := netlink.LinkList()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	var vlans []*Interface

	// Filter for VLANs
	for _, link := range links {
		vlanLink, ok := link.(*netlink.Vlan)
		if !ok {
			continue
		}

		// Get parent link
		parentLink, err := netlink.LinkByIndex(vlanLink.Attrs().ParentIndex)
		if err != nil {
			continue // Skip if parent not found
		}

		// Determine state
		state := StateNone
		attrs := vlanLink.Attrs()
		if attrs.Flags&net.FlagUp != 0 {
			state = StateActive
		}

		// Build config
		config := &Config{
			ID:              vlanLink.VlanId,
			ParentInterface: parentLink.Attrs().Name,
			Name:            attrs.Name,
			Priority:        0,
			MTU:             attrs.MTU,
			Enabled:         state == StateActive,
		}

		// Build interface
		iface := &Interface{
			Config:      config,
			SystemName:  attrs.Name,
			ParentIndex: attrs.ParentIndex,
			State:       state,
		}

		vlans = append(vlans, iface)
	}

	return vlans, nil
}

// Update updates a VLAN interface configuration
func (m *LinuxManager) Update(name string, config *Config) error {
	// Get the link
	link, err := netlink.LinkByName(name)
	if err != nil {
		return ErrVLANNotFound
	}

	// Verify it's a VLAN
	if _, ok := link.(*netlink.Vlan); !ok {
		return fmt.Errorf("interface %s is not a VLAN", name)
	}

	// Update MTU if specified
	if config.MTU > 0 && config.MTU != link.Attrs().MTU {
		if err := netlink.LinkSetMTU(link, config.MTU); err != nil {
			return fmt.Errorf("failed to update MTU: %w", err)
		}
	}

	// Update link state
	if config.Enabled {
		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("failed to bring interface up: %w", err)
		}
	} else {
		if err := netlink.LinkSetDown(link); err != nil {
			return fmt.Errorf("failed to bring interface down: %w", err)
		}
	}

	// Note: VLAN ID and parent cannot be changed after creation
	// Would need to delete and recreate

	return nil
}

// Exists checks if a VLAN interface exists
func (m *LinuxManager) Exists(name string) (bool, error) {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if _, ok := err.(netlink.LinkNotFoundError); ok {
			return false, nil
		}
		return false, err
	}

	// Verify it's actually a VLAN
	if _, ok := link.(*netlink.Vlan); !ok {
		return false, nil
	}

	return true, nil
}

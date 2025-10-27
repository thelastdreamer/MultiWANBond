package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/network/vlan"
)

func main() {
	fmt.Println("=== MultiWANBond VLAN Management Test ===\n")

	// Create VLAN manager
	fmt.Println("Creating VLAN manager...")
	manager, err := vlan.NewManager()
	if err != nil {
		fmt.Printf("Error creating VLAN manager: %v\n", err)
		fmt.Println("\nNote: VLAN management requires root/administrator privileges.")
		os.Exit(1)
	}
	fmt.Println("✓ VLAN manager created successfully\n")

	// List existing VLANs
	fmt.Println("--- Existing VLANs ---")
	existingVLANs, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing VLANs: %v\n", err)
	} else {
		if len(existingVLANs) == 0 {
			fmt.Println("No existing VLANs found")
		} else {
			for _, v := range existingVLANs {
				fmt.Printf("  - %s (ID: %d, Parent: %s, State: %s)\n",
					v.SystemName, v.Config.ID, v.Config.ParentInterface, v.State)
			}
		}
	}
	fmt.Println()

	// Interactive VLAN creation test
	fmt.Println("--- VLAN Creation Test ---")
	fmt.Println("This test will attempt to create a VLAN interface.")
	fmt.Println("Press Ctrl+C to skip, or follow the prompts.\n")

	// Get parent interface
	var parentInterface string
	fmt.Print("Enter parent interface name (e.g., eth0, en0, Ethernet): ")
	fmt.Scanln(&parentInterface)

	if parentInterface == "" {
		fmt.Println("No parent interface provided, skipping VLAN creation test.")
		return
	}

	// Get VLAN ID
	var vlanID int
	fmt.Print("Enter VLAN ID (2-4094): ")
	fmt.Scanln(&vlanID)

	if vlanID < 2 || vlanID > 4094 {
		fmt.Println("Invalid VLAN ID, skipping VLAN creation test.")
		return
	}

	// Create test VLAN
	fmt.Printf("\nCreating VLAN %d on %s...\n", vlanID, parentInterface)
	config := &vlan.Config{
		ID:              vlanID,
		ParentInterface: parentInterface,
		Name:            "", // Auto-generate
		DisplayName:     "Test VLAN",
		Priority:        0,
		MTU:             1500,
		Enabled:         true,
		AutoCreate:      false,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	iface, err := manager.Create(config)
	if err != nil {
		fmt.Printf("✗ Error creating VLAN: %v\n", err)
		fmt.Println("\nCommon issues:")
		fmt.Println("  - Parent interface doesn't exist")
		fmt.Println("  - VLAN ID already in use")
		fmt.Println("  - Insufficient permissions (need root/administrator)")
		fmt.Println("  - Platform doesn't support VLAN creation")
		return
	}

	fmt.Printf("✓ VLAN created successfully: %s\n", iface.SystemName)
	fmt.Printf("  System Name: %s\n", iface.SystemName)
	fmt.Printf("  VLAN ID: %d\n", iface.Config.ID)
	fmt.Printf("  Parent: %s\n", iface.Config.ParentInterface)
	fmt.Printf("  State: %s\n", iface.State)
	fmt.Println()

	// Test Get
	fmt.Println("--- Testing Get VLAN ---")
	retrievedIface, err := manager.Get(iface.SystemName)
	if err != nil {
		fmt.Printf("✗ Error retrieving VLAN: %v\n", err)
	} else {
		fmt.Printf("✓ Successfully retrieved VLAN: %s\n", retrievedIface.SystemName)
	}
	fmt.Println()

	// Test Exists
	fmt.Println("--- Testing Exists ---")
	exists, err := manager.Exists(iface.SystemName)
	if err != nil {
		fmt.Printf("✗ Error checking existence: %v\n", err)
	} else {
		fmt.Printf("✓ VLAN exists: %v\n", exists)
	}
	fmt.Println()

	// Test Update
	fmt.Println("--- Testing Update ---")
	updateConfig := &vlan.Config{
		ID:              vlanID,
		ParentInterface: parentInterface,
		Name:            iface.SystemName,
		DisplayName:     "Updated Test VLAN",
		Priority:        0,
		MTU:             1400,
		Enabled:         true,
	}

	err = manager.Update(iface.SystemName, updateConfig)
	if err != nil {
		fmt.Printf("✗ Error updating VLAN: %v\n", err)
	} else {
		fmt.Println("✓ VLAN updated successfully")
	}
	fmt.Println()

	// Test List (should now show our created VLAN)
	fmt.Println("--- Testing List (after creation) ---")
	vlans, err := manager.List()
	if err != nil {
		fmt.Printf("✗ Error listing VLANs: %v\n", err)
	} else {
		fmt.Printf("✓ Found %d VLAN(s):\n", len(vlans))
		for _, v := range vlans {
			fmt.Printf("  - %s (ID: %d, Parent: %s, State: %s)\n",
				v.SystemName, v.Config.ID, v.Config.ParentInterface, v.State)
		}
	}
	fmt.Println()

	// Ask if user wants to delete the test VLAN
	fmt.Print("Delete the test VLAN? (y/n): ")
	var response string
	fmt.Scanln(&response)

	if response == "y" || response == "Y" {
		fmt.Printf("\nDeleting VLAN %s...\n", iface.SystemName)
		err = manager.Delete(iface.SystemName)
		if err != nil {
			fmt.Printf("✗ Error deleting VLAN: %v\n", err)
		} else {
			fmt.Println("✓ VLAN deleted successfully")
		}

		// Verify deletion
		exists, err := manager.Exists(iface.SystemName)
		if err != nil {
			fmt.Printf("✗ Error checking existence: %v\n", err)
		} else if exists {
			fmt.Println("✗ VLAN still exists after deletion")
		} else {
			fmt.Println("✓ VLAN no longer exists")
		}
	} else {
		fmt.Printf("\nTest VLAN %s left in place. You can manually delete it with:\n", iface.SystemName)
		fmt.Printf("  Linux:   sudo ip link delete %s\n", iface.SystemName)
		fmt.Printf("  macOS:   sudo ifconfig %s destroy\n", iface.SystemName)
		fmt.Printf("  Windows: Remove via Network Connections\n")
	}

	fmt.Println("\n=== VLAN Test Complete ===")
}

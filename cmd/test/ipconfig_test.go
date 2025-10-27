package main

import (
	"fmt"
	"os"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/network/ipconfig"
)

func main() {
	fmt.Println("=== MultiWANBond IP Configuration Test ===\n")

	// Create IP configuration manager
	fmt.Println("Creating IP configuration manager...")
	manager, err := ipconfig.NewManager()
	if err != nil {
		fmt.Printf("Error creating IP configuration manager: %v\n", err)
		fmt.Println("\nNote: IP configuration requires root/administrator privileges.")
		os.Exit(1)
	}
	fmt.Println("✓ IP configuration manager created successfully\n")

	// List all interfaces
	fmt.Println("--- Current Interface States ---")
	states, err := manager.List()
	if err != nil {
		fmt.Printf("Error listing interfaces: %v\n", err)
	} else {
		if len(states) == 0 {
			fmt.Println("No interfaces found")
		} else {
			for name, state := range states {
				fmt.Printf("\nInterface: %s\n", name)
				fmt.Printf("  Status: Up=%v, Carrier=%v\n", state.IsUp, state.HasCarrier)

				if len(state.IPv4Addresses) > 0 {
					fmt.Printf("  IPv4 Addresses:\n")
					for _, addr := range state.IPv4Addresses {
						fmt.Printf("    - %s\n", addr.String())
					}
				}

				if state.IPv4Gateway != nil {
					fmt.Printf("  IPv4 Gateway: %s\n", state.IPv4Gateway.String())
				}

				if len(state.IPv6Addresses) > 0 {
					fmt.Printf("  IPv6 Addresses:\n")
					for _, addr := range state.IPv6Addresses {
						fmt.Printf("    - %s\n", addr.String())
					}
				}

				if len(state.DNSServers) > 0 {
					fmt.Printf("  DNS Servers:\n")
					for _, dns := range state.DNSServers {
						fmt.Printf("    - %s\n", dns.String())
					}
				}
			}
		}
	}
	fmt.Println()

	// List routes
	fmt.Println("--- Current Routes ---")
	routes, err := manager.ListRoutes()
	if err != nil {
		fmt.Printf("Error listing routes: %v\n", err)
	} else {
		if len(routes) == 0 {
			fmt.Println("No routes found")
		} else {
			// Group and limit output
			defaultRoutes := 0
			otherRoutes := 0

			fmt.Println("Default Routes:")
			for _, route := range routes {
				if route.Destination == "0.0.0.0/0" || route.Destination == "::/0" {
					fmt.Printf("  - %s via %s", route.Destination, route.Gateway)
					if route.Interface != "" {
						fmt.Printf(" dev %s", route.Interface)
					}
					if route.Metric > 0 {
						fmt.Printf(" metric %d", route.Metric)
					}
					fmt.Println()
					defaultRoutes++
					if defaultRoutes >= 5 {
						break
					}
				}
			}

			fmt.Println("\nOther Routes (showing first 10):")
			for _, route := range routes {
				if route.Destination != "0.0.0.0/0" && route.Destination != "::/0" {
					fmt.Printf("  - %s", route.Destination)
					if route.Gateway != "" {
						fmt.Printf(" via %s", route.Gateway)
					}
					if route.Interface != "" {
						fmt.Printf(" dev %s", route.Interface)
					}
					fmt.Println()
					otherRoutes++
					if otherRoutes >= 10 {
						fmt.Println("  ... and more")
						break
					}
				}
			}
		}
	}
	fmt.Println()

	// Interactive configuration test (optional)
	fmt.Println("--- Interactive Configuration Test ---")
	fmt.Println("WARNING: The following test will modify network configuration!")
	fmt.Println("Only proceed if you understand the risks.")
	fmt.Print("\nDo you want to test IP configuration? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("Skipping interactive test. Exiting safely.")
		return
	}

	// Get interface name
	var interfaceName string
	fmt.Print("\nEnter interface name to configure: ")
	fmt.Scanln(&interfaceName)

	if interfaceName == "" {
		fmt.Println("No interface specified, exiting.")
		return
	}

	// Verify interface exists
	state, err := manager.Get(interfaceName)
	if err != nil {
		fmt.Printf("Error: Interface %s not found: %v\n", interfaceName, err)
		return
	}

	fmt.Printf("\nCurrent configuration for %s:\n", interfaceName)
	fmt.Printf("  IPv4: %v\n", state.IPv4Addresses)
	fmt.Printf("  Gateway: %v\n", state.IPv4Gateway)

	// Choose configuration method
	fmt.Println("\nConfiguration methods:")
	fmt.Println("  1. Static IP")
	fmt.Println("  2. DHCP")
	fmt.Print("Choose method (1 or 2): ")

	var method int
	fmt.Scanln(&method)

	var config *ipconfig.IPConfig

	if method == 1 {
		// Static IP
		var ipAddress, netmask, gateway string

		fmt.Print("Enter IP address (e.g., 192.168.1.100): ")
		fmt.Scanln(&ipAddress)

		fmt.Print("Enter netmask or CIDR (e.g., 255.255.255.0 or 24): ")
		fmt.Scanln(&netmask)

		fmt.Print("Enter gateway (optional, press Enter to skip): ")
		fmt.Scanln(&gateway)

		config = &ipconfig.IPConfig{
			InterfaceName: interfaceName,
			IPv4Method:    ipconfig.ConfigMethodStatic,
			IPv4Address:   ipAddress,
			IPv4Netmask:   netmask,
			GatewayMethod: ipconfig.GatewayMethodDisable,
		}

		if gateway != "" {
			config.GatewayMethod = ipconfig.GatewayMethodStatic
			config.IPv4Gateway = gateway
		}

	} else if method == 2 {
		// DHCP
		config = &ipconfig.IPConfig{
			InterfaceName: interfaceName,
			IPv4Method:    ipconfig.ConfigMethodDHCP,
			DHCPTimeout:   30 * time.Second,
		}
	} else {
		fmt.Println("Invalid method selected.")
		return
	}

	// Apply configuration
	fmt.Printf("\nApplying configuration to %s...\n", interfaceName)
	if err := manager.Apply(config); err != nil {
		fmt.Printf("✗ Error applying configuration: %v\n", err)
		return
	}

	fmt.Println("✓ Configuration applied successfully!")

	// Wait a moment for configuration to take effect
	time.Sleep(2 * time.Second)

	// Verify new configuration
	fmt.Println("\nVerifying new configuration...")
	newState, err := manager.Get(interfaceName)
	if err != nil {
		fmt.Printf("Error getting new state: %v\n", err)
	} else {
		fmt.Printf("New IPv4 Addresses: %v\n", newState.IPv4Addresses)
		fmt.Printf("New Gateway: %v\n", newState.IPv4Gateway)
	}

	// Ask about restoring
	fmt.Print("\nDo you want to restore the original configuration? (yes/no): ")
	fmt.Scanln(&response)

	if response == "yes" {
		fmt.Println("To restore manually, you may need to:")
		fmt.Println("  - Use your system's network configuration tools")
		fmt.Println("  - Or set DHCP: sudo dhclient", interfaceName)
		fmt.Println("  - Or set static IP using system tools")
	}

	fmt.Println("\n=== IP Configuration Test Complete ===")
}

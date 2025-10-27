package main

import (
	"context"
	"fmt"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/network"
)

func main() {
	fmt.Println("MultiWANBond Network Detection Test")
	fmt.Println("====================================\n")

	// Create detector
	detector, err := network.NewDetector()
	if err != nil {
		fmt.Printf("Error creating detector: %v\n", err)
		return
	}

	// Detect all interfaces
	fmt.Println("Detecting all network interfaces...")
	interfaces, err := detector.DetectAll()
	if err != nil {
		fmt.Printf("Error detecting interfaces: %v\n", err)
		return
	}

	fmt.Printf("\nFound %d network interfaces:\n\n", len(interfaces))

	// Display each interface
	for i, iface := range interfaces {
		fmt.Printf("%d. %s (%s)\n", i+1, iface.SystemName, iface.Type)
		fmt.Printf("   Display Name: %s\n", iface.DisplayName)
		fmt.Printf("   MAC Address:  %s\n", iface.MACAddress)
		fmt.Printf("   MTU:          %d\n", iface.MTU)
		fmt.Printf("   Admin State:  %s\n", iface.AdminState)
		fmt.Printf("   Oper State:   %s\n", iface.OperState)
		fmt.Printf("   Has Carrier:  %v\n", iface.HasCarrier)

		if iface.Speed > 0 {
			speedMbps := iface.Speed / 1000000
			fmt.Printf("   Speed:        %d Mbps (%s)\n", speedMbps, iface.Duplex)
		} else {
			fmt.Printf("   Speed:        Unknown\n")
		}

		if iface.Driver != "" {
			fmt.Printf("   Driver:       %s\n", iface.Driver)
		}

		// IP addresses
		if len(iface.IPv4Addresses) > 0 {
			fmt.Printf("   IPv4:         ")
			for j, ip := range iface.IPv4Addresses {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", ip)
			}
			fmt.Println()
		}

		if len(iface.IPv6Addresses) > 0 {
			fmt.Printf("   IPv6:         ")
			for j, ip := range iface.IPv6Addresses {
				if j > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", ip)
			}
			fmt.Println()
		}

		// Connectivity
		fmt.Printf("   Has IP:       %v\n", iface.HasIP)
		fmt.Printf("   Has Internet: %v", iface.HasInternet)
		if iface.HasInternet {
			fmt.Printf(" (latency: %v)", iface.TestLatency)
		}
		fmt.Println()

		// VLAN info
		if iface.VLANInfo != nil {
			fmt.Printf("   VLAN ID:      %d\n", iface.VLANInfo.ID)
			fmt.Printf("   VLAN Parent:  %s\n", iface.VLANInfo.Parent)
		}

		// Bond info
		if iface.BondInfo != nil {
			fmt.Printf("   Bond Mode:    %s\n", iface.BondInfo.Mode)
			fmt.Printf("   Bond Slaves:  %v\n", iface.BondInfo.Slaves)
		}

		// Bridge info
		if iface.BridgeInfo != nil {
			fmt.Printf("   Bridge Members: %v\n", iface.BridgeInfo.Members)
		}

		// Statistics
		if iface.RxBytes > 0 || iface.TxBytes > 0 {
			fmt.Printf("   Statistics:\n")
			fmt.Printf("     RX: %d bytes, %d packets, %d errors\n",
				iface.RxBytes, iface.RxPackets, iface.RxErrors)
			fmt.Printf("     TX: %d bytes, %d packets, %d errors\n",
				iface.TxBytes, iface.TxPackets, iface.TxErrors)
		}

		fmt.Println()
	}

	// Test specific interface detection
	if len(interfaces) > 0 {
		testIface := interfaces[0].SystemName
		fmt.Printf("Testing detection of specific interface: %s\n", testIface)
		specific, err := detector.DetectByName(testIface)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Successfully detected: %s\n", specific.SystemName)
		}
		fmt.Println()
	}

	// Test capabilities
	for _, iface := range interfaces {
		if network.IsInterfaceUsable(iface) {
			fmt.Printf("Testing capabilities for %s...\n", iface.SystemName)
			caps, err := detector.GetCapabilities(iface.SystemName)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("  VLAN Support:    %v\n", caps.SupportsVLAN)
				fmt.Printf("  Bonding Support: %v\n", caps.SupportsBonding)
				fmt.Printf("  Bridge Support:  %v\n", caps.SupportsBridge)
				fmt.Printf("  TSO:             %v\n", caps.SupportsTSO)
				fmt.Printf("  GSO:             %v\n", caps.SupportsGSO)
				fmt.Printf("  GRO:             %v\n", caps.SupportsGRO)
			}
			fmt.Println()
			break // Just test one usable interface
		}
	}

	// Filter usable interfaces
	usable := network.GetUsableInterfaces(interfaces)
	fmt.Printf("Usable interfaces for WAN bonding: %d\n", len(usable))
	for i, iface := range usable {
		fmt.Printf("  %d. %s - %s", i+1, iface.SystemName, iface.Type)
		if iface.Speed > 0 {
			fmt.Printf(" (%d Mbps)", iface.Speed/1000000)
		}
		if iface.HasInternet {
			fmt.Printf(" [Internet OK - %v]", iface.TestLatency)
		}
		fmt.Println()
	}
	fmt.Println()

	// Test monitoring (just for a few seconds)
	fmt.Println("Testing interface monitoring for 10 seconds...")
	fmt.Println("(Try unplugging/plugging a cable if you want to see changes)")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	changeCh, err := detector.Monitor(ctx)
	if err != nil {
		fmt.Printf("Error starting monitor: %v\n", err)
		return
	}

	changeCount := 0
	monitorDone := make(chan bool)

	go func() {
		for change := range changeCh {
			changeCount++
			fmt.Printf("  [%v] %s: %s (%s -> %s)\n",
				change.Timestamp.Format("15:04:05"),
				change.InterfaceName,
				change.ChangeType,
				change.OldState,
				change.NewState)
		}
		monitorDone <- true
	}()

	<-ctx.Done()
	<-monitorDone

	if changeCount == 0 {
		fmt.Println("  No changes detected")
	} else {
		fmt.Printf("  Detected %d changes\n", changeCount)
	}

	fmt.Println("\nNetwork detection test completed successfully!")
}

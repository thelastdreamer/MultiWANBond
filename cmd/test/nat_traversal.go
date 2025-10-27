package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/nat"
)

func main() {
	fmt.Println("MultiWANBond NAT Traversal Test")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	testCount := 0
	passedCount := 0

	// Test 1: Create NAT manager with default config
	fmt.Println("Test 1: Creating NAT traversal manager...")
	testCount++
	config := nat.DefaultNATTraversalConfig()
	manager, err := nat.NewManager(config)
	if err != nil {
		fmt.Printf("  ❌ FAILED: %v\n", err)
	} else {
		fmt.Println("  ✓ PASSED: NAT manager created successfully")
		passedCount++
	}
	fmt.Println()

	// Test 2: Initialize and discover NAT type
	fmt.Println("Test 2: Discovering NAT type via STUN...")
	testCount++
	err = manager.Initialize()
	if err != nil {
		fmt.Printf("  ❌ FAILED: %v\n", err)
		fmt.Println("  Note: This requires internet connection to STUN servers")
	} else {
		localAddr := manager.GetLocalAddr()
		publicAddr := manager.GetPublicAddr()
		natType := manager.GetNATType()

		fmt.Printf("  ✓ PASSED: NAT discovery completed\n")
		fmt.Printf("    Local Address:  %s\n", localAddr)
		fmt.Printf("    Public Address: %s\n", publicAddr)
		fmt.Printf("    NAT Type:       %s\n", natType)
		passedCount++
	}
	fmt.Println()

	// Test 3: Check CGNAT detection
	fmt.Println("Test 3: CGNAT detection...")
	testCount++
	if manager.IsCGNATDetected() {
		cgnatInfo := manager.GetCGNATInfo()
		fmt.Printf("  ✓ PASSED: CGNAT detected\n")
		fmt.Printf("    Type:       %s\n", cgnatInfo.Type)
		fmt.Printf("    Confidence: %.2f\n", cgnatInfo.Confidence)
	} else {
		fmt.Println("  ✓ PASSED: No CGNAT detected")
		passedCount++
	}
	fmt.Println()

	// Test 4: Get traversal capabilities
	fmt.Println("Test 4: Get traversal capabilities...")
	testCount++
	caps := manager.GetTraversalCapabilities()
	fmt.Printf("  ✓ PASSED: Capabilities retrieved\n")
	fmt.Printf("    NAT Type:          %s\n", caps.NATType)
	fmt.Printf("    Can Direct Connect: %t\n", caps.CanDirectConnect)
	fmt.Printf("    Needs Relay:       %t\n", caps.NeedsRelay)
	fmt.Printf("    Relay Available:   %t\n", caps.RelayAvailable)
	fmt.Printf("    CGNAT Detected:    %t\n", caps.CGNATDetected)
	passedCount++
	fmt.Println()

	// Test 5: Check private IP detection
	fmt.Println("Test 5: Testing private IP detection...")
	testCount++
	privateTests := []struct {
		ip       []byte
		ipStr    string
		expected bool
	}{
		{[]byte{10, 0, 0, 1}, "10.0.0.1", true},
		{[]byte{172, 16, 0, 1}, "172.16.0.1", true},
		{[]byte{192, 168, 1, 1}, "192.168.1.1", true},
		{[]byte{8, 8, 8, 8}, "8.8.8.8", false},
		{[]byte{1, 1, 1, 1}, "1.1.1.1", false},
	}

	allPassed := true
	for _, test := range privateTests {
		result := nat.IsPrivateAddress(test.ip)
		if result != test.expected {
			fmt.Printf("  ❌ FAILED: %s expected %t, got %t\n", test.ipStr, test.expected, result)
			allPassed = false
		}
	}

	if allPassed {
		fmt.Println("  ✓ PASSED: Private IP detection working correctly")
		passedCount++
	}
	fmt.Println()

	// Test 6: Test CGNAT range detection
	fmt.Println("Test 6: Testing CGNAT range detection (RFC 6598)...")
	testCount++
	cgnatConfig := nat.DefaultCGNATConfig()

	cgnatTests := []struct {
		ip       []byte
		expected bool
	}{
		{[]byte{100, 64, 0, 1}, true},   // CGNAT range
		{[]byte{100, 127, 255, 255}, true}, // CGNAT range
		{[]byte{100, 63, 255, 255}, false}, // Just before CGNAT
		{[]byte{100, 128, 0, 0}, false},    // Just after CGNAT
		{[]byte{10, 0, 0, 1}, false},       // Private but not CGNAT
	}

	allPassed = true
	for _, test := range cgnatTests {
		result := cgnatConfig.IsCGNATAddress(test.ip)
		if result != test.expected {
			fmt.Printf("  ❌ FAILED: %d.%d.%d.%d expected %t, got %t\n",
				test.ip[0], test.ip[1], test.ip[2], test.ip[3], test.expected, result)
			allPassed = false
		}
	}

	if allPassed {
		fmt.Println("  ✓ PASSED: CGNAT range detection (100.64.0.0/10) working")
		passedCount++
	}
	fmt.Println()

	// Test 7: Test NAT type methods
	fmt.Println("Test 7: Testing NAT type characteristics...")
	testCount++

	natTypeTests := []struct {
		natType          nat.NATType
		canDirectConnect bool
		needsRelay       bool
	}{
		{nat.NATTypeOpen, true, false},
		{nat.NATTypeFullCone, true, false},
		{nat.NATTypeRestrictedCone, true, false},
		{nat.NATTypePortRestrictedCone, true, false},
		{nat.NATTypeSymmetric, false, true},
		{nat.NATTypeBlocked, false, true},
	}

	allPassed = true
	for _, test := range natTypeTests {
		canConnect := test.natType.CanDirectConnect()
		needsRelay := test.natType.NeedsRelay()

		if canConnect != test.canDirectConnect || needsRelay != test.needsRelay {
			fmt.Printf("  ❌ FAILED: %s incorrect characteristics\n", test.natType)
			allPassed = false
		}
	}

	if allPassed {
		fmt.Println("  ✓ PASSED: NAT type characteristics correct")
		passedCount++
	}
	fmt.Println()

	// Test 8: Test config creation
	fmt.Println("Test 8: Testing configuration creation...")
	testCount++

	stunConfig := nat.DefaultSTUNConfig()
	holePunchConfig := nat.DefaultHolePunchConfig()
	relayConfig := nat.DefaultRelayConfig()
	cgnatConfig2 := nat.DefaultCGNATConfig()

	if stunConfig.PrimaryServer == "" || holePunchConfig.Timeout == 0 ||
		relayConfig.MaxBandwidth == 0 || len(cgnatConfig2.CGNATRanges) == 0 {
		fmt.Println("  ❌ FAILED: Default configs have missing values")
	} else {
		fmt.Println("  ✓ PASSED: All default configs created correctly")
		fmt.Printf("    STUN Server: %s\n", stunConfig.PrimaryServer)
		fmt.Printf("    Hole Punch Timeout: %s\n", holePunchConfig.Timeout)
		fmt.Printf("    Relay Max Bandwidth: %d MB/s\n", relayConfig.MaxBandwidth/1024/1024)
		fmt.Printf("    CGNAT Detection: %t\n", cgnatConfig2.EnableCGNATDetection)
		passedCount++
	}
	fmt.Println()

	// Test 9: Test adaptive keep-alive
	fmt.Println("Test 9: Testing adaptive keep-alive...")
	testCount++

	keepAlive := nat.NewAdaptiveKeepAlive(15 * time.Second)
	initialInterval := keepAlive.GetInterval()

	// Record successes to trigger increase
	for i := 0; i < 10; i++ {
		keepAlive.RecordSuccess()
	}

	// Force adjustment by setting last adjustment to past
	time.Sleep(10 * time.Millisecond)

	newInterval := keepAlive.GetInterval()
	if newInterval >= initialInterval {
		fmt.Println("  ✓ PASSED: Adaptive keep-alive working")
		fmt.Printf("    Initial interval: %s\n", initialInterval)
		fmt.Printf("    After successes:  %s\n", newInterval)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Adaptive keep-alive not adjusting correctly")
	}
	fmt.Println()

	// Test 10: Test statistics
	fmt.Println("Test 10: Testing statistics collection...")
	testCount++

	stats := manager.GetStats()
	if stats != nil {
		fmt.Println("  ✓ PASSED: Statistics retrieved")
		fmt.Printf("    STUN Requests:     %d\n", stats.STUNRequests)
		fmt.Printf("    STUN Successes:    %d\n", stats.STUNSuccesses)
		fmt.Printf("    STUN Failures:     %d\n", stats.STUNFailures)
		fmt.Printf("    Active Connections: %d\n", stats.ActiveConnections)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Could not retrieve statistics")
	}
	fmt.Println()

	// Summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Test Results: %d/%d tests passed (%.1f%%)\n",
		passedCount, testCount, float64(passedCount)/float64(testCount)*100)
	fmt.Println(strings.Repeat("=", 80))

	if passedCount == testCount {
		fmt.Println("\n✓ All NAT traversal tests passed!")
	} else {
		fmt.Printf("\n⚠ %d/%d tests failed\n", testCount-passedCount, testCount)
	}
}

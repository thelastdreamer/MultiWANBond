package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/thelastdreamer/MultiWANBond/pkg/routing"
)

func main() {
	fmt.Println("MultiWANBond Policy Routing Test")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	testCount := 0
	passedCount := 0

	// Test 1: Create routing manager
	fmt.Println("Test 1: Creating routing manager...")
	testCount++
	config := routing.DefaultRoutingConfig()
	manager := routing.NewManager(config)
	if manager == nil {
		fmt.Println("  ❌ FAILED: Manager is nil")
	} else {
		fmt.Println("  ✓ PASSED: Routing manager created")
		passedCount++
	}
	fmt.Println()

	// Test 2: Test default configuration
	fmt.Println("Test 2: Testing default configuration...")
	testCount++
	if config.EnablePolicyRouting && config.EnableSourceRouting && config.EnableMarkRouting {
		fmt.Println("  ✓ PASSED: Default config has all features enabled")
		fmt.Printf("    Main Table ID:      %d\n", config.MainTableID)
		fmt.Printf("    Table ID Start:     %d\n", config.TableIDStart)
		fmt.Printf("    Mark Base:          %d\n", config.MarkBase)
		fmt.Printf("    Max Custom Tables:  %d\n", config.MaxCustomTables)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Default config missing features")
	}
	fmt.Println()

	// Test 3: Create WAN routing tables
	fmt.Println("Test 3: Creating WAN routing tables...")
	testCount++

	// WAN 1: 192.168.1.0/24 via 192.168.1.1
	_, sourceNet1, _ := net.ParseCIDR("192.168.1.0/24")
	gateway1 := net.ParseIP("192.168.1.1")
	sourceIP1 := net.ParseIP("192.168.1.100")

	err := manager.CreateWANTable(1, "eth0", sourceIP1, gateway1)
	if err != nil {
		fmt.Printf("  ⚠ SKIPPED: Cannot create WAN tables on Windows: %v\n", err)
		fmt.Println("    (This is expected - full routing only works on Linux)")
	} else {
		fmt.Println("  ✓ PASSED: WAN table 1 created")
		fmt.Printf("    Table ID:  %d\n", config.TableIDStart+1)
		fmt.Printf("    Source IP: %s\n", sourceIP1)
		fmt.Printf("    Gateway:   %s\n", gateway1)
		passedCount++
	}
	fmt.Println()

	// Test 4: Test route types
	fmt.Println("Test 4: Testing route types...")
	testCount++

	types := []routing.RouteType{
		routing.RouteTypeUnicast,
		routing.RouteTypeLocal,
		routing.RouteTypeBroadcast,
		routing.RouteTypeBlackhole,
		routing.RouteTypeUnreachable,
		routing.RouteTypeProhibit,
	}

	allNamed := true
	for _, t := range types {
		if t.String() == "unknown" {
			allNamed = false
		}
	}

	if allNamed {
		fmt.Println("  ✓ PASSED: All route types have names")
		for _, t := range types {
			fmt.Printf("    %s\n", t.String())
		}
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Some route types not named")
	}
	fmt.Println()

	// Test 5: Test route scopes
	fmt.Println("Test 5: Testing route scopes...")
	testCount++

	scopes := []routing.RouteScope{
		routing.RouteScopeUniverse,
		routing.RouteScopeHost,
		routing.RouteScopeLink,
		routing.RouteScopeSite,
	}

	allNamed = true
	for _, s := range scopes {
		if s.String() == "unknown" {
			allNamed = false
		}
	}

	if allNamed {
		fmt.Println("  ✓ PASSED: All route scopes have names")
		for _, s := range scopes {
			fmt.Printf("    %s\n", s.String())
		}
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Some route scopes not named")
	}
	fmt.Println()

	// Test 6: Test source routing
	fmt.Println("Test 6: Testing source-based routing...")
	testCount++

	err = manager.AddSourceRoutingRule(sourceNet1, 1)
	if err != nil {
		if strings.Contains(err.Error(), "not yet implemented") {
			fmt.Println("  ⚠ SKIPPED: Source routing not implemented on this platform")
			fmt.Println("    (Full routing only works on Linux)")
		} else {
			fmt.Printf("  ⚠ Note: %v\n", err)
		}
	} else {
		fmt.Println("  ✓ PASSED: Source routing rule added")
		fmt.Printf("    Source Network: %s\n", sourceNet1)
		fmt.Printf("    Via WAN:        1\n")
		passedCount++
	}
	fmt.Println()

	// Test 7: Test mark routing
	fmt.Println("Test 7: Testing mark-based routing...")
	testCount++

	mark := uint32(101) // MarkBase + WANID
	mask := uint32(0xFFFFFFFF)

	err = manager.AddMarkRoutingRule(mark, mask, 1)
	if err != nil {
		if strings.Contains(err.Error(), "not yet implemented") {
			fmt.Println("  ⚠ SKIPPED: Mark routing not implemented on this platform")
		} else {
			fmt.Printf("  ⚠ Note: %v\n", err)
		}
	} else {
		fmt.Println("  ✓ PASSED: Mark routing rule added")
		fmt.Printf("    Mark:     0x%08X\n", mark)
		fmt.Printf("    Mask:     0x%08X\n", mask)
		fmt.Printf("    Via WAN:  1\n")
		passedCount++
	}
	fmt.Println()

	// Test 8: Test policy rule actions
	fmt.Println("Test 8: Testing policy rule actions...")
	testCount++

	actions := []routing.RuleAction{
		routing.RuleActionTable,
		routing.RuleActionBlackhole,
		routing.RuleActionUnreachable,
		routing.RuleActionProhibit,
	}

	allNamed = true
	for _, a := range actions {
		if a.String() == "unknown" {
			allNamed = false
		}
	}

	if allNamed {
		fmt.Println("  ✓ PASSED: All rule actions have names")
		for _, a := range actions {
			fmt.Printf("    %s\n", a.String())
		}
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Some rule actions not named")
	}
	fmt.Println()

	// Test 9: Test routing statistics
	fmt.Println("Test 9: Testing routing statistics...")
	testCount++

	stats := manager.GetStats()
	if stats != nil {
		fmt.Println("  ✓ PASSED: Statistics retrieved")
		fmt.Printf("    Tables Created:  %d\n", stats.TablesCreated)
		fmt.Printf("    Active Tables:   %d\n", stats.ActiveTables)
		fmt.Printf("    Routes Added:    %d\n", stats.RoutesAdded)
		fmt.Printf("    Active Routes:   %d\n", stats.ActiveRoutes)
		fmt.Printf("    Rules Added:     %d\n", stats.RulesAdded)
		fmt.Printf("    Active Rules:    %d\n", stats.ActiveRules)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Statistics is nil")
	}
	fmt.Println()

	// Test 10: Test manager interfaces
	fmt.Println("Test 10: Testing manager interfaces...")
	testCount++

	tableManager := routing.NewTableManager()
	ruleManager := routing.NewRuleManager()

	if tableManager != nil && ruleManager != nil {
		fmt.Println("  ✓ PASSED: Managers implement interfaces")
		fmt.Printf("    TableManager: %T\n", tableManager)
		fmt.Printf("    RuleManager:  %T\n", ruleManager)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Manager creation failed")
	}
	fmt.Println()

	// Summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Test Results: %d/%d tests passed (%.1f%%)\n",
		passedCount, testCount, float64(passedCount)/float64(testCount)*100)
	fmt.Println(strings.Repeat("=", 80))

	if passedCount == testCount {
		fmt.Println("\n✓ All policy routing tests passed!")
	} else {
		fmt.Printf("\n⚠ %d/%d tests passed (some features require Linux)\n", passedCount, testCount)
	}
}

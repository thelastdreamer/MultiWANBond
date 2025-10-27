package main

import (
	"fmt"
	"net"
	"strings"

	"github.com/thelastdreamer/MultiWANBond/pkg/dpi"
)

func main() {
	fmt.Println("MultiWANBond Deep Packet Inspection Test")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	testCount := 0
	passedCount := 0

	// Test 1: Create DPI classifier
	fmt.Println("Test 1: Creating DPI classifier...")
	testCount++
	config := dpi.DefaultDPIConfig()
	classifier := dpi.NewClassifier(config)
	if classifier == nil {
		fmt.Println("  ❌ FAILED: Classifier is nil")
	} else {
		fmt.Println("  ✓ PASSED: DPI classifier created")
		passedCount++
	}
	fmt.Println()

	// Test 2: Test protocol names
	fmt.Println("Test 2: Testing protocol names (58 protocols)...")
	testCount++
	protocols := []dpi.Protocol{
		dpi.ProtocolHTTP, dpi.ProtocolHTTPS, dpi.ProtocolHTTP2,
		dpi.ProtocolYouTube, dpi.ProtocolNetflix, dpi.ProtocolSpotify,
		dpi.ProtocolFacebook, dpi.ProtocolWhatsApp, dpi.ProtocolZoom,
		dpi.ProtocolSteam, dpi.ProtocolMinecraft, dpi.ProtocolDiscord,
		dpi.ProtocolSSH, dpi.ProtocolFTP, dpi.ProtocolDNS,
	}

	allNamed := true
	for _, p := range protocols {
		if p.String() == "Unknown" && p != dpi.ProtocolUnknown {
			allNamed = false
			break
		}
	}

	if allNamed {
		fmt.Println("  ✓ PASSED: All protocols have names")
		fmt.Println("    Sample protocols:")
		for i, p := range protocols {
			if i < 10 {
				fmt.Printf("      - %s\n", p)
			}
		}
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Some protocols not named")
	}
	fmt.Println()

	// Test 3: Test protocol categories
	fmt.Println("Test 3: Testing protocol categories...")
	testCount++
	categoryTests := map[dpi.Protocol]dpi.Category{
		dpi.ProtocolHTTP:      dpi.CategoryWeb,
		dpi.ProtocolYouTube:   dpi.CategoryStreaming,
		dpi.ProtocolFacebook:  dpi.CategorySocialMedia,
		dpi.ProtocolSteam:     dpi.CategoryGaming,
		dpi.ProtocolZoom:      dpi.CategoryCommunication,
		dpi.ProtocolFTP:       dpi.CategoryFileTransfer,
		dpi.ProtocolSMTP:      dpi.CategoryEmail,
		dpi.ProtocolOpenVPN:   dpi.CategoryVPN,
		dpi.ProtocolDNS:       dpi.CategorySystem,
	}

	categoriesCorrect := true
	for proto, expectedCat := range categoryTests {
		actualCat := proto.GetCategory()
		if actualCat != expectedCat {
			fmt.Printf("  ❌ %s: expected %s, got %s\n", proto, expectedCat, actualCat)
			categoriesCorrect = false
		}
	}

	if categoriesCorrect {
		fmt.Println("  ✓ PASSED: Protocol categories correct")
		for proto, cat := range categoryTests {
			fmt.Printf("    %s -> %s\n", proto, cat)
		}
		passedCount++
	}
	fmt.Println()

	// Test 4: Test traffic classes
	fmt.Println("Test 4: Testing traffic classes...")
	testCount++
	classTests := map[dpi.Protocol]dpi.TrafficClass{
		dpi.ProtocolZoom:            dpi.ClassRealTime,
		dpi.ProtocolSteam:           dpi.ClassInteractive,
		dpi.ProtocolYouTube:         dpi.ClassStreaming,
		dpi.ProtocolFTP:             dpi.ClassBulk,
		dpi.ProtocolTorrent:         dpi.ClassBackground,
	}

	classesCorrect := true
	for proto, expectedClass := range classTests {
		actualClass := proto.GetTrafficClass()
		if actualClass != expectedClass {
			fmt.Printf("  ❌ %s: expected %s, got %s\n", proto, expectedClass, actualClass)
			classesCorrect = false
		}
	}

	if classesCorrect {
		fmt.Println("  ✓ PASSED: Traffic classes correct")
		for proto, class := range classTests {
			fmt.Printf("    %s -> %s (priority %d)\n", proto, class, class.GetPriority())
		}
		passedCount++
	}
	fmt.Println()

	// Test 5: Test port-based detection
	fmt.Println("Test 5: Testing port-based protocol detection...")
	testCount++
	detector := dpi.NewDetector(config)

	portTests := []struct {
		port     uint16
		expected dpi.Protocol
	}{
		{80, dpi.ProtocolHTTP},
		{443, dpi.ProtocolHTTPS},
		{22, dpi.ProtocolSSH},
		{21, dpi.ProtocolFTP},
		{53, dpi.ProtocolDNS},
		{3389, dpi.ProtocolRDP},
	}

	portDetectionWorks := true
	for _, test := range portTests {
		payload := []byte{}
		classification := detector.Classify(payload, 0, test.port)
		if classification.Protocol != test.expected {
			portDetectionWorks = false
			break
		}
	}

	if portDetectionWorks {
		fmt.Println("  ✓ PASSED: Port-based detection working")
		for _, test := range portTests {
			fmt.Printf("    Port %d -> %s\n", test.port, test.expected)
		}
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Port-based detection issues")
	}
	fmt.Println()

	// Test 6: Test HTTP detection
	fmt.Println("Test 6: Testing HTTP signature detection...")
	testCount++
	httpPayload := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	httpClass := detector.Classify(httpPayload, 12345, 80)

	if httpClass.Protocol == dpi.ProtocolHTTP && httpClass.Confidence > 0.8 {
		fmt.Println("  ✓ PASSED: HTTP detection working")
		fmt.Printf("    Protocol:   %s\n", httpClass.Protocol)
		fmt.Printf("    Confidence: %.2f\n", httpClass.Confidence)
		passedCount++
	} else {
		fmt.Printf("  ❌ FAILED: HTTP not detected (got %s, confidence %.2f)\n",
			httpClass.Protocol, httpClass.Confidence)
	}
	fmt.Println()

	// Test 7: Test flow tracking
	fmt.Println("Test 7: Testing flow tracking...")
	testCount++

	srcIP := net.ParseIP("192.168.1.100")
	dstIP := net.ParseIP("8.8.8.8")
	srcPort := uint16(12345)
	dstPort := uint16(80)

	classification, flow := classifier.ClassifyPacket(srcIP, dstIP, srcPort, dstPort, 6, httpPayload, true)

	if flow != nil && classification != nil {
		fmt.Println("  ✓ PASSED: Flow tracking working")
		fmt.Printf("    Flow:       %s:%d -> %s:%d\n", srcIP, srcPort, dstIP, dstPort)
		fmt.Printf("    Protocol:   %s\n", classification.Protocol)
		fmt.Printf("    Category:   %s\n", classification.Category)
		fmt.Printf("    Packets:    %d\n", flow.Packets)
		fmt.Printf("    Bytes:      %d\n", flow.Bytes)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Flow tracking not working")
	}
	fmt.Println()

	// Test 8: Test application policies
	fmt.Println("Test 8: Testing application policies...")
	testCount++

	policy := &dpi.ApplicationPolicy{
		Name:          "YouTube-WAN1",
		Protocol:      dpi.ProtocolYouTube,
		WANID:         1,
		Mark:          101,
		Priority:      10,
		BandwidthLimit: 10 * 1024 * 1024, // 10 MB/s
		TrafficClass:  dpi.ClassStreaming,
		Enabled:       true,
	}

	err := classifier.AddPolicy(policy)
	if err != nil {
		fmt.Printf("  ❌ FAILED: %v\n", err)
	} else {
		fmt.Println("  ✓ PASSED: Application policy added")
		fmt.Printf("    Name:      %s\n", policy.Name)
		fmt.Printf("    Protocol:  %s\n", policy.Protocol)
		fmt.Printf("    WAN:       %d\n", policy.WANID)
		fmt.Printf("    Mark:      %d\n", policy.Mark)
		fmt.Printf("    Class:     %s\n", policy.TrafficClass)
		passedCount++
	}
	fmt.Println()

	// Test 9: Test statistics
	fmt.Println("Test 9: Testing DPI statistics...")
	testCount++

	stats := classifier.GetStats()
	if stats != nil {
		fmt.Println("  ✓ PASSED: Statistics retrieved")
		fmt.Printf("    Total Flows:      %d\n", stats.TotalFlows)
		fmt.Printf("    Active Flows:     %d\n", stats.ActiveFlows)
		fmt.Printf("    Classified Flows: %d\n", stats.ClassifiedFlows)
		fmt.Printf("    Total Packets:    %d\n", stats.TotalPackets)
		fmt.Printf("    Total Bytes:      %d\n", stats.TotalBytes)
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: Statistics is nil")
	}
	fmt.Println()

	// Test 10: Test signature count
	fmt.Println("Test 10: Testing protocol signatures...")
	testCount++

	signatures := detector.GetSignatures()
	if len(signatures) > 0 {
		fmt.Println("  ✓ PASSED: Protocol signatures loaded")
		fmt.Printf("    Total signatures: %d\n", len(signatures))
		fmt.Println("    Sample signatures:")
		for i, sig := range signatures {
			if i < 5 {
				fmt.Printf("      - %s (%s)\n", sig.Name, sig.Protocol)
			}
		}
		passedCount++
	} else {
		fmt.Println("  ❌ FAILED: No signatures loaded")
	}
	fmt.Println()

	// Summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("Test Results: %d/%d tests passed (%.1f%%)\n",
		passedCount, testCount, float64(passedCount)/float64(testCount)*100)
	fmt.Println(strings.Repeat("=", 80))

	if passedCount == testCount {
		fmt.Println("\n✓ All DPI tests passed!")
	} else {
		fmt.Printf("\n⚠ %d/%d tests failed\n", testCount-passedCount, testCount)
	}
}

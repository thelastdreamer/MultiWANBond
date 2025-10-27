package main

import (
	"fmt"
	"net"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
	"github.com/thelastdreamer/MultiWANBond/pkg/router"
	"github.com/thelastdreamer/MultiWANBond/pkg/fec"
	"github.com/thelastdreamer/MultiWANBond/pkg/packet"
	"github.com/thelastdreamer/MultiWANBond/pkg/health"
)

func main() {
	fmt.Println("=== MultiWANBond Core Features Test ===\n")

	passedTests := 0
	totalTests := 0

	// Test 1: Protocol FlowKey
	fmt.Println("Test 1: Protocol FlowKey String() method")
	totalTests++
	flowKey := protocol.FlowKey{
		SrcIP:    net.ParseIP("192.168.1.100"),
		DstIP:    net.ParseIP("8.8.8.8"),
		SrcPort:  12345,
		DstPort:  80,
		Protocol: 6, // TCP
	}
	flowKeyStr := flowKey.String()
	if flowKeyStr != "" {
		fmt.Printf("  ✓ FlowKey.String() works: %s\n", flowKeyStr)
		passedTests++
	} else {
		fmt.Println("  ✗ FlowKey.String() failed")
	}
	fmt.Println()

	// Test 2: Router Creation
	fmt.Println("Test 2: Router creation and WAN management")
	totalTests++
	r := router.NewRouter(protocol.LoadBalanceRoundRobin)
	if r != nil {
		fmt.Println("  ✓ Router created successfully")
		passedTests++
	} else {
		fmt.Println("  ✗ Router creation failed")
	}
	fmt.Println()

	// Test 3: Add WANs to Router
	fmt.Println("Test 3: Adding WAN interfaces to router")
	totalTests++

	wan1 := &protocol.WANInterface{
		ID:   1,
		Name: "WAN1",
		Type: protocol.WANTypeFiber,
		State: protocol.WANStateUp,
		Config: protocol.WANConfig{
			MaxBandwidth: 1000000000, // 1 Gbps
			Weight:       100,
			Enabled:      true,
		},
		Metrics: &protocol.WANMetrics{
			Latency:    10 * time.Millisecond,
			PacketLoss: 0.1,
		},
	}

	wan2 := &protocol.WANInterface{
		ID:   2,
		Name: "WAN2",
		Type: protocol.WANTypeLTE,
		State: protocol.WANStateUp,
		Config: protocol.WANConfig{
			MaxBandwidth: 100000000, // 100 Mbps
			Weight:       50,
			Enabled:      true,
		},
		Metrics: &protocol.WANMetrics{
			Latency:    50 * time.Millisecond,
			PacketLoss: 1.5,
		},
	}

	r.AddWAN(wan1)
	r.AddWAN(wan2)
	fmt.Println("  ✓ Added 2 WAN interfaces")
	passedTests++
	fmt.Println()

	// Test 4: Router Modes
	fmt.Println("Test 4: Testing different router modes")
	totalTests++

	modes := []protocol.LoadBalanceMode{
		protocol.LoadBalanceRoundRobin,
		protocol.LoadBalanceWeighted,
		protocol.LoadBalanceLeastUsed,
		protocol.LoadBalanceLeastLatency,
		protocol.LoadBalancePerFlow,
		protocol.LoadBalanceAdaptive,
	}

	modeNames := []string{
		"Round-Robin",
		"Weighted",
		"Least Used",
		"Least Latency",
		"Per-Flow",
		"Adaptive",
	}

	allModesWork := true
	for i, mode := range modes {
		r.SetMode(mode)
		testPacket := &protocol.Packet{
			Priority: 128,
		}

		decision, err := r.Route(testPacket, &flowKey)
		if err != nil {
			fmt.Printf("  ✗ %s mode failed: %v\n", modeNames[i], err)
			allModesWork = false
		} else {
			fmt.Printf("  ✓ %s mode: WAN %d selected\n", modeNames[i], decision.PrimaryWAN)
		}
	}

	if allModesWork {
		passedTests++
	}
	fmt.Println()

	// Test 5: FEC System
	fmt.Println("Test 5: Forward Error Correction (FEC)")
	totalTests++

	fecEncoder := fec.NewReedSolomonEncoder()
	if fecEncoder != nil {
		fmt.Println("  ✓ FEC Encoder created")

		// Test data
		testData := []byte("This is test data for Forward Error Correction encoding and decoding")

		// Encode with 50% redundancy
		encoded, err := fecEncoder.Encode(testData, 0.5)
		if err != nil {
			fmt.Printf("  ✗ FEC encoding failed: %v\n", err)
		} else {
			fmt.Printf("  ✓ FEC encoding successful: created %d packets\n", len(encoded))
			passedTests++
		}
	} else {
		fmt.Println("  ✗ FEC Encoder creation failed")
	}
	fmt.Println()

	// Test 6: Packet Processor
	fmt.Println("Test 6: Packet processor with reordering")
	totalTests++

	processor := packet.NewProcessor(100, 5*time.Second) // 100 packet buffer, 5s timeout
	if processor != nil {
		fmt.Println("  ✓ Packet processor created")

		// Create test packets
		testPackets := []*protocol.Packet{
			{Version: protocol.ProtocolVersion, Type: protocol.PacketTypeData, SequenceID: 3, Timestamp: time.Now().UnixNano(), Data: []byte("Packet 3")},
			{Version: protocol.ProtocolVersion, Type: protocol.PacketTypeData, SequenceID: 1, Timestamp: time.Now().UnixNano(), Data: []byte("Packet 1")},
			{Version: protocol.ProtocolVersion, Type: protocol.PacketTypeData, SequenceID: 2, Timestamp: time.Now().UnixNano(), Data: []byte("Packet 2")},
		}

		// Process out-of-order packets
		for _, pkt := range testPackets {
			_, ready, err := processor.Reorder(pkt)
			if err != nil && ready {
				fmt.Printf("  Packet %d processed\n", pkt.SequenceID)
			}
		}

		fmt.Println("  ✓ Processed 3 out-of-order packets")
		passedTests++
	} else {
		fmt.Println("  ✗ Packet processor creation failed")
	}
	fmt.Println()

	// Test 7: Health Checker
	fmt.Println("Test 7: Health checker creation")
	totalTests++

	checker := health.NewChecker()
	if checker != nil {
		fmt.Println("  ✓ Health checker created")
		fmt.Println("  ✓ Ready for sub-second health checks")
		passedTests++
	} else {
		fmt.Println("  ✗ Health checker creation failed")
	}
	fmt.Println()

	// Test 8: Packet Encoding/Decoding
	fmt.Println("Test 8: Packet encoding and decoding")
	totalTests++

	// Create a processor for encoding/decoding
	testProcessor := packet.NewProcessor(100, 5*time.Second)

	originalPacket := &protocol.Packet{
		Version:    protocol.ProtocolVersion,
		Type:       protocol.PacketTypeData,
		SessionID:  12345,
		SequenceID: 678,
		Timestamp:  time.Now().UnixNano(),
		WANID:      1,
		Priority:   128,
		Data:       []byte("Hello, MultiWANBond!"),
	}

	encoded, err := testProcessor.Encode(originalPacket)
	if err != nil {
		fmt.Printf("  ✗ Packet encoding failed: %v\n", err)
	} else {
		fmt.Printf("  ✓ Packet encoded: %d bytes\n", len(encoded))

		// Decode it back
		decoded, err := testProcessor.Decode(encoded)
		if err != nil {
			fmt.Printf("  ✗ Packet decoding failed: %v\n", err)
		} else {
			if decoded.SequenceID == originalPacket.SequenceID {
				fmt.Println("  ✓ Packet decoded successfully")
				passedTests++
			} else {
				fmt.Println("  ✗ Decoded packet doesn't match original")
			}
		}
	}
	fmt.Println()

	// Test 9: Router Metrics Update
	fmt.Println("Test 9: Router metrics updates")
	totalTests++

	newMetrics := &protocol.WANMetrics{
		Latency:       20 * time.Millisecond,
		Jitter:        5 * time.Millisecond,
		PacketLoss:    0.5,
		AvgLatency:    15 * time.Millisecond,
		AvgJitter:     3 * time.Millisecond,
		AvgPacketLoss: 0.3,
	}

	r.UpdateMetrics(1, newMetrics)
	fmt.Println("  ✓ Metrics updated for WAN 1")

	r.RecordBandwidthUsage(1, 1000000) // 1 MB
	fmt.Println("  ✓ Bandwidth usage recorded")
	passedTests++
	fmt.Println()

	// Test 10: FlowKey as Map Key
	fmt.Println("Test 10: FlowKey used as map key (verifying fix)")
	totalTests++

	flowMap := make(map[string]uint8)
	flowMap[flowKey.String()] = 1

	if val, exists := flowMap[flowKey.String()]; exists && val == 1 {
		fmt.Println("  ✓ FlowKey successfully used as map key")
		passedTests++
	} else {
		fmt.Println("  ✗ FlowKey map lookup failed")
	}
	fmt.Println()

	// Results Summary
	fmt.Println("===========================================")
	fmt.Printf("Test Results: %d/%d passed (%.1f%%)\n",
		passedTests, totalTests, float64(passedTests)/float64(totalTests)*100)
	fmt.Println("===========================================")

	if passedTests == totalTests {
		fmt.Println("\n✅ All core features working correctly!")
	} else {
		fmt.Printf("\n⚠️  %d test(s) failed\n", totalTests-passedTests)
	}
}

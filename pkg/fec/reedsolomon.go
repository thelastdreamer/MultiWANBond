package fec

import (
	"fmt"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// ReedSolomonEncoder implements FEC using Reed-Solomon coding
// This is a simplified implementation - in production, use a library like github.com/klauspost/reedsolomon
type ReedSolomonEncoder struct {
	dataShards   int
	parityShards int
}

// NewReedSolomonEncoder creates a new Reed-Solomon FEC encoder
func NewReedSolomonEncoder() *ReedSolomonEncoder {
	return &ReedSolomonEncoder{
		dataShards:   4, // Default: 4 data shards
		parityShards: 2, // Default: 2 parity shards (50% redundancy)
	}
}

// Encode adds FEC redundancy to data
// Returns multiple packets: data packets + FEC parity packets
func (e *ReedSolomonEncoder) Encode(data []byte, redundancy float64) ([][]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	if redundancy < 0 || redundancy > 1 {
		return nil, fmt.Errorf("redundancy must be between 0 and 1")
	}

	// Calculate number of data and parity shards based on redundancy
	dataShards := 4
	parityShards := int(float64(dataShards) * redundancy)
	if parityShards < 1 {
		parityShards = 1
	}

	e.dataShards = dataShards
	e.parityShards = parityShards

	// Calculate shard size
	shardSize := (len(data) + dataShards - 1) / dataShards

	// Create data shards
	dataPackets := make([][]byte, dataShards)
	for i := 0; i < dataShards; i++ {
		start := i * shardSize
		end := start + shardSize
		if end > len(data) {
			end = len(data)
		}

		// Pad to shard size if needed
		shard := make([]byte, shardSize)
		copy(shard, data[start:end])
		dataPackets[i] = shard
	}

	// Generate parity shards (simplified XOR-based for demonstration)
	// In production, use proper Reed-Solomon library
	parityPackets := make([][]byte, parityShards)
	for i := 0; i < parityShards; i++ {
		parity := make([]byte, shardSize)

		// Simple XOR-based parity (not true Reed-Solomon, but demonstrates concept)
		for j := 0; j < dataShards; j++ {
			for k := 0; k < shardSize; k++ {
				parity[k] ^= dataPackets[j][k]
			}
		}

		parityPackets[i] = parity
	}

	// Combine data and parity packets
	allPackets := make([][]byte, dataShards+parityShards)
	copy(allPackets, dataPackets)
	copy(allPackets[dataShards:], parityPackets)

	return allPackets, nil
}

// Decode recovers data from FEC packets (may have missing packets)
// packets: all received packets (data + parity, may have nils for missing)
// missing: indices of missing packets
func (e *ReedSolomonEncoder) Decode(packets [][]byte, missing []int) ([]byte, error) {
	if len(packets) == 0 {
		return nil, fmt.Errorf("no packets to decode")
	}

	totalShards := e.dataShards + e.parityShards
	if len(packets) < totalShards {
		return nil, fmt.Errorf("not enough packets: got %d, need %d", len(packets), totalShards)
	}

	// Count how many packets we have
	receivedCount := 0
	for _, pkt := range packets {
		if pkt != nil {
			receivedCount++
		}
	}

	// Check if we can recover
	if !e.CanRecover(totalShards, receivedCount) {
		return nil, fmt.Errorf("cannot recover: need at least %d packets, have %d", e.dataShards, receivedCount)
	}

	// If no packets are missing, just reconstruct from data shards
	if len(missing) == 0 {
		return e.reconstructData(packets[:e.dataShards])
	}

	// Recover missing data shards using parity
	// Simplified recovery - in production, use proper Reed-Solomon decoding
	shardSize := len(packets[0])
	for _, missIdx := range missing {
		if missIdx < e.dataShards {
			// Recover missing data shard
			recovered := make([]byte, shardSize)

			// XOR all available shards (simplified recovery)
			for i, pkt := range packets {
				if pkt != nil && i != missIdx {
					for k := 0; k < shardSize; k++ {
						recovered[k] ^= pkt[k]
					}
				}
			}

			packets[missIdx] = recovered
		}
	}

	// Reconstruct data from (now complete) data shards
	return e.reconstructData(packets[:e.dataShards])
}

// reconstructData combines data shards back into original data
func (e *ReedSolomonEncoder) reconstructData(dataShards [][]byte) ([]byte, error) {
	if len(dataShards) == 0 {
		return nil, fmt.Errorf("no data shards")
	}

	shardSize := len(dataShards[0])
	data := make([]byte, 0, len(dataShards)*shardSize)

	for _, shard := range dataShards {
		data = append(data, shard...)
	}

	return data, nil
}

// CanRecover checks if data can be recovered given packet loss
func (e *ReedSolomonEncoder) CanRecover(totalPackets, receivedPackets int) bool {
	// We need at least as many packets as data shards
	return receivedPackets >= e.dataShards
}

// SetShardCount configures the number of data and parity shards
func (e *ReedSolomonEncoder) SetShardCount(dataShards, parityShards int) error {
	if dataShards < 1 {
		return fmt.Errorf("data shards must be >= 1")
	}
	if parityShards < 1 {
		return fmt.Errorf("parity shards must be >= 1")
	}

	e.dataShards = dataShards
	e.parityShards = parityShards
	return nil
}

// FECManager manages FEC encoding/decoding for the protocol
type FECManager struct {
	encoder protocol.FECEncoder
	enabled bool
}

// NewFECManager creates a new FEC manager
func NewFECManager() *FECManager {
	return &FECManager{
		encoder: NewReedSolomonEncoder(),
		enabled: false,
	}
}

// Enable enables FEC
func (m *FECManager) Enable() {
	m.enabled = true
}

// Disable disables FEC
func (m *FECManager) Disable() {
	m.enabled = false
}

// IsEnabled returns whether FEC is enabled
func (m *FECManager) IsEnabled() bool {
	return m.enabled
}

// EncodePacket encodes a packet with FEC
func (m *FECManager) EncodePacket(data []byte, redundancy float64) ([][]byte, error) {
	if !m.enabled {
		return [][]byte{data}, nil
	}

	return m.encoder.Encode(data, redundancy)
}

// DecodePackets decodes packets with FEC
func (m *FECManager) DecodePackets(packets [][]byte, missing []int) ([]byte, error) {
	if !m.enabled || len(missing) == 0 {
		// If FEC disabled or no missing packets, return first packet
		for _, pkt := range packets {
			if pkt != nil {
				return pkt, nil
			}
		}
		return nil, fmt.Errorf("no valid packets")
	}

	return m.encoder.Decode(packets, missing)
}

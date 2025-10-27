package packet

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// Processor handles packet encoding, decoding, and reordering
type Processor struct {
	mu              sync.RWMutex
	reorderBuffer   map[uint64]*protocol.Packet
	nextExpectedSeq uint64
	bufferSize      int
	timeout         time.Duration
	lastCleanup     time.Time
}

// NewProcessor creates a new packet processor
func NewProcessor(bufferSize int, timeout time.Duration) *Processor {
	return &Processor{
		reorderBuffer:   make(map[uint64]*protocol.Packet),
		nextExpectedSeq: 0,
		bufferSize:      bufferSize,
		timeout:         timeout,
		lastCleanup:     time.Now(),
	}
}

// Encode encodes a packet for transmission
func (p *Processor) Encode(packet *protocol.Packet) ([]byte, error) {
	if packet == nil {
		return nil, fmt.Errorf("packet is nil")
	}

	// Calculate total size
	// Header: Version(1) + Type(1) + Flags(2) + SessionID(8) + SequenceID(8) +
	//         Timestamp(8) + WANID(1) + Priority(1) + DataLen(4) + Checksum(4) = 38 bytes
	headerSize := 38
	totalSize := headerSize + len(packet.Data)

	buf := make([]byte, totalSize)

	// Encode header
	offset := 0

	buf[offset] = packet.Version
	offset++

	buf[offset] = byte(packet.Type)
	offset++

	binary.BigEndian.PutUint16(buf[offset:], packet.Flags)
	offset += 2

	binary.BigEndian.PutUint64(buf[offset:], packet.SessionID)
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:], packet.SequenceID)
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:], uint64(packet.Timestamp))
	offset += 8

	buf[offset] = packet.WANID
	offset++

	buf[offset] = packet.Priority
	offset++

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(packet.Data)))
	offset += 4

	// Copy data
	copy(buf[offset:], packet.Data)
	offset += len(packet.Data)

	// Calculate checksum (excluding checksum field itself)
	checksum := crc32.ChecksumIEEE(buf[:offset])
	binary.BigEndian.PutUint32(buf[offset:], checksum)

	return buf, nil
}

// Decode decodes a received packet
func (p *Processor) Decode(data []byte) (*protocol.Packet, error) {
	if len(data) < 38 {
		return nil, fmt.Errorf("packet too small: %d bytes", len(data))
	}

	packet := &protocol.Packet{}
	offset := 0

	// Decode header
	packet.Version = data[offset]
	offset++

	if packet.Version != protocol.ProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", packet.Version)
	}

	packet.Type = protocol.PacketType(data[offset])
	offset++

	packet.Flags = binary.BigEndian.Uint16(data[offset:])
	offset += 2

	packet.SessionID = binary.BigEndian.Uint64(data[offset:])
	offset += 8

	packet.SequenceID = binary.BigEndian.Uint64(data[offset:])
	offset += 8

	packet.Timestamp = int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	packet.WANID = data[offset]
	offset++

	packet.Priority = data[offset]
	offset++

	dataLen := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	// Validate data length
	if int(dataLen) > len(data)-offset-4 {
		return nil, fmt.Errorf("invalid data length: %d", dataLen)
	}

	// Extract data
	packet.Data = make([]byte, dataLen)
	copy(packet.Data, data[offset:offset+int(dataLen)])
	offset += int(dataLen)

	// Verify checksum
	receivedChecksum := binary.BigEndian.Uint32(data[offset:])
	calculatedChecksum := crc32.ChecksumIEEE(data[:offset])

	if receivedChecksum != calculatedChecksum {
		return nil, fmt.Errorf("checksum mismatch: received %d, calculated %d", receivedChecksum, calculatedChecksum)
	}

	packet.Checksum = receivedChecksum

	return packet, nil
}

// Reorder handles packet reordering
// Returns: data (if packet can be delivered), ready (true if data is ready), error
func (p *Processor) Reorder(packet *protocol.Packet) ([]byte, bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Special packet types bypass reordering
	if packet.Type == protocol.PacketTypeHeartbeat || packet.Type == protocol.PacketTypeControl {
		return packet.Data, true, nil
	}

	// Check if this is the next expected packet
	if packet.SequenceID == p.nextExpectedSeq {
		p.nextExpectedSeq++

		// Check if we have buffered packets that are now in order
		deliverable := [][]byte{packet.Data}

		for {
			if buffered, exists := p.reorderBuffer[p.nextExpectedSeq]; exists {
				deliverable = append(deliverable, buffered.Data)
				delete(p.reorderBuffer, p.nextExpectedSeq)
				p.nextExpectedSeq++
			} else {
				break
			}
		}

		// For simplicity, return first packet's data
		// In production, you might want to combine or handle multiple packets
		return deliverable[0], true, nil
	}

	// Packet is out of order
	if packet.SequenceID > p.nextExpectedSeq {
		// Future packet - buffer it
		p.reorderBuffer[packet.SequenceID] = packet

		// Check buffer size
		if len(p.reorderBuffer) > p.bufferSize {
			// Buffer overflow - force delivery of oldest packets
			return p.forceDelivery()
		}

		// Not ready yet
		return nil, false, nil
	}

	// Old packet (SequenceID < nextExpectedSeq) - it's a duplicate or very late
	// Drop it
	return nil, false, fmt.Errorf("duplicate or late packet: seq=%d, expected=%d", packet.SequenceID, p.nextExpectedSeq)
}

// forceDelivery forces delivery of buffered packets when buffer is full
func (p *Processor) forceDelivery() ([]byte, bool, error) {
	// Find oldest packet in buffer
	var oldestSeq uint64 = 1<<64 - 1
	var oldestPacket *protocol.Packet

	for seq, pkt := range p.reorderBuffer {
		if seq < oldestSeq {
			oldestSeq = seq
			oldestPacket = pkt
		}
	}

	if oldestPacket == nil {
		return nil, false, fmt.Errorf("buffer overflow but no packets found")
	}

	// Deliver oldest packet and advance sequence
	delete(p.reorderBuffer, oldestSeq)
	p.nextExpectedSeq = oldestSeq + 1

	return oldestPacket.Data, true, nil
}

// CleanupExpired removes expired packets from reorder buffer
func (p *Processor) CleanupExpired() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	if now.Sub(p.lastCleanup) < p.timeout {
		return
	}

	// Remove packets older than timeout
	cutoff := now.Add(-p.timeout).UnixNano()

	for seq, pkt := range p.reorderBuffer {
		if pkt.Timestamp < cutoff {
			delete(p.reorderBuffer, seq)
		}
	}

	p.lastCleanup = now
}

// Reset resets the reorder buffer
func (p *Processor) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.reorderBuffer = make(map[uint64]*protocol.Packet)
	p.nextExpectedSeq = 0
	p.lastCleanup = time.Now()
}

// SetNextExpectedSeq sets the next expected sequence number
func (p *Processor) SetNextExpectedSeq(seq uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.nextExpectedSeq = seq
}

// GetBufferSize returns current reorder buffer size
func (p *Processor) GetBufferSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.reorderBuffer)
}

// GetNextExpectedSeq returns the next expected sequence number
func (p *Processor) GetNextExpectedSeq() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.nextExpectedSeq
}

// DeduplicateCache handles duplicate packet detection
type DeduplicateCache struct {
	mu      sync.RWMutex
	cache   map[uint64]time.Time // SequenceID -> receive time
	maxSize int
	ttl     time.Duration
}

// NewDeduplicateCache creates a new duplicate detection cache
func NewDeduplicateCache(maxSize int, ttl time.Duration) *DeduplicateCache {
	return &DeduplicateCache{
		cache:   make(map[uint64]time.Time),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// IsDuplicate checks if a packet is a duplicate
func (dc *DeduplicateCache) IsDuplicate(seqID uint64) bool {
	dc.mu.RLock()
	_, exists := dc.cache[seqID]
	dc.mu.RUnlock()

	if exists {
		return true
	}

	// Not a duplicate - add to cache
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.cache[seqID] = time.Now()

	// Cleanup if cache is too large
	if len(dc.cache) > dc.maxSize {
		dc.cleanup()
	}

	return false
}

// cleanup removes expired entries from cache
func (dc *DeduplicateCache) cleanup() {
	now := time.Now()
	cutoff := now.Add(-dc.ttl)

	for seq, timestamp := range dc.cache {
		if timestamp.Before(cutoff) {
			delete(dc.cache, seq)
		}
	}
}

// Clear clears the cache
func (dc *DeduplicateCache) Clear() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.cache = make(map[uint64]time.Time)
}

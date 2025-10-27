package server

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// NATEngine handles Network Address Translation for client sessions
type NATEngine struct {
	mu              sync.RWMutex
	sessionManager  *SessionManager
	portAllocator   *PortAllocator
	mappingTimeout  time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// PortAllocator manages allocation of NAT ports
type PortAllocator struct {
	mu        sync.RWMutex
	allocated map[uint16]bool // Port -> is allocated
	minPort   uint16
	maxPort   uint16
}

// NewNATEngine creates a new NAT engine
func NewNATEngine(sessionManager *SessionManager) *NATEngine {
	return &NATEngine{
		sessionManager:  sessionManager,
		portAllocator:   NewPortAllocator(10000, 65535),
		mappingTimeout:  5 * time.Minute,
		cleanupInterval: 30 * time.Second,
		stopChan:        make(chan struct{}),
	}
}

// Start starts the NAT engine
func (ne *NATEngine) Start() {
	ne.wg.Add(1)
	go ne.cleanupRoutine()
}

// Stop stops the NAT engine
func (ne *NATEngine) Stop() {
	close(ne.stopChan)
	ne.wg.Wait()
}

// TranslateOutbound translates an outbound packet (client -> internet)
func (ne *NATEngine) TranslateOutbound(sessionID string, srcIP net.IP, srcPort uint16, dstIP net.IP, dstPort uint16, protocol uint8) (*NATMapping, error) {
	// Get session
	session, err := ne.sessionManager.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if mapping exists
	mapping := session.NATMappings.GetMapping(srcIP, srcPort, protocol)
	if mapping != nil {
		// Update existing mapping
		mapping.LastUsed = time.Now()
		return mapping, nil
	}

	// Allocate new public port
	publicPort, err := ne.portAllocator.Allocate()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate port: %w", err)
	}

	// Create new mapping
	mapping = &NATMapping{
		SourceIP:   srcIP,
		SourcePort: srcPort,
		PublicIP:   session.PublicIP,
		PublicPort: publicPort,
		DestIP:     dstIP,
		DestPort:   dstPort,
		Protocol:   protocol,
		Created:    time.Now(),
		LastUsed:   time.Now(),
	}

	// Add to session's NAT table
	session.NATMappings.AddMapping(mapping)

	return mapping, nil
}

// TranslateInbound translates an inbound packet (internet -> client)
func (ne *NATEngine) TranslateInbound(publicIP net.IP, publicPort uint16, protocol uint8) (*NATMapping, string, error) {
	// Find session with matching public IP
	sessions := ne.sessionManager.GetAllSessions()

	for _, session := range sessions {
		if session.PublicIP.Equal(publicIP) {
			// Look through session's NAT mappings for matching public port
			session.NATMappings.mu.RLock()
			for _, mapping := range session.NATMappings.mappings {
				if mapping.PublicPort == publicPort && mapping.Protocol == protocol {
					session.NATMappings.mu.RUnlock()

					// Update last used
					mapping.LastUsed = time.Now()
					return mapping, session.ID, nil
				}
			}
			session.NATMappings.mu.RUnlock()
		}
	}

	return nil, "", fmt.Errorf("no NAT mapping found for %s:%d", publicIP, publicPort)
}

// UpdateMappingStats updates statistics for a NAT mapping
func (ne *NATEngine) UpdateMappingStats(sessionID string, srcIP net.IP, srcPort uint16, protocol uint8, bytesForward, bytesReverse uint64) {
	session, err := ne.sessionManager.GetSession(sessionID)
	if err != nil {
		return
	}

	mapping := session.NATMappings.GetMapping(srcIP, srcPort, protocol)
	if mapping != nil {
		mapping.BytesForward += bytesForward
		mapping.BytesReverse += bytesReverse
		mapping.PacketsForward++
		mapping.LastUsed = time.Now()
	}
}

// cleanupRoutine periodically removes expired NAT mappings
func (ne *NATEngine) cleanupRoutine() {
	defer ne.wg.Done()

	ticker := time.NewTicker(ne.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ne.stopChan:
			return
		case <-ticker.C:
			ne.cleanupExpiredMappings()
		}
	}
}

// cleanupExpiredMappings removes expired NAT mappings
func (ne *NATEngine) cleanupExpiredMappings() {
	sessions := ne.sessionManager.GetAllSessions()
	now := time.Now()

	for _, session := range sessions {
		session.NATMappings.mu.Lock()

		toRemove := make([]string, 0)
		for key, mapping := range session.NATMappings.mappings {
			if now.Sub(mapping.LastUsed) > ne.mappingTimeout {
				toRemove = append(toRemove, key)
				// Release port
				ne.portAllocator.Release(mapping.PublicPort)
			}
		}

		for _, key := range toRemove {
			delete(session.NATMappings.mappings, key)
		}

		session.NATMappings.mu.Unlock()
	}
}

// GetMappingCount returns the total number of active NAT mappings
func (ne *NATEngine) GetMappingCount() int {
	sessions := ne.sessionManager.GetAllSessions()
	count := 0

	for _, session := range sessions {
		session.NATMappings.mu.RLock()
		count += len(session.NATMappings.mappings)
		session.NATMappings.mu.RUnlock()
	}

	return count
}

// NewPortAllocator creates a new port allocator
func NewPortAllocator(minPort, maxPort uint16) *PortAllocator {
	return &PortAllocator{
		allocated: make(map[uint16]bool),
		minPort:   minPort,
		maxPort:   maxPort,
	}
}

// Allocate allocates a random available port
func (pa *PortAllocator) Allocate() (uint16, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Try random allocation first (faster)
	for i := 0; i < 100; i++ {
		port := pa.randomPort()
		if !pa.allocated[port] {
			pa.allocated[port] = true
			return port, nil
		}
	}

	// Fall back to sequential search
	for port := pa.minPort; port <= pa.maxPort; port++ {
		if !pa.allocated[port] {
			pa.allocated[port] = true
			return port, nil
		}
	}

	return 0, fmt.Errorf("no ports available")
}

// Release releases an allocated port
func (pa *PortAllocator) Release(port uint16) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	delete(pa.allocated, port)
}

// GetAllocated returns the number of allocated ports
func (pa *PortAllocator) GetAllocated() int {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	return len(pa.allocated)
}

// randomPort generates a random port in the allowed range
func (pa *PortAllocator) randomPort() uint16 {
	var b [2]byte
	rand.Read(b[:])
	port := binary.BigEndian.Uint16(b[:])

	// Ensure port is in range
	portRange := pa.maxPort - pa.minPort + 1
	return pa.minPort + (port % portRange)
}

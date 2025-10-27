package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Manager coordinates all NAT traversal operations
type Manager struct {
	config *NATTraversalConfig
	mu     sync.RWMutex

	// Components
	stunClient    *STUNClient
	holePuncher   *HolePuncher
	relayClient   *RelayClient
	cgnatDetector *CGNATDetector

	// State
	localAddr  *net.UDPAddr
	publicAddr *net.UDPAddr
	natType    NATType
	cgnatInfo  *CGNATInfo

	// Peer connections
	connections map[string]*ConnectionInfo

	// Stats
	stats *TraversalStats

	// Control
	running bool
	stopCh  chan struct{}
}

// NewManager creates a new NAT traversal manager
func NewManager(config *NATTraversalConfig) (*Manager, error) {
	if config == nil {
		config = DefaultNATTraversalConfig()
	}

	// Create STUN client
	stunClient, err := NewSTUNClient(config.STUN)
	if err != nil {
		return nil, fmt.Errorf("failed to create STUN client: %w", err)
	}

	m := &Manager{
		config:        config,
		stunClient:    stunClient,
		cgnatDetector: NewCGNATDetector(config.CGNAT),
		connections:   make(map[string]*ConnectionInfo),
		stats:         &TraversalStats{},
		stopCh:        make(chan struct{}),
	}

	// Create hole puncher (uses same connection as STUN)
	m.holePuncher = NewHolePuncher(stunClient.conn, config.HolePunch)

	// Create relay client if enabled
	if config.Relay.EnableRelay {
		relayClient, err := NewRelayClient(stunClient.conn, config.Relay)
		if err == nil {
			m.relayClient = relayClient
		}
		// Don't fail if relay unavailable, just continue without it
	}

	return m, nil
}

// Initialize performs initial NAT discovery
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Discover NAT mapping via STUN
	mapping, err := m.stunClient.DiscoverNATMapping()
	if err != nil {
		m.stats.STUNRequests++
		m.stats.STUNFailures++
		return fmt.Errorf("STUN discovery failed: %w", err)
	}

	m.stats.STUNRequests++
	m.stats.STUNSuccesses++

	m.localAddr = mapping.LocalAddr
	m.publicAddr = mapping.PublicAddr
	m.natType = mapping.MappingType

	// Detect CGNAT
	if m.config.CGNAT.EnableCGNATDetection {
		cgnatInfo, err := m.cgnatDetector.DetectCGNAT(m.localAddr, m.publicAddr)
		if err == nil && cgnatInfo.Detected {
			m.cgnatInfo = cgnatInfo
			m.stats.CGNATDetected++
		}
	}

	return nil
}

// Start starts the NAT traversal manager
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("already running")
	}
	m.running = true
	m.mu.Unlock()

	// Start STUN refresh routine
	m.stunClient.StartRefreshRoutine()

	// Start hole puncher keep-alive
	m.holePuncher.StartKeepAliveRoutine()

	// Start relay keep-alive if enabled
	if m.relayClient != nil {
		m.relayClient.StartKeepAliveRoutine()
	}

	return nil
}

// Stop stops the NAT traversal manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)

	// Close STUN client
	if err := m.stunClient.Close(); err != nil {
		return err
	}

	return nil
}

// Connect establishes a connection to a peer
func (m *Manager) Connect(peerInfo *PeerInfo) (*ConnectionInfo, error) {
	m.mu.RLock()
	// Check if already connected
	if conn, exists := m.connections[peerInfo.PeerID]; exists {
		m.mu.RUnlock()
		return conn, nil
	}
	m.mu.RUnlock()

	// Determine best connection method
	method := m.selectConnectionMethod(peerInfo)

	var connInfo *ConnectionInfo
	var err error

	switch method {
	case TraversalMethodDirect:
		connInfo, err = m.connectDirect(peerInfo)
	case TraversalMethodSTUN, TraversalMethodHolePunch:
		connInfo, err = m.connectViaHolePunch(peerInfo)
	case TraversalMethodRelay:
		connInfo, err = m.connectViaRelay(peerInfo)
	default:
		return nil, fmt.Errorf("unknown traversal method")
	}

	if err != nil {
		return nil, err
	}

	// Store connection
	m.mu.Lock()
	m.connections[peerInfo.PeerID] = connInfo
	m.stats.ActiveConnections++
	m.mu.Unlock()

	return connInfo, nil
}

// selectConnectionMethod selects the best connection method
func (m *Manager) selectConnectionMethod(peerInfo *PeerInfo) TraversalMethod {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If both peers are open (no NAT), use direct
	if m.natType == NATTypeOpen && peerInfo.NATType == NATTypeOpen {
		return TraversalMethodDirect
	}

	// If CGNAT detected and relay is forced, use relay
	if m.cgnatInfo != nil && m.cgnatInfo.Detected && m.config.CGNAT.ForceRelay {
		return TraversalMethodRelay
	}

	// If both NAT types can do direct connection, try hole punch
	if m.natType.CanDirectConnect() && peerInfo.NATType.CanDirectConnect() {
		return TraversalMethodHolePunch
	}

	// If either is symmetric NAT, prefer relay (or try aggressive hole punch)
	if m.natType.NeedsRelay() || peerInfo.NATType.NeedsRelay() {
		if m.relayClient != nil && m.config.Relay.EnableRelay {
			return TraversalMethodRelay
		}
		// Try aggressive hole punch as fallback
		if m.config.CGNAT.AggressivePunch {
			return TraversalMethodHolePunch
		}
	}

	// Default to hole punch
	return TraversalMethodHolePunch
}

// connectDirect establishes direct connection (no NAT)
func (m *Manager) connectDirect(peerInfo *PeerInfo) (*ConnectionInfo, error) {
	connInfo := &ConnectionInfo{
		PeerID:       peerInfo.PeerID,
		LocalAddr:    m.localAddr,
		RemoteAddr:   peerInfo.PublicAddr,
		Method:       TraversalMethodDirect,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}

	m.mu.Lock()
	m.stats.DirectConnections++
	m.mu.Unlock()

	return connInfo, nil
}

// connectViaHolePunch establishes connection via hole punching
func (m *Manager) connectViaHolePunch(peerInfo *PeerInfo) (*ConnectionInfo, error) {
	m.mu.Lock()
	m.stats.HolePunchAttempts++
	m.mu.Unlock()

	// Attempt hole punch
	result, err := m.holePuncher.SimultaneousPunch(peerInfo, true)
	if err != nil || !result.Success {
		m.mu.Lock()
		m.stats.HolePunchAttempts--
		m.mu.Unlock()

		// Fallback to relay if hole punch fails
		if m.relayClient != nil && m.config.Relay.PreferDirect {
			return m.connectViaRelay(peerInfo)
		}

		return nil, fmt.Errorf("hole punch failed: %w", err)
	}

	m.mu.Lock()
	m.stats.HolePunchSuccesses++
	m.mu.Unlock()

	connInfo := &ConnectionInfo{
		PeerID:       peerInfo.PeerID,
		LocalAddr:    m.localAddr,
		RemoteAddr:   result.RemoteAddr,
		Method:       TraversalMethodHolePunch,
		Established:  time.Now(),
		LastActivity: time.Now(),
		RTT:          result.RTT,
	}

	return connInfo, nil
}

// connectViaRelay establishes connection via relay server
func (m *Manager) connectViaRelay(peerInfo *PeerInfo) (*ConnectionInfo, error) {
	if m.relayClient == nil {
		return nil, fmt.Errorf("relay client not available")
	}

	// Get peer's relay ID (would be exchanged via signaling)
	peerRelayID := peerInfo.PeerID + "_relay" // Placeholder

	relayConn, err := m.relayClient.EstablishRelayConnection(peerInfo.PeerID, peerRelayID)
	if err != nil {
		return nil, fmt.Errorf("relay connection failed: %w", err)
	}

	m.mu.Lock()
	m.stats.RelayConnections++
	m.mu.Unlock()

	connInfo := &ConnectionInfo{
		PeerID:       peerInfo.PeerID,
		LocalAddr:    relayConn.LocalAddr,
		RemoteAddr:   relayConn.RelayAddr,
		Method:       TraversalMethodRelay,
		Established:  relayConn.Established,
		LastActivity: relayConn.LastActivity,
	}

	return connInfo, nil
}

// Disconnect closes connection to a peer
func (m *Manager) Disconnect(peerID string) error {
	m.mu.Lock()
	conn, exists := m.connections[peerID]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("no connection to peer %s", peerID)
	}
	delete(m.connections, peerID)
	m.stats.ActiveConnections--
	m.mu.Unlock()

	// Clean up based on connection method
	switch conn.Method {
	case TraversalMethodHolePunch:
		m.holePuncher.CloseSession(peerID)
	case TraversalMethodRelay:
		if m.relayClient != nil {
			m.relayClient.CloseRelayConnection(peerID)
		}
	}

	return nil
}

// GetLocalAddr returns the local address
func (m *Manager) GetLocalAddr() *net.UDPAddr {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.localAddr
}

// GetPublicAddr returns the public address discovered via STUN
func (m *Manager) GetPublicAddr() *net.UDPAddr {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.publicAddr
}

// GetNATType returns the detected NAT type
func (m *Manager) GetNATType() NATType {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.natType
}

// IsCGNATDetected returns whether CGNAT was detected
func (m *Manager) IsCGNATDetected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cgnatInfo != nil && m.cgnatInfo.Detected
}

// GetCGNATInfo returns CGNAT information
func (m *Manager) GetCGNATInfo() *CGNATInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cgnatInfo
}

// GetConnection returns connection info for a peer
func (m *Manager) GetConnection(peerID string) (*ConnectionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[peerID]
	return conn, exists
}

// GetAllConnections returns all active connections
func (m *Manager) GetAllConnections() map[string]*ConnectionInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	conns := make(map[string]*ConnectionInfo, len(m.connections))
	for k, v := range m.connections {
		conns[k] = v
	}
	return conns
}

// GetStats returns NAT traversal statistics
func (m *Manager) GetStats() *TraversalStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	stats := *m.stats
	return &stats
}

// RefreshNATMapping manually refreshes the NAT mapping
func (m *Manager) RefreshNATMapping() error {
	err := m.stunClient.RefreshMapping()
	if err != nil {
		m.mu.Lock()
		m.stats.STUNRequests++
		m.stats.STUNFailures++
		m.mu.Unlock()
		return err
	}

	m.mu.Lock()
	m.stats.STUNRequests++
	m.stats.STUNSuccesses++

	// Update addresses
	mapping := m.stunClient.GetMapping()
	if mapping != nil {
		m.publicAddr = mapping.PublicAddr
	}
	m.mu.Unlock()

	return nil
}

// GetRelayID returns the relay ID if relay is active
func (m *Manager) GetRelayID() string {
	if m.relayClient == nil {
		return ""
	}
	return m.relayClient.GetRelayID()
}

// GetTraversalCapabilities returns information about traversal capabilities
func (m *Manager) GetTraversalCapabilities() *TraversalCapabilities {
	m.mu.RLock()
	defer m.mu.RUnlock()

	caps := &TraversalCapabilities{
		LocalAddr:        m.localAddr,
		PublicAddr:       m.publicAddr,
		NATType:          m.natType,
		CanDirectConnect: m.natType.CanDirectConnect(),
		NeedsRelay:       m.natType.NeedsRelay(),
		RelayAvailable:   m.relayClient != nil,
		CGNATDetected:    m.cgnatInfo != nil && m.cgnatInfo.Detected,
	}

	if caps.CGNATDetected {
		caps.CGNATInfo = m.cgnatInfo
		strategy := m.cgnatDetector.GetRecommendedStrategy()
		caps.RecommendedStrategy = &strategy
	}

	return caps
}

// TraversalCapabilities contains information about NAT traversal capabilities
type TraversalCapabilities struct {
	LocalAddr           *net.UDPAddr
	PublicAddr          *net.UDPAddr
	NATType             NATType
	CanDirectConnect    bool
	NeedsRelay          bool
	RelayAvailable      bool
	CGNATDetected       bool
	CGNATInfo           *CGNATInfo
	RecommendedStrategy *TraversalStrategy
}

// UpdateConnection updates connection activity timestamp
func (m *Manager) UpdateConnection(peerID string, bytesSent, bytesReceived uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, exists := m.connections[peerID]; exists {
		conn.LastActivity = time.Now()
		conn.BytesSent += bytesSent
		conn.BytesReceived += bytesReceived
	}
}

// MeasureRTT measures round-trip time to a peer
func (m *Manager) MeasureRTT(peerID string) (time.Duration, error) {
	m.mu.RLock()
	conn, exists := m.connections[peerID]
	m.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("no connection to peer %s", peerID)
	}

	// If using relay, measure relay RTT
	if conn.Method == TraversalMethodRelay && m.relayClient != nil {
		rtt, err := m.relayClient.MeasureRelayRTT()
		if err == nil {
			m.mu.Lock()
			conn.RTT = rtt
			m.mu.Unlock()
		}
		return rtt, err
	}

	// For direct/hole punch, would need to implement ping-pong
	// Returning stored RTT for now
	return conn.RTT, nil
}

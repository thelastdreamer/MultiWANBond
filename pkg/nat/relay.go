package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// RelayConnection represents a connection through a relay server
type RelayConnection struct {
	PeerID         string
	LocalAddr      *net.UDPAddr
	RelayAddr      *net.UDPAddr
	PeerRelayID    string
	Established    time.Time
	LastActivity   time.Time
	BytesSent      uint64
	BytesReceived  uint64
	RTT            time.Duration
}

// RelayClient manages relay connections (TURN-like functionality)
type RelayClient struct {
	config *RelayConfig
	conn   *net.UDPConn
	mu     sync.RWMutex

	// Current relay server
	relayAddr *net.UDPAddr

	// Allocated relay ID from server
	relayID string

	// Active relay connections
	connections map[string]*RelayConnection

	// Bandwidth tracking
	totalSent     uint64
	totalReceived uint64
	lastBWCheck   time.Time

	// Stats
	established uint64
	failed      uint64
}

// NewRelayClient creates a new relay client
func NewRelayClient(conn *net.UDPConn, config *RelayConfig) (*RelayClient, error) {
	if config == nil {
		config = DefaultRelayConfig()
	}

	if !config.EnableRelay {
		return nil, fmt.Errorf("relay is disabled")
	}

	if len(config.RelayServers) == 0 {
		return nil, fmt.Errorf("no relay servers configured")
	}

	rc := &RelayClient{
		config:      config,
		conn:        conn,
		connections: make(map[string]*RelayConnection),
		lastBWCheck: time.Now(),
	}

	// Connect to first relay server
	if err := rc.connectToRelay(config.RelayServers[0]); err != nil {
		return nil, fmt.Errorf("failed to connect to relay: %w", err)
	}

	return rc, nil
}

// connectToRelay establishes connection with a relay server
func (rc *RelayClient) connectToRelay(relayServer string) error {
	addr, err := net.ResolveUDPAddr("udp4", relayServer)
	if err != nil {
		return fmt.Errorf("failed to resolve relay server: %w", err)
	}

	rc.mu.Lock()
	rc.relayAddr = addr
	rc.mu.Unlock()

	// Send allocation request
	allocMsg := []byte("RELAY:ALLOC")
	_, err = rc.conn.WriteToUDP(allocMsg, addr)
	if err != nil {
		return fmt.Errorf("failed to send allocation request: %w", err)
	}

	// Wait for allocation response
	rc.conn.SetReadDeadline(time.Now().Add(rc.config.Timeout))
	buffer := make([]byte, 1500)
	n, _, err := rc.conn.ReadFromUDP(buffer)
	if err != nil {
		return fmt.Errorf("failed to receive allocation response: %w", err)
	}

	// Parse response to get relay ID
	if n > 12 && string(buffer[:12]) == "RELAY:ALLOC:" {
		rc.relayID = string(buffer[12:n])
	} else {
		return fmt.Errorf("invalid allocation response")
	}

	return nil
}

// EstablishRelayConnection establishes a relay connection to a peer
func (rc *RelayClient) EstablishRelayConnection(peerID, peerRelayID string) (*RelayConnection, error) {
	rc.mu.RLock()
	if conn, exists := rc.connections[peerID]; exists {
		rc.mu.RUnlock()
		return conn, nil
	}
	rc.mu.RUnlock()

	// Check bandwidth limit
	if rc.config.MaxBandwidth > 0 {
		if err := rc.checkBandwidthLimit(); err != nil {
			rc.mu.Lock()
			rc.failed++
			rc.mu.Unlock()
			return nil, err
		}
	}

	// Send connect request to relay
	connectMsg := []byte(fmt.Sprintf("RELAY:CONNECT:%s:%s", rc.relayID, peerRelayID))
	_, err := rc.conn.WriteToUDP(connectMsg, rc.relayAddr)
	if err != nil {
		rc.mu.Lock()
		rc.failed++
		rc.mu.Unlock()
		return nil, fmt.Errorf("failed to send connect request: %w", err)
	}

	// Wait for connect response
	rc.conn.SetReadDeadline(time.Now().Add(rc.config.Timeout))
	buffer := make([]byte, 1500)
	n, _, err := rc.conn.ReadFromUDP(buffer)
	if err != nil {
		rc.mu.Lock()
		rc.failed++
		rc.mu.Unlock()
		return nil, fmt.Errorf("failed to receive connect response: %w", err)
	}

	// Check response
	if n < 13 || string(buffer[:13]) != "RELAY:CONNOK:" {
		rc.mu.Lock()
		rc.failed++
		rc.mu.Unlock()
		return nil, fmt.Errorf("relay connection failed")
	}

	// Create connection
	connection := &RelayConnection{
		PeerID:       peerID,
		LocalAddr:    rc.conn.LocalAddr().(*net.UDPAddr),
		RelayAddr:    rc.relayAddr,
		PeerRelayID:  peerRelayID,
		Established:  time.Now(),
		LastActivity: time.Now(),
	}

	rc.mu.Lock()
	rc.connections[peerID] = connection
	rc.established++
	rc.mu.Unlock()

	return connection, nil
}

// SendViaRelay sends data to a peer via relay
func (rc *RelayClient) SendViaRelay(peerID string, data []byte) error {
	rc.mu.RLock()
	conn, exists := rc.connections[peerID]
	relayAddr := rc.relayAddr
	relayID := rc.relayID
	rc.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no relay connection to peer %s", peerID)
	}

	// Check bandwidth limit
	if rc.config.MaxBandwidth > 0 {
		if err := rc.checkBandwidthLimit(); err != nil {
			return err
		}
	}

	// Build relay message: RELAY:DATA:fromID:toID:payload
	header := fmt.Sprintf("RELAY:DATA:%s:%s:", relayID, conn.PeerRelayID)
	message := append([]byte(header), data...)

	// Send to relay
	n, err := rc.conn.WriteToUDP(message, relayAddr)
	if err != nil {
		return fmt.Errorf("failed to send via relay: %w", err)
	}

	// Update stats
	rc.mu.Lock()
	conn.BytesSent += uint64(n)
	conn.LastActivity = time.Now()
	rc.totalSent += uint64(n)
	rc.mu.Unlock()

	return nil
}

// ReceiveViaRelay receives data from relay (should be called in a loop)
func (rc *RelayClient) ReceiveViaRelay() (peerID string, data []byte, err error) {
	buffer := make([]byte, 2000) // Slightly larger for relay overhead

	rc.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, remoteAddr, err := rc.conn.ReadFromUDP(buffer)
	if err != nil {
		return "", nil, err
	}

	// Check if from relay server
	rc.mu.RLock()
	relayAddr := rc.relayAddr
	rc.mu.RUnlock()

	if remoteAddr.String() != relayAddr.String() {
		return "", nil, fmt.Errorf("not from relay server")
	}

	// Parse relay message: RELAY:DATA:fromID:toID:payload
	if n < 11 || string(buffer[:11]) != "RELAY:DATA:" {
		return "", nil, fmt.Errorf("invalid relay message")
	}

	// Extract fromID, toID, and payload
	payload := buffer[11:n]

	// Find delimiters
	firstColon := -1
	secondColon := -1
	for i, b := range payload {
		if b == ':' {
			if firstColon == -1 {
				firstColon = i
			} else if secondColon == -1 {
				secondColon = i
				break
			}
		}
	}

	if firstColon == -1 || secondColon == -1 {
		return "", nil, fmt.Errorf("malformed relay message")
	}

	fromRelayID := string(payload[:firstColon])
	// toRelayID := string(payload[firstColon+1:secondColon])
	data = payload[secondColon+1:]

	// Find peer by relay ID
	rc.mu.Lock()
	var foundPeerID string
	for pid, conn := range rc.connections {
		if conn.PeerRelayID == fromRelayID {
			foundPeerID = pid
			conn.BytesReceived += uint64(len(data))
			conn.LastActivity = time.Now()
			break
		}
	}
	rc.totalReceived += uint64(len(data))
	rc.mu.Unlock()

	if foundPeerID == "" {
		return "", nil, fmt.Errorf("unknown relay peer")
	}

	return foundPeerID, data, nil
}

// checkBandwidthLimit checks if bandwidth limit is exceeded
func (rc *RelayClient) checkBandwidthLimit() error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rc.lastBWCheck)

	if elapsed < 1*time.Second {
		return nil // Don't check too frequently
	}

	// Calculate current bandwidth (bytes/sec)
	currentBW := rc.totalSent / uint64(elapsed.Seconds())

	if currentBW > rc.config.MaxBandwidth {
		return fmt.Errorf("bandwidth limit exceeded: %d/%d bytes/sec", currentBW, rc.config.MaxBandwidth)
	}

	// Reset counters
	rc.totalSent = 0
	rc.totalReceived = 0
	rc.lastBWCheck = now

	return nil
}

// CloseRelayConnection closes a relay connection
func (rc *RelayClient) CloseRelayConnection(peerID string) error {
	rc.mu.Lock()
	conn, exists := rc.connections[peerID]
	if !exists {
		rc.mu.Unlock()
		return fmt.Errorf("no connection to peer %s", peerID)
	}
	delete(rc.connections, peerID)
	relayAddr := rc.relayAddr
	relayID := rc.relayID
	rc.mu.Unlock()

	// Send disconnect message
	disconnectMsg := []byte(fmt.Sprintf("RELAY:DISCONNECT:%s:%s", relayID, conn.PeerRelayID))
	rc.conn.WriteToUDP(disconnectMsg, relayAddr)

	return nil
}

// GetRelayID returns the allocated relay ID
func (rc *RelayClient) GetRelayID() string {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.relayID
}

// GetConnection returns a relay connection
func (rc *RelayClient) GetConnection(peerID string) (*RelayConnection, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	conn, exists := rc.connections[peerID]
	return conn, exists
}

// GetActiveConnections returns all active relay connections
func (rc *RelayClient) GetActiveConnections() map[string]*RelayConnection {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	// Return a copy
	conns := make(map[string]*RelayConnection, len(rc.connections))
	for k, v := range rc.connections {
		conns[k] = v
	}
	return conns
}

// StartKeepAliveRoutine sends keep-alive messages to relay
func (rc *RelayClient) StartKeepAliveRoutine() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			rc.mu.RLock()
			relayAddr := rc.relayAddr
			relayID := rc.relayID
			rc.mu.RUnlock()

			if relayAddr == nil {
				continue
			}

			// Send keep-alive
			keepAliveMsg := []byte("RELAY:KEEPALIVE:" + relayID)
			rc.conn.WriteToUDP(keepAliveMsg, relayAddr)

			// Clean up idle connections
			rc.cleanupIdleConnections()
		}
	}()
}

// cleanupIdleConnections removes idle relay connections
func (rc *RelayClient) cleanupIdleConnections() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	now := time.Now()
	idleTimeout := 5 * time.Minute

	for peerID, conn := range rc.connections {
		if now.Sub(conn.LastActivity) > idleTimeout {
			delete(rc.connections, peerID)
		}
	}
}

// GetStats returns relay client statistics
func (rc *RelayClient) GetStats() (established, failed uint64, totalSent, totalReceived uint64) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.established, rc.failed, rc.totalSent, rc.totalReceived
}

// SwitchRelay switches to a different relay server
func (rc *RelayClient) SwitchRelay(relayServer string) error {
	// Close existing connections
	rc.mu.Lock()
	oldConnections := rc.connections
	rc.connections = make(map[string]*RelayConnection)
	rc.mu.Unlock()

	// Send disconnect to old relay
	if rc.relayAddr != nil {
		disconnectMsg := []byte("RELAY:DISCONNECT:" + rc.relayID)
		rc.conn.WriteToUDP(disconnectMsg, rc.relayAddr)
	}

	// Connect to new relay
	if err := rc.connectToRelay(relayServer); err != nil {
		// Restore old connections on failure
		rc.mu.Lock()
		rc.connections = oldConnections
		rc.mu.Unlock()
		return err
	}

	return nil
}

// MeasureRelayRTT measures round-trip time to relay server
func (rc *RelayClient) MeasureRelayRTT() (time.Duration, error) {
	rc.mu.RLock()
	relayAddr := rc.relayAddr
	relayID := rc.relayID
	rc.mu.RUnlock()

	if relayAddr == nil {
		return 0, fmt.Errorf("not connected to relay")
	}

	// Send ping
	pingMsg := []byte("RELAY:PING:" + relayID)
	start := time.Now()

	_, err := rc.conn.WriteToUDP(pingMsg, relayAddr)
	if err != nil {
		return 0, fmt.Errorf("failed to send ping: %w", err)
	}

	// Wait for pong
	rc.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buffer := make([]byte, 100)
	n, _, err := rc.conn.ReadFromUDP(buffer)
	if err != nil {
		return 0, fmt.Errorf("failed to receive pong: %w", err)
	}

	rtt := time.Since(start)

	// Check response
	if n < 11 || string(buffer[:11]) != "RELAY:PONG:" {
		return 0, fmt.Errorf("invalid pong response")
	}

	return rtt, nil
}

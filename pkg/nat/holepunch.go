package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// PunchResult represents the result of a hole punching attempt
type PunchResult struct {
	Success    bool
	RemoteAddr *net.UDPAddr
	Method     TraversalMethod
	RTT        time.Duration
	Attempts   int
	Error      error
}

// HolePuncher manages UDP hole punching operations
type HolePuncher struct {
	config *HolePunchConfig
	conn   *net.UDPConn
	mu     sync.RWMutex

	// Active punch sessions
	sessions map[string]*punchSession

	// Stats
	attempts  uint64
	successes uint64
	failures  uint64
}

// punchSession represents an ongoing hole punching session
type punchSession struct {
	peerID       string
	localAddr    *net.UDPAddr
	peerLocalAddr *net.UDPAddr
	peerPublicAddr *net.UDPAddr
	startTime    time.Time
	attempts     int
	lastPunch    time.Time
	established  bool
	conn         *net.UDPConn
}

// NewHolePuncher creates a new hole puncher
func NewHolePuncher(conn *net.UDPConn, config *HolePunchConfig) *HolePuncher {
	if config == nil {
		config = DefaultHolePunchConfig()
	}

	return &HolePuncher{
		config:   config,
		conn:     conn,
		sessions: make(map[string]*punchSession),
	}
}

// Punch attempts to establish a P2P connection with a peer via hole punching
func (hp *HolePuncher) Punch(peerInfo *PeerInfo) (*PunchResult, error) {
	hp.mu.Lock()

	// Check if session already exists
	if session, exists := hp.sessions[peerInfo.PeerID]; exists {
		if session.established {
			hp.mu.Unlock()
			return &PunchResult{
				Success:    true,
				RemoteAddr: session.peerPublicAddr,
				Method:     TraversalMethodHolePunch,
				Attempts:   session.attempts,
			}, nil
		}
	}

	// Create new session
	session := &punchSession{
		peerID:         peerInfo.PeerID,
		localAddr:      hp.conn.LocalAddr().(*net.UDPAddr),
		peerLocalAddr:  peerInfo.LocalAddr,
		peerPublicAddr: peerInfo.PublicAddr,
		startTime:      time.Now(),
		conn:           hp.conn,
	}

	hp.sessions[peerInfo.PeerID] = session
	hp.attempts++
	hp.mu.Unlock()

	// Perform the punch
	result := hp.performPunch(session, peerInfo.NATType)

	hp.mu.Lock()
	if result.Success {
		session.established = true
		hp.successes++
	} else {
		hp.failures++
		delete(hp.sessions, peerInfo.PeerID)
	}
	hp.mu.Unlock()

	return result, nil
}

// performPunch performs the actual hole punching
func (hp *HolePuncher) performPunch(session *punchSession, peerNATType NATType) *PunchResult {
	result := &PunchResult{
		Success: false,
	}

	// Create punch message
	punchMsg := []byte("PUNCH:" + session.peerID)

	// Strategy depends on NAT types
	// For most NAT types, we need to punch to both local and public addresses
	addrs := hp.selectPunchAddresses(session, peerNATType)

	deadline := time.Now().Add(hp.config.Timeout)
	attemptDeadline := time.Now().Add(hp.config.RetryInterval)

	// Listen for responses in background
	responseChan := make(chan *net.UDPAddr, 1)
	stopChan := make(chan struct{})
	defer close(stopChan)

	go hp.listenForPunchResponse(session.peerID, responseChan, stopChan)

	// Punch loop
	for time.Now().Before(deadline) && result.Attempts < hp.config.MaxAttempts {
		// Send punch packets to all candidate addresses
		for _, addr := range addrs {
			hp.conn.WriteToUDP(punchMsg, addr)
		}

		result.Attempts++
		session.attempts++
		session.lastPunch = time.Now()

		// Wait for response or timeout
		attemptDeadline = time.Now().Add(hp.config.RetryInterval)
		select {
		case remoteAddr := <-responseChan:
			// Success! We got a response
			result.Success = true
			result.RemoteAddr = remoteAddr
			result.Method = TraversalMethodHolePunch
			result.RTT = time.Since(session.lastPunch)
			return result
		case <-time.After(time.Until(attemptDeadline)):
			// Try again
			continue
		}
	}

	result.Error = fmt.Errorf("hole punch timeout after %d attempts", result.Attempts)
	return result
}

// selectPunchAddresses selects which addresses to punch based on NAT types
func (hp *HolePuncher) selectPunchAddresses(session *punchSession, peerNATType NATType) []*net.UDPAddr {
	addrs := make([]*net.UDPAddr, 0, 3)

	// Always try public address
	if session.peerPublicAddr != nil {
		addrs = append(addrs, session.peerPublicAddr)
	}

	// Try local address for LAN peers or full cone NAT
	if session.peerLocalAddr != nil {
		if hp.isLikelyLAN(session.peerLocalAddr) || peerNATType == NATTypeFullCone {
			addrs = append(addrs, session.peerLocalAddr)
		}
	}

	// For symmetric NAT, try port prediction
	if peerNATType == NATTypeSymmetric {
		predictedAddrs := hp.predictSymmetricPorts(session.peerPublicAddr)
		addrs = append(addrs, predictedAddrs...)
	}

	return addrs
}

// isLikelyLAN checks if address is likely on same LAN
func (hp *HolePuncher) isLikelyLAN(addr *net.UDPAddr) bool {
	localAddr := hp.conn.LocalAddr().(*net.UDPAddr)

	// Check if in same /24 subnet (simple heuristic)
	if addr.IP[0] == localAddr.IP[0] &&
	   addr.IP[1] == localAddr.IP[1] &&
	   addr.IP[2] == localAddr.IP[2] {
		return true
	}

	return false
}

// predictSymmetricPorts predicts likely ports for symmetric NAT
func (hp *HolePuncher) predictSymmetricPorts(baseAddr *net.UDPAddr) []*net.UDPAddr {
	if baseAddr == nil {
		return nil
	}

	// Try ports around the base port (common NAT behavior)
	addrs := make([]*net.UDPAddr, 0, 10)
	basePort := baseAddr.Port

	// Try sequential ports (many NATs allocate sequentially)
	for offset := -5; offset <= 5; offset++ {
		if offset == 0 {
			continue // Already tried base port
		}
		port := basePort + offset
		if port > 0 && port <= 65535 {
			addrs = append(addrs, &net.UDPAddr{
				IP:   baseAddr.IP,
				Port: port,
			})
		}
	}

	return addrs
}

// listenForPunchResponse listens for punch response from peer
func (hp *HolePuncher) listenForPunchResponse(peerID string, responseChan chan *net.UDPAddr, stopChan chan struct{}) {
	buffer := make([]byte, 1500)

	for {
		select {
		case <-stopChan:
			return
		default:
			hp.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, remoteAddr, err := hp.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			// Check if this is a punch response
			if n > 6 && string(buffer[:6]) == "PUNCH:" {
				receivedPeerID := string(buffer[6:n])
				if receivedPeerID == peerID {
					select {
					case responseChan <- remoteAddr:
						return
					case <-stopChan:
						return
					}
				}
			}
		}
	}
}

// MaintainHole sends keep-alive packets to maintain the NAT hole
func (hp *HolePuncher) MaintainHole(peerID string) error {
	hp.mu.RLock()
	session, exists := hp.sessions[peerID]
	hp.mu.RUnlock()

	if !exists || !session.established {
		return fmt.Errorf("no established session for peer %s", peerID)
	}

	// Send keep-alive
	keepAliveMsg := []byte("KEEPALIVE:" + peerID)
	_, err := hp.conn.WriteToUDP(keepAliveMsg, session.peerPublicAddr)
	if err != nil {
		return fmt.Errorf("failed to send keep-alive: %w", err)
	}

	session.lastPunch = time.Now()
	return nil
}

// StartKeepAliveRoutine starts automatic keep-alive for all established sessions
func (hp *HolePuncher) StartKeepAliveRoutine() {
	go func() {
		ticker := time.NewTicker(hp.config.KeepAliveInterval)
		defer ticker.Stop()

		for range ticker.C {
			hp.mu.RLock()
			peerIDs := make([]string, 0, len(hp.sessions))
			for peerID, session := range hp.sessions {
				if session.established {
					peerIDs = append(peerIDs, peerID)
				}
			}
			hp.mu.RUnlock()

			// Send keep-alives
			for _, peerID := range peerIDs {
				hp.MaintainHole(peerID)
			}

			// Clean up old sessions
			hp.cleanupSessions()
		}
	}()
}

// cleanupSessions removes expired sessions
func (hp *HolePuncher) cleanupSessions() {
	hp.mu.Lock()
	defer hp.mu.Unlock()

	now := time.Now()
	timeout := 60 * time.Second

	for peerID, session := range hp.sessions {
		if now.Sub(session.lastPunch) > timeout {
			delete(hp.sessions, peerID)
		}
	}
}

// GetSession returns information about a punch session
func (hp *HolePuncher) GetSession(peerID string) (*punchSession, bool) {
	hp.mu.RLock()
	defer hp.mu.RUnlock()
	session, exists := hp.sessions[peerID]
	return session, exists
}

// CloseSession closes a punch session
func (hp *HolePuncher) CloseSession(peerID string) {
	hp.mu.Lock()
	defer hp.mu.Unlock()
	delete(hp.sessions, peerID)
}

// GetStats returns hole punching statistics
func (hp *HolePuncher) GetStats() (attempts, successes, failures uint64) {
	hp.mu.RLock()
	defer hp.mu.RUnlock()
	return hp.attempts, hp.successes, hp.failures
}

// SimultaneousPunch performs simultaneous punch with a peer
// This is more reliable when both peers are behind NAT
func (hp *HolePuncher) SimultaneousPunch(peerInfo *PeerInfo, coordinated bool) (*PunchResult, error) {
	if !coordinated {
		// If not coordinated, just do regular punch
		return hp.Punch(peerInfo)
	}

	// For coordinated punch, we expect both peers to start punching at roughly the same time
	// This is typically coordinated through a signaling server

	hp.mu.Lock()
	session := &punchSession{
		peerID:         peerInfo.PeerID,
		localAddr:      hp.conn.LocalAddr().(*net.UDPAddr),
		peerLocalAddr:  peerInfo.LocalAddr,
		peerPublicAddr: peerInfo.PublicAddr,
		startTime:      time.Now(),
		conn:           hp.conn,
	}
	hp.sessions[peerInfo.PeerID] = session
	hp.attempts++
	hp.mu.Unlock()

	result := &PunchResult{
		Success: false,
	}

	// Create punch message
	punchMsg := []byte("SIMPUNCH:" + session.peerID)

	// Select addresses
	addrs := hp.selectPunchAddresses(session, peerInfo.NATType)

	// Listen for responses
	responseChan := make(chan *net.UDPAddr, 1)
	stopChan := make(chan struct{})
	defer close(stopChan)

	go hp.listenForSimultaneousPunchResponse(session.peerID, responseChan, stopChan)

	// Rapid fire punches for first 2 seconds
	rapidDeadline := time.Now().Add(2 * time.Second)
	deadline := time.Now().Add(hp.config.Timeout)

	for time.Now().Before(deadline) && result.Attempts < hp.config.MaxAttempts {
		// Send to all addresses
		for _, addr := range addrs {
			hp.conn.WriteToUDP(punchMsg, addr)
		}

		result.Attempts++
		session.attempts++
		session.lastPunch = time.Now()

		// Use shorter interval during rapid phase
		interval := hp.config.RetryInterval
		if time.Now().Before(rapidDeadline) {
			interval = 100 * time.Millisecond
		}

		select {
		case remoteAddr := <-responseChan:
			result.Success = true
			result.RemoteAddr = remoteAddr
			result.Method = TraversalMethodHolePunch
			result.RTT = time.Since(session.lastPunch)

			hp.mu.Lock()
			session.established = true
			hp.successes++
			hp.mu.Unlock()

			return result, nil
		case <-time.After(interval):
			continue
		}
	}

	hp.mu.Lock()
	hp.failures++
	delete(hp.sessions, peerInfo.PeerID)
	hp.mu.Unlock()

	result.Error = fmt.Errorf("simultaneous punch timeout after %d attempts", result.Attempts)
	return result, result.Error
}

// listenForSimultaneousPunchResponse listens for simultaneous punch response
func (hp *HolePuncher) listenForSimultaneousPunchResponse(peerID string, responseChan chan *net.UDPAddr, stopChan chan struct{}) {
	buffer := make([]byte, 1500)

	for {
		select {
		case <-stopChan:
			return
		default:
			hp.conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			n, remoteAddr, err := hp.conn.ReadFromUDP(buffer)
			if err != nil {
				continue
			}

			// Check for simultaneous punch response
			if n > 9 && string(buffer[:9]) == "SIMPUNCH:" {
				receivedPeerID := string(buffer[9:n])
				if receivedPeerID == peerID {
					select {
					case responseChan <- remoteAddr:
						return
					case <-stopChan:
						return
					}
				}
			}
		}
	}
}

package server

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// SessionManager manages multiple client sessions
type SessionManager struct {
	mu               sync.RWMutex
	sessions         map[string]*ClientSession // SessionID -> Session
	clientSessions   map[string][]*ClientSession // ClientID -> Sessions
	ipSessions       map[string][]*ClientSession // IP -> Sessions
	natPool          *NATPool
	config           *ServerConfig
	stats            *ServerStats
	eventChan        chan SessionEvent
	stopChan         chan struct{}
	wg               sync.WaitGroup
}

// NATPool manages allocation of NAT IPs
type NATPool struct {
	mu          sync.RWMutex
	available   []net.IP
	allocated   map[string]net.IP // SessionID -> IP
	startIP     net.IP
	size        int
}

// NewSessionManager creates a new session manager
func NewSessionManager(config *ServerConfig) *SessionManager {
	sm := &SessionManager{
		sessions:       make(map[string]*ClientSession),
		clientSessions: make(map[string][]*ClientSession),
		ipSessions:     make(map[string][]*ClientSession),
		config:         config,
		stats: &ServerStats{
			StartTime:      time.Now(),
			WANUtilization: make(map[uint8]float64),
		},
		eventChan: make(chan SessionEvent, 100),
		stopChan:  make(chan struct{}),
	}

	// Initialize NAT pool
	sm.natPool = NewNATPool(config.NATPoolStart, config.NATPoolSize)

	return sm
}

// Start starts the session manager
func (sm *SessionManager) Start() {
	sm.wg.Add(1)
	go sm.cleanupRoutine()
}

// Stop stops the session manager
func (sm *SessionManager) Stop() {
	close(sm.stopChan)
	sm.wg.Wait()
}

// CreateSession creates a new client session
func (sm *SessionManager) CreateSession(clientID string, remoteAddr *net.UDPAddr, config *ClientConfig) (*ClientSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check connection limits
	if len(sm.sessions) >= sm.config.MaxClients {
		return nil, fmt.Errorf("max clients reached (%d)", sm.config.MaxClients)
	}

	// Check per-IP limit
	ipSessions := sm.ipSessions[remoteAddr.IP.String()]
	if len(ipSessions) >= sm.config.MaxClientsPerIP {
		return nil, fmt.Errorf("max clients per IP reached (%d)", sm.config.MaxClientsPerIP)
	}

	// Check per-client limit
	clientSessions := sm.clientSessions[clientID]
	if len(clientSessions) >= sm.config.MaxSessionsPerClient {
		return nil, fmt.Errorf("max sessions per client reached (%d)", sm.config.MaxSessionsPerClient)
	}

	// Generate session ID
	sessionID := fmt.Sprintf("%s-%d", clientID, time.Now().UnixNano())

	// Allocate NAT IP
	publicIP, err := sm.natPool.Allocate(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate NAT IP: %w", err)
	}

	// Use provided config or default
	if config == nil {
		config = sm.config.DefaultClientConfig
	}

	// Create session
	session := &ClientSession{
		ID:            sessionID,
		ClientID:      clientID,
		RemoteAddr:    remoteAddr,
		PublicIP:      publicIP,
		WANInterfaces: make(map[uint8]*ClientWANState),
		NATMappings:   NewNATTable(),
		BandwidthQuota: &BandwidthQuota{
			MaxUpload:   config.MaxUploadBandwidth,
			MaxDownload: config.MaxDownloadBandwidth,
			LastReset:   time.Now(),
		},
		StartTime: time.Now(),
		LastSeen:  time.Now(),
		Config:    config,
		State:     ClientStateConnecting,
		Metadata:  make(map[string]interface{}),
	}

	// Add to maps
	sm.sessions[sessionID] = session
	sm.clientSessions[clientID] = append(sm.clientSessions[clientID], session)
	sm.ipSessions[remoteAddr.IP.String()] = append(sm.ipSessions[remoteAddr.IP.String()], session)

	// Update stats
	sm.stats.TotalSessions++
	sm.stats.ActiveSessions++
	sm.stats.TotalClients++
	sm.stats.ActiveClients = len(sm.clientSessions)

	// Send event
	sm.sendEvent(SessionEvent{
		Type:      EventSessionCreated,
		SessionID: sessionID,
		ClientID:  clientID,
		Timestamp: time.Now(),
		Details:   fmt.Sprintf("Client %s connected from %s", clientID, remoteAddr),
	})

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*ClientSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// GetClientSessions retrieves all sessions for a client
func (sm *SessionManager) GetClientSessions(clientID string) []*ClientSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := sm.clientSessions[clientID]
	result := make([]*ClientSession, len(sessions))
	copy(result, sessions)
	return result
}

// RemoveSession removes a session
func (sm *SessionManager) RemoveSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Release NAT IP
	sm.natPool.Release(sessionID)

	// Remove from maps
	delete(sm.sessions, sessionID)

	// Remove from client sessions
	clientSessions := sm.clientSessions[session.ClientID]
	for i, s := range clientSessions {
		if s.ID == sessionID {
			sm.clientSessions[session.ClientID] = append(clientSessions[:i], clientSessions[i+1:]...)
			break
		}
	}

	// Remove from IP sessions
	ipKey := session.RemoteAddr.IP.String()
	ipSessions := sm.ipSessions[ipKey]
	for i, s := range ipSessions {
		if s.ID == sessionID {
			sm.ipSessions[ipKey] = append(ipSessions[:i], ipSessions[i+1:]...)
			break
		}
	}

	// Update stats
	sm.stats.ActiveSessions--
	sm.stats.ActiveClients = len(sm.clientSessions)

	// Send event
	sm.sendEvent(SessionEvent{
		Type:      EventSessionDisconnected,
		SessionID: sessionID,
		ClientID:  session.ClientID,
		Timestamp: time.Now(),
		Details:   "Session terminated",
	})

	return nil
}

// UpdateSessionActivity updates the last seen time for a session
func (sm *SessionManager) UpdateSessionActivity(sessionID string) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if exists {
		session.mu.Lock()
		session.LastSeen = time.Now()
		session.mu.Unlock()
	}
}

// GetAllSessions returns all active sessions
func (sm *SessionManager) GetAllSessions() []*ClientSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*ClientSession, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// GetStats returns server statistics
func (sm *SessionManager) GetStats() *ServerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy
	stats := *sm.stats
	return &stats
}

// cleanupRoutine periodically cleans up idle/expired sessions
func (sm *SessionManager) cleanupRoutine() {
	defer sm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.cleanupIdleSessions()
		}
	}
}

// cleanupIdleSessions removes idle and expired sessions
func (sm *SessionManager) cleanupIdleSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	toRemove := make([]string, 0)

	for sessionID, session := range sm.sessions {
		session.mu.RLock()

		// Check idle timeout
		idleTime := now.Sub(session.LastSeen)
		if idleTime > session.Config.IdleTimeout {
			toRemove = append(toRemove, sessionID)
			session.mu.RUnlock()
			continue
		}

		// Check session timeout
		sessionDuration := now.Sub(session.StartTime)
		if sessionDuration > session.Config.SessionTimeout {
			toRemove = append(toRemove, sessionID)
			session.mu.RUnlock()
			continue
		}

		session.mu.RUnlock()
	}

	// Remove idle sessions
	for _, sessionID := range toRemove {
		session := sm.sessions[sessionID]

		// Release NAT IP
		sm.natPool.Release(sessionID)

		// Remove from maps
		delete(sm.sessions, sessionID)

		// Remove from client sessions
		clientSessions := sm.clientSessions[session.ClientID]
		for i, s := range clientSessions {
			if s.ID == sessionID {
				sm.clientSessions[session.ClientID] = append(clientSessions[:i], clientSessions[i+1:]...)
				break
			}
		}

		// Remove from IP sessions
		ipKey := session.RemoteAddr.IP.String()
		ipSessions := sm.ipSessions[ipKey]
		for i, s := range ipSessions {
			if s.ID == sessionID {
				sm.ipSessions[ipKey] = append(ipSessions[:i], ipSessions[i+1:]...)
				break
			}
		}

		// Update stats
		sm.stats.ActiveSessions--
		sm.stats.ActiveClients = len(sm.clientSessions)

		// Send event
		sm.sendEvent(SessionEvent{
			Type:      EventSessionIdle,
			SessionID: sessionID,
			ClientID:  session.ClientID,
			Timestamp: time.Now(),
			Details:   "Session removed due to inactivity",
		})
	}
}

// sendEvent sends a session event (non-blocking)
func (sm *SessionManager) sendEvent(event SessionEvent) {
	select {
	case sm.eventChan <- event:
	default:
		// Channel full, skip event
	}
}

// GetEventChannel returns the event channel
func (sm *SessionManager) GetEventChannel() <-chan SessionEvent {
	return sm.eventChan
}

// NewNATPool creates a new NAT IP pool
func NewNATPool(startIP net.IP, size int) *NATPool {
	pool := &NATPool{
		available: make([]net.IP, 0, size),
		allocated: make(map[string]net.IP),
		startIP:   startIP,
		size:      size,
	}

	// Generate IP range
	ip := make(net.IP, len(startIP))
	copy(ip, startIP)

	for i := 0; i < size; i++ {
		pool.available = append(pool.available, net.IP(append([]byte(nil), ip...)))

		// Increment IP
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	return pool
}

// Allocate allocates an IP from the pool
func (np *NATPool) Allocate(sessionID string) (net.IP, error) {
	np.mu.Lock()
	defer np.mu.Unlock()

	if len(np.available) == 0 {
		return nil, fmt.Errorf("NAT pool exhausted")
	}

	// Take first available IP
	ip := np.available[0]
	np.available = np.available[1:]
	np.allocated[sessionID] = ip

	return ip, nil
}

// Release releases an IP back to the pool
func (np *NATPool) Release(sessionID string) {
	np.mu.Lock()
	defer np.mu.Unlock()

	ip, exists := np.allocated[sessionID]
	if !exists {
		return
	}

	delete(np.allocated, sessionID)
	np.available = append(np.available, ip)
}

// GetAvailable returns the number of available IPs
func (np *NATPool) GetAvailable() int {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return len(np.available)
}

// GetAllocated returns the number of allocated IPs
func (np *NATPool) GetAllocated() int {
	np.mu.RLock()
	defer np.mu.RUnlock()
	return len(np.allocated)
}

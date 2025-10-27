// Package security - Security manager
package security

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager coordinates all security operations
type Manager struct {
	config        *SecurityConfig
	encryptor     *Encryptor
	authenticator *Authenticator
	rateLimiter   *RateLimiter
	authChecker   *AuthorizationChecker

	// Event handling
	events    []*SecurityEvent
	eventsMu  sync.RWMutex
	maxEvents int

	// Background tasks
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a new security manager
func NewManager(config *SecurityConfig) *Manager {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:        config,
		encryptor:     NewEncryptor(config),
		authenticator: NewAuthenticator(config),
		rateLimiter:   NewRateLimiter(config.RateLimitWindow, config.MaxConnectionsPerIP),
		authChecker:   NewAuthorizationChecker(),
		events:        make([]*SecurityEvent, 0),
		maxEvents:     1000,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the security manager
func (m *Manager) Start() error {
	// Start key rotation if enabled
	if m.config.KeyRotationEnabled {
		m.wg.Add(1)
		go m.keyRotationWorker()
	}

	// Start cleanup worker
	m.wg.Add(1)
	go m.cleanupWorker()

	return nil
}

// Stop stops the security manager
func (m *Manager) Stop() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

// Encrypt encrypts data for a recipient
func (m *Manager) Encrypt(data []byte, recipientID string) (*EncryptedPacket, error) {
	if !m.config.EncryptionEnabled {
		return &EncryptedPacket{
			Header: PacketHeader{
				Version:        1,
				EncryptionType: EncryptionNone,
				Timestamp:      time.Now(),
				RecipientID:    recipientID,
			},
			Payload: data,
		}, nil
	}

	packet, err := m.encryptor.Encrypt(data, recipientID)
	if err != nil {
		m.recordEvent(NewSecurityEvent(EventEncryptionError, "error",
			fmt.Sprintf("Encryption failed: %v", err), "", ""))
		return nil, err
	}

	return packet, nil
}

// Decrypt decrypts an encrypted packet
func (m *Manager) Decrypt(packet *EncryptedPacket) ([]byte, error) {
	data, err := m.encryptor.Decrypt(packet)
	if err != nil {
		m.recordEvent(NewSecurityEvent(EventEncryptionError, "error",
			fmt.Sprintf("Decryption failed: %v", err), packet.Header.RecipientID, ""))
		return nil, err
	}

	return data, nil
}

// Authenticate authenticates a peer
func (m *Manager) Authenticate(peerID, ip string, credentials interface{}) (*Session, error) {
	// Check rate limit
	if m.config.EnableRateLimit {
		if !m.rateLimiter.Allow(ip) {
			m.recordEvent(NewSecurityEvent(EventRateLimitExceeded, "warning",
				"Rate limit exceeded", peerID, ip))
			return nil, fmt.Errorf("rate limit exceeded for IP: %s", ip)
		}
	}

	// Authenticate
	session, err := m.authenticator.Authenticate(peerID, credentials)
	if err != nil {
		m.recordEvent(NewSecurityEvent(EventAuthFailure, "warning",
			fmt.Sprintf("Authentication failed: %v", err), peerID, ip))
		return nil, err
	}

	m.recordEvent(NewSecurityEvent(EventAuthSuccess, "info",
		"Authentication successful", peerID, ip))

	return session, nil
}

// CheckAuthorization checks if a peer is authorized
func (m *Manager) CheckAuthorization(peerID, ip, policyID string) bool {
	allowed := m.authChecker.CheckAccess(peerID, ip, policyID)

	if !allowed {
		m.recordEvent(NewSecurityEvent(EventUnauthorizedAccess, "warning",
			"Unauthorized access attempt", peerID, ip))
	}

	return allowed
}

// GenerateToken generates an authentication token
func (m *Manager) GenerateToken(peerID string, validity time.Duration) (string, error) {
	return m.authenticator.GenerateToken(peerID, validity)
}

// GetSession retrieves a session
func (m *Manager) GetSession(sessionID string) (*Session, error) {
	return m.authenticator.GetSession(sessionID)
}

// RevokeSession revokes a session
func (m *Manager) RevokeSession(sessionID string) {
	m.authenticator.RevokeSession(sessionID)
}

// AddPeer adds a trusted peer
func (m *Manager) AddPeer(peer *Peer) {
	m.authenticator.GetTrustStore().AddPeer(peer)
	m.recordEvent(NewSecurityEvent(EventPeerConnected, "info",
		"Peer added to trust store", peer.ID, ""))
}

// RemovePeer removes a peer
func (m *Manager) RemovePeer(peerID string) {
	m.authenticator.GetTrustStore().RemovePeer(peerID)
	m.recordEvent(NewSecurityEvent(EventPeerDisconnected, "info",
		"Peer removed from trust store", peerID, ""))
}

// GetPeer retrieves a peer
func (m *Manager) GetPeer(peerID string) (*Peer, bool) {
	return m.authenticator.GetTrustStore().GetPeer(peerID)
}

// AddPolicy adds a security policy
func (m *Manager) AddPolicy(policy *SecurityPolicy) {
	m.authChecker.AddPolicy(policy)
}

// GetPolicy retrieves a security policy
func (m *Manager) GetPolicy(policyID string) (*SecurityPolicy, bool) {
	return m.authChecker.GetPolicy(policyID)
}

// RemovePolicy removes a security policy
func (m *Manager) RemovePolicy(policyID string) {
	m.authChecker.RemovePolicy(policyID)
}

// GetEvents returns security events
func (m *Manager) GetEvents() []*SecurityEvent {
	m.eventsMu.RLock()
	defer m.eventsMu.RUnlock()

	result := make([]*SecurityEvent, len(m.events))
	copy(result, m.events)
	return result
}

// GetRecentEvents returns recent security events
func (m *Manager) GetRecentEvents(limit int) []*SecurityEvent {
	m.eventsMu.RLock()
	defer m.eventsMu.RUnlock()

	start := len(m.events) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*SecurityEvent, len(m.events)-start)
	copy(result, m.events[start:])
	return result
}

// recordEvent records a security event
func (m *Manager) recordEvent(event *SecurityEvent) {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()

	m.events = append(m.events, event)

	// Trim events if exceeding max
	if len(m.events) > m.maxEvents {
		m.events = m.events[len(m.events)-m.maxEvents:]
	}
}

// keyRotationWorker periodically rotates encryption keys
func (m *Manager) keyRotationWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.KeyRotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			rotated := m.encryptor.keyStore.RotateKeys()
			m.recordEvent(NewSecurityEvent(EventKeyRotation, "info",
				fmt.Sprintf("Rotated %d encryption keys", rotated), "", ""))
		}
	}
}

// cleanupWorker periodically cleans up expired data
func (m *Manager) cleanupWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Cleanup expired keys
			keysRemoved := m.encryptor.keyStore.CleanupExpired()

			// Cleanup expired sessions and tokens
			sessionsRemoved := m.authenticator.CleanupExpired()

			total := keysRemoved + sessionsRemoved
			if total > 0 {
				m.recordEvent(NewSecurityEvent(EventKeyRotation, "info",
					fmt.Sprintf("Cleaned up %d expired items", total), "", ""))
			}
		}
	}
}

// GetStats returns security statistics
func (m *Manager) GetStats() *SecurityStats {
	stats := &SecurityStats{
		EncryptionEnabled: m.config.EncryptionEnabled,
		EncryptionType:    m.config.EncryptionType.String(),
		AuthEnabled:       m.config.AuthEnabled,
		AuthType:          m.config.AuthType.String(),
		TotalEvents:       len(m.events),
	}

	// Count event types
	m.eventsMu.RLock()
	authSuccessCount := 0
	authFailureCount := 0
	encryptionErrorCount := 0
	unauthorizedCount := 0

	for _, event := range m.events {
		switch event.Type {
		case EventAuthSuccess:
			authSuccessCount++
		case EventAuthFailure:
			authFailureCount++
		case EventEncryptionError:
			encryptionErrorCount++
		case EventUnauthorizedAccess:
			unauthorizedCount++
		}
	}
	m.eventsMu.RUnlock()

	stats.AuthSuccessCount = authSuccessCount
	stats.AuthFailureCount = authFailureCount
	stats.EncryptionErrorCount = encryptionErrorCount
	stats.UnauthorizedAccessCount = unauthorizedCount

	return stats
}

// SecurityStats represents security statistics
type SecurityStats struct {
	EncryptionEnabled      bool
	EncryptionType         string
	AuthEnabled            bool
	AuthType               string
	TotalEvents            int
	AuthSuccessCount       int
	AuthFailureCount       int
	EncryptionErrorCount   int
	UnauthorizedAccessCount int
}

// Package security provides encryption and security features
package security

import (
	"crypto/x509"
	"sync"
	"time"
)

// EncryptionType defines the type of encryption
type EncryptionType int

const (
	// EncryptionNone no encryption (not recommended)
	EncryptionNone EncryptionType = iota
	// EncryptionAES256GCM AES-256-GCM encryption
	EncryptionAES256GCM
	// EncryptionChaCha20Poly1305 ChaCha20-Poly1305 encryption
	EncryptionChaCha20Poly1305
	// EncryptionWireGuard WireGuard-style encryption (Noise protocol)
	EncryptionWireGuard
)

// String returns the string representation of the encryption type
func (e EncryptionType) String() string {
	switch e {
	case EncryptionNone:
		return "none"
	case EncryptionAES256GCM:
		return "aes256gcm"
	case EncryptionChaCha20Poly1305:
		return "chacha20poly1305"
	case EncryptionWireGuard:
		return "wireguard"
	default:
		return "unknown"
	}
}

// AuthType defines the authentication type
type AuthType int

const (
	// AuthNone no authentication (not recommended)
	AuthNone AuthType = iota
	// AuthPSK pre-shared key authentication
	AuthPSK
	// AuthCertificate certificate-based authentication
	AuthCertificate
	// AuthToken token-based authentication
	AuthToken
	// AuthMutualTLS mutual TLS authentication
	AuthMutualTLS
)

// String returns the string representation of the auth type
func (a AuthType) String() string {
	switch a {
	case AuthNone:
		return "none"
	case AuthPSK:
		return "psk"
	case AuthCertificate:
		return "certificate"
	case AuthToken:
		return "token"
	case AuthMutualTLS:
		return "mtls"
	default:
		return "unknown"
	}
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	// Encryption settings
	EncryptionEnabled bool
	EncryptionType    EncryptionType

	// Authentication settings
	AuthEnabled bool
	AuthType    AuthType

	// Key rotation
	KeyRotationEnabled  bool
	KeyRotationInterval time.Duration

	// Certificate settings
	CertFile   string
	KeyFile    string
	CAFile     string
	VerifyPeer bool

	// PSK settings
	PreSharedKey string

	// Token settings
	TokenSecret      string
	TokenExpiration  time.Duration

	// Security policies
	RequireEncryption bool
	MinKeySize        int
	AllowedCiphers    []string

	// Rate limiting
	EnableRateLimit    bool
	MaxConnectionsPerIP int
	RateLimitWindow     time.Duration
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		EncryptionEnabled:   true,
		EncryptionType:      EncryptionChaCha20Poly1305,
		AuthEnabled:         true,
		AuthType:            AuthPSK,
		KeyRotationEnabled:  true,
		KeyRotationInterval: 24 * time.Hour,
		VerifyPeer:          true,
		TokenExpiration:     1 * time.Hour,
		RequireEncryption:   true,
		MinKeySize:          256,
		AllowedCiphers: []string{
			"AES256-GCM",
			"CHACHA20-POLY1305",
		},
		EnableRateLimit:     true,
		MaxConnectionsPerIP: 100,
		RateLimitWindow:     1 * time.Minute,
	}
}

// KeyPair represents a cryptographic key pair
type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ID         string
}

// NewKeyPair creates a new key pair
func NewKeyPair(id string, publicKey, privateKey []byte, validity time.Duration) *KeyPair {
	now := time.Now()
	return &KeyPair{
		ID:         id,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		CreatedAt:  now,
		ExpiresAt:  now.Add(validity),
	}
}

// IsExpired checks if the key pair is expired
func (kp *KeyPair) IsExpired() bool {
	return time.Now().After(kp.ExpiresAt)
}

// Certificate represents an X.509 certificate
type Certificate struct {
	ID           string
	Cert         *x509.Certificate
	RawCert      []byte
	PrivateKey   interface{}
	CreatedAt    time.Time
	ExpiresAt    time.Time
	Subject      string
	Issuer       string
	SerialNumber string
}

// NewCertificate creates a new certificate wrapper
func NewCertificate(id string, cert *x509.Certificate, rawCert []byte, privateKey interface{}) *Certificate {
	return &Certificate{
		ID:           id,
		Cert:         cert,
		RawCert:      rawCert,
		PrivateKey:   privateKey,
		CreatedAt:    cert.NotBefore,
		ExpiresAt:    cert.NotAfter,
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		SerialNumber: cert.SerialNumber.String(),
	}
}

// IsExpired checks if the certificate is expired
func (c *Certificate) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// IsValid checks if the certificate is currently valid
func (c *Certificate) IsValid() bool {
	now := time.Now()
	return now.After(c.CreatedAt) && now.Before(c.ExpiresAt)
}

// SessionKey represents an encryption session key
type SessionKey struct {
	ID        string
	Key       []byte
	Nonce     []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	PeerID    string
	Used      bool
	mu        sync.Mutex
}

// NewSessionKey creates a new session key
func NewSessionKey(id string, key, nonce []byte, peerID string, validity time.Duration) *SessionKey {
	now := time.Now()
	return &SessionKey{
		ID:        id,
		Key:       key,
		Nonce:     nonce,
		CreatedAt: now,
		ExpiresAt: now.Add(validity),
		PeerID:    peerID,
		Used:      false,
	}
}

// IsExpired checks if the session key is expired
func (sk *SessionKey) IsExpired() bool {
	return time.Now().After(sk.ExpiresAt)
}

// MarkUsed marks the session key as used
func (sk *SessionKey) MarkUsed() {
	sk.mu.Lock()
	defer sk.mu.Unlock()
	sk.Used = true
}

// Peer represents a trusted peer
type Peer struct {
	ID           string
	PublicKey    []byte
	Endpoint     string
	AllowedIPs   []string
	LastHandshake time.Time
	BytesSent    uint64
	BytesReceived uint64
	Trusted      bool
	CreatedAt    time.Time
	mu           sync.RWMutex
}

// NewPeer creates a new peer
func NewPeer(id string, publicKey []byte, endpoint string, allowedIPs []string) *Peer {
	return &Peer{
		ID:         id,
		PublicKey:  publicKey,
		Endpoint:   endpoint,
		AllowedIPs: allowedIPs,
		CreatedAt:  time.Now(),
		Trusted:    false,
	}
}

// UpdateHandshake updates the last handshake time
func (p *Peer) UpdateHandshake() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastHandshake = time.Now()
}

// UpdateTraffic updates traffic counters
func (p *Peer) UpdateTraffic(sent, received uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.BytesSent = sent
	p.BytesReceived = received
}

// SetTrusted marks the peer as trusted
func (p *Peer) SetTrusted(trusted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Trusted = trusted
}

// SecurityPolicy represents a security policy
type SecurityPolicy struct {
	ID          string
	Name        string
	Description string

	// Encryption requirements
	RequireEncryption bool
	AllowedCiphers    []EncryptionType

	// Authentication requirements
	RequireAuth    bool
	AllowedAuthTypes []AuthType

	// Access control
	AllowedPeers []string
	DeniedPeers  []string
	AllowedIPs   []string
	DeniedIPs    []string

	// Rate limiting
	MaxConnectionRate int
	MaxBandwidth      uint64

	// Timeouts
	HandshakeTimeout time.Duration
	SessionTimeout   time.Duration

	CreatedAt time.Time
	UpdatedAt time.Time
	Enabled   bool
}

// NewSecurityPolicy creates a new security policy
func NewSecurityPolicy(id, name, description string) *SecurityPolicy {
	now := time.Now()
	return &SecurityPolicy{
		ID:                id,
		Name:              name,
		Description:       description,
		RequireEncryption: true,
		AllowedCiphers: []EncryptionType{
			EncryptionAES256GCM,
			EncryptionChaCha20Poly1305,
		},
		RequireAuth: true,
		AllowedAuthTypes: []AuthType{
			AuthPSK,
			AuthCertificate,
			AuthMutualTLS,
		},
		MaxConnectionRate: 100,
		MaxBandwidth:      0, // unlimited
		HandshakeTimeout:  30 * time.Second,
		SessionTimeout:    1 * time.Hour,
		CreatedAt:         now,
		UpdatedAt:         now,
		Enabled:           true,
	}
}

// IsAllowedPeer checks if a peer is allowed
func (sp *SecurityPolicy) IsAllowedPeer(peerID string) bool {
	// Check denied list first
	for _, denied := range sp.DeniedPeers {
		if denied == peerID {
			return false
		}
	}

	// If allowed list is empty, allow all (except denied)
	if len(sp.AllowedPeers) == 0 {
		return true
	}

	// Check allowed list
	for _, allowed := range sp.AllowedPeers {
		if allowed == peerID {
			return true
		}
	}

	return false
}

// IsAllowedIP checks if an IP is allowed
func (sp *SecurityPolicy) IsAllowedIP(ip string) bool {
	// Check denied list first
	for _, denied := range sp.DeniedIPs {
		if denied == ip {
			return false
		}
	}

	// If allowed list is empty, allow all (except denied)
	if len(sp.AllowedIPs) == 0 {
		return true
	}

	// Check allowed list
	for _, allowed := range sp.AllowedIPs {
		if allowed == ip {
			return true
		}
	}

	return false
}

// EncryptedPacket represents an encrypted packet
type EncryptedPacket struct {
	Header    PacketHeader
	Payload   []byte
	Signature []byte
}

// PacketHeader represents packet metadata
type PacketHeader struct {
	Version       uint8
	EncryptionType EncryptionType
	Flags         uint8
	SequenceNum   uint64
	Timestamp     time.Time
	SenderID      string
	RecipientID   string
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID          string
	Type        SecurityEventType
	Severity    string // "info", "warning", "critical"
	Description string
	PeerID      string
	IP          string
	Timestamp   time.Time
	Details     map[string]interface{}
}

// SecurityEventType defines types of security events
type SecurityEventType int

const (
	// EventAuthSuccess successful authentication
	EventAuthSuccess SecurityEventType = iota
	// EventAuthFailure failed authentication
	EventAuthFailure
	// EventEncryptionError encryption/decryption error
	EventEncryptionError
	// EventCertificateExpired certificate expired
	EventCertificateExpired
	// EventRateLimitExceeded rate limit exceeded
	EventRateLimitExceeded
	// EventUnauthorizedAccess unauthorized access attempt
	EventUnauthorizedAccess
	// EventKeyRotation key rotation event
	EventKeyRotation
	// EventPeerConnected peer connected
	EventPeerConnected
	// EventPeerDisconnected peer disconnected
	EventPeerDisconnected
)

// String returns the string representation of the event type
func (e SecurityEventType) String() string {
	switch e {
	case EventAuthSuccess:
		return "auth_success"
	case EventAuthFailure:
		return "auth_failure"
	case EventEncryptionError:
		return "encryption_error"
	case EventCertificateExpired:
		return "certificate_expired"
	case EventRateLimitExceeded:
		return "rate_limit_exceeded"
	case EventUnauthorizedAccess:
		return "unauthorized_access"
	case EventKeyRotation:
		return "key_rotation"
	case EventPeerConnected:
		return "peer_connected"
	case EventPeerDisconnected:
		return "peer_disconnected"
	default:
		return "unknown"
	}
}

// NewSecurityEvent creates a new security event
func NewSecurityEvent(eventType SecurityEventType, severity, description, peerID, ip string) *SecurityEvent {
	return &SecurityEvent{
		ID:          generateID(),
		Type:        eventType,
		Severity:    severity,
		Description: description,
		PeerID:      peerID,
		IP:          ip,
		Timestamp:   time.Now(),
		Details:     make(map[string]interface{}),
	}
}

// generateID generates a unique ID
func generateID() string {
	return time.Now().Format("20060102150405") + "-" + string(rune(time.Now().UnixNano()%1000))
}

// TrustStore manages trusted peers and certificates
type TrustStore struct {
	peers        map[string]*Peer
	certificates map[string]*Certificate
	mu           sync.RWMutex
}

// NewTrustStore creates a new trust store
func NewTrustStore() *TrustStore {
	return &TrustStore{
		peers:        make(map[string]*Peer),
		certificates: make(map[string]*Certificate),
	}
}

// AddPeer adds a peer to the trust store
func (ts *TrustStore) AddPeer(peer *Peer) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.peers[peer.ID] = peer
}

// GetPeer retrieves a peer from the trust store
func (ts *TrustStore) GetPeer(id string) (*Peer, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	peer, exists := ts.peers[id]
	return peer, exists
}

// RemovePeer removes a peer from the trust store
func (ts *TrustStore) RemovePeer(id string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.peers, id)
}

// AddCertificate adds a certificate to the trust store
func (ts *TrustStore) AddCertificate(cert *Certificate) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.certificates[cert.ID] = cert
}

// GetCertificate retrieves a certificate from the trust store
func (ts *TrustStore) GetCertificate(id string) (*Certificate, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	cert, exists := ts.certificates[id]
	return cert, exists
}

// RemoveCertificate removes a certificate from the trust store
func (ts *TrustStore) RemoveCertificate(id string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.certificates, id)
}

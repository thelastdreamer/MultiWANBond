// Package security - Authentication implementation
package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrAuthFailed authentication failed
	ErrAuthFailed = errors.New("authentication failed")
	// ErrInvalidToken invalid token
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken expired token
	ErrExpiredToken = errors.New("expired token")
	// ErrInvalidCredentials invalid credentials
	ErrInvalidCredentials = errors.New("invalid credentials")
)

// Authenticator handles authentication
type Authenticator struct {
	config     *SecurityConfig
	trustStore *TrustStore
	tokens     map[string]*Token
	sessions   map[string]*Session
	mu         sync.RWMutex
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(config *SecurityConfig) *Authenticator {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &Authenticator{
		config:     config,
		trustStore: NewTrustStore(),
		tokens:     make(map[string]*Token),
		sessions:   make(map[string]*Session),
	}
}

// Authenticate authenticates a peer
func (a *Authenticator) Authenticate(peerID string, credentials interface{}) (*Session, error) {
	if !a.config.AuthEnabled {
		// Create unauthenticated session
		return a.createSession(peerID, AuthNone), nil
	}

	switch a.config.AuthType {
	case AuthPSK:
		return a.authenticatePSK(peerID, credentials)
	case AuthToken:
		return a.authenticateToken(peerID, credentials)
	case AuthCertificate, AuthMutualTLS:
		return a.authenticateCertificate(peerID, credentials)
	default:
		return nil, fmt.Errorf("unsupported auth type: %v", a.config.AuthType)
	}
}

// authenticatePSK authenticates using pre-shared key
func (a *Authenticator) authenticatePSK(peerID string, credentials interface{}) (*Session, error) {
	psk, ok := credentials.(string)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(psk), []byte(a.config.PreSharedKey)) != 1 {
		return nil, ErrAuthFailed
	}

	return a.createSession(peerID, AuthPSK), nil
}

// authenticateToken authenticates using token
func (a *Authenticator) authenticateToken(peerID string, credentials interface{}) (*Session, error) {
	tokenStr, ok := credentials.(string)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	// Verify token
	token, err := a.VerifyToken(tokenStr)
	if err != nil {
		return nil, err
	}

	// Check if token is for the correct peer
	if token.PeerID != peerID {
		return nil, ErrAuthFailed
	}

	return a.createSession(peerID, AuthToken), nil
}

// authenticateCertificate authenticates using certificate
func (a *Authenticator) authenticateCertificate(peerID string, credentials interface{}) (*Session, error) {
	cert, ok := credentials.(*Certificate)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	// Verify certificate is valid
	if !cert.IsValid() {
		return nil, errors.New("invalid or expired certificate")
	}

	// Check if certificate is in trust store
	_, exists := a.trustStore.GetCertificate(cert.ID)
	if !exists {
		return nil, errors.New("untrusted certificate")
	}

	return a.createSession(peerID, AuthCertificate), nil
}

// createSession creates a new authenticated session
func (a *Authenticator) createSession(peerID string, authType AuthType) *Session {
	session := NewSession(peerID, authType, a.config.TokenExpiration)

	a.mu.Lock()
	a.sessions[session.ID] = session
	a.mu.Unlock()

	return session
}

// GetSession retrieves a session
func (a *Authenticator) GetSession(sessionID string) (*Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	session, exists := a.sessions[sessionID]
	if !exists {
		return nil, errors.New("session not found")
	}

	if session.IsExpired() {
		return nil, errors.New("session expired")
	}

	return session, nil
}

// RevokeSession revokes a session
func (a *Authenticator) RevokeSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, sessionID)
}

// GenerateToken generates a new authentication token
func (a *Authenticator) GenerateToken(peerID string, validity time.Duration) (string, error) {
	token := &Token{
		ID:        generateID(),
		PeerID:    peerID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(validity),
	}

	// Create token payload
	payload := map[string]interface{}{
		"id":         token.ID,
		"peer_id":    token.PeerID,
		"issued_at":  token.IssuedAt.Unix(),
		"expires_at": token.ExpiresAt.Unix(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token: %w", err)
	}

	// Encode payload
	encodedPayload := base64.URLEncoding.EncodeToString(payloadJSON)

	// Generate signature
	signature := a.signToken(encodedPayload)

	// Combine payload and signature
	tokenStr := encodedPayload + "." + signature

	// Store token
	a.mu.Lock()
	a.tokens[token.ID] = token
	a.mu.Unlock()

	return tokenStr, nil
}

// VerifyToken verifies an authentication token
func (a *Authenticator) VerifyToken(tokenStr string) (*Token, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, ErrInvalidToken
	}

	encodedPayload := parts[0]
	signature := parts[1]

	// Verify signature
	expectedSignature := a.signToken(encodedPayload)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSignature)) != 1 {
		return nil, ErrInvalidToken
	}

	// Decode payload
	payloadJSON, err := base64.URLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, ErrInvalidToken
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return nil, ErrInvalidToken
	}

	// Extract token data
	tokenID, _ := payload["id"].(string)
	peerID, _ := payload["peer_id"].(string)
	issuedAt := time.Unix(int64(payload["issued_at"].(float64)), 0)
	expiresAt := time.Unix(int64(payload["expires_at"].(float64)), 0)

	token := &Token{
		ID:        tokenID,
		PeerID:    peerID,
		IssuedAt:  issuedAt,
		ExpiresAt: expiresAt,
	}

	// Check expiration
	if token.IsExpired() {
		return nil, ErrExpiredToken
	}

	return token, nil
}

// signToken signs a token payload
func (a *Authenticator) signToken(payload string) string {
	h := hmac.New(sha256.New, []byte(a.config.TokenSecret))
	h.Write([]byte(payload))
	signature := h.Sum(nil)
	return base64.URLEncoding.EncodeToString(signature)
}

// CleanupExpired removes expired sessions and tokens
func (a *Authenticator) CleanupExpired() int {
	a.mu.Lock()
	defer a.mu.Unlock()

	removed := 0

	// Clean sessions
	for id, session := range a.sessions {
		if session.IsExpired() {
			delete(a.sessions, id)
			removed++
		}
	}

	// Clean tokens
	for id, token := range a.tokens {
		if token.IsExpired() {
			delete(a.tokens, id)
			removed++
		}
	}

	return removed
}

// GetTrustStore returns the trust store
func (a *Authenticator) GetTrustStore() *TrustStore {
	return a.trustStore
}

// Token represents an authentication token
type Token struct {
	ID        string
	PeerID    string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// IsExpired checks if the token is expired
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// Session represents an authenticated session
type Session struct {
	ID        string
	PeerID    string
	AuthType  AuthType
	CreatedAt time.Time
	ExpiresAt time.Time
	LastAccess time.Time
	Attributes map[string]interface{}
	mu        sync.RWMutex
}

// NewSession creates a new session
func NewSession(peerID string, authType AuthType, validity time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:         generateID(),
		PeerID:     peerID,
		AuthType:   authType,
		CreatedAt:  now,
		ExpiresAt:  now.Add(validity),
		LastAccess: now,
		Attributes: make(map[string]interface{}),
	}
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// UpdateAccess updates the last access time
func (s *Session) UpdateAccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LastAccess = time.Now()
}

// SetAttribute sets a session attribute
func (s *Session) SetAttribute(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes[key] = value
}

// GetAttribute gets a session attribute
func (s *Session) GetAttribute(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.Attributes[key]
	return value, exists
}

// RateLimiter implements rate limiting
type RateLimiter struct {
	limits    map[string]*RateLimit
	mu        sync.RWMutex
	window    time.Duration
	maxRate   int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(window time.Duration, maxRate int) *RateLimiter {
	return &RateLimiter{
		limits:  make(map[string]*RateLimit),
		window:  window,
		maxRate: maxRate,
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	limit, exists := rl.limits[identifier]
	if !exists {
		limit = &RateLimit{
			WindowStart: now,
			Count:       0,
		}
		rl.limits[identifier] = limit
	}

	// Reset window if expired
	if now.Sub(limit.WindowStart) > rl.window {
		limit.WindowStart = now
		limit.Count = 0
	}

	// Check rate limit
	if limit.Count >= rl.maxRate {
		return false
	}

	limit.Count++
	return true
}

// Reset resets the rate limit for an identifier
func (rl *RateLimiter) Reset(identifier string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limits, identifier)
}

// RateLimit tracks rate limit state
type RateLimit struct {
	WindowStart time.Time
	Count       int
}

// AuthorizationChecker checks if a peer is authorized for an action
type AuthorizationChecker struct {
	policies map[string]*SecurityPolicy
	mu       sync.RWMutex
}

// NewAuthorizationChecker creates a new authorization checker
func NewAuthorizationChecker() *AuthorizationChecker {
	return &AuthorizationChecker{
		policies: make(map[string]*SecurityPolicy),
	}
}

// AddPolicy adds a security policy
func (ac *AuthorizationChecker) AddPolicy(policy *SecurityPolicy) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.policies[policy.ID] = policy
}

// CheckAccess checks if a peer is authorized
func (ac *AuthorizationChecker) CheckAccess(peerID, ip string, policyID string) bool {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	policy, exists := ac.policies[policyID]
	if !exists || !policy.Enabled {
		// No policy or disabled - allow by default
		return true
	}

	// Check peer access
	if !policy.IsAllowedPeer(peerID) {
		return false
	}

	// Check IP access
	if !policy.IsAllowedIP(ip) {
		return false
	}

	return true
}

// GetPolicy retrieves a security policy
func (ac *AuthorizationChecker) GetPolicy(policyID string) (*SecurityPolicy, bool) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	policy, exists := ac.policies[policyID]
	return policy, exists
}

// RemovePolicy removes a security policy
func (ac *AuthorizationChecker) RemovePolicy(policyID string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	delete(ac.policies, policyID)
}

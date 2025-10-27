// Package security - Encryption implementation
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

var (
	// ErrInvalidKeySize invalid key size
	ErrInvalidKeySize = errors.New("invalid key size")
	// ErrEncryptionFailed encryption failed
	ErrEncryptionFailed = errors.New("encryption failed")
	// ErrDecryptionFailed decryption failed
	ErrDecryptionFailed = errors.New("decryption failed")
	// ErrInvalidNonce invalid nonce
	ErrInvalidNonce = errors.New("invalid nonce")
	// ErrExpiredKey expired key
	ErrExpiredKey = errors.New("expired key")
)

// Encryptor handles packet encryption/decryption
type Encryptor struct {
	config    *SecurityConfig
	keyStore  *KeyStore
	sequenceNum uint64
	mu        sync.Mutex
}

// NewEncryptor creates a new encryptor
func NewEncryptor(config *SecurityConfig) *Encryptor {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &Encryptor{
		config:   config,
		keyStore: NewKeyStore(),
	}
}

// Encrypt encrypts data using the configured encryption type
func (e *Encryptor) Encrypt(data []byte, recipientID string) (*EncryptedPacket, error) {
	if !e.config.EncryptionEnabled {
		// No encryption, return plaintext wrapped in packet
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

	// Get or generate session key
	sessionKey, err := e.keyStore.GetOrCreateSessionKey(recipientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session key: %w", err)
	}

	// Increment sequence number
	e.mu.Lock()
	e.sequenceNum++
	seqNum := e.sequenceNum
	e.mu.Unlock()

	// Encrypt based on type
	var encrypted []byte
	var encType EncryptionType

	switch e.config.EncryptionType {
	case EncryptionAES256GCM:
		encrypted, err = e.encryptAESGCM(data, sessionKey.Key, sessionKey.Nonce, seqNum)
		encType = EncryptionAES256GCM
	case EncryptionChaCha20Poly1305:
		encrypted, err = e.encryptChaCha20(data, sessionKey.Key, sessionKey.Nonce, seqNum)
		encType = EncryptionChaCha20Poly1305
	default:
		return nil, fmt.Errorf("unsupported encryption type: %v", e.config.EncryptionType)
	}

	if err != nil {
		return nil, err
	}

	// Create packet
	packet := &EncryptedPacket{
		Header: PacketHeader{
			Version:        1,
			EncryptionType: encType,
			SequenceNum:    seqNum,
			Timestamp:      time.Now(),
			RecipientID:    recipientID,
		},
		Payload: encrypted,
	}

	return packet, nil
}

// Decrypt decrypts an encrypted packet
func (e *Encryptor) Decrypt(packet *EncryptedPacket) ([]byte, error) {
	if packet.Header.EncryptionType == EncryptionNone {
		return packet.Payload, nil
	}

	// Get session key
	sessionKey, err := e.keyStore.GetSessionKey(packet.Header.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session key: %w", err)
	}

	// Decrypt based on type
	var decrypted []byte

	switch packet.Header.EncryptionType {
	case EncryptionAES256GCM:
		decrypted, err = e.decryptAESGCM(packet.Payload, sessionKey.Key, sessionKey.Nonce, packet.Header.SequenceNum)
	case EncryptionChaCha20Poly1305:
		decrypted, err = e.decryptChaCha20(packet.Payload, sessionKey.Key, sessionKey.Nonce, packet.Header.SequenceNum)
	default:
		return nil, fmt.Errorf("unsupported encryption type: %v", packet.Header.EncryptionType)
	}

	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

// encryptAESGCM encrypts data using AES-256-GCM
func (e *Encryptor) encryptAESGCM(data, key, baseNonce []byte, seqNum uint64) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create unique nonce by combining base nonce with sequence number
	nonce := make([]byte, gcm.NonceSize())
	copy(nonce, baseNonce[:gcm.NonceSize()])
	binary.BigEndian.PutUint64(nonce[gcm.NonceSize()-8:], seqNum)

	// Encrypt
	encrypted := gcm.Seal(nil, nonce, data, nil)
	return encrypted, nil
}

// decryptAESGCM decrypts data using AES-256-GCM
func (e *Encryptor) decryptAESGCM(encrypted, key, baseNonce []byte, seqNum uint64) ([]byte, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Recreate nonce
	nonce := make([]byte, gcm.NonceSize())
	copy(nonce, baseNonce[:gcm.NonceSize()])
	binary.BigEndian.PutUint64(nonce[gcm.NonceSize()-8:], seqNum)

	// Decrypt
	decrypted, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return decrypted, nil
}

// encryptChaCha20 encrypts data using ChaCha20-Poly1305
func (e *Encryptor) encryptChaCha20(data, key, baseNonce []byte, seqNum uint64) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, ErrInvalidKeySize
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20: %w", err)
	}

	// Create unique nonce
	nonce := make([]byte, aead.NonceSize())
	copy(nonce, baseNonce[:aead.NonceSize()])
	binary.BigEndian.PutUint64(nonce[aead.NonceSize()-8:], seqNum)

	// Encrypt
	encrypted := aead.Seal(nil, nonce, data, nil)
	return encrypted, nil
}

// decryptChaCha20 decrypts data using ChaCha20-Poly1305
func (e *Encryptor) decryptChaCha20(encrypted, key, baseNonce []byte, seqNum uint64) ([]byte, error) {
	if len(key) != chacha20poly1305.KeySize {
		return nil, ErrInvalidKeySize
	}

	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create ChaCha20: %w", err)
	}

	// Recreate nonce
	nonce := make([]byte, aead.NonceSize())
	copy(nonce, baseNonce[:aead.NonceSize()])
	binary.BigEndian.PutUint64(nonce[aead.NonceSize()-8:], seqNum)

	// Decrypt
	decrypted, err := aead.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return decrypted, nil
}

// KeyStore manages encryption keys
type KeyStore struct {
	sessionKeys map[string]*SessionKey
	keyPairs    map[string]*KeyPair
	mu          sync.RWMutex
}

// NewKeyStore creates a new key store
func NewKeyStore() *KeyStore {
	return &KeyStore{
		sessionKeys: make(map[string]*SessionKey),
		keyPairs:    make(map[string]*KeyPair),
	}
}

// GetOrCreateSessionKey gets or creates a session key for a peer
func (ks *KeyStore) GetOrCreateSessionKey(peerID string) (*SessionKey, error) {
	ks.mu.RLock()
	key, exists := ks.sessionKeys[peerID]
	ks.mu.RUnlock()

	if exists && !key.IsExpired() {
		return key, nil
	}

	// Create new session key
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Double-check after acquiring write lock
	if key, exists := ks.sessionKeys[peerID]; exists && !key.IsExpired() {
		return key, nil
	}

	// Generate new key and nonce
	keyBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	nonceBytes := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	sessionKey := NewSessionKey(generateID(), keyBytes, nonceBytes, peerID, 24*time.Hour)
	ks.sessionKeys[peerID] = sessionKey

	return sessionKey, nil
}

// GetSessionKey gets a session key for a peer
func (ks *KeyStore) GetSessionKey(peerID string) (*SessionKey, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	key, exists := ks.sessionKeys[peerID]
	if !exists {
		return nil, errors.New("session key not found")
	}

	if key.IsExpired() {
		return nil, ErrExpiredKey
	}

	return key, nil
}

// AddKeyPair adds a key pair to the store
func (ks *KeyStore) AddKeyPair(keyPair *KeyPair) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.keyPairs[keyPair.ID] = keyPair
}

// GetKeyPair gets a key pair from the store
func (ks *KeyStore) GetKeyPair(id string) (*KeyPair, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	keyPair, exists := ks.keyPairs[id]
	if !exists {
		return nil, errors.New("key pair not found")
	}

	if keyPair.IsExpired() {
		return nil, ErrExpiredKey
	}

	return keyPair, nil
}

// RotateKeys rotates all session keys
func (ks *KeyStore) RotateKeys() int {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	rotated := 0
	newKeys := make(map[string]*SessionKey)

	for peerID := range ks.sessionKeys {
		// Generate new key
		keyBytes := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, keyBytes); err != nil {
			continue
		}

		nonceBytes := make([]byte, 12)
		if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
			continue
		}

		sessionKey := NewSessionKey(generateID(), keyBytes, nonceBytes, peerID, 24*time.Hour)
		newKeys[peerID] = sessionKey
		rotated++
	}

	ks.sessionKeys = newKeys
	return rotated
}

// CleanupExpired removes expired keys
func (ks *KeyStore) CleanupExpired() int {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	removed := 0

	// Clean session keys
	for peerID, key := range ks.sessionKeys {
		if key.IsExpired() {
			delete(ks.sessionKeys, peerID)
			removed++
		}
	}

	// Clean key pairs
	for id, keyPair := range ks.keyPairs {
		if keyPair.IsExpired() {
			delete(ks.keyPairs, id)
			removed++
		}
	}

	return removed
}

// GenerateKey generates a new encryption key
func GenerateKey(size int) ([]byte, error) {
	if size != 16 && size != 24 && size != 32 {
		return nil, ErrInvalidKeySize
	}

	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	return key, nil
}

// DeriveKey derives a key from a password using SHA-256
func DeriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// GenerateNonce generates a random nonce
func GenerateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

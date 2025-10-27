// Package main demonstrates Security functionality
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/security"
)

func main() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("MultiWANBond - Security & Encryption Demo")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	testResults := make(map[string]bool)

	// Test 1: Security configuration
	fmt.Println("Test 1: Security Configuration")
	fmt.Println(strings.Repeat("-", 80))

	config := security.DefaultSecurityConfig()
	fmt.Printf("Encryption Enabled: %v\n", config.EncryptionEnabled)
	fmt.Printf("Encryption Type: %s\n", config.EncryptionType.String())
	fmt.Printf("Authentication Enabled: %v\n", config.AuthEnabled)
	fmt.Printf("Authentication Type: %s\n", config.AuthType.String())
	fmt.Printf("Key Rotation Enabled: %v\n", config.KeyRotationEnabled)
	fmt.Printf("Key Rotation Interval: %v\n", config.KeyRotationInterval)
	fmt.Printf("Min Key Size: %d bits\n", config.MinKeySize)

	testResults["Security Configuration"] = config.EncryptionEnabled && config.AuthEnabled
	fmt.Println()

	// Test 2: Security manager creation
	fmt.Println("Test 2: Security Manager Creation")
	fmt.Println(strings.Repeat("-", 80))

	manager := security.NewManager(config)
	if manager != nil {
		fmt.Println("Security manager created successfully")
		testResults["Manager Creation"] = true
	} else {
		fmt.Println("Failed to create security manager")
		testResults["Manager Creation"] = false
	}

	// Start manager
	if err := manager.Start(); err != nil {
		fmt.Printf("Failed to start manager: %v\n", err)
		testResults["Manager Creation"] = false
	} else {
		fmt.Println("Security manager started")
	}
	fmt.Println()

	// Test 3: Encryption/Decryption (AES-256-GCM)
	fmt.Println("Test 3: AES-256-GCM Encryption")
	fmt.Println(strings.Repeat("-", 80))

	config.EncryptionType = security.EncryptionAES256GCM
	encryptor := security.NewEncryptor(config)

	testData := []byte("This is a test message for encryption!")
	fmt.Printf("Original data: %s\n", string(testData))
	fmt.Printf("Data length: %d bytes\n", len(testData))

	encrypted, err := encryptor.Encrypt(testData, "peer1")
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		testResults["AES-256-GCM Encryption"] = false
	} else {
		fmt.Printf("Encrypted successfully\n")
		fmt.Printf("Encrypted length: %d bytes\n", len(encrypted.Payload))
		fmt.Printf("Encryption type: %s\n", encrypted.Header.EncryptionType.String())
		fmt.Printf("Sequence number: %d\n", encrypted.Header.SequenceNum)

		// Decrypt
		decrypted, err := encryptor.Decrypt(encrypted)
		if err != nil {
			fmt.Printf("Decryption failed: %v\n", err)
			testResults["AES-256-GCM Encryption"] = false
		} else {
			fmt.Printf("Decrypted successfully\n")
			fmt.Printf("Decrypted data: %s\n", string(decrypted))

			if string(decrypted) == string(testData) {
				fmt.Println("✓ Decrypted data matches original")
				testResults["AES-256-GCM Encryption"] = true
			} else {
				fmt.Println("✗ Decrypted data does not match")
				testResults["AES-256-GCM Encryption"] = false
			}
		}
	}
	fmt.Println()

	// Test 4: ChaCha20-Poly1305 encryption
	fmt.Println("Test 4: ChaCha20-Poly1305 Encryption")
	fmt.Println(strings.Repeat("-", 80))

	config.EncryptionType = security.EncryptionChaCha20Poly1305
	encryptor2 := security.NewEncryptor(config)

	testData2 := []byte("Testing ChaCha20-Poly1305 encryption algorithm!")
	fmt.Printf("Original data: %s\n", string(testData2))

	encrypted2, err := encryptor2.Encrypt(testData2, "peer2")
	if err != nil {
		fmt.Printf("Encryption failed: %v\n", err)
		testResults["ChaCha20 Encryption"] = false
	} else {
		fmt.Printf("Encrypted successfully\n")
		fmt.Printf("Encryption type: %s\n", encrypted2.Header.EncryptionType.String())

		decrypted2, err := encryptor2.Decrypt(encrypted2)
		if err != nil {
			fmt.Printf("Decryption failed: %v\n", err)
			testResults["ChaCha20 Encryption"] = false
		} else {
			if string(decrypted2) == string(testData2) {
				fmt.Println("✓ ChaCha20 encryption/decryption successful")
				testResults["ChaCha20 Encryption"] = true
			} else {
				fmt.Println("✗ ChaCha20 decryption failed")
				testResults["ChaCha20 Encryption"] = false
			}
		}
	}
	fmt.Println()

	// Test 5: PSK Authentication
	fmt.Println("Test 5: Pre-Shared Key Authentication")
	fmt.Println(strings.Repeat("-", 80))

	config.PreSharedKey = "super-secret-psk-key-12345"
	authenticator := security.NewAuthenticator(config)

	// Correct PSK
	session, err := authenticator.Authenticate("peer1", config.PreSharedKey)
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		testResults["PSK Authentication"] = false
	} else {
		fmt.Printf("✓ Authentication successful\n")
		fmt.Printf("Session ID: %s\n", session.ID)
		fmt.Printf("Peer ID: %s\n", session.PeerID)
		fmt.Printf("Auth Type: %s\n", session.AuthType.String())
		fmt.Printf("Expires: %s\n", session.ExpiresAt.Format("15:04:05"))
		testResults["PSK Authentication"] = true
	}

	// Wrong PSK
	_, err = authenticator.Authenticate("peer2", "wrong-psk")
	if err != nil {
		fmt.Printf("✓ Wrong PSK correctly rejected\n")
	} else {
		fmt.Println("✗ Wrong PSK incorrectly accepted")
		testResults["PSK Authentication"] = false
	}
	fmt.Println()

	// Test 6: Token authentication
	fmt.Println("Test 6: Token-Based Authentication")
	fmt.Println(strings.Repeat("-", 80))

	config.TokenSecret = "token-secret-key-67890"
	config.AuthType = security.AuthToken
	authenticator2 := security.NewAuthenticator(config)

	// Generate token
	token, err := authenticator2.GenerateToken("peer3", 1*time.Hour)
	if err != nil {
		fmt.Printf("Token generation failed: %v\n", err)
		testResults["Token Authentication"] = false
	} else {
		fmt.Printf("Token generated: %s...\n", token[:50])

		// Authenticate with token
		session2, err := authenticator2.Authenticate("peer3", token)
		if err != nil {
			fmt.Printf("Token authentication failed: %v\n", err)
			testResults["Token Authentication"] = false
		} else {
			fmt.Printf("✓ Token authentication successful\n")
			fmt.Printf("Session ID: %s\n", session2.ID)
			testResults["Token Authentication"] = true
		}

		// Try invalid token
		_, err = authenticator2.Authenticate("peer3", "invalid.token.here")
		if err != nil {
			fmt.Println("✓ Invalid token correctly rejected")
		} else {
			fmt.Println("✗ Invalid token incorrectly accepted")
			testResults["Token Authentication"] = false
		}
	}
	fmt.Println()

	// Test 7: Security policies
	fmt.Println("Test 7: Security Policies")
	fmt.Println(strings.Repeat("-", 80))

	policy := security.NewSecurityPolicy("policy1", "Test Policy", "Test security policy")
	policy.AllowedPeers = []string{"peer1", "peer2"}
	policy.DeniedPeers = []string{"peer3"}
	policy.AllowedIPs = []string{"192.168.1.10", "192.168.1.20"}
	policy.RequireEncryption = true
	policy.RequireAuth = true

	fmt.Printf("Policy ID: %s\n", policy.ID)
	fmt.Printf("Policy Name: %s\n", policy.Name)
	fmt.Printf("Require Encryption: %v\n", policy.RequireEncryption)
	fmt.Printf("Require Auth: %v\n", policy.RequireAuth)
	fmt.Printf("Allowed Peers: %v\n", policy.AllowedPeers)
	fmt.Printf("Denied Peers: %v\n", policy.DeniedPeers)

	// Test policy checks
	tests := []struct {
		peerID   string
		expected bool
	}{
		{"peer1", true},
		{"peer2", true},
		{"peer3", false},
		{"peer4", false},
	}

	allPassed := true
	for _, test := range tests {
		result := policy.IsAllowedPeer(test.peerID)
		status := "✓"
		if result != test.expected {
			status = "✗"
			allPassed = false
		}
		fmt.Printf("%s Peer %s: allowed=%v (expected %v)\n", status, test.peerID, result, test.expected)
	}

	testResults["Security Policies"] = allPassed
	fmt.Println()

	// Test 8: Rate limiting
	fmt.Println("Test 8: Rate Limiting")
	fmt.Println(strings.Repeat("-", 80))

	rateLimiter := security.NewRateLimiter(1*time.Second, 5)
	clientIP := "192.168.1.100"

	fmt.Printf("Rate limit: 5 requests per second\n")
	fmt.Printf("Testing with client IP: %s\n", clientIP)

	allowedCount := 0
	deniedCount := 0

	// Try 10 requests
	for i := 1; i <= 10; i++ {
		if rateLimiter.Allow(clientIP) {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	fmt.Printf("Allowed requests: %d\n", allowedCount)
	fmt.Printf("Denied requests: %d\n", deniedCount)

	if allowedCount == 5 && deniedCount == 5 {
		fmt.Println("✓ Rate limiting working correctly")
		testResults["Rate Limiting"] = true
	} else {
		fmt.Println("✗ Rate limiting not working as expected")
		testResults["Rate Limiting"] = false
	}
	fmt.Println()

	// Test 9: Peer management
	fmt.Println("Test 9: Peer Management")
	fmt.Println(strings.Repeat("-", 80))

	peer1 := security.NewPeer("peer1", []byte("publickey1"), "192.168.1.10:8080", []string{"10.0.0.0/24"})
	peer1.SetTrusted(true)
	peer1.UpdateTraffic(1000000, 2000000)

	manager.AddPeer(peer1)

	retrievedPeer, exists := manager.GetPeer("peer1")
	if exists {
		fmt.Printf("✓ Peer retrieved successfully\n")
		fmt.Printf("Peer ID: %s\n", retrievedPeer.ID)
		fmt.Printf("Endpoint: %s\n", retrievedPeer.Endpoint)
		fmt.Printf("Trusted: %v\n", retrievedPeer.Trusted)
		fmt.Printf("Bytes sent: %d\n", retrievedPeer.BytesSent)
		fmt.Printf("Bytes received: %d\n", retrievedPeer.BytesReceived)
		testResults["Peer Management"] = true
	} else {
		fmt.Println("✗ Failed to retrieve peer")
		testResults["Peer Management"] = false
	}
	fmt.Println()

	// Test 10: Security events
	fmt.Println("Test 10: Security Events")
	fmt.Println(strings.Repeat("-", 80))

	// Wait a bit for background events
	time.Sleep(200 * time.Millisecond)

	events := manager.GetEvents()
	fmt.Printf("Total security events: %d\n", len(events))

	if len(events) > 0 {
		fmt.Println("Recent events:")
		recentEvents := manager.GetRecentEvents(5)
		for i, event := range recentEvents {
			fmt.Printf("  %d. [%s] %s: %s (peer: %s)\n",
				i+1, event.Severity, event.Type.String(), event.Description, event.PeerID)
		}
	}

	stats := manager.GetStats()
	fmt.Printf("\nSecurity Statistics:\n")
	fmt.Printf("  Encryption: %s (enabled: %v)\n", stats.EncryptionType, stats.EncryptionEnabled)
	fmt.Printf("  Authentication: %s (enabled: %v)\n", stats.AuthType, stats.AuthEnabled)
	fmt.Printf("  Auth successes: %d\n", stats.AuthSuccessCount)
	fmt.Printf("  Auth failures: %d\n", stats.AuthFailureCount)
	fmt.Printf("  Encryption errors: %d\n", stats.EncryptionErrorCount)
	fmt.Printf("  Unauthorized access: %d\n", stats.UnauthorizedAccessCount)

	testResults["Security Events"] = len(events) > 0
	fmt.Println()

	// Stop manager
	if err := manager.Stop(); err != nil {
		fmt.Printf("Error stopping manager: %v\n", err)
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Test Summary")
	fmt.Println(strings.Repeat("=", 80))

	passed := 0
	total := len(testResults)

	for test, result := range testResults {
		status := "❌ FAIL"
		if result {
			status = "✓ PASS"
			passed++
		}
		fmt.Printf("%s: %s\n", status, test)
	}

	fmt.Println()
	fmt.Printf("Tests Passed: %d/%d (%.0f%%)\n", passed, total, float64(passed)/float64(total)*100)
	fmt.Println(strings.Repeat("=", 80))

	// Print feature summary
	fmt.Println()
	fmt.Println("Phase 9 Features Implemented:")
	fmt.Println("  - AES-256-GCM encryption")
	fmt.Println("  - ChaCha20-Poly1305 encryption")
	fmt.Println("  - Pre-shared key (PSK) authentication")
	fmt.Println("  - Token-based authentication")
	fmt.Println("  - Certificate-based authentication support")
	fmt.Println("  - Security policy enforcement")
	fmt.Println("  - Rate limiting")
	fmt.Println("  - Peer trust management")
	fmt.Println("  - Session management")
	fmt.Println("  - Automatic key rotation")
	fmt.Println("  - Security event logging")
	fmt.Println("  - Authorization checking")
	fmt.Println()
	fmt.Println("Ready for secure communications!")
}

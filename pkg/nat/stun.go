package nat

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// STUN message types (RFC 5389)
const (
	stunBindingRequest         uint16 = 0x0001
	stunBindingResponse        uint16 = 0x0101
	stunBindingErrorResponse   uint16 = 0x0111
)

// STUN attribute types
const (
	stunAttrMappedAddress     uint16 = 0x0001
	stunAttrChangeRequest     uint16 = 0x0003
	stunAttrSourceAddress     uint16 = 0x0004
	stunAttrChangedAddress    uint16 = 0x0005
	stunAttrXorMappedAddress  uint16 = 0x0020
	stunAttrXorMappedAddress2 uint16 = 0x8020
)

// STUN magic cookie (RFC 5389)
const stunMagicCookie uint32 = 0x2112A442

// STUNClient handles STUN protocol operations
type STUNClient struct {
	config *STUNConfig
	conn   *net.UDPConn
	mu     sync.RWMutex

	// Current NAT mapping
	mapping *NATMapping

	// Stats
	requests  uint64
	successes uint64
	failures  uint64
}

// NewSTUNClient creates a new STUN client
func NewSTUNClient(config *STUNConfig) (*STUNClient, error) {
	if config == nil {
		config = DefaultSTUNConfig()
	}

	// Bind to local port
	localAddr := &net.UDPAddr{
		IP:   net.IPv4zero,
		Port: config.LocalPort,
	}

	conn, err := net.ListenUDP("udp4", localAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP: %w", err)
	}

	return &STUNClient{
		config: config,
		conn:   conn,
	}, nil
}

// Close closes the STUN client
func (c *STUNClient) Close() error {
	return c.conn.Close()
}

// GetLocalAddr returns the local address
func (c *STUNClient) GetLocalAddr() *net.UDPAddr {
	return c.conn.LocalAddr().(*net.UDPAddr)
}

// DiscoverNATMapping discovers the public IP:port via STUN
func (c *STUNClient) DiscoverNATMapping() (*NATMapping, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Test 1: Basic binding request to primary server
	publicAddr, err := c.sendBindingRequest(c.config.PrimaryServer, false, false)
	if err != nil {
		c.failures++
		return nil, fmt.Errorf("test 1 failed: %w", err)
	}

	c.requests++
	c.successes++

	// Create initial mapping
	mapping := &NATMapping{
		LocalAddr:   c.conn.LocalAddr().(*net.UDPAddr),
		PublicAddr:  publicAddr,
		Discovered:  time.Now(),
		LastRefresh: time.Now(),
		TTL:         30 * time.Second,
		STUNServer:  c.config.PrimaryServer,
	}

	// Check if we're behind NAT
	if publicAddr.IP.Equal(mapping.LocalAddr.IP) && publicAddr.Port == mapping.LocalAddr.Port {
		mapping.MappingType = NATTypeOpen
		c.mapping = mapping
		return mapping, nil
	}

	// Test 2: Request with change IP and port to detect NAT type
	_, err = c.sendBindingRequest(c.config.PrimaryServer, true, true)
	if err == nil {
		// Response from different IP:port means Full Cone NAT
		mapping.MappingType = NATTypeFullCone
		c.mapping = mapping
		return mapping, nil
	}

	// Test 3: Request with change port only
	_, err = c.sendBindingRequest(c.config.PrimaryServer, false, true)
	if err == nil {
		// Response from different port means Restricted Cone
		mapping.MappingType = NATTypeRestrictedCone
		c.mapping = mapping
		return mapping, nil
	} else {
		// No response means Port-Restricted Cone
		mapping.MappingType = NATTypePortRestrictedCone
	}

	// Test 4: Request to secondary server to check for symmetric NAT
	if c.config.SecondaryServer != "" {
		publicAddr2, err := c.sendBindingRequest(c.config.SecondaryServer, false, false)
		if err == nil {
			// If we get different public port for different server, it's Symmetric NAT
			if publicAddr2.Port != publicAddr.Port {
				mapping.MappingType = NATTypeSymmetric
			}
		}
	}

	c.mapping = mapping
	return mapping, nil
}

// RefreshMapping refreshes the NAT mapping by sending keep-alive
func (c *STUNClient) RefreshMapping() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.mapping == nil {
		return fmt.Errorf("no mapping to refresh")
	}

	publicAddr, err := c.sendBindingRequest(c.config.PrimaryServer, false, false)
	if err != nil {
		c.failures++
		return err
	}

	c.requests++
	c.successes++

	// Update mapping
	c.mapping.PublicAddr = publicAddr
	c.mapping.LastRefresh = time.Now()

	return nil
}

// GetMapping returns the current NAT mapping
func (c *STUNClient) GetMapping() *NATMapping {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mapping
}

// sendBindingRequest sends a STUN binding request
func (c *STUNClient) sendBindingRequest(serverAddr string, changeIP, changePort bool) (*net.UDPAddr, error) {
	// Parse server address
	addr, err := net.ResolveUDPAddr("udp4", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve STUN server: %w", err)
	}

	// Generate transaction ID
	transactionID := make([]byte, 12)
	rand.Read(transactionID)

	// Build STUN message
	message := c.buildBindingRequest(transactionID, changeIP, changePort)

	// Set deadline
	deadline := time.Now().Add(c.config.Timeout)
	c.conn.SetReadDeadline(deadline)

	// Send request
	_, err = c.conn.WriteToUDP(message, addr)
	if err != nil {
		return nil, fmt.Errorf("failed to send STUN request: %w", err)
	}

	// Wait for response
	buffer := make([]byte, 1500)
	n, _, err := c.conn.ReadFromUDP(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to receive STUN response: %w", err)
	}

	// Parse response
	publicAddr, err := c.parseBindingResponse(buffer[:n], transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse STUN response: %w", err)
	}

	return publicAddr, nil
}

// buildBindingRequest builds a STUN binding request message
func (c *STUNClient) buildBindingRequest(transactionID []byte, changeIP, changePort bool) []byte {
	message := make([]byte, 20) // Header only for basic request

	// Message Type (2 bytes)
	binary.BigEndian.PutUint16(message[0:2], stunBindingRequest)

	// Message Length (2 bytes) - will update if we add attributes
	messageLength := 0

	// Magic Cookie (4 bytes)
	binary.BigEndian.PutUint32(message[4:8], stunMagicCookie)

	// Transaction ID (12 bytes)
	copy(message[8:20], transactionID)

	// Add CHANGE-REQUEST attribute if needed
	if changeIP || changePort {
		attr := make([]byte, 8)
		binary.BigEndian.PutUint16(attr[0:2], stunAttrChangeRequest)
		binary.BigEndian.PutUint16(attr[2:4], 4) // Length

		changeFlags := uint32(0)
		if changeIP {
			changeFlags |= 0x04
		}
		if changePort {
			changeFlags |= 0x02
		}
		binary.BigEndian.PutUint32(attr[4:8], changeFlags)

		message = append(message, attr...)
		messageLength += 8
	}

	// Update message length
	binary.BigEndian.PutUint16(message[2:4], uint16(messageLength))

	return message
}

// parseBindingResponse parses a STUN binding response
func (c *STUNClient) parseBindingResponse(data []byte, expectedTxID []byte) (*net.UDPAddr, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("message too short")
	}

	// Check message type
	msgType := binary.BigEndian.Uint16(data[0:2])
	if msgType != stunBindingResponse {
		return nil, fmt.Errorf("not a binding response: 0x%04x", msgType)
	}

	// Check magic cookie
	cookie := binary.BigEndian.Uint32(data[4:8])
	if cookie != stunMagicCookie {
		return nil, fmt.Errorf("invalid magic cookie")
	}

	// Check transaction ID
	txID := data[8:20]
	for i := 0; i < 12; i++ {
		if txID[i] != expectedTxID[i] {
			return nil, fmt.Errorf("transaction ID mismatch")
		}
	}

	// Parse attributes
	messageLength := binary.BigEndian.Uint16(data[2:4])
	pos := 20

	var mappedAddr *net.UDPAddr

	for pos < 20+int(messageLength) {
		if pos+4 > len(data) {
			break
		}

		attrType := binary.BigEndian.Uint16(data[pos : pos+2])
		attrLength := binary.BigEndian.Uint16(data[pos+2 : pos+4])
		pos += 4

		if pos+int(attrLength) > len(data) {
			break
		}

		attrValue := data[pos : pos+int(attrLength)]

		switch attrType {
		case stunAttrMappedAddress:
			mappedAddr = c.parseMappedAddress(attrValue, false)
		case stunAttrXorMappedAddress, stunAttrXorMappedAddress2:
			mappedAddr = c.parseMappedAddress(attrValue, true)
		}

		// Advance to next attribute (attributes are padded to 4-byte boundary)
		pos += int(attrLength)
		if attrLength%4 != 0 {
			pos += 4 - int(attrLength%4)
		}
	}

	if mappedAddr == nil {
		return nil, fmt.Errorf("no mapped address in response")
	}

	return mappedAddr, nil
}

// parseMappedAddress parses a MAPPED-ADDRESS or XOR-MAPPED-ADDRESS attribute
func (c *STUNClient) parseMappedAddress(data []byte, xor bool) *net.UDPAddr {
	if len(data) < 8 {
		return nil
	}

	// Skip first byte (reserved)
	family := data[1]
	if family != 0x01 { // IPv4
		return nil
	}

	port := binary.BigEndian.Uint16(data[2:4])
	ip := net.IPv4(data[4], data[5], data[6], data[7])

	if xor {
		// XOR with magic cookie
		port ^= uint16(stunMagicCookie >> 16)
		cookieBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(cookieBytes, stunMagicCookie)
		for i := 0; i < 4; i++ {
			ip[i] ^= cookieBytes[i]
		}
	}

	return &net.UDPAddr{
		IP:   ip,
		Port: int(port),
	}
}

// StartRefreshRoutine starts automatic NAT mapping refresh
func (c *STUNClient) StartRefreshRoutine() {
	go func() {
		ticker := time.NewTicker(c.config.RefreshInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := c.RefreshMapping(); err != nil {
				// Mapping refresh failed, try to rediscover
				c.DiscoverNATMapping()
			}
		}
	}()
}

// GetStats returns STUN client statistics
func (c *STUNClient) GetStats() (requests, successes, failures uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.requests, c.successes, c.failures
}

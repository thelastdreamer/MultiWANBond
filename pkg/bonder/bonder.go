package bonder

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/config"
	"github.com/thelastdreamer/MultiWANBond/pkg/fec"
	"github.com/thelastdreamer/MultiWANBond/pkg/health"
	"github.com/thelastdreamer/MultiWANBond/pkg/packet"
	"github.com/thelastdreamer/MultiWANBond/pkg/plugin"
	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
	"github.com/thelastdreamer/MultiWANBond/pkg/router"
)

// Bonder is the main bonding implementation
type Bonder struct {
	mu              sync.RWMutex
	session         *protocol.Session
	healthChecker   *health.Checker
	router          *router.Router
	processor       *packet.Processor
	fecManager      *fec.FECManager
	pluginManager   *plugin.Manager
	wans            map[uint8]*protocol.WANInterface
	sendChan        chan []byte
	recvChan        chan []byte
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	running         atomic.Bool
	sequenceID      atomic.Uint64
}

// New creates a new Bonder instance
func New(cfg *config.BondConfig) (*Bonder, error) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Convert session config
	sessionConfig, err := cfg.Session.ToSessionConfig()
	if err != nil {
		return nil, fmt.Errorf("invalid session config: %w", err)
	}

	// Create session
	session := &protocol.Session{
		ID:             uint64(time.Now().UnixNano()),
		LocalEndpoint:  cfg.Session.LocalEndpoint,
		RemoteEndpoint: cfg.Session.RemoteEndpoint,
		WANInterfaces:  make(map[uint8]*protocol.WANInterface),
		StartTime:     time.Now(),
		Config:        sessionConfig,
	}

	// Create components
	routingMode := config.ParseLoadBalanceMode(cfg.Routing.Mode)

	bonder := &Bonder{
		session:       session,
		healthChecker: health.NewChecker(),
		router:        router.NewRouter(routingMode),
		processor:     packet.NewProcessor(sessionConfig.ReorderBuffer, sessionConfig.ReorderTimeout),
		fecManager:    fec.NewFECManager(),
		pluginManager: plugin.NewManager(),
		wans:          make(map[uint8]*protocol.WANInterface),
		sendChan:      make(chan []byte, 1000),
		recvChan:      make(chan []byte, 1000),
	}

	// Configure FEC
	if cfg.FEC.Enabled {
		bonder.fecManager.Enable()
		session.Config.FECEnabled = true
		session.Config.FECRedundancy = cfg.FEC.Redundancy
	}

	// Add WANs from config
	for _, wanCfg := range cfg.WANs {
		if err := bonder.addWANFromConfig(&wanCfg); err != nil {
			return nil, fmt.Errorf("failed to add WAN %s: %w", wanCfg.Name, err)
		}
	}

	return bonder, nil
}

// Start starts the bonding service
func (b *Bonder) Start(ctx context.Context) error {
	if b.running.Load() {
		return fmt.Errorf("bonder already running")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.ctx, b.cancel = context.WithCancel(ctx)

	// Start health checker
	if err := b.healthChecker.Start(b.ctx); err != nil {
		return fmt.Errorf("failed to start health checker: %w", err)
	}

	// Start plugins
	if err := b.pluginManager.StartAll(b.ctx); err != nil {
		b.healthChecker.Stop()
		return fmt.Errorf("failed to start plugins: %w", err)
	}

	// Start sender goroutine
	b.wg.Add(1)
	go b.senderLoop()

	// Start receiver goroutines for each WAN
	for _, wan := range b.wans {
		b.wg.Add(1)
		go b.receiverLoop(wan)
	}

	// Start health event handler
	b.wg.Add(1)
	go b.healthEventLoop()

	b.running.Store(true)

	return nil
}

// Stop stops the bonding service
func (b *Bonder) Stop() error {
	if !b.running.Load() {
		return fmt.Errorf("bonder not running")
	}

	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
	}
	b.mu.Unlock()

	// Wait for goroutines
	b.wg.Wait()

	// Stop components
	b.healthChecker.Stop()
	b.pluginManager.StopAll()

	// Close connections
	b.mu.Lock()
	for _, wan := range b.wans {
		if wan.Conn != nil {
			wan.Conn.Close()
		}
	}
	b.mu.Unlock()

	b.running.Store(false)

	return nil
}

// AddWAN adds a new WAN interface to the bond
func (b *Bonder) AddWAN(wan *protocol.WANInterface) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.wans[wan.ID]; exists {
		return fmt.Errorf("WAN %d already exists", wan.ID)
	}

	// Create UDP connection
	if wan.Conn == nil {
		addr, err := net.ResolveUDPAddr("udp", wan.LocalAddr.String()+":0")
		if err != nil {
			return fmt.Errorf("failed to resolve local address: %w", err)
		}

		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return fmt.Errorf("failed to create UDP connection: %w", err)
		}

		wan.Conn = conn
	}

	b.wans[wan.ID] = wan
	b.session.WANInterfaces[wan.ID] = wan

	// Add to components
	b.healthChecker.AddWAN(wan)
	b.router.AddWAN(wan)

	// If running, start receiver for this WAN
	if b.running.Load() {
		b.wg.Add(1)
		go b.receiverLoop(wan)
	}

	return nil
}

// RemoveWAN removes a WAN interface from the bond
func (b *Bonder) RemoveWAN(wanID uint8) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	wan, exists := b.wans[wanID]
	if !exists {
		return fmt.Errorf("WAN %d not found", wanID)
	}

	// Close connection
	if wan.Conn != nil {
		wan.Conn.Close()
	}

	// Remove from components
	b.healthChecker.RemoveWAN(wanID)
	b.router.RemoveWAN(wanID)

	delete(b.wans, wanID)
	delete(b.session.WANInterfaces, wanID)

	return nil
}

// GetWANs returns all active WAN interfaces
func (b *Bonder) GetWANs() map[uint8]*protocol.WANInterface {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Return copy
	wans := make(map[uint8]*protocol.WANInterface)
	for id, wan := range b.wans {
		wans[id] = wan
	}

	return wans
}

// GetMetrics returns current metrics for all WANs
func (b *Bonder) GetMetrics() map[uint8]*protocol.WANMetrics {
	b.mu.RLock()
	defer b.mu.RUnlock()

	metrics := make(map[uint8]*protocol.WANMetrics)
	for id := range b.wans {
		if m, err := b.healthChecker.GetMetrics(id); err == nil {
			metrics[id] = m
		}
	}

	return metrics
}

// Send sends data through the bonded connection
func (b *Bonder) Send(data []byte) error {
	if !b.running.Load() {
		return fmt.Errorf("bonder not running")
	}

	select {
	case b.sendChan <- data:
		return nil
	case <-b.ctx.Done():
		return fmt.Errorf("bonder stopped")
	default:
		return fmt.Errorf("send buffer full")
	}
}

// Receive returns a channel for receiving data
func (b *Bonder) Receive() <-chan []byte {
	return b.recvChan
}

// GetSession returns the current session
func (b *Bonder) GetSession() *protocol.Session {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.session
}

// UpdateConfig updates the session configuration
func (b *Bonder) UpdateConfig(config *protocol.SessionConfig) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.session.Config = config

	// Update FEC
	if config.FECEnabled {
		b.fecManager.Enable()
	} else {
		b.fecManager.Disable()
	}

	return nil
}

// senderLoop handles sending packets
func (b *Bonder) senderLoop() {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			return

		case data := <-b.sendChan:
			if err := b.sendPacket(data); err != nil {
				// Alert on send error
				b.pluginManager.Alert(protocol.AlertLevelWarning, "Send error", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// sendPacket sends a single packet
func (b *Bonder) sendPacket(data []byte) error {
	// Create packet
	pkt := &protocol.Packet{
		Version:    protocol.ProtocolVersion,
		Type:       protocol.PacketTypeData,
		SessionID:  b.session.ID,
		SequenceID: b.sequenceID.Add(1),
		Timestamp:  time.Now().UnixNano(),
		Priority:   128, // Default priority
		Data:       data,
	}

	// Apply outgoing filters
	filtered, err := b.pluginManager.FilterOutgoing(pkt)
	if err != nil {
		return fmt.Errorf("filter error: %w", err)
	}
	if filtered == nil {
		// Packet dropped by filter
		return nil
	}
	pkt = filtered

	// Get routing decision
	decision, err := b.router.Route(pkt, nil)
	if err != nil {
		return fmt.Errorf("routing error: %w", err)
	}

	// Encode packet
	encoded, err := b.processor.Encode(pkt)
	if err != nil {
		return fmt.Errorf("encode error: %w", err)
	}

	// Send on primary WAN
	b.mu.RLock()
	primaryWAN := b.wans[decision.PrimaryWAN]
	b.mu.RUnlock()

	if primaryWAN == nil || primaryWAN.RemoteAddr == nil {
		return fmt.Errorf("primary WAN not available")
	}

	_, err = primaryWAN.Conn.WriteToUDP(encoded, primaryWAN.RemoteAddr)
	if err != nil {
		return fmt.Errorf("send error: %w", err)
	}

	// Record metrics
	b.pluginManager.RecordPacket(decision.PrimaryWAN, pkt, true)

	// Send on backup WANs if needed
	for _, wanID := range decision.BackupWANs {
		b.mu.RLock()
		backupWAN := b.wans[wanID]
		b.mu.RUnlock()

		if backupWAN != nil && backupWAN.RemoteAddr != nil {
			backupWAN.Conn.WriteToUDP(encoded, backupWAN.RemoteAddr)
			b.pluginManager.RecordPacket(wanID, pkt, true)
		}
	}

	return nil
}

// receiverLoop handles receiving packets on a WAN
func (b *Bonder) receiverLoop(wan *protocol.WANInterface) {
	defer b.wg.Done()

	buf := make([]byte, protocol.MaxPacketSize)

	for {
		select {
		case <-b.ctx.Done():
			return

		default:
			// Set read deadline
			wan.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))

			n, addr, err := wan.Conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				// Other error
				continue
			}

			// Update remote address if not set
			if wan.RemoteAddr == nil {
				wan.RemoteAddr = addr
			}

			// Decode packet
			pkt, err := b.processor.Decode(buf[:n])
			if err != nil {
				continue
			}

			// Apply incoming filters
			filtered, err := b.pluginManager.FilterIncoming(pkt)
			if err != nil || filtered == nil {
				continue
			}
			pkt = filtered

			// Record metrics
			b.pluginManager.RecordPacket(wan.ID, pkt, false)

			// Handle packet based on type
			switch pkt.Type {
			case protocol.PacketTypeHeartbeat:
				// Echo heartbeat back
				wan.Conn.WriteToUDP(buf[:n], addr)

			case protocol.PacketTypeData:
				// Reorder and deliver
				data, ready, err := b.processor.Reorder(pkt)
				if err == nil && ready {
					select {
					case b.recvChan <- data:
					default:
						// Receive buffer full
					}
				}

			case protocol.PacketTypeControl:
				// Handle control packet
				// TODO: Implement control message handling

			case protocol.PacketTypeMulticast:
				// Handle multicast
				// TODO: Implement multicast handling
			}
		}
	}
}

// healthEventLoop handles health events
func (b *Bonder) healthEventLoop() {
	defer b.wg.Done()

	events := b.healthChecker.Subscribe()

	for {
		select {
		case <-b.ctx.Done():
			return

		case event := <-events:
			// Update router with new metrics
			if event.Metrics != nil {
				b.router.UpdateMetrics(event.WANID, event.Metrics)
			}

			// Send alerts for state changes
			if event.OldState != event.NewState {
				level := protocol.AlertLevelInfo
				if event.NewState == protocol.WANStateDown {
					level = protocol.AlertLevelError
				}

				b.pluginManager.Alert(level, "WAN state changed", map[string]interface{}{
					"wan_id":    event.WANID,
					"old_state": event.OldState,
					"new_state": event.NewState,
				})
			}
		}
	}
}

// addWANFromConfig adds a WAN from configuration
func (b *Bonder) addWANFromConfig(cfg *config.WANInterfaceConfig) error {
	wanConfig, err := cfg.ToWANConfig()
	if err != nil {
		return err
	}

	// Parse addresses
	localIP := net.ParseIP(cfg.LocalAddr)
	if localIP == nil {
		return fmt.Errorf("invalid local address: %s", cfg.LocalAddr)
	}

	var remoteAddr *net.UDPAddr
	if cfg.RemoteAddr != "" {
		remoteAddr, err = net.ResolveUDPAddr("udp", cfg.RemoteAddr)
		if err != nil {
			return fmt.Errorf("invalid remote address: %w", err)
		}
	}

	wan := &protocol.WANInterface{
		ID:         cfg.ID,
		Name:       cfg.Name,
		Type:       config.ParseWANType(cfg.Type),
		LocalAddr:  localIP,
		RemoteAddr: remoteAddr,
		Metrics:    &protocol.WANMetrics{},
		State:      protocol.WANStateStarting,
		Config:     *wanConfig,
		LastSeen:   time.Now(),
	}

	return b.AddWAN(wan)
}

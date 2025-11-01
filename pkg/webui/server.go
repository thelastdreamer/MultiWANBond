package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/config"
	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// Server provides web-based management interface
type Server struct {
	config *Config
	mu     sync.RWMutex

	// HTTP server
	httpServer *http.Server

	// WebSocket clients
	wsClients map[*WSClient]bool
	wsMu      sync.RWMutex

	// Event channel
	eventChan chan *Event

	// System state
	startTime  time.Time
	stats      *DashboardStats
	wanStatuses []*WANStatus

	// Configuration management
	configFile   string
	bondConfig   *config.BondConfig
	configMu     sync.RWMutex

	// Backend component references
	metricsData  *MetricsData
	metricsMu    sync.RWMutex

	// Control
	running bool
	stopCh  chan struct{}
}

// MetricsData holds backend metrics for the Web UI
type MetricsData struct {
	WANMetrics   map[uint8]*protocol.WANMetrics
	Flows        []FlowInfo
	Alerts       []Alert
	NATInfo      *NATInfo
	HealthChecks []HealthCheckInfo
	TrafficStats *TrafficStats
	LastUpdate   time.Time
}

// NewServer creates a new web UI server
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	return &Server{
		config:      config,
		wsClients:   make(map[*WSClient]bool),
		eventChan:   make(chan *Event, 1000),
		startTime:   time.Now(),
		stats:       &DashboardStats{},
		stopCh:      make(chan struct{}),
		metricsData: &MetricsData{
			WANMetrics: make(map[uint8]*protocol.WANMetrics),
			Flows:      make([]FlowInfo, 0),
			Alerts:     make([]Alert, 0),
		},
	}
}

// Start starts the web server
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	// Setup routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.ListenAddr, s.config.ListenPort)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.corsMiddleware(s.authMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start event broadcaster
	go s.broadcastEvents()

	// Start server
	go func() {
		var err error
		if s.config.EnableTLS {
			err = s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
		} else {
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("Web server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the web server
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)

	if s.httpServer != nil {
		return s.httpServer.Close()
	}

	return nil
}

// setupRoutes configures HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// API endpoints
	mux.HandleFunc("/api/dashboard", s.handleDashboard)
	mux.HandleFunc("/api/wans", s.handleWANs)
	mux.HandleFunc("/api/wans/status", s.handleWANStatus)
	mux.HandleFunc("/api/flows", s.handleFlows)
	mux.HandleFunc("/api/traffic", s.handleTraffic)
	mux.HandleFunc("/api/nat", s.handleNATInfo)
	mux.HandleFunc("/api/health", s.handleHealthChecks)
	mux.HandleFunc("/api/routing", s.handleRouting)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/alerts", s.handleAlerts)

	// WebSocket endpoint
	mux.HandleFunc("/ws", s.handleWebSocket)

	// Metrics endpoint
	if s.config.EnableMetrics {
		mux.HandleFunc(s.config.MetricsPath, s.handleMetrics)
	}

	// Static files
	if s.config.StaticDir != "" {
		fs := http.FileServer(http.Dir(s.config.StaticDir))
		mux.Handle("/", fs)
	}
}

// handleDashboard returns dashboard statistics
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	// Return a copy of the stats with current system info
	stats := &DashboardStats{
		Uptime:        time.Since(s.startTime),
		Version:       "1.0.0",
		Platform:      runtime.GOOS,
		ActiveWANs:    s.stats.ActiveWANs,
		TotalWANs:     s.stats.TotalWANs,
		HealthyWANs:   s.stats.HealthyWANs,
		DegradedWANs:  s.stats.DegradedWANs,
		DownWANs:      s.stats.DownWANs,
		TotalPackets:  s.stats.TotalPackets,
		TotalBytes:    s.stats.TotalBytes,
		CurrentPPS:    s.stats.CurrentPPS,
		CurrentBPS:    s.stats.CurrentBPS,
		ActiveFlows:   s.stats.ActiveFlows,
		TotalSessions: s.stats.TotalSessions,
		NATType:       s.stats.NATType,
		PublicIP:      s.stats.PublicIP,
		CGNATDetected: s.stats.CGNATDetected,
		Timestamp:     time.Now(),
	}
	s.mu.RUnlock()

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// handleWANs handles WAN interface queries
func (s *Server) handleWANs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Get specific WAN by ID or list all
		idParam := r.URL.Query().Get("id")
		if idParam != "" {
			// Return specific WAN
			s.configMu.RLock()
			cfg := s.bondConfig
			s.configMu.RUnlock()

			if cfg == nil {
				s.sendError(w, "No configuration loaded", http.StatusInternalServerError)
				return
			}

			var id uint8
			fmt.Sscanf(idParam, "%d", &id)

			for _, wan := range cfg.WANs {
				if wan.ID == id {
					s.sendJSON(w, APIResponse{
						Success: true,
						Data:    toWANConfig(wan),
					})
					return
				}
			}

			s.sendError(w, "WAN not found", http.StatusNotFound)
			return
		}

		// Return list of all WANs
		s.configMu.RLock()
		cfg := s.bondConfig
		s.configMu.RUnlock()

		if cfg == nil {
			s.sendJSON(w, APIResponse{
				Success: true,
				Data:    make([]*WANConfig, 0),
			})
			return
		}

		wans := make([]*WANConfig, 0, len(cfg.WANs))
		for _, wan := range cfg.WANs {
			wans = append(wans, toWANConfig(wan))
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Data:    wans,
		})

	case http.MethodPost:
		// Add new WAN
		var wanCfg WANConfig
		if err := json.NewDecoder(r.Body).Decode(&wanCfg); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.configMu.Lock()
		if s.bondConfig == nil {
			s.configMu.Unlock()
			s.sendError(w, "No configuration loaded", http.StatusInternalServerError)
			return
		}

		// Add WAN to configuration
		newWAN := fromWANConfig(&wanCfg)
		s.bondConfig.WANs = append(s.bondConfig.WANs, newWAN)
		s.configMu.Unlock()

		// Save configuration
		if err := s.SaveConfig(); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "WAN added successfully (restart required for changes to take effect)",
		})

	case http.MethodPut:
		// Update WAN
		var wanCfg WANConfig
		if err := json.NewDecoder(r.Body).Decode(&wanCfg); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.configMu.Lock()
		if s.bondConfig == nil {
			s.configMu.Unlock()
			s.sendError(w, "No configuration loaded", http.StatusInternalServerError)
			return
		}

		// Find and update WAN
		found := false
		for i, wan := range s.bondConfig.WANs {
			if wan.ID == wanCfg.ID {
				s.bondConfig.WANs[i] = fromWANConfig(&wanCfg)
				found = true
				break
			}
		}
		s.configMu.Unlock()

		if !found {
			s.sendError(w, "WAN not found", http.StatusNotFound)
			return
		}

		// Save configuration
		if err := s.SaveConfig(); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "WAN updated successfully (restart required for changes to take effect)",
		})

	case http.MethodDelete:
		// Delete WAN
		idParam := r.URL.Query().Get("id")
		if idParam == "" {
			s.sendError(w, "Missing id parameter", http.StatusBadRequest)
			return
		}

		var id uint8
		fmt.Sscanf(idParam, "%d", &id)

		s.configMu.Lock()
		if s.bondConfig == nil {
			s.configMu.Unlock()
			s.sendError(w, "No configuration loaded", http.StatusInternalServerError)
			return
		}

		// Find and delete WAN
		found := false
		newWANs := make([]config.WANInterfaceConfig, 0, len(s.bondConfig.WANs))
		for _, wan := range s.bondConfig.WANs {
			if wan.ID == id {
				found = true
				continue
			}
			newWANs = append(newWANs, wan)
		}
		s.bondConfig.WANs = newWANs
		s.configMu.Unlock()

		if !found {
			s.sendError(w, "WAN not found", http.StatusNotFound)
			return
		}

		// Save configuration
		if err := s.SaveConfig(); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "WAN deleted successfully (restart required for changes to take effect)",
		})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleWANStatus returns real-time WAN status
func (s *Server) handleWANStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	statuses := s.wanStatuses
	s.mu.RUnlock()

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    statuses,
	})
}

// handleFlows returns active flows
func (s *Server) handleFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metricsMu.RLock()
	flows := s.metricsData.Flows
	s.metricsMu.RUnlock()

	if flows == nil {
		flows = make([]FlowInfo, 0)
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    flows,
	})
}

// handleTraffic returns traffic statistics
func (s *Server) handleTraffic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metricsMu.RLock()
	stats := s.metricsData.TrafficStats
	s.metricsMu.RUnlock()

	if stats == nil {
		stats = &TrafficStats{
			Timestamp:     time.Now(),
			BytesPerWAN:   make(map[uint8]uint64),
			PacketsPerWAN: make(map[uint8]uint64),
			TopProtocols:  make([]ProtocolStat, 0),
			TopFlows:      make([]FlowInfo, 0),
		}
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    stats,
	})
}

// handleNATInfo returns NAT traversal information
func (s *Server) handleNATInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metricsMu.RLock()
	natInfo := s.metricsData.NATInfo
	s.metricsMu.RUnlock()

	if natInfo == nil {
		// Return default NAT info if not yet available
		natInfo = &NATInfo{
			NATType:       "Unknown",
			LocalAddr:     "",
			PublicAddr:    "",
			CGNATDetected: false,
			CanDirect:     false,
			NeedsRelay:    false,
		}
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    natInfo,
	})
}

// handleHealthChecks returns health check information
func (s *Server) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.metricsMu.RLock()
	checks := s.metricsData.HealthChecks
	s.metricsMu.RUnlock()

	if checks == nil {
		checks = make([]HealthCheckInfo, 0)
	}

	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    checks,
	})
}

// handleRouting handles routing policy queries
func (s *Server) handleRouting(w http.ResponseWriter, r *http.Request) {
	s.configMu.RLock()
	cfg := s.bondConfig
	s.configMu.RUnlock()

	if cfg == nil {
		s.sendError(w, "Configuration not loaded", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return stored routing policies
		policies := make([]*RoutingPolicy, 0, len(cfg.Routing.Policies))
		for _, p := range cfg.Routing.Policies {
			policies = append(policies, &RoutingPolicy{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				Type:        p.Type,
				Match:       p.Match,
				TargetWAN:   p.TargetWAN,
				Priority:    p.Priority,
				Enabled:     p.Enabled,
			})
		}
		s.sendJSON(w, APIResponse{
			Success: true,
			Data:    policies,
		})

	case http.MethodPost:
		var policy RoutingPolicy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Add new routing policy to configuration
		s.configMu.Lock()

		// Generate ID
		maxID := 0
		for _, p := range s.bondConfig.Routing.Policies {
			if p.ID > maxID {
				maxID = p.ID
			}
		}
		policy.ID = maxID + 1

		// Add to config
		s.bondConfig.Routing.Policies = append(s.bondConfig.Routing.Policies, config.RoutingPolicy{
			ID:          policy.ID,
			Name:        policy.Name,
			Description: policy.Description,
			Type:        policy.Type,
			Match:       policy.Match,
			TargetWAN:   policy.TargetWAN,
			Priority:    policy.Priority,
			Enabled:     policy.Enabled,
		})

		// Save to file
		if err := s.SaveConfig(); err != nil {
			s.configMu.Unlock()
			s.sendError(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
			return
		}
		s.configMu.Unlock()

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "Routing policy added successfully (restart required for changes to take effect)",
			Data:    policy,
		})

	case http.MethodDelete:
		idParam := r.URL.Query().Get("id")
		if idParam == "" {
			s.sendError(w, "Missing policy ID", http.StatusBadRequest)
			return
		}

		var id int
		if _, err := fmt.Sscanf(idParam, "%d", &id); err != nil {
			s.sendError(w, "Invalid policy ID", http.StatusBadRequest)
			return
		}

		// Remove routing policy from configuration
		s.configMu.Lock()

		// Find and remove policy
		found := false
		newPolicies := make([]config.RoutingPolicy, 0, len(s.bondConfig.Routing.Policies)-1)
		for _, p := range s.bondConfig.Routing.Policies {
			if p.ID == id {
				found = true
				continue
			}
			newPolicies = append(newPolicies, p)
		}

		if !found {
			s.configMu.Unlock()
			s.sendError(w, "Routing policy not found", http.StatusNotFound)
			return
		}

		s.bondConfig.Routing.Policies = newPolicies

		// Save to file
		if err := s.SaveConfig(); err != nil {
			s.configMu.Unlock()
			s.sendError(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
			return
		}
		s.configMu.Unlock()

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "Routing policy deleted successfully (restart required for changes to take effect)",
		})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfig handles system configuration
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.configMu.RLock()
		cfg := s.bondConfig
		s.configMu.RUnlock()

		if cfg == nil {
			s.sendJSON(w, APIResponse{
				Success: true,
				Data:    &SystemConfig{},
			})
			return
		}

		// Convert to SystemConfig
		sysConfig := &SystemConfig{
			LoadBalanceMode: string(cfg.Routing.Mode),
			EnableFEC:       cfg.FEC.Enabled,
			FECDataShards:   cfg.FEC.DataShards,
			FECParityShards: cfg.FEC.ParityShards,
			EnableDPI:       false, // Not in config yet
			EnableQoS:       false, // Not in config yet
			EnableNATT:      true,  // Assume enabled
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Data:    sysConfig,
		})

	case http.MethodPut:
		var sysConfig SystemConfig
		if err := json.NewDecoder(r.Body).Decode(&sysConfig); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.configMu.Lock()
		if s.bondConfig == nil {
			s.configMu.Unlock()
			s.sendError(w, "No configuration loaded", http.StatusInternalServerError)
			return
		}

		// Update configuration
		s.bondConfig.Routing.Mode = sysConfig.LoadBalanceMode
		s.bondConfig.FEC.Enabled = sysConfig.EnableFEC
		s.bondConfig.FEC.DataShards = sysConfig.FECDataShards
		s.bondConfig.FEC.ParityShards = sysConfig.FECParityShards
		s.bondConfig.FEC.Redundancy = float64(sysConfig.FECParityShards) / float64(sysConfig.FECDataShards)

		s.configMu.Unlock()

		// Save configuration
		if err := s.SaveConfig(); err != nil {
			s.sendError(w, fmt.Sprintf("Failed to save configuration: %v", err), http.StatusInternalServerError)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "Configuration updated successfully (restart required for changes to take effect)",
		})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleLogs returns system logs
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logs := make([]*LogEntry, 0)
	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    logs,
	})
}

// handleAlerts returns system alerts
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.metricsMu.RLock()
		alerts := s.metricsData.Alerts
		s.metricsMu.RUnlock()

		if alerts == nil {
			alerts = make([]Alert, 0)
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Data:    alerts,
		})

	case http.MethodDelete:
		s.ClearAlerts()
		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "All alerts cleared",
		})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleMetrics returns Prometheus-style metrics
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# MultiWANBond Metrics\n")
	fmt.Fprintf(w, "multiwanbond_uptime_seconds %.0f\n", time.Since(s.startTime).Seconds())
	fmt.Fprintf(w, "multiwanbond_goroutines %d\n", runtime.NumGoroutine())
}

// sendJSON sends a JSON response
func (s *Server) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// authMiddleware provides basic authentication
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.config.EnableAuth {
			next.ServeHTTP(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok || username != s.config.Username || password != s.config.Password {
			w.Header().Set("WWW-Authenticate", `Basic realm="MultiWANBond"`)
			s.sendError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.config.EnableCORS {
			origin := "*"
			if len(s.config.AllowedOrigins) > 0 {
				origin = s.config.AllowedOrigins[0]
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

// PublishEvent publishes an event to WebSocket clients
func (s *Server) PublishEvent(event *Event) {
	select {
	case s.eventChan <- event:
	default:
		// Channel full, drop event
	}
}

// broadcastEvents broadcasts events to all WebSocket clients
func (s *Server) broadcastEvents() {
	for {
		select {
		case <-s.stopCh:
			return
		case event := <-s.eventChan:
			s.wsMu.RLock()
			for client := range s.wsClients {
				select {
				case client.send <- event:
				default:
					// Client send buffer full, skip
				}
			}
			s.wsMu.RUnlock()
		}
	}
}

// GetAddress returns the server address
func (s *Server) GetAddress() string {
	scheme := "http"
	if s.config.EnableTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, s.config.ListenAddr, s.config.ListenPort)
}

// UpdateStats updates dashboard statistics from bonder metrics
func (s *Server) UpdateStats(metrics map[uint8]*protocol.WANMetrics, wans map[uint8]*protocol.WANInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Reset counters
	s.stats.TotalWANs = len(wans)
	s.stats.ActiveWANs = 0
	s.stats.HealthyWANs = 0
	s.stats.DegradedWANs = 0
	s.stats.DownWANs = 0
	s.stats.TotalPackets = 0
	s.stats.TotalBytes = 0

	// Build WAN statuses
	s.wanStatuses = make([]*WANStatus, 0, len(wans))

	// Process each WAN
	for id, wan := range wans {
		if wan == nil {
			continue
		}

		// Determine WAN state
		status := "down"
		switch wan.State {
		case protocol.WANStateUp:
			s.stats.ActiveWANs++
			s.stats.HealthyWANs++
			status = "up"
		case protocol.WANStateDegraded:
			s.stats.ActiveWANs++
			s.stats.DegradedWANs++
			status = "degraded"
		default: // Down or other
			s.stats.DownWANs++
		}

		// Get metrics for this WAN
		wanStatus := &WANStatus{
			ID:        id,
			Name:      wan.Name,
			Interface: wan.Name,
			Status:    status,
			Weight:    wan.Config.Weight,
		}

		if m, exists := metrics[id]; exists && m != nil {
			s.stats.TotalPackets += m.PacketsSent + m.PacketsRecv
			s.stats.TotalBytes += m.BytesSent + m.BytesReceived

			wanStatus.Latency = m.AvgLatency.Milliseconds()
			wanStatus.Jitter = m.AvgJitter.Milliseconds()
			wanStatus.PacketLoss = m.AvgPacketLoss
			wanStatus.BytesSent = m.BytesSent
			wanStatus.BytesReceived = m.BytesReceived
			wanStatus.PacketsSent = m.PacketsSent
			wanStatus.PacketsReceived = m.PacketsRecv
		}

		s.wanStatuses = append(s.wanStatuses, wanStatus)
	}

	s.stats.Uptime = time.Since(s.startTime)
	s.stats.Timestamp = time.Now()

	// Update metrics data for other handlers
	s.metricsMu.Lock()
	s.metricsData.WANMetrics = metrics
	s.metricsData.LastUpdate = time.Now()
	s.metricsMu.Unlock()
}

// UpdateNATInfo updates NAT traversal information
func (s *Server) UpdateNATInfo(natInfo *NATInfo) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metricsData.NATInfo = natInfo
}

// AddAlert adds a new alert
func (s *Server) AddAlert(alert Alert) {
	s.metricsMu.Lock()
	s.metricsData.Alerts = append(s.metricsData.Alerts, alert)
	s.metricsMu.Unlock()

	// Publish alert event to WebSocket clients
	s.PublishEvent(&Event{
		Type:      EventSystemAlert,
		Timestamp: time.Now(),
		Message:   alert.Message,
		Data:      alert,
		Severity:  alert.Severity,
	})
}

// UpdateFlows updates active network flows
func (s *Server) UpdateFlows(flows []FlowInfo) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metricsData.Flows = flows
}

// UpdateHealthChecks updates health check information
func (s *Server) UpdateHealthChecks(checks []HealthCheckInfo) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metricsData.HealthChecks = checks
}

// UpdateTrafficStats updates traffic statistics
func (s *Server) UpdateTrafficStats(stats *TrafficStats) {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metricsData.TrafficStats = stats

	// Publish traffic update event
	s.PublishEvent(&Event{
		Type:      EventTrafficUpdate,
		Timestamp: time.Now(),
		Data:      stats,
	})
}

// ClearAlerts clears all alerts
func (s *Server) ClearAlerts() {
	s.metricsMu.Lock()
	defer s.metricsMu.Unlock()
	s.metricsData.Alerts = make([]Alert, 0)
}

// SetConfigFile sets the configuration file path and loads it
func (s *Server) SetConfigFile(path string) error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	s.configFile = path

	// Load initial configuration
	cfg, err := config.LoadBondConfig(path)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	s.bondConfig = cfg
	return nil
}

// LoadConfig reloads configuration from file
func (s *Server) LoadConfig() error {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	if s.configFile == "" {
		return fmt.Errorf("no config file set")
	}

	cfg, err := config.LoadBondConfig(s.configFile)
	if err != nil {
		return err
	}

	s.bondConfig = cfg
	return nil
}

// SaveConfig saves current configuration to file
func (s *Server) SaveConfig() error {
	s.configMu.RLock()
	cfg := s.bondConfig
	file := s.configFile
	s.configMu.RUnlock()

	if file == "" {
		return fmt.Errorf("no config file set")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(file, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// toWANConfig converts config.WANInterfaceConfig to WANConfig for API
func toWANConfig(wan config.WANInterfaceConfig) *WANConfig {
	// Parse durations from strings
	maxLatency, _ := time.ParseDuration(wan.MaxLatency)
	maxJitter, _ := time.ParseDuration(wan.MaxJitter)
	healthInterval, _ := time.ParseDuration(wan.HealthCheckInterval)

	return &WANConfig{
		ID:                  wan.ID,
		Name:                wan.Name,
		Interface:           wan.LocalAddr,
		Priority:            wan.Weight, // Note: config uses Weight, not Priority field
		Weight:              wan.Weight,
		MaxBandwidth:        wan.MaxBandwidth,
		MaxLatency:          maxLatency.Milliseconds(),
		MaxJitter:           maxJitter.Milliseconds(),
		MaxPacketLoss:       wan.MaxPacketLoss,
		HealthCheckInterval: healthInterval.Milliseconds(),
		Enabled:             wan.Enabled,
	}
}

// fromWANConfig converts WANConfig to config.WANInterfaceConfig
func fromWANConfig(wanCfg *WANConfig) config.WANInterfaceConfig {
	return config.WANInterfaceConfig{
		ID:                  wanCfg.ID,
		Name:                wanCfg.Name,
		Type:                "ethernet", // Default type
		LocalAddr:           wanCfg.Interface,
		RemoteAddr:          "", // Will be set from session config
		MaxBandwidth:        wanCfg.MaxBandwidth,
		MaxLatency:          fmt.Sprintf("%dms", wanCfg.MaxLatency),
		MaxJitter:           fmt.Sprintf("%dms", wanCfg.MaxJitter),
		MaxPacketLoss:       wanCfg.MaxPacketLoss,
		HealthCheckInterval: fmt.Sprintf("%dms", wanCfg.HealthCheckInterval),
		FailureThreshold:    3,
		Weight:              wanCfg.Weight,
		Enabled:             wanCfg.Enabled,
	}
}

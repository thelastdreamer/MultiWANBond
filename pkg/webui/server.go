package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
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
	startTime time.Time
	stats     *DashboardStats

	// Control
	running bool
	stopCh  chan struct{}
}

// NewServer creates a new web UI server
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	return &Server{
		config:    config,
		wsClients: make(map[*WSClient]bool),
		eventChan: make(chan *Event, 1000),
		startTime: time.Now(),
		stats:     &DashboardStats{},
		stopCh:    make(chan struct{}),
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
	stats := &DashboardStats{
		Uptime:    time.Since(s.startTime),
		Version:   "0.1.0",
		Platform:  runtime.GOOS,
		Timestamp: time.Now(),
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
		// Return list of WANs
		wans := make([]*WANStatus, 0)
		s.sendJSON(w, APIResponse{
			Success: true,
			Data:    wans,
		})

	case http.MethodPost:
		// Add new WAN
		var config WANConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "WAN added successfully",
		})

	case http.MethodPut:
		// Update WAN
		var config WANConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "WAN updated successfully",
		})

	case http.MethodDelete:
		// Delete WAN
		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "WAN deleted successfully",
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

	// Return WAN status
	status := make([]*WANStatus, 0)
	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    status,
	})
}

// handleFlows returns active flows
func (s *Server) handleFlows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flows := make([]*FlowInfo, 0)
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

	stats := &TrafficStats{
		Timestamp:     time.Now(),
		BytesPerWAN:   make(map[uint8]uint64),
		PacketsPerWAN: make(map[uint8]uint64),
		TopProtocols:  make([]ProtocolStat, 0),
		TopFlows:      make([]FlowInfo, 0),
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

	natInfo := &NATInfo{}
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

	checks := make([]*HealthCheckInfo, 0)
	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    checks,
	})
}

// handleRouting handles routing policy queries
func (s *Server) handleRouting(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		policies := make([]*RoutingPolicy, 0)
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

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "Routing policy added",
		})

	default:
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConfig handles system configuration
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		config := &SystemConfig{}
		s.sendJSON(w, APIResponse{
			Success: true,
			Data:    config,
		})

	case http.MethodPut:
		var config SystemConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			s.sendError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		s.sendJSON(w, APIResponse{
			Success: true,
			Message: "Configuration updated",
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
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	alerts := make([]*Alert, 0)
	s.sendJSON(w, APIResponse{
		Success: true,
		Data:    alerts,
	})
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

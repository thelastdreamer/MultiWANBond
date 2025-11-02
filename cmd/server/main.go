package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/bonder"
	"github.com/thelastdreamer/MultiWANBond/pkg/config"
	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
	"github.com/thelastdreamer/MultiWANBond/pkg/setup"
	"github.com/thelastdreamer/MultiWANBond/pkg/webui"
)

const (
	version = "1.0.0"
)

func main() {
	// Parse command if provided
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "setup":
			runSetup()
			return
		case "version", "--version", "-v":
			fmt.Printf("MultiWANBond v%s\n", version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		}
	}

	// Run server mode
	runServer()
}

func runSetup() {
	// Parse setup flags
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	configFile := fs.String("config", "", "Path to save configuration file")
	fs.Parse(os.Args[2:])

	// If no config file specified, use default based on OS
	if *configFile == "" {
		if runtime.GOOS == "windows" {
			// Use ProgramData on Windows (same as installer)
			*configFile = filepath.Join(os.Getenv("ProgramData"), "MultiWANBond", "config.json")
		} else {
			// Use ~/.config on Linux/macOS
			if homeDir, err := os.UserHomeDir(); err == nil {
				*configFile = filepath.Join(homeDir, ".config", "multiwanbond", "config.json")
			} else {
				*configFile = "config.json"
			}
		}
	}

	fmt.Println("\n================================================================")
	fmt.Println("       MultiWANBond Setup Wizard")
	fmt.Println("================================================================\n")

	// Create wizard
	wizard, err := setup.NewWizard()
	if err != nil {
		log.Fatalf("Failed to create setup wizard: %v", err)
	}

	// Run wizard
	cfg, err := wizard.Run()
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}

	// Get network detector for conversion
	detector, err := wizard.GetDetector()
	if err != nil {
		log.Fatalf("Failed to get network detector: %v", err)
	}

	// Save configuration as BondConfig format
	if err := cfg.SaveAsBondConfig(*configFile, detector); err != nil {
		log.Fatalf("Failed to save configuration: %v", err)
	}

	fmt.Printf("\n[OK] Configuration saved to: %s\n", *configFile)
	fmt.Println("\nTo start MultiWANBond, run:")
	fmt.Printf("  multiwanbond --config %s\n", *configFile)
	fmt.Println("")
}

func runServer() {
	// Parse server flags
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	configFile := fs.String("config", "configs/example.json", "Path to configuration file")
	showStats := fs.Bool("stats", true, "Show statistics")
	statsInterval := fs.Duration("stats-interval", 10*time.Second, "Statistics interval")
	fs.Parse(os.Args[1:])

	// Check if config file exists
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		log.Printf("Configuration file not found: %s", *configFile)
		log.Println("\nPlease run the setup wizard first:")
		log.Println("  multiwanbond setup")
		log.Println("")
		os.Exit(1)
	}

	// Load configuration
	log.Printf("Loading configuration from %s", *configFile)
	cfg, err := config.LoadBondConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration has at least one WAN
	if len(cfg.WANs) == 0 {
		log.Fatalf("Configuration must have at least one WAN interface")
	}

	// Create bonder with optional remote address
	log.Println("Creating MultiWANBond instance...")
	b, err := bonder.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bonder: %v", err)
	}

	// Start bonding service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Starting MultiWANBond service...")
	if err := b.Start(ctx); err != nil {
		log.Fatalf("Failed to start bonder: %v", err)
	}

	// Start Web UI
	log.Println("Starting Web UI server...")
	webConfig := webui.DefaultConfig()
	webConfig.ListenPort = 8080

	// Load credentials from config if available
	if cfg.WebUI != nil && cfg.WebUI.Enabled {
		webConfig.EnableAuth = true
		webConfig.Username = cfg.WebUI.Username
		webConfig.Password = cfg.WebUI.Password
		log.Println("Web UI authentication enabled")
	} else {
		log.Println("WARNING: Web UI running without authentication!")
	}

	webServer := webui.NewServer(webConfig)

	// Set configuration file for web UI management
	if err := webServer.SetConfigFile(*configFile); err != nil {
		log.Printf("Warning: Failed to load config into Web UI: %v", err)
	}

	if err := webServer.Start(); err != nil {
		log.Printf("Warning: Failed to start Web UI: %v", err)
	} else {
		if cfg.WebUI != nil && cfg.WebUI.Enabled {
			log.Printf("Web UI available at: http://localhost:8080 (Username: %s)", cfg.WebUI.Username)
		} else {
			log.Printf("Web UI available at: http://localhost:8080 (No authentication)")
		}
	}

	// Start metrics bridge to update Web UI
	go metricsUpdater(b, webServer, 1*time.Second)

	// Print WAN status
	wans := b.GetWANs()
	log.Printf("Active WANs: %d", len(wans))
	for _, wan := range wans {
		log.Printf("  - WAN %d (%s): %s @ %s", wan.ID, wan.Name, wan.Type, wan.LocalAddr)
	}

	// Print mode information
	if cfg.Session.RemoteEndpoint != "" {
		log.Printf("Mode: Client - Connected to server at %s", cfg.Session.RemoteEndpoint)
	} else {
		log.Printf("Mode: Standalone - Not connected to any server")
		log.Printf("You can configure a server address later by editing: %s", *configFile)
	}

	// Start receiver goroutine
	go receiver(b)

	// Start statistics printer if enabled
	if *showStats {
		go statsMonitor(b, *statsInterval)
	}

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("MultiWANBond is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutting down...")
	if err := b.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func receiver(b *bonder.Bonder) {
	recvChan := b.Receive()
	for data := range recvChan {
		log.Printf("Received %d bytes: %s", len(data), string(data))

		// Echo back
		if err := b.Send([]byte("ACK: " + string(data))); err != nil {
			log.Printf("Failed to send response: %v", err)
		}
	}
}

func statsMonitor(b *bonder.Bonder, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var prevMetrics map[uint8]*protocol.WANMetrics

	for range ticker.C {
		metrics := b.GetMetrics()

		// Only print if metrics have changed
		if metricsChanged(prevMetrics, metrics) {
			printStats(b)
			prevMetrics = copyMetrics(metrics)
		}
	}
}

// metricsChanged checks if metrics have meaningfully changed
func metricsChanged(prev, current map[uint8]*protocol.WANMetrics) bool {
	if prev == nil {
		return true // First run, show stats
	}

	if len(prev) != len(current) {
		return true // Number of WANs changed
	}

	for id, curr := range current {
		old, exists := prev[id]
		if !exists {
			return true // New WAN appeared
		}

		// Check if significant changes occurred
		if old.PacketsSent != curr.PacketsSent ||
			old.PacketsRecv != curr.PacketsRecv ||
			old.BytesSent != curr.BytesSent ||
			old.BytesReceived != curr.BytesReceived {
			return true // Traffic changed
		}
	}

	return false // No changes
}

// copyMetrics creates a deep copy of metrics map
func copyMetrics(metrics map[uint8]*protocol.WANMetrics) map[uint8]*protocol.WANMetrics {
	if metrics == nil {
		return nil
	}

	copy := make(map[uint8]*protocol.WANMetrics, len(metrics))
	for id, m := range metrics {
		if m == nil {
			continue
		}
		metricsCopy := *m
		copy[id] = &metricsCopy
	}
	return copy
}

// metricsUpdater continuously updates Web UI with latest metrics
func metricsUpdater(b *bonder.Bonder, server *webui.Server, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Alert thresholds
	const (
		highLatencyThreshold  = 200 * time.Millisecond
		highJitterThreshold   = 50 * time.Millisecond
		highPacketLossPercent = 5.0
	)

	// Track previous states to detect changes
	prevStates := make(map[uint8]interface{})
	alertCounter := 0

	for range ticker.C {
		metrics := b.GetMetrics()
		wans := b.GetWANs()

		// Update Web UI statistics
		server.UpdateStats(metrics, wans)

		// Update health checks
		healthChecks := make([]webui.HealthCheckInfo, 0, len(metrics))
		for id, m := range metrics {
			wan := wans[id]
			if wan == nil {
				continue
			}

			healthChecks = append(healthChecks, webui.HealthCheckInfo{
				WANID:      id,
				Method:     "ICMP",
				Target:     wan.RemoteAddr.String(),
				Interval:   200, // 200ms interval
				LastCheck:  time.Now(),
				Status:     getHealthStatus(m),
				Latency:    m.Latency.Milliseconds(),
				Jitter:     m.Jitter.Milliseconds(),
				PacketLoss: m.PacketLoss,
				Successes:  int(m.PacketsRecv),
				Failures:   int(m.PacketsLost),
			})
		}
		server.UpdateHealthChecks(healthChecks)

		// Update traffic statistics
		totalBytes := uint64(0)
		totalPackets := uint64(0)
		bytesPerWAN := make(map[uint8]uint64)
		packetsPerWAN := make(map[uint8]uint64)

		for id, m := range metrics {
			bytesPerWAN[id] = m.BytesSent + m.BytesReceived
			packetsPerWAN[id] = m.PacketsSent + m.PacketsRecv
			totalBytes += bytesPerWAN[id]
			totalPackets += packetsPerWAN[id]
		}

		trafficStats := &webui.TrafficStats{
			Timestamp:     time.Now(),
			TotalBytes:    totalBytes,
			TotalPackets:  totalPackets,
			BytesPerWAN:   bytesPerWAN,
			PacketsPerWAN: packetsPerWAN,
			TopProtocols:  make([]webui.ProtocolStat, 0),
			TopFlows:      make([]webui.FlowInfo, 0),
		}
		server.UpdateTrafficStats(trafficStats)

		// Generate alerts for health issues
		for id, m := range metrics {
			wan := wans[id]
			if wan == nil {
				continue
			}

			// Check for WAN state changes
			if prevState, exists := prevStates[id]; exists {
				if prevState != wan.State {
					alertCounter++
					server.AddAlert(webui.Alert{
						ID:        fmt.Sprintf("alert-%d", alertCounter),
						Type:      "wan_state_change",
						Severity:  getAlertSeverity(wan.State),
						Message:   fmt.Sprintf("WAN %d (%s) state changed: %s", id, wan.Name, getStateName(wan.State)),
						Timestamp: time.Now(),
						Resolved:  false,
					})
				}
			}
			prevStates[id] = wan.State

			// Check latency
			if m.Latency > highLatencyThreshold {
				alertCounter++
				server.AddAlert(webui.Alert{
					ID:        fmt.Sprintf("alert-%d", alertCounter),
					Type:      "high_latency",
					Severity:  "warning",
					Message:   fmt.Sprintf("WAN %d (%s) high latency: %v (threshold: %v)", id, wan.Name, m.Latency, highLatencyThreshold),
					Timestamp: time.Now(),
					Resolved:  false,
				})
			}

			// Check jitter
			if m.Jitter > highJitterThreshold {
				alertCounter++
				server.AddAlert(webui.Alert{
					ID:        fmt.Sprintf("alert-%d", alertCounter),
					Type:      "high_jitter",
					Severity:  "warning",
					Message:   fmt.Sprintf("WAN %d (%s) high jitter: %v (threshold: %v)", id, wan.Name, m.Jitter, highJitterThreshold),
					Timestamp: time.Now(),
					Resolved:  false,
				})
			}

			// Check packet loss
			if m.PacketLoss > highPacketLossPercent {
				alertCounter++
				server.AddAlert(webui.Alert{
					ID:        fmt.Sprintf("alert-%d", alertCounter),
					Type:      "high_packet_loss",
					Severity:  "error",
					Message:   fmt.Sprintf("WAN %d (%s) high packet loss: %.2f%% (threshold: %.2f%%)", id, wan.Name, m.PacketLoss, highPacketLossPercent),
					Timestamp: time.Now(),
					Resolved:  false,
				})
			}
		}

		// Update NAT info if NAT manager is available
		natMgr := b.GetNATManager()
		if natMgr != nil {
			natInfo := &webui.NATInfo{
				NATType:        natMgr.GetNATType().String(),
				CGNATDetected:  natMgr.IsCGNATDetected(),
				CanDirect:      natMgr.GetNATType().CanDirectConnect(),
				NeedsRelay:     natMgr.GetNATType().NeedsRelay(),
				RelayAvailable: natMgr.GetRelayID() != "",
			}

			// Get public and local addresses
			if publicAddr := natMgr.GetPublicAddr(); publicAddr != nil {
				natInfo.PublicAddr = publicAddr.String()
			}
			if localAddr := natMgr.GetLocalAddr(); localAddr != nil {
				natInfo.LocalAddr = localAddr.String()
			}

			server.UpdateNATInfo(natInfo)
		}

		// Update flows if DPI classifier is available
		dpiClassifier := b.GetDPIClassifier()
		if dpiClassifier != nil {
			activeFlows := dpiClassifier.GetAllFlows()
			flows := make([]webui.FlowInfo, 0, len(activeFlows))

			for _, flow := range activeFlows {
				// Calculate duration
				duration := time.Since(flow.FirstSeen).Milliseconds()

				flowInfo := webui.FlowInfo{
					SrcIP:       flow.SrcIP.String(),
					DstIP:       flow.DstIP.String(),
					SrcPort:     flow.SrcPort,
					DstPort:     flow.DstPort,
					Protocol:    flow.Protocol.String(),
					Application: flow.Protocol.String(), // Use protocol as application name
					Category:    flow.Category.String(),
					Packets:     flow.Packets,
					Bytes:       flow.Bytes,
					Duration:    duration,
					FirstSeen:   flow.FirstSeen,
					LastSeen:    flow.LastSeen,
					WANID:       0, // TODO: Track which WAN this flow uses
				}

				flows = append(flows, flowInfo)
			}

			server.UpdateFlows(flows)
		}
	}
}

// getHealthStatus returns health status based on metrics
func getHealthStatus(m *protocol.WANMetrics) string {
	if m.PacketLoss > 10 {
		return "critical"
	}
	if m.PacketLoss > 5 || m.Latency > 200*time.Millisecond {
		return "degraded"
	}
	if m.Latency > 100*time.Millisecond {
		return "warning"
	}
	return "healthy"
}

// getAlertSeverity returns alert severity based on WAN state
func getAlertSeverity(state interface{}) string {
	switch s := state.(type) {
	case uint8:
		switch s {
		case 0: // Down
			return "error"
		case 3: // Degraded
			return "warning"
		default:
			return "info"
		}
	default:
		return "info"
	}
}

func printStats(b *bonder.Bonder) {
	metrics := b.GetMetrics()
	wans := b.GetWANs()

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("MultiWANBond Statistics")
	fmt.Println(strings.Repeat("=", 80))

	for id, wan := range wans {
		m := metrics[id]
		if m == nil {
			continue
		}

		fmt.Printf("\nWAN %d: %s (%s)\n", id, wan.Name, wan.Type)
		fmt.Printf("  State:        %s\n", getStateName(wan.State))
		fmt.Printf("  Latency:      %v (avg: %v)\n", m.Latency, m.AvgLatency)
		fmt.Printf("  Jitter:       %v (avg: %v)\n", m.Jitter, m.AvgJitter)
		fmt.Printf("  Packet Loss:  %.2f%% (avg: %.2f%%)\n", m.PacketLoss, m.AvgPacketLoss)
		fmt.Printf("  Packets:      Sent: %d, Recv: %d, Lost: %d\n",
			m.PacketsSent, m.PacketsRecv, m.PacketsLost)
		fmt.Printf("  Bandwidth:    Sent: %.2f MB, Recv: %.2f MB\n",
			float64(m.BytesSent)/1024/1024, float64(m.BytesReceived)/1024/1024)
		fmt.Printf("  Last Update:  %v\n", m.LastUpdate.Format("15:04:05"))
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
}

func getStateName(state interface{}) string {
	// Type assertion to handle the state
	switch s := state.(type) {
	case uint8:
		switch s {
		case 0:
			return "Down"
		case 1:
			return "Starting"
		case 2:
			return "Up"
		case 3:
			return "Degraded"
		case 4:
			return "Recovering"
		default:
			return "Unknown"
		}
	default:
		return fmt.Sprintf("%v", state)
	}
}

func printHelp() {
	fmt.Printf("MultiWANBond v%s - Multi-WAN Bonding Solution\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  multiwanbond [command] [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  setup              Run interactive setup wizard")
	fmt.Println("  (no command)       Run MultiWANBond server")
	fmt.Println("  version            Show version information")
	fmt.Println("  help               Show this help message")
	fmt.Println("")
	fmt.Println("Server Options:")
	fmt.Println("  --config <file>    Path to configuration file (default: configs/example.json)")
	fmt.Println("  --stats            Show statistics (default: true)")
	fmt.Println("  --stats-interval   Statistics display interval (default: 10s)")
	fmt.Println("")
	fmt.Println("Setup Options:")
	fmt.Println("  --config <file>    Path to save configuration file")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Run setup wizard")
	fmt.Println("  multiwanbond setup")
	fmt.Println("")
	fmt.Println("  # Run setup wizard with custom config path")
	fmt.Println("  multiwanbond setup --config /etc/multiwanbond/config.json")
	fmt.Println("")
	fmt.Println("  # Start server with custom config")
	fmt.Println("  multiwanbond --config /etc/multiwanbond/config.json")
	fmt.Println("")
	fmt.Println("  # Start server without statistics")
	fmt.Println("  multiwanbond --config config.json --stats=false")
	fmt.Println("")
}

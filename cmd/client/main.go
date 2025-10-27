package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/bonder"
	"github.com/thelastdreamer/MultiWANBond/pkg/config"
)

var (
	configFile = flag.String("config", "configs/example.json", "Path to configuration file")
	interactive = flag.Bool("interactive", true, "Interactive mode")
	message    = flag.String("message", "", "Single message to send (non-interactive)")
	count      = flag.Int("count", 1, "Number of times to send message (non-interactive)")
)

func main() {
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from %s", *configFile)
	cfg, err := config.LoadBondConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify remote endpoint is configured
	if cfg.Session.RemoteEndpoint == "" {
		log.Fatal("Remote endpoint not configured. Please set 'remote_endpoint' in config.")
	}

	// Create bonder
	log.Println("Creating MultiWANBond instance...")
	b, err := bonder.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create bonder: %v", err)
	}

	// Start bonding service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Println("Starting MultiWANBond client...")
	if err := b.Start(ctx); err != nil {
		log.Fatalf("Failed to start bonder: %v", err)
	}

	// Print WAN status
	wans := b.GetWANs()
	log.Printf("Active WANs: %d", len(wans))
	for _, wan := range wans {
		log.Printf("  - WAN %d (%s): %s @ %s -> %s",
			wan.ID, wan.Name, wan.Type, wan.LocalAddr, wan.RemoteAddr)
	}

	// Start receiver goroutine
	go receiver(b)

	// Handle termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if *interactive {
		// Interactive mode
		go interactiveMode(b)
		log.Println("Client is ready. Type messages to send (Ctrl+C to quit):")
	} else {
		// Non-interactive mode
		go nonInteractiveMode(b)
	}

	<-sigChan

	log.Println("Shutting down...")
	if err := b.Stop(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Client stopped")
}

func receiver(b *bonder.Bonder) {
	recvChan := b.Receive()
	for data := range recvChan {
		fmt.Printf("\n<< Received: %s\n", string(data))
		fmt.Print(">> ")
	}
}

func interactiveMode(b *bonder.Bonder) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print(">> ")

	for scanner.Scan() {
		text := scanner.Text()
		text = strings.TrimSpace(text)

		if text == "" {
			fmt.Print(">> ")
			continue
		}

		// Handle commands
		if strings.HasPrefix(text, "/") {
			handleCommand(b, text)
			fmt.Print(">> ")
			continue
		}

		// Send message
		if err := b.Send([]byte(text)); err != nil {
			log.Printf("Failed to send: %v", err)
		} else {
			fmt.Printf("Sent: %s\n", text)
		}

		fmt.Print(">> ")
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Scanner error: %v", err)
	}
}

func nonInteractiveMode(b *bonder.Bonder) {
	if *message == "" {
		log.Fatal("Message required in non-interactive mode. Use -message flag.")
	}

	// Give some time for connections to establish
	time.Sleep(1 * time.Second)

	for i := 0; i < *count; i++ {
		msg := fmt.Sprintf("%s [%d/%d]", *message, i+1, *count)
		if err := b.Send([]byte(msg)); err != nil {
			log.Printf("Failed to send message %d: %v", i+1, err)
		} else {
			log.Printf("Sent message %d/%d", i+1, *count)
		}

		if i < *count-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Wait a bit for responses
	time.Sleep(2 * time.Second)
}

func handleCommand(b *bonder.Bonder, cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/help":
		printHelp()

	case "/status":
		printStatus(b)

	case "/wans":
		printWANs(b)

	case "/metrics":
		printMetrics(b)

	case "/quit", "/exit":
		log.Println("Exiting...")
		os.Exit(0)

	default:
		fmt.Printf("Unknown command: %s (type /help for help)\n", parts[0])
	}
}

func printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  /help      - Show this help")
	fmt.Println("  /status    - Show connection status")
	fmt.Println("  /wans      - Show WAN interfaces")
	fmt.Println("  /metrics   - Show detailed metrics")
	fmt.Println("  /quit      - Exit client")
	fmt.Println()
}

func printStatus(b *bonder.Bonder) {
	session := b.GetSession()
	wans := b.GetWANs()

	fmt.Println("\nConnection Status:")
	fmt.Printf("  Session ID:       %d\n", session.ID)
	fmt.Printf("  Local Endpoint:   %s\n", session.LocalEndpoint)
	fmt.Printf("  Remote Endpoint:  %s\n", session.RemoteEndpoint)
	fmt.Printf("  Active WANs:      %d\n", len(wans))
	fmt.Printf("  Uptime:           %v\n", time.Since(session.StartTime).Round(time.Second))
	fmt.Println()
}

func printWANs(b *bonder.Bonder) {
	wans := b.GetWANs()

	fmt.Println("\nWAN Interfaces:")
	for _, wan := range wans {
		fmt.Printf("  WAN %d: %s (%s)\n", wan.ID, wan.Name, wan.Type)
		fmt.Printf("    Local:   %s\n", wan.LocalAddr)
		fmt.Printf("    Remote:  %s\n", wan.RemoteAddr)
		fmt.Printf("    State:   %v\n", wan.State)
		fmt.Printf("    Enabled: %v\n", wan.Config.Enabled)
		fmt.Println()
	}
}

func printMetrics(b *bonder.Bonder) {
	metrics := b.GetMetrics()
	wans := b.GetWANs()

	fmt.Println("\nDetailed Metrics:")
	for id, wan := range wans {
		m := metrics[id]
		if m == nil {
			fmt.Printf("  WAN %d: No metrics available\n", id)
			continue
		}

		fmt.Printf("  WAN %d: %s (%s)\n", id, wan.Name, wan.Type)
		fmt.Printf("    Latency:      %v (avg: %v)\n", m.Latency, m.AvgLatency)
		fmt.Printf("    Jitter:       %v (avg: %v)\n", m.Jitter, m.AvgJitter)
		fmt.Printf("    Packet Loss:  %.2f%% (avg: %.2f%%)\n", m.PacketLoss, m.AvgPacketLoss)
		fmt.Printf("    Bandwidth:    %.2f Mbps\n", float64(m.Bandwidth)*8/1024/1024)
		fmt.Printf("    Packets:      Sent: %d, Recv: %d, Lost: %d\n",
			m.PacketsSent, m.PacketsRecv, m.PacketsLost)
		fmt.Printf("    Data:         Sent: %.2f MB, Recv: %.2f MB\n",
			float64(m.BytesSent)/1024/1024, float64(m.BytesReceived)/1024/1024)
		fmt.Println()
	}
}

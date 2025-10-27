package main

import (
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
	showStats  = flag.Bool("stats", true, "Show statistics")
	statsInterval = flag.Duration("stats-interval", 10*time.Second, "Statistics interval")
)

func main() {
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from %s", *configFile)
	cfg, err := config.LoadBondConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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

	log.Println("Starting MultiWANBond service...")
	if err := b.Start(ctx); err != nil {
		log.Fatalf("Failed to start bonder: %v", err)
	}

	// Print WAN status
	wans := b.GetWANs()
	log.Printf("Active WANs: %d", len(wans))
	for _, wan := range wans {
		log.Printf("  - WAN %d (%s): %s @ %s", wan.ID, wan.Name, wan.Type, wan.LocalAddr)
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

	log.Println("MultiWANBond server is running. Press Ctrl+C to stop.")
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

	for range ticker.C {
		printStats(b)
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

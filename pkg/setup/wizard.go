package setup

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/thelastdreamer/MultiWANBond/pkg/network"
)

// Wizard provides interactive setup
type Wizard struct {
	scanner  *bufio.Scanner
	detector *network.UniversalDetector
}

// NewWizard creates a new setup wizard
func NewWizard() (*Wizard, error) {
	detector, err := network.NewDetector()
	if err != nil {
		return nil, fmt.Errorf("failed to create network detector: %w", err)
	}

	return &Wizard{
		scanner:  bufio.NewScanner(os.Stdin),
		detector: detector,
	}, nil
}

// GetDetector returns the network detector
func (w *Wizard) GetDetector() (*network.UniversalDetector, error) {
	if w.detector == nil {
		return nil, fmt.Errorf("detector not initialized")
	}
	return w.detector, nil
}

// Run starts the interactive setup wizard
func (w *Wizard) Run() (*Config, error) {
	fmt.Println()
	printHeader("MultiWANBond Setup Wizard")
	fmt.Println()
	fmt.Println("This wizard will help you set up MultiWANBond in just a few steps.")
	fmt.Println()

	config := &Config{
		Version: "1.0",
	}

	// Step 1: Select mode
	mode, err := w.selectMode()
	if err != nil {
		return nil, err
	}
	config.Mode = mode

	// Step 2: Select network interfaces
	interfaces, err := w.selectInterfaces()
	if err != nil {
		return nil, err
	}

	// Step 3: Configure WANs
	wans, err := w.configureWANs(interfaces, mode)
	if err != nil {
		return nil, err
	}
	config.WANs = wans

	// Step 4: Configure server settings (if needed)
	if mode != ModeStandalone {
		server, err := w.configureServer(mode)
		if err != nil {
			return nil, err
		}
		config.Server = server
	}

	// Step 5: Security settings
	security, err := w.configureSecurity()
	if err != nil {
		return nil, err
	}
	config.Security = security

	// Step 6: Generate Web UI credentials
	webui, err := w.configureWebUI()
	if err != nil {
		return nil, err
	}
	config.WebUI = webui

	// Step 7: Review and confirm
	fmt.Println()
	printHeader("Configuration Summary")
	fmt.Println()
	w.printSummary(config)
	fmt.Println()

	if !w.confirm("Save this configuration?") {
		return nil, fmt.Errorf("setup cancelled by user")
	}

	return config, nil
}

// selectMode prompts user to select operation mode
func (w *Wizard) selectMode() (Mode, error) {
	fmt.Println()
	printSection("Step 1: Select Operation Mode")
	fmt.Println()
	fmt.Println("  1. Standalone   - Run on a single machine (testing/development)")
	fmt.Println("  2. Client       - Connect to a remote server")
	fmt.Println("  3. Server       - Accept connections from clients")
	fmt.Println()

	for {
		choice := w.prompt("Select mode [1-3]")

		switch choice {
		case "1":
			return ModeStandalone, nil
		case "2":
			return ModeClient, nil
		case "3":
			return ModeServer, nil
		default:
			fmt.Println("Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}

// selectInterfaces prompts user to select network interfaces
func (w *Wizard) selectInterfaces() ([]*network.NetworkInterface, error) {
	fmt.Println()
	printSection("Step 2: Select Network Interfaces")
	fmt.Println()
	fmt.Println("Detecting available network interfaces...")
	fmt.Println()

	allInterfaces, err := w.detector.DetectAll()
	if err != nil {
		return nil, fmt.Errorf("failed to detect interfaces: %w", err)
	}

	// Filter to usable interfaces (physical, up, has IP)
	var usable []*network.NetworkInterface
	for _, iface := range allInterfaces {
		if w.isUsable(iface) {
			usable = append(usable, iface)
		}
	}

	if len(usable) == 0 {
		return nil, fmt.Errorf("no usable network interfaces found")
	}

	// Display available interfaces
	fmt.Println("Available network interfaces:")
	fmt.Println()
	for i, iface := range usable {
		status := "DOWN"
		if iface.OperState == "up" {
			status = "UP"
		}

		fmt.Printf("  %d. %s\n", i+1, iface.SystemName)
		fmt.Printf("     Status: %s | Type: %s\n", status, iface.Type)

		if len(iface.IPv4Addresses) > 0 {
			// Convert []net.IP to []string
			var ipStrs []string
			for _, ip := range iface.IPv4Addresses {
				ipStrs = append(ipStrs, ip.String())
			}
			fmt.Printf("     IPv4: %s\n", strings.Join(ipStrs, ", "))
		}
		if iface.Speed > 0 {
			fmt.Printf("     Speed: %d Mbps\n", iface.Speed/1000000)
		}
		fmt.Println()
	}

	// Prompt for selection
	fmt.Println("Select interfaces to use for WAN bonding.")
	fmt.Println("Enter numbers separated by commas (e.g., 1,2,3)")
	fmt.Println()

	for {
		input := w.prompt("Select interfaces")
		selected, err := w.parseSelection(input, len(usable))
		if err != nil {
			fmt.Printf("Invalid selection: %v\n", err)
			continue
		}

		if len(selected) == 0 {
			fmt.Println("You must select at least one interface.")
			continue
		}

		// Build result
		var result []*network.NetworkInterface
		for _, idx := range selected {
			result = append(result, usable[idx])
		}

		return result, nil
	}
}

// configureWANs creates WAN configurations from selected interfaces
func (w *Wizard) configureWANs(interfaces []*network.NetworkInterface, mode Mode) ([]*WANConfig, error) {
	fmt.Println()
	printSection("Step 3: Configure WAN Interfaces")
	fmt.Println()

	var wans []*WANConfig

	for i, iface := range interfaces {
		fmt.Printf("Configuring WAN %d: %s\n", i+1, iface.SystemName)
		fmt.Println()

		wan := &WANConfig{
			ID:        uint8(i + 1),
			Name:      fmt.Sprintf("WAN%d", i+1),
			Interface: iface.SystemName,
			Enabled:   true,
			Weight:    100,
		}

		// Get friendly name
		defaultName := fmt.Sprintf("WAN%d-%s", i+1, iface.SystemName)
		name := w.promptWithDefault("  Friendly name", defaultName)
		if name != "" {
			wan.Name = name
		}

		// Get weight
		weightStr := w.promptWithDefault("  Weight (1-1000)", "100")
		if weight, err := strconv.Atoi(weightStr); err == nil && weight > 0 && weight <= 1000 {
			wan.Weight = weight
		}

		wans = append(wans, wan)
		fmt.Println()
	}

	return wans, nil
}

// configureServer configures server/client settings
func (w *Wizard) configureServer(mode Mode) (*ServerConfig, error) {
	fmt.Println()
	printSection("Step 4: Server Configuration")
	fmt.Println()

	config := &ServerConfig{
		ListenPort: 9000,
	}

	if mode == ModeServer {
		fmt.Println("Configure server listening address:")
		fmt.Println()

		addr := w.promptWithDefault("  Listen address", "0.0.0.0")
		config.ListenAddress = addr

		portStr := w.promptWithDefault("  Listen port", "9000")
		if port, err := strconv.Atoi(portStr); err == nil && port > 0 && port <= 65535 {
			config.ListenPort = port
		}
	} else if mode == ModeClient {
		fmt.Println("Configure remote server address:")
		fmt.Println()
		fmt.Println("  Leave empty to configure later.")
		fmt.Println()

		addr := w.prompt("  Server address (e.g., server.example.com:9000)")
		if addr != "" {
			config.RemoteAddress = addr
		} else {
			fmt.Println()
			fmt.Println("  ⚠ No server address configured.")
			fmt.Println("    You can add it later by editing the config file.")
		}
	}

	return config, nil
}

// configureSecurity configures security settings
func (w *Wizard) configureSecurity() (*SecurityConfig, error) {
	fmt.Println()
	printSection("Step 5: Security Settings")
	fmt.Println()

	config := &SecurityConfig{
		EncryptionEnabled: true,
		EncryptionType:    "chacha20poly1305",
	}

	if w.confirm("Enable encryption? (recommended)") {
		fmt.Println()
		fmt.Println("  1. ChaCha20-Poly1305 (fast, recommended)")
		fmt.Println("  2. AES-256-GCM (hardware accelerated)")
		fmt.Println()

		choice := w.promptWithDefault("Select encryption [1-2]", "1")
		if choice == "2" {
			config.EncryptionType = "aes256gcm"
		}

		fmt.Println()
		fmt.Println("A pre-shared key is required for encryption.")
		fmt.Println("This must be the same on client and server.")
		fmt.Println()

		key := w.prompt("  Enter pre-shared key (leave empty to generate)")
		if key == "" {
			key = generateRandomKey(32)
			fmt.Printf("  Generated key: %s\n", key)
			fmt.Println("  ⚠ Save this key! You'll need it for the other side.")
		}
		config.PreSharedKey = key
	} else {
		config.EncryptionEnabled = false
		fmt.Println()
		fmt.Println("  ⚠ Encryption disabled. All traffic will be unencrypted!")
	}

	return config, nil
}

// configureWebUI generates Web UI credentials
func (w *Wizard) configureWebUI() (*WebUIConfig, error) {
	fmt.Println()
	printSection("Step 6: Web UI Security")
	fmt.Println()

	config := &WebUIConfig{
		Username: "admin",
		Enabled:  true,
	}

	fmt.Println("Generating secure password for Web UI access...")
	fmt.Println()

	password, err := generatePassword(16)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	config.Password = password

	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println("  ⚠️  IMPORTANT: Web UI Credentials - SAVE THESE SECURELY!")
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Println("  Web UI URL:  http://localhost:8080")
	fmt.Println("  Username:    admin")
	fmt.Printf("  Password:    %s\n", password)
	fmt.Println()
	fmt.Println("  ⚠️  Write this password down NOW!")
	fmt.Println("  ⚠️  You'll need it to access the Web UI dashboard.")
	fmt.Println("  ⚠️  This password will be saved to your config file.")
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	fmt.Print("Press Enter to continue...")
	w.scanner.Scan()

	return config, nil
}

// Helper functions

func (w *Wizard) isUsable(iface *network.NetworkInterface) bool {
	// Must be physical or virtual (not loopback)
	if iface.Type == network.InterfaceLoopback {
		return false
	}

	// Must be up
	if iface.OperState != "up" {
		return false
	}

	// Must have at least one IP address
	if len(iface.IPv4Addresses) == 0 && len(iface.IPv6Addresses) == 0 {
		return false
	}

	return true
}

func (w *Wizard) parseSelection(input string, max int) ([]int, error) {
	parts := strings.Split(input, ",")
	var result []int

	for _, part := range parts {
		part = strings.TrimSpace(part)
		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid number: %s", part)
		}

		if num < 1 || num > max {
			return nil, fmt.Errorf("number out of range: %d (must be 1-%d)", num, max)
		}

		result = append(result, num-1) // Convert to 0-indexed
	}

	return result, nil
}

func (w *Wizard) prompt(message string) string {
	fmt.Printf("%s: ", message)
	w.scanner.Scan()
	return strings.TrimSpace(w.scanner.Text())
}

func (w *Wizard) promptWithDefault(message, defaultValue string) string {
	fmt.Printf("%s [%s]: ", message, defaultValue)
	w.scanner.Scan()
	value := strings.TrimSpace(w.scanner.Text())
	if value == "" {
		return defaultValue
	}
	return value
}

func (w *Wizard) confirm(message string) bool {
	for {
		response := w.prompt(message + " [Y/n]")
		response = strings.ToLower(response)

		if response == "" || response == "y" || response == "yes" {
			return true
		}
		if response == "n" || response == "no" {
			return false
		}

		fmt.Println("Please answer 'y' or 'n'")
	}
}

func (w *Wizard) printSummary(config *Config) {
	fmt.Printf("Mode:          %s\n", config.Mode)
	fmt.Printf("WAN Count:     %d\n", len(config.WANs))
	fmt.Println()

	fmt.Println("WAN Interfaces:")
	for _, wan := range config.WANs {
		status := "enabled"
		if !wan.Enabled {
			status = "disabled"
		}
		fmt.Printf("  - %s (%s) - weight: %d - %s\n",
			wan.Name, wan.Interface, wan.Weight, status)
	}
	fmt.Println()

	if config.Server != nil {
		if config.Mode == ModeServer {
			fmt.Printf("Server:        %s:%d\n", config.Server.ListenAddress, config.Server.ListenPort)
		} else if config.Mode == ModeClient {
			if config.Server.RemoteAddress != "" {
				fmt.Printf("Remote Server: %s\n", config.Server.RemoteAddress)
			} else {
				fmt.Println("Remote Server: (not configured)")
			}
		}
		fmt.Println()
	}

	if config.Security != nil {
		if config.Security.EncryptionEnabled {
			fmt.Printf("Encryption:    %s\n", config.Security.EncryptionType)
		} else {
			fmt.Println("Encryption:    disabled")
		}
	}

	fmt.Println()
	if config.WebUI != nil && config.WebUI.Enabled {
		fmt.Println("Web UI Access:")
		fmt.Println("  URL:      http://localhost:8080")
		fmt.Printf("  Username: %s\n", config.WebUI.Username)
		fmt.Printf("  Password: %s (saved in config)\n", config.WebUI.Password)
	}
}

func printHeader(title string) {
	line := strings.Repeat("=", 60)
	fmt.Println(line)
	fmt.Printf("%s\n", title)
	fmt.Println(line)
}

func printSection(title string) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("-", 60))
}

func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based if crypto/rand fails (shouldn't happen)
		panic(fmt.Sprintf("failed to generate random key: %v", err))
	}
	// Use URL-safe base64 encoding and trim to desired length
	encoded := base64.URLEncoding.EncodeToString(bytes)
	if len(encoded) > length {
		return encoded[:length]
	}
	return encoded
}

// generatePassword generates a cryptographically secure random password
func generatePassword(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	for i := 0; i < length; i++ {
		result[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(result), nil
}

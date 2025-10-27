package network

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Detector interface defines cross-platform network interface detection
type Detector interface {
	// DetectAll detects all network interfaces
	DetectAll() ([]*NetworkInterface, error)

	// DetectByName detects a specific interface by system name
	DetectByName(name string) (*NetworkInterface, error)

	// GetCapabilities returns capabilities of an interface
	GetCapabilities(name string) (*InterfaceCapabilities, error)

	// TestConnectivity tests internet connectivity on an interface
	TestConnectivity(ifaceName string, target string, method string) (*ConnectivityTest, error)

	// Monitor starts monitoring for interface changes
	Monitor(ctx context.Context) (<-chan *InterfaceChange, error)
}

// UniversalDetector provides cross-platform network detection
type UniversalDetector struct {
	mu             sync.RWMutex
	platformImpl   Detector
	cachedIfaces   map[string]*NetworkInterface
	lastUpdate     time.Time
	cacheTimeout   time.Duration
}

// newPlatformDetector is implemented by platform-specific files
// (detector_init_linux.go, detector_init_windows.go, detector_init_darwin.go)

// NewDetector creates a new cross-platform detector
func NewDetector() (*UniversalDetector, error) {
	impl, err := newPlatformDetector()
	if err != nil {
		return nil, fmt.Errorf("failed to create platform detector: %w", err)
	}

	return &UniversalDetector{
		platformImpl: impl,
		cachedIfaces: make(map[string]*NetworkInterface),
		cacheTimeout: 5 * time.Second,
	}, nil
}

// DetectAll detects all network interfaces
func (d *UniversalDetector) DetectAll() ([]*NetworkInterface, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check cache
	if time.Since(d.lastUpdate) < d.cacheTimeout && len(d.cachedIfaces) > 0 {
		result := make([]*NetworkInterface, 0, len(d.cachedIfaces))
		for _, iface := range d.cachedIfaces {
			result = append(result, iface)
		}
		return result, nil
	}

	// Detect using platform implementation
	ifaces, err := d.platformImpl.DetectAll()
	if err != nil {
		return nil, err
	}

	// Update cache
	d.cachedIfaces = make(map[string]*NetworkInterface)
	for _, iface := range ifaces {
		d.cachedIfaces[iface.SystemName] = iface
	}
	d.lastUpdate = time.Now()

	return ifaces, nil
}

// DetectByName detects a specific interface
func (d *UniversalDetector) DetectByName(name string) (*NetworkInterface, error) {
	d.mu.RLock()
	// Check cache first
	if iface, ok := d.cachedIfaces[name]; ok && time.Since(d.lastUpdate) < d.cacheTimeout {
		d.mu.RUnlock()
		return iface, nil
	}
	d.mu.RUnlock()

	return d.platformImpl.DetectByName(name)
}

// GetCapabilities returns interface capabilities
func (d *UniversalDetector) GetCapabilities(name string) (*InterfaceCapabilities, error) {
	return d.platformImpl.GetCapabilities(name)
}

// TestConnectivity tests internet connectivity
func (d *UniversalDetector) TestConnectivity(ifaceName string, target string, method string) (*ConnectivityTest, error) {
	if target == "" {
		target = "8.8.8.8" // Default to Google DNS
	}
	if method == "" {
		method = "ping" // Default method
	}

	return d.platformImpl.TestConnectivity(ifaceName, target, method)
}

// Monitor monitors interface changes
func (d *UniversalDetector) Monitor(ctx context.Context) (<-chan *InterfaceChange, error) {
	return d.platformImpl.Monitor(ctx)
}

// ClearCache clears the interface cache
func (d *UniversalDetector) ClearCache() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cachedIfaces = make(map[string]*NetworkInterface)
	d.lastUpdate = time.Time{}
}

// Helper functions

// GetDefaultGateway returns the default gateway for the system
func GetDefaultGateway() (net.IP, error) {
	// This is a simplified version
	// Platform-specific implementations will provide more accurate results
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// IsInterfaceUp checks if an interface is up
func IsInterfaceUp(iface *NetworkInterface) bool {
	return iface.AdminState == "up" && iface.OperState == "up" && iface.HasCarrier
}

// IsInterfaceUsable checks if an interface is usable for bonding
func IsInterfaceUsable(iface *NetworkInterface) bool {
	// Must be up, have a carrier, and have an IP
	return IsInterfaceUp(iface) && iface.HasIP && iface.Type != InterfaceLoopback
}

// FilterInterfaces filters interfaces by type
func FilterInterfaces(ifaces []*NetworkInterface, filterFunc func(*NetworkInterface) bool) []*NetworkInterface {
	result := make([]*NetworkInterface, 0)
	for _, iface := range ifaces {
		if filterFunc(iface) {
			result = append(result, iface)
		}
	}
	return result
}

// GetPhysicalInterfaces returns only physical interfaces
func GetPhysicalInterfaces(ifaces []*NetworkInterface) []*NetworkInterface {
	return FilterInterfaces(ifaces, func(i *NetworkInterface) bool {
		return i.Type == InterfacePhysical
	})
}

// GetUsableInterfaces returns interfaces usable for WAN bonding
func GetUsableInterfaces(ifaces []*NetworkInterface) []*NetworkInterface {
	return FilterInterfaces(ifaces, IsInterfaceUsable)
}

// GetInterfaceByName finds an interface by system name
func GetInterfaceByName(ifaces []*NetworkInterface, name string) *NetworkInterface {
	for _, iface := range ifaces {
		if iface.SystemName == name {
			return iface
		}
	}
	return nil
}

// SortInterfacesBySpeed sorts interfaces by speed (descending)
func SortInterfacesBySpeed(ifaces []*NetworkInterface) {
	// Simple bubble sort for small lists
	for i := 0; i < len(ifaces)-1; i++ {
		for j := 0; j < len(ifaces)-i-1; j++ {
			if ifaces[j].Speed < ifaces[j+1].Speed {
				ifaces[j], ifaces[j+1] = ifaces[j+1], ifaces[j]
			}
		}
	}
}

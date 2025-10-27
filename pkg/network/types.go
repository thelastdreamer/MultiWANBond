package network

import (
	"net"
	"time"
)

// InterfaceType defines the type of network interface
type InterfaceType string

const (
	InterfacePhysical InterfaceType = "physical"
	InterfaceVLAN     InterfaceType = "vlan"
	InterfaceBridge   InterfaceType = "bridge"
	InterfaceBond     InterfaceType = "bond"
	InterfaceTunnel   InterfaceType = "tunnel"
	InterfacePPPoE    InterfaceType = "pppoe"
	InterfaceVirtual  InterfaceType = "virtual"
	InterfaceLoopback InterfaceType = "loopback"
)

// DuplexMode represents the duplex mode of an interface
type DuplexMode string

const (
	DuplexFull    DuplexMode = "full"
	DuplexHalf    DuplexMode = "half"
	DuplexUnknown DuplexMode = "unknown"
)

// NetworkInterface represents a detected network interface
type NetworkInterface struct {
	// System identifiers
	SystemName string        // e.g., "eth0", "wlan0"
	Index      int           // Interface index
	Type       InterfaceType // Interface type

	// Display (user-configurable)
	DisplayName string // e.g., "Office Fiber Primary"
	Description string // User notes

	// Hardware info
	MACAddress string
	Driver     string
	PCIAddress string // For physical NICs
	Vendor     string
	Model      string

	// Capabilities
	Speed  uint64     // Link speed in bps (0 if unknown)
	Duplex DuplexMode // Full, half, or unknown
	MTU    int        // Maximum transmission unit

	// State
	Enabled    bool
	AdminState string // up, down
	OperState  string // up, down, testing, unknown
	Flags      []string

	// IP configuration
	IPv4Addresses []net.IP
	IPv6Addresses []net.IP
	IPv4Gateway   net.IP
	IPv6Gateway   net.IP

	// Connectivity
	HasCarrier  bool          // Physical link detected
	HasIP       bool          // Has IP address assigned
	HasInternet bool          // Can reach internet
	TestLatency time.Duration // Latency to test target (0 if no internet)

	// VLAN specific (if Type == InterfaceVLAN)
	VLANInfo *VLANInfo

	// Bond specific (if Type == InterfaceBond)
	BondInfo *BondInfo

	// Bridge specific (if Type == InterfaceBridge)
	BridgeInfo *BridgeInfo

	// Statistics
	RxBytes   uint64
	TxBytes   uint64
	RxPackets uint64
	TxPackets uint64
	RxErrors  uint64
	TxErrors  uint64
	RxDropped uint64
	TxDropped uint64

	// Timestamps
	DetectedAt time.Time
	UpdatedAt  time.Time
}

// VLANInfo contains VLAN-specific information
type VLANInfo struct {
	ID       int    // VLAN ID (1-4094)
	Parent   string // Parent interface name
	Priority uint8  // 802.1p priority (0-7)
}

// BondInfo contains bonding-specific information
type BondInfo struct {
	Mode   string   // bonding mode (802.3ad, active-backup, etc.)
	Slaves []string // Slave interface names
	Active string   // Currently active slave
}

// BridgeInfo contains bridge-specific information
type BridgeInfo struct {
	Members []string // Bridge member interfaces
	STP     bool     // Spanning Tree Protocol enabled
}

// InterfaceCapabilities represents what an interface can do
type InterfaceCapabilities struct {
	SupportsVLAN    bool
	SupportsBonding bool
	SupportsBridge  bool
	SupportsTSO     bool // TCP Segmentation Offload
	SupportsGSO     bool // Generic Segmentation Offload
	SupportsGRO     bool // Generic Receive Offload
	SupportsLRO     bool // Large Receive Offload
}

// ConnectivityTest represents an internet connectivity test result
type ConnectivityTest struct {
	Interface   string
	Target      string
	Method      string // ping, http, dns
	Success     bool
	Latency     time.Duration
	Error       string
	TestedAt    time.Time
}

// InterfaceChange represents a state change event
type InterfaceChange struct {
	InterfaceName string
	ChangeType    ChangeType
	OldState      string
	NewState      string
	Timestamp     time.Time
}

// ChangeType defines types of interface changes
type ChangeType string

const (
	ChangeAdded         ChangeType = "added"
	ChangeRemoved       ChangeType = "removed"
	ChangeAdminState    ChangeType = "admin_state"
	ChangeOperState     ChangeType = "oper_state"
	ChangeCarrier       ChangeType = "carrier"
	ChangeIPAddress     ChangeType = "ip_address"
	ChangeMTU           ChangeType = "mtu"
	ChangeSpeed         ChangeType = "speed"
)

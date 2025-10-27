package bridge

import (
	"time"
)

// STPState defines the Spanning Tree Protocol state
type STPState string

const (
	// STPStateDisabled STP is disabled
	STPStateDisabled STPState = "disabled"

	// STPStateListening bridge is listening for BPDUs
	STPStateListening STPState = "listening"

	// STPStateLearning bridge is learning MAC addresses
	STPStateLearning STPState = "learning"

	// STPStateForwarding bridge is forwarding frames
	STPStateForwarding STPState = "forwarding"

	// STPStateBlocking bridge port is blocking
	STPStateBlocking STPState = "blocking"
)

// BridgeProtocol defines the bridging protocol
type BridgeProtocol string

const (
	// BridgeProtocolSTP Standard Spanning Tree Protocol (802.1D)
	BridgeProtocolSTP BridgeProtocol = "stp"

	// BridgeProtocolRSTP Rapid Spanning Tree Protocol (802.1w)
	BridgeProtocolRSTP BridgeProtocol = "rstp"

	// BridgeProtocolMSTP Multiple Spanning Tree Protocol (802.1s)
	BridgeProtocolMSTP BridgeProtocol = "mstp"

	// BridgeProtocolNone No spanning tree protocol
	BridgeProtocolNone BridgeProtocol = "none"
)

// AGingMethod defines how MAC addresses are aged
type AgeingMethod string

const (
	// AgeingMethodTime age entries based on time
	AgeingMethodTime AgeingMethod = "time"

	// AgeingMethodTraffic age entries based on traffic activity
	AgeingMethodTraffic AgeingMethod = "traffic"
)

// BridgeConfig represents a bridge interface configuration
type BridgeConfig struct {
	// Name of the bridge interface (e.g., "br0")
	Name string

	// Ports is a list of interface names to add to the bridge
	Ports []string

	// STPEnabled enables Spanning Tree Protocol
	STPEnabled bool

	// STPProtocol defines which STP variant to use
	STPProtocol BridgeProtocol

	// STPPriority is the bridge priority (0-65535, lower is better)
	// Default: 32768
	STPPriority int

	// STPForwardDelay in seconds (4-30)
	// Time spent in listening and learning states
	// Default: 15 seconds
	STPForwardDelay int

	// STPHelloTime in seconds (1-10)
	// Time between sending BPDUs
	// Default: 2 seconds
	STPHelloTime int

	// STPMaxAge in seconds (6-40)
	// Maximum age of received BPDUs
	// Default: 20 seconds
	STPMaxAge int

	// AgeingTime in seconds (0-1000000)
	// Time before MAC address entries are removed
	// Default: 300 seconds (5 minutes)
	AgeingTime int

	// AgeingMethod defines how to age MAC addresses
	AgeingMethod AgeingMethod

	// VLANFiltering enables VLAN filtering on the bridge
	VLANFiltering bool

	// VLANDefaultPVID is the default VLAN ID for untagged traffic
	// Default: 1
	VLANDefaultPVID int

	// MulticastSnooping enables IGMP/MLD snooping
	MulticastSnooping bool

	// MulticastQuerier enables multicast querier
	MulticastQuerier bool

	// MulticastRouter enables multicast router discovery
	MulticastRouter bool

	// HashMax maximum size of multicast hash table (1-4096)
	// Default: 512
	HashMax int

	// HashElasticity multicast hash elasticity (0-16)
	// Default: 4
	HashElasticity int

	// MulticastLastMemberCount number of queries before stopping
	// Default: 2
	MulticastLastMemberCount int

	// MulticastStartupQueryCount startup queries to send
	// Default: 2
	MulticastStartupQueryCount int

	// GroupFwdMask bitmask of forwarded group addresses
	// Default: 0 (forward none)
	GroupFwdMask int

	// MTU for the bridge interface
	MTU int

	// MACAddress for the bridge (if empty, uses lowest MAC of member ports)
	MACAddress string

	// Enabled indicates if the interface should be up
	Enabled bool
}

// BridgeInfo represents runtime information about a bridge
type BridgeInfo struct {
	// Name of the bridge
	Name string

	// State of the bridge (up/down)
	State string

	// MACAddress of the bridge
	MACAddress string

	// MTU of the bridge
	MTU int

	// Ports currently attached
	Ports []PortInfo

	// STPEnabled indicates if STP is enabled
	STPEnabled bool

	// STPProtocol currently configured
	STPProtocol BridgeProtocol

	// STPState current STP state
	STPState STPState

	// RootID bridge ID of root bridge
	RootID string

	// RootPriority priority of root bridge
	RootPriority int

	// BridgeID this bridge's ID
	BridgeID string

	// BridgePriority this bridge's priority
	BridgePriority int

	// DesignatedRoot designated root bridge
	DesignatedRoot string

	// RootPort port leading to root
	RootPort string

	// RootPathCost cost of path to root
	RootPathCost int

	// AgeingTime configured
	AgeingTime int

	// VLANFiltering status
	VLANFiltering bool

	// VLANDefaultPVID configured
	VLANDefaultPVID int

	// MulticastSnooping status
	MulticastSnooping bool

	// MACCount number of MAC addresses in forwarding database
	MACCount int

	// Created timestamp
	Created time.Time

	// LastModified timestamp
	LastModified time.Time
}

// PortInfo represents information about a bridge port
type PortInfo struct {
	// Name of the port interface
	Name string

	// State of the port (up/down)
	State string

	// STPState current STP state for this port
	STPState STPState

	// Priority port priority (0-63)
	Priority int

	// PathCost STP path cost for this port
	PathCost int

	// Designated indicates if this is a designated port
	Designated bool

	// DesignatedRoot designated root for this port
	DesignatedRoot string

	// DesignatedBridge designated bridge for this port
	DesignatedBridge string

	// DesignatedPort designated port ID
	DesignatedPort string

	// DesignatedCost cost to designated root
	DesignatedCost int

	// IsEdge indicates if this is an edge port (connects to end device)
	IsEdge bool

	// FastLeave enables fast leave for multicast
	FastLeave bool

	// Guard BPDU guard status
	Guard bool

	// Hairpin mode (allow frames to exit same port they entered)
	Hairpin bool

	// Learning enables MAC learning on this port
	Learning bool

	// Flood enables flooding on this port
	Flood bool

	// ProxyARP enables proxy ARP on this port
	ProxyARP bool

	// ProxyARPWiFi enables proxy ARP for WiFi on this port
	ProxyARPWiFi bool

	// Isolated port (can't communicate with other isolated ports)
	Isolated bool
}

// FDBEntry represents a forwarding database entry
type FDBEntry struct {
	// MACAddress
	MACAddress string

	// Port interface name
	Port string

	// VLANID (0 if VLAN filtering disabled)
	VLANID int

	// IsLocal indicates if this is a local address
	IsLocal bool

	// IsStatic indicates if this is a static entry
	IsStatic bool

	// AgingTimer time until entry expires (seconds)
	AgingTimer int

	// LastUsed timestamp of last use
	LastUsed time.Time
}

// BridgeStats represents statistics for a bridge
type BridgeStats struct {
	// Name of the bridge
	Name string

	// RXBytes received
	RXBytes uint64

	// RXPackets received
	RXPackets uint64

	// RXErrors
	RXErrors uint64

	// RXDropped
	RXDropped uint64

	// TXBytes transmitted
	TXBytes uint64

	// TXPackets transmitted
	TXPackets uint64

	// TXErrors
	TXErrors uint64

	// TXDropped
	TXDropped uint64

	// MulticastRX received multicast packets
	MulticastRX uint64

	// MulticastTX transmitted multicast packets
	MulticastTX uint64

	// PortStats per-port statistics
	PortStats map[string]*PortStats
}

// PortStats represents statistics for a bridge port
type PortStats struct {
	// Name of the port
	Name string

	// RXBytes received
	RXBytes uint64

	// RXPackets received
	RXPackets uint64

	// RXErrors
	RXErrors uint64

	// TXBytes transmitted
	TXBytes uint64

	// TXPackets transmitted
	TXPackets uint64

	// TXErrors
	TXErrors uint64

	// Dropped packets
	Dropped uint64

	// Multicast packets
	Multicast uint64
}

// DefaultBridgeConfig returns a default bridge configuration
func DefaultBridgeConfig(name string) *BridgeConfig {
	return &BridgeConfig{
		Name:                       name,
		Ports:                      []string{},
		STPEnabled:                 false, // Disabled by default
		STPProtocol:                BridgeProtocolRSTP,
		STPPriority:                32768,
		STPForwardDelay:            15,
		STPHelloTime:               2,
		STPMaxAge:                  20,
		AgeingTime:                 300,
		AgeingMethod:               AgeingMethodTime,
		VLANFiltering:              false,
		VLANDefaultPVID:            1,
		MulticastSnooping:          true,
		MulticastQuerier:           false,
		MulticastRouter:            true,
		HashMax:                    512,
		HashElasticity:             4,
		MulticastLastMemberCount:   2,
		MulticastStartupQueryCount: 2,
		GroupFwdMask:               0,
		MTU:                        1500,
		MACAddress:                 "",
		Enabled:                    true,
	}
}

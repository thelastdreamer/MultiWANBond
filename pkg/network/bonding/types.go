package bonding

import (
	"time"
)

// BondMode defines the bonding mode
type BondMode string

const (
	// BondModeRoundRobin transmits packets in sequential order from first to last slave
	BondModeRoundRobin BondMode = "balance-rr" // Mode 0

	// BondModeActiveBackup only one slave is active, another becomes active if the active slave fails
	BondModeActiveBackup BondMode = "active-backup" // Mode 1

	// BondModeXOR transmits based on selected hash policy
	BondModeXOR BondMode = "balance-xor" // Mode 2

	// BondModeBroadcast transmits everything on all slave interfaces
	BondModeBroadcast BondMode = "broadcast" // Mode 3

	// BondMode8023AD IEEE 802.3ad Dynamic link aggregation (LACP)
	BondMode8023AD BondMode = "802.3ad" // Mode 4

	// BondModeTLB Adaptive transmit load balancing
	BondModeTLB BondMode = "balance-tlb" // Mode 5

	// BondModeALB Adaptive load balancing
	BondModeALB BondMode = "balance-alb" // Mode 6
)

// XmitHashPolicy defines the hash policy for bond mode 2 and 4
type XmitHashPolicy string

const (
	// XmitHashLayer2 uses MAC addresses for hash
	XmitHashLayer2 XmitHashPolicy = "layer2" // Default

	// XmitHashLayer23 uses MAC and IP addresses for hash
	XmitHashLayer23 XmitHashPolicy = "layer2+3"

	// XmitHashLayer34 uses IP addresses and ports for hash
	XmitHashLayer34 XmitHashPolicy = "layer3+4"

	// XmitHashEncap23 uses encapsulated layer 2+3 for hash
	XmitHashEncap23 XmitHashPolicy = "encap2+3"

	// XmitHashEncap34 uses encapsulated layer 3+4 for hash
	XmitHashEncap34 XmitHashPolicy = "encap3+4"
)

// LACPRate defines the LACP PDU transmission rate
type LACPRate string

const (
	// LACPRateSlow transmits LACPDUs every 30 seconds
	LACPRateSlow LACPRate = "slow" // Default

	// LACPRateFast transmits LACPDUs every 1 second
	LACPRateFast LACPRate = "fast"
)

// ADSelect defines the 802.3ad aggregation selection logic
type ADSelect string

const (
	// ADSelectStable reselect the active aggregator only if the link goes down
	ADSelectStable ADSelect = "stable" // Default

	// ADSelectBandwidth select the aggregator with highest bandwidth
	ADSelectBandwidth ADSelect = "bandwidth"

	// ADSelectCount select the aggregator with largest number of slaves
	ADSelectCount ADSelect = "count"
)

// PrimaryReselect defines when primary slave is chosen as the active slave
type PrimaryReselect string

const (
	// PrimaryReselectAlways primary becomes active whenever it comes back up
	PrimaryReselectAlways PrimaryReselect = "always"

	// PrimaryReselectBetter primary becomes active if it comes back and is better than current
	PrimaryReselectBetter PrimaryReselect = "better"

	// PrimaryReselectFailure primary becomes active only if current active fails
	PrimaryReselectFailure PrimaryReselect = "failure" // Default
)

// FailOverMAC defines how MAC addresses are handled during failover
type FailOverMAC string

const (
	// FailOverMACNone bond MAC doesn't change
	FailOverMACNone FailOverMAC = "none" // Default

	// FailOverMACActive bond takes MAC of current active slave
	FailOverMACActive FailOverMAC = "active"

	// FailOverMACFollow bond MAC changes to match the new active slave
	FailOverMACFollow FailOverMAC = "follow"
)

// ARPValidate defines how ARP monitoring validates slaves
type ARPValidate string

const (
	// ARPValidateNone no validation
	ARPValidateNone ARPValidate = "none" // Default

	// ARPValidateActive validate only active slave
	ARPValidateActive ARPValidate = "active"

	// ARPValidateBackup validate only backup slaves
	ARPValidateBackup ARPValidate = "backup"

	// ARPValidateAll validate all slaves
	ARPValidateAll ARPValidate = "all"
)

// BondConfig represents a bonding interface configuration
type BondConfig struct {
	// Name of the bond interface (e.g., "bond0")
	Name string

	// Mode defines the bonding mode
	Mode BondMode

	// Slaves is a list of interface names to bond together
	Slaves []string

	// MIIMonInterval in milliseconds (link monitoring frequency)
	// Set to 0 to disable MII monitoring
	MIIMonInterval int

	// UpDelay in milliseconds (delay before enabling a slave after link up)
	// Must be a multiple of MIIMonInterval
	UpDelay int

	// DownDelay in milliseconds (delay before disabling a slave after link down)
	// Must be a multiple of MIIMonInterval
	DownDelay int

	// UseCarrier uses netif_carrier_ok() instead of MII ioctls
	UseCarrier bool

	// ARPInterval in milliseconds for ARP monitoring
	// Set to 0 to disable ARP monitoring
	ARPInterval int

	// ARPIPTargets is a list of IP addresses for ARP monitoring
	ARPIPTargets []string

	// ARPValidate defines how to validate ARP responses
	ARPValidate ARPValidate

	// ARPAllTargets requires all or any ARP targets to be reachable
	// true = all targets must respond, false = any target can respond
	ARPAllTargets bool

	// Primary specifies which slave is preferred as active
	// Only used in active-backup mode
	Primary string

	// PrimaryReselect defines when primary becomes active
	PrimaryReselect PrimaryReselect

	// FailOverMAC defines MAC address handling during failover
	FailOverMAC FailOverMAC

	// XmitHashPolicy for balance-xor and 802.3ad modes
	XmitHashPolicy XmitHashPolicy

	// LACPRate for 802.3ad mode (LACP PDU transmission rate)
	LACPRate LACPRate

	// ADSelect for 802.3ad mode (aggregation selection logic)
	ADSelect ADSelect

	// MinLinks minimum number of links that must be active before bond is considered up
	// Only for 802.3ad mode
	MinLinks int

	// NumGratARPPeer number of peer notifications to send after failover
	NumGratARPPeer int

	// NumUnsolicNA number of IPv6 neighbor advertisements after failover
	NumUnsolicNA int

	// MTU for the bond interface
	MTU int

	// MACAddress for the bond interface (if empty, uses first slave's MAC)
	MACAddress string

	// Enabled indicates if the interface should be up
	Enabled bool
}

// BondInfo represents runtime information about a bonding interface
type BondInfo struct {
	// Name of the bond interface
	Name string

	// Mode currently configured
	Mode BondMode

	// State of the bond (up/down)
	State string

	// MACAddress of the bond
	MACAddress string

	// MTU of the bond
	MTU int

	// Slaves currently attached
	Slaves []SlaveInfo

	// ActiveSlave is the name of currently active slave (for active-backup mode)
	ActiveSlave string

	// MIIStatus overall MII status
	MIIStatus string

	// MIIMonInterval configured
	MIIMonInterval int

	// ARPInterval configured
	ARPInterval int

	// ARPIPTargets configured
	ARPIPTargets []string

	// XmitHashPolicy configured
	XmitHashPolicy XmitHashPolicy

	// LACPRate configured (for 802.3ad)
	LACPRate LACPRate

	// MinLinks configured
	MinLinks int

	// Created timestamp
	Created time.Time

	// LastModified timestamp
	LastModified time.Time
}

// SlaveInfo represents information about a slave interface in a bond
type SlaveInfo struct {
	// Name of the slave interface
	Name string

	// State of the slave (up/down)
	State string

	// MACAddress of the slave
	MACAddress string

	// Speed in Mbps
	Speed int

	// Duplex (full/half)
	Duplex string

	// LinkStatus (up/down)
	LinkStatus string

	// MIIStatus of the slave
	MIIStatus string

	// QueueID for this slave
	QueueID int

	// ADActorOperPortState for 802.3ad mode
	ADActorOperPortState int

	// ADPartnerOperPortState for 802.3ad mode
	ADPartnerOperPortState int

	// IsActive indicates if this is the currently active slave
	IsActive bool

	// IsPrimary indicates if this is the designated primary slave
	IsPrimary bool
}

// BondStats represents statistics for a bonding interface
type BondStats struct {
	// Name of the bond interface
	Name string

	// RXBytes received by bond
	RXBytes uint64

	// RXPackets received by bond
	RXPackets uint64

	// RXErrors on bond
	RXErrors uint64

	// RXDropped packets on bond
	RXDropped uint64

	// TXBytes transmitted by bond
	TXBytes uint64

	// TXPackets transmitted by bond
	TXPackets uint64

	// TXErrors on bond
	TXErrors uint64

	// TXDropped packets on bond
	TXDropped uint64

	// SlaveStats per-slave statistics
	SlaveStats map[string]*SlaveStats
}

// SlaveStats represents statistics for a slave interface
type SlaveStats struct {
	// Name of the slave
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

	// LinkFailureCount number of times link has failed
	LinkFailureCount uint64

	// LastLinkFailure timestamp of last link failure
	LastLinkFailure time.Time
}

// DefaultBondConfig returns a default bonding configuration
func DefaultBondConfig(name string) *BondConfig {
	return &BondConfig{
		Name:            name,
		Mode:            BondModeActiveBackup,
		Slaves:          []string{},
		MIIMonInterval:  100, // 100ms monitoring
		UpDelay:         200, // 200ms delay before marking up
		DownDelay:       200, // 200ms delay before marking down
		UseCarrier:      true,
		ARPInterval:     0, // Disabled by default
		ARPIPTargets:    []string{},
		ARPValidate:     ARPValidateNone,
		ARPAllTargets:   false,
		Primary:         "",
		PrimaryReselect: PrimaryReselectFailure,
		FailOverMAC:     FailOverMACNone,
		XmitHashPolicy:  XmitHashLayer2,
		LACPRate:        LACPRateSlow,
		ADSelect:        ADSelectStable,
		MinLinks:        0,
		NumGratARPPeer:  1,
		NumUnsolicNA:    1,
		MTU:             1500,
		MACAddress:      "",
		Enabled:         true,
	}
}

// LACP8023ADConfig returns a configuration optimized for 802.3ad/LACP bonding
func LACP8023ADConfig(name string, slaves []string) *BondConfig {
	return &BondConfig{
		Name:            name,
		Mode:            BondMode8023AD,
		Slaves:          slaves,
		MIIMonInterval:  100,
		UpDelay:         200,
		DownDelay:       200,
		UseCarrier:      true,
		ARPInterval:     0, // Not used with 802.3ad
		ARPIPTargets:    []string{},
		ARPValidate:     ARPValidateNone,
		ARPAllTargets:   false,
		XmitHashPolicy:  XmitHashLayer34, // Layer 3+4 for better distribution
		LACPRate:        LACPRateFast,    // Fast for quicker convergence
		ADSelect:        ADSelectBandwidth,
		MinLinks:        1,
		NumGratARPPeer:  1,
		NumUnsolicNA:    1,
		MTU:             1500,
		MACAddress:      "",
		Enabled:         true,
	}
}

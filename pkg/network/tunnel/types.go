package tunnel

import (
	"time"
)

// TunnelType defines the type of tunnel
type TunnelType string

const (
	// TunnelTypeGRE Generic Routing Encapsulation
	TunnelTypeGRE TunnelType = "gre"

	// TunnelTypeGRETap GRE with TAP (Layer 2)
	TunnelTypeGRETap TunnelType = "gretap"

	// TunnelTypeIPIP IP in IP tunnel
	TunnelTypeIPIP TunnelType = "ipip"

	// TunnelTypeSIT Simple Internet Transition (IPv6 over IPv4)
	TunnelTypeSIT TunnelType = "sit"

	// TunnelTypeVTI Virtual Tunnel Interface (for IPsec)
	TunnelTypeVTI TunnelType = "vti"

	// TunnelTypeWireGuard WireGuard VPN tunnel
	TunnelTypeWireGuard TunnelType = "wireguard"

	// TunnelTypeVXLAN Virtual eXtensible LAN
	TunnelTypeVXLAN TunnelType = "vxlan"

	// TunnelTypeGeneve Generic Network Virtualization Encapsulation
	TunnelTypeGeneve TunnelType = "geneve"
)

// EncapType defines the encapsulation type
type EncapType string

const (
	// EncapTypeNone no encapsulation
	EncapTypeNone EncapType = "none"

	// EncapTypeFOU Foo-Over-UDP
	EncapTypeFOU EncapType = "fou"

	// EncapTypeGUE Generic UDP Encapsulation
	EncapTypeGUE EncapType = "gue"
)

// TunnelConfig represents a tunnel interface configuration
type TunnelConfig struct {
	// Name of the tunnel interface (e.g., "tun0", "wg0")
	Name string

	// Type of tunnel
	Type TunnelType

	// LocalAddress is the local endpoint address
	LocalAddress string

	// RemoteAddress is the remote endpoint address
	RemoteAddress string

	// TTL for tunneled packets (0 = inherit from inner packet)
	TTL int

	// TOS Type of Service field (0 = inherit)
	TOS int

	// MTU for the tunnel interface
	MTU int

	// Key for GRE tunnels (0 = no key)
	GREKey uint32

	// IKey input key for GRE
	IKey uint32

	// OKey output key for GRE
	OKey uint32

	// Checksum enables GRE checksum
	Checksum bool

	// Seq enables GRE sequence numbers
	Seq bool

	// Csum6 enables IPv6 checksum for GRE
	Csum6 bool

	// RemoteCsum enables remote checksum offload
	RemoteCsum bool

	// EncapType for UDP encapsulation
	EncapType EncapType

	// EncapSport source port for encapsulation
	EncapSport uint16

	// EncapDport destination port for encapsulation
	EncapDport uint16

	// EncapCsum enables encapsulation checksum
	EncapCsum bool

	// EncapCsum6 enables IPv6 encapsulation checksum
	EncapCsum6 bool

	// EncapRemCsum enables remote encapsulation checksum
	EncapRemCsum bool

	// WireGuard-specific fields
	WireGuardPrivateKey string
	WireGuardPublicKey  string
	WireGuardListenPort int
	WireGuardPeers      []WireGuardPeer

	// VXLAN-specific fields
	VXLANID       uint32
	VXLANGroup    string
	VXLANPort     uint16
	VXLANSrcPort  uint16
	VXLANLearning bool
	VXLANRSC      bool

	// Geneve-specific fields
	GeneveID       uint32
	GeneveRemote   string
	GenevePort     uint16
	GeneveInnerProto uint16

	// MACAddress for the tunnel interface (if applicable)
	MACAddress string

	// Enabled indicates if the interface should be up
	Enabled bool
}

// WireGuardPeer represents a WireGuard peer configuration
type WireGuardPeer struct {
	// PublicKey of the peer
	PublicKey string

	// PresharedKey for additional security (optional)
	PresharedKey string

	// Endpoint address (IP:port)
	Endpoint string

	// AllowedIPs list of IP ranges allowed for this peer
	AllowedIPs []string

	// PersistentKeepalive interval in seconds (0 = disabled)
	PersistentKeepalive int
}

// TunnelInfo represents runtime information about a tunnel
type TunnelInfo struct {
	// Name of the tunnel
	Name string

	// Type of tunnel
	Type TunnelType

	// State of the tunnel (up/down)
	State string

	// LocalAddress
	LocalAddress string

	// RemoteAddress
	RemoteAddress string

	// MTU
	MTU int

	// MACAddress (if applicable)
	MACAddress string

	// TTL configured
	TTL int

	// TOS configured
	TOS int

	// GREKey (if applicable)
	GREKey uint32

	// EncapType configured
	EncapType EncapType

	// EncapSport configured
	EncapSport uint16

	// EncapDport configured
	EncapDport uint16

	// WireGuard info (if applicable)
	WireGuardInfo *WireGuardInfo

	// VXLAN info (if applicable)
	VXLANID uint32

	// Geneve info (if applicable)
	GeneveID uint32

	// Created timestamp
	Created time.Time

	// LastModified timestamp
	LastModified time.Time
}

// WireGuardInfo represents WireGuard-specific runtime information
type WireGuardInfo struct {
	// PublicKey of this interface
	PublicKey string

	// ListenPort
	ListenPort int

	// Peers connected
	Peers []WireGuardPeerInfo

	// Fwmark firewall mark
	Fwmark uint32
}

// WireGuardPeerInfo represents runtime information about a WireGuard peer
type WireGuardPeerInfo struct {
	// PublicKey
	PublicKey string

	// Endpoint
	Endpoint string

	// AllowedIPs
	AllowedIPs []string

	// LatestHandshake timestamp
	LatestHandshake time.Time

	// TransferRX bytes received
	TransferRX uint64

	// TransferTX bytes transmitted
	TransferTX uint64

	// PersistentKeepalive interval
	PersistentKeepalive int

	// LastSeen timestamp
	LastSeen time.Time
}

// TunnelStats represents statistics for a tunnel
type TunnelStats struct {
	// Name of the tunnel
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

	// RXCompressed (for some tunnel types)
	RXCompressed uint64

	// TXCompressed
	TXCompressed uint64

	// Collisions
	Collisions uint64
}

// DefaultTunnelConfig returns a default tunnel configuration
func DefaultTunnelConfig(name string, tunnelType TunnelType) *TunnelConfig {
	return &TunnelConfig{
		Name:          name,
		Type:          tunnelType,
		LocalAddress:  "",
		RemoteAddress: "",
		TTL:           64,
		TOS:           0,
		MTU:           1420, // Common for tunnels to avoid fragmentation
		GREKey:        0,
		IKey:          0,
		OKey:          0,
		Checksum:      false,
		Seq:           false,
		Csum6:         false,
		RemoteCsum:    false,
		EncapType:     EncapTypeNone,
		EncapSport:    0,
		EncapDport:    0,
		EncapCsum:     false,
		EncapCsum6:    false,
		EncapRemCsum:  false,
		MACAddress:    "",
		Enabled:       true,
	}
}

// DefaultGRETunnelConfig returns a default GRE tunnel configuration
func DefaultGRETunnelConfig(name, local, remote string) *TunnelConfig {
	config := DefaultTunnelConfig(name, TunnelTypeGRE)
	config.LocalAddress = local
	config.RemoteAddress = remote
	config.MTU = 1476 // 1500 - 24 (IP + GRE header)
	return config
}

// DefaultIPIPTunnelConfig returns a default IPIP tunnel configuration
func DefaultIPIPTunnelConfig(name, local, remote string) *TunnelConfig {
	config := DefaultTunnelConfig(name, TunnelTypeIPIP)
	config.LocalAddress = local
	config.RemoteAddress = remote
	config.MTU = 1480 // 1500 - 20 (IP header)
	return config
}

// DefaultWireGuardConfig returns a default WireGuard tunnel configuration
func DefaultWireGuardConfig(name string) *TunnelConfig {
	config := DefaultTunnelConfig(name, TunnelTypeWireGuard)
	config.WireGuardListenPort = 51820
	config.WireGuardPeers = []WireGuardPeer{}
	config.MTU = 1420 // Default WireGuard MTU
	return config
}

// DefaultVXLANConfig returns a default VXLAN tunnel configuration
func DefaultVXLANConfig(name string, vni uint32) *TunnelConfig {
	config := DefaultTunnelConfig(name, TunnelTypeVXLAN)
	config.VXLANID = vni
	config.VXLANPort = 4789 // Default VXLAN port
	config.VXLANLearning = true
	config.VXLANRSC = false
	return config
}

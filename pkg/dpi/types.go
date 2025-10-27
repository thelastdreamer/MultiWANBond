// Package dpi provides Deep Packet Inspection capabilities for traffic classification
package dpi

import (
	"net"
	"time"
)

// Protocol represents an application protocol
type Protocol int

const (
	// ProtocolUnknown is unidentified traffic
	ProtocolUnknown Protocol = iota

	// Web protocols
	ProtocolHTTP
	ProtocolHTTPS
	ProtocolHTTP2
	ProtocolHTTP3
	ProtocolWebSocket

	// Streaming
	ProtocolYouTube
	ProtocolNetflix
	ProtocolTwitch
	ProtocolSpotify
	ProtocolAppleMusic

	// Social media
	ProtocolFacebook
	ProtocolInstagram
	ProtocolTwitter
	ProtocolTikTok
	ProtocolWhatsApp

	// Gaming
	ProtocolSteam
	ProtocolEpicGames
	ProtocolMinecraft
	ProtocolLeagueOfLegends
	ProtocolFortnite

	// Communication
	ProtocolZoom
	ProtocolTeams
	ProtocolSkype
	ProtocolDiscord
	ProtocolSlack

	// File transfer
	ProtocolFTP
	ProtocolSFTP
	ProtocolSCP
	ProtocolTorrent
	ProtocolDropbox

	// Email
	ProtocolSMTP
	ProtocolIMAP
	ProtocolPOP3

	// DNS
	ProtocolDNS
	ProtocolDNSOverHTTPS
	ProtocolDNSOverTLS

	// VPN
	ProtocolOpenVPN
	ProtocolWireGuard
	ProtocolIPSec
	ProtocolL2TP

	// Other
	ProtocolSSH
	ProtocolTelnet
	ProtocolRDP
	ProtocolVNC
	ProtocolNTP
	ProtocolDHCP
	ProtocolSNMP
)

// String returns string representation of protocol
func (p Protocol) String() string {
	names := map[Protocol]string{
		ProtocolUnknown:         "Unknown",
		ProtocolHTTP:            "HTTP",
		ProtocolHTTPS:           "HTTPS",
		ProtocolHTTP2:           "HTTP/2",
		ProtocolHTTP3:           "HTTP/3",
		ProtocolWebSocket:       "WebSocket",
		ProtocolYouTube:         "YouTube",
		ProtocolNetflix:         "Netflix",
		ProtocolTwitch:          "Twitch",
		ProtocolSpotify:         "Spotify",
		ProtocolAppleMusic:      "Apple Music",
		ProtocolFacebook:        "Facebook",
		ProtocolInstagram:       "Instagram",
		ProtocolTwitter:         "Twitter",
		ProtocolTikTok:          "TikTok",
		ProtocolWhatsApp:        "WhatsApp",
		ProtocolSteam:           "Steam",
		ProtocolEpicGames:       "Epic Games",
		ProtocolMinecraft:       "Minecraft",
		ProtocolLeagueOfLegends: "League of Legends",
		ProtocolFortnite:        "Fortnite",
		ProtocolZoom:            "Zoom",
		ProtocolTeams:           "Microsoft Teams",
		ProtocolSkype:           "Skype",
		ProtocolDiscord:         "Discord",
		ProtocolSlack:           "Slack",
		ProtocolFTP:             "FTP",
		ProtocolSFTP:            "SFTP",
		ProtocolSCP:             "SCP",
		ProtocolTorrent:         "BitTorrent",
		ProtocolDropbox:         "Dropbox",
		ProtocolSMTP:            "SMTP",
		ProtocolIMAP:            "IMAP",
		ProtocolPOP3:            "POP3",
		ProtocolDNS:             "DNS",
		ProtocolDNSOverHTTPS:    "DNS over HTTPS",
		ProtocolDNSOverTLS:      "DNS over TLS",
		ProtocolOpenVPN:         "OpenVPN",
		ProtocolWireGuard:       "WireGuard",
		ProtocolIPSec:           "IPSec",
		ProtocolL2TP:            "L2TP",
		ProtocolSSH:             "SSH",
		ProtocolTelnet:          "Telnet",
		ProtocolRDP:             "RDP",
		ProtocolVNC:             "VNC",
		ProtocolNTP:             "NTP",
		ProtocolDHCP:            "DHCP",
		ProtocolSNMP:            "SNMP",
	}

	if name, ok := names[p]; ok {
		return name
	}
	return "Unknown"
}

// Category represents application category
type Category int

const (
	CategoryUnknown Category = iota
	CategoryWeb
	CategoryStreaming
	CategorySocialMedia
	CategoryGaming
	CategoryCommunication
	CategoryFileTransfer
	CategoryEmail
	CategoryVPN
	CategorySystem
)

// String returns string representation of category
func (c Category) String() string {
	switch c {
	case CategoryWeb:
		return "Web"
	case CategoryStreaming:
		return "Streaming"
	case CategorySocialMedia:
		return "Social Media"
	case CategoryGaming:
		return "Gaming"
	case CategoryCommunication:
		return "Communication"
	case CategoryFileTransfer:
		return "File Transfer"
	case CategoryEmail:
		return "Email"
	case CategoryVPN:
		return "VPN"
	case CategorySystem:
		return "System"
	default:
		return "Unknown"
	}
}

// GetCategory returns the category for a protocol
func (p Protocol) GetCategory() Category {
	categoryMap := map[Protocol]Category{
		ProtocolHTTP:            CategoryWeb,
		ProtocolHTTPS:           CategoryWeb,
		ProtocolHTTP2:           CategoryWeb,
		ProtocolHTTP3:           CategoryWeb,
		ProtocolWebSocket:       CategoryWeb,
		ProtocolYouTube:         CategoryStreaming,
		ProtocolNetflix:         CategoryStreaming,
		ProtocolTwitch:          CategoryStreaming,
		ProtocolSpotify:         CategoryStreaming,
		ProtocolAppleMusic:      CategoryStreaming,
		ProtocolFacebook:        CategorySocialMedia,
		ProtocolInstagram:       CategorySocialMedia,
		ProtocolTwitter:         CategorySocialMedia,
		ProtocolTikTok:          CategorySocialMedia,
		ProtocolWhatsApp:        CategorySocialMedia,
		ProtocolSteam:           CategoryGaming,
		ProtocolEpicGames:       CategoryGaming,
		ProtocolMinecraft:       CategoryGaming,
		ProtocolLeagueOfLegends: CategoryGaming,
		ProtocolFortnite:        CategoryGaming,
		ProtocolZoom:            CategoryCommunication,
		ProtocolTeams:           CategoryCommunication,
		ProtocolSkype:           CategoryCommunication,
		ProtocolDiscord:         CategoryCommunication,
		ProtocolSlack:           CategoryCommunication,
		ProtocolFTP:             CategoryFileTransfer,
		ProtocolSFTP:            CategoryFileTransfer,
		ProtocolSCP:             CategoryFileTransfer,
		ProtocolTorrent:         CategoryFileTransfer,
		ProtocolDropbox:         CategoryFileTransfer,
		ProtocolSMTP:            CategoryEmail,
		ProtocolIMAP:            CategoryEmail,
		ProtocolPOP3:            CategoryEmail,
		ProtocolOpenVPN:         CategoryVPN,
		ProtocolWireGuard:       CategoryVPN,
		ProtocolIPSec:           CategoryVPN,
		ProtocolL2TP:            CategoryVPN,
		ProtocolDNS:             CategorySystem,
		ProtocolDNSOverHTTPS:    CategorySystem,
		ProtocolDNSOverTLS:      CategorySystem,
		ProtocolNTP:             CategorySystem,
		ProtocolDHCP:            CategorySystem,
		ProtocolSNMP:            CategorySystem,
	}

	if cat, ok := categoryMap[p]; ok {
		return cat
	}
	return CategoryUnknown
}

// Flow represents a network flow
type Flow struct {
	// Source and destination
	SrcIP   net.IP
	DstIP   net.IP
	SrcPort uint16
	DstPort uint16
	Proto   uint8

	// Classification
	Protocol   Protocol
	Category   Category
	Confidence float64

	// Timing
	FirstSeen time.Time
	LastSeen  time.Time

	// Statistics
	Packets      uint64
	Bytes        uint64
	PacketsUp    uint64
	PacketsDown  uint64
	BytesUp      uint64
	BytesDown    uint64

	// State
	Established bool
}

// Classification represents the result of DPI classification
type Classification struct {
	Protocol   Protocol
	Category   Category
	Confidence float64
	Matched    []string // Matched signatures
	Timestamp  time.Time
}

// Signature represents a protocol signature for detection
type Signature struct {
	Name       string
	Protocol   Protocol
	Port       uint16
	PortRange  [2]uint16
	Pattern    []byte
	Offset     int
	Depth      int
	IsRegex    bool
	Weight     float64
}

// TrafficClass represents a traffic class for QoS
type TrafficClass int

const (
	// ClassRealTime for voice, video conferencing (highest priority)
	ClassRealTime TrafficClass = iota

	// ClassInteractive for gaming, remote desktop
	ClassInteractive

	// ClassStreaming for video/audio streaming
	ClassStreaming

	// ClassBulk for file downloads, backups
	ClassBulk

	// ClassBackground for system updates, P2P
	ClassBackground

	// ClassDefault for unclassified traffic
	ClassDefault
)

// String returns string representation of traffic class
func (c TrafficClass) String() string {
	switch c {
	case ClassRealTime:
		return "Real-Time"
	case ClassInteractive:
		return "Interactive"
	case ClassStreaming:
		return "Streaming"
	case ClassBulk:
		return "Bulk"
	case ClassBackground:
		return "Background"
	case ClassDefault:
		return "Default"
	default:
		return "Unknown"
	}
}

// GetPriority returns priority value (lower = higher priority)
func (c TrafficClass) GetPriority() int {
	priorities := map[TrafficClass]int{
		ClassRealTime:    10,
		ClassInteractive: 20,
		ClassStreaming:   30,
		ClassDefault:     40,
		ClassBulk:        50,
		ClassBackground:  60,
	}

	if pri, ok := priorities[c]; ok {
		return pri
	}
	return 40 // Default
}

// GetTrafficClass returns the traffic class for a protocol
func (p Protocol) GetTrafficClass() TrafficClass {
	classMap := map[Protocol]TrafficClass{
		// Real-time
		ProtocolZoom:   ClassRealTime,
		ProtocolTeams:  ClassRealTime,
		ProtocolSkype:  ClassRealTime,

		// Interactive
		ProtocolSteam:           ClassInteractive,
		ProtocolEpicGames:       ClassInteractive,
		ProtocolMinecraft:       ClassInteractive,
		ProtocolLeagueOfLegends: ClassInteractive,
		ProtocolFortnite:        ClassInteractive,
		ProtocolRDP:             ClassInteractive,
		ProtocolVNC:             ClassInteractive,
		ProtocolSSH:             ClassInteractive,

		// Streaming
		ProtocolYouTube:    ClassStreaming,
		ProtocolNetflix:    ClassStreaming,
		ProtocolTwitch:     ClassStreaming,
		ProtocolSpotify:    ClassStreaming,
		ProtocolAppleMusic: ClassStreaming,

		// Bulk
		ProtocolFTP:     ClassBulk,
		ProtocolSFTP:    ClassBulk,
		ProtocolSCP:     ClassBulk,
		ProtocolDropbox: ClassBulk,

		// Background
		ProtocolTorrent: ClassBackground,
	}

	if class, ok := classMap[p]; ok {
		return class
	}
	return ClassDefault
}

// ApplicationPolicy represents a routing policy for an application
type ApplicationPolicy struct {
	Name        string
	Protocol    Protocol
	Category    Category
	WANID       uint8
	Mark        uint32
	Priority    int
	BandwidthLimit uint64
	TrafficClass TrafficClass
	Enabled     bool
	Created     time.Time
}

// DPIConfig contains DPI configuration
type DPIConfig struct {
	EnableDPI              bool
	EnableProtocolDetection bool
	EnableCategoryDetection bool
	MaxFlows               int
	FlowTimeout            time.Duration
	InspectionDepth        int // Bytes to inspect per packet
	EnableQoS              bool
	EnableBandwidthShaping bool
}

// DefaultDPIConfig returns default DPI configuration
func DefaultDPIConfig() *DPIConfig {
	return &DPIConfig{
		EnableDPI:              true,
		EnableProtocolDetection: true,
		EnableCategoryDetection: true,
		MaxFlows:               100000,
		FlowTimeout:            300 * time.Second,
		InspectionDepth:        1024,
		EnableQoS:              true,
		EnableBandwidthShaping: true,
	}
}

// DPIStats contains DPI statistics
type DPIStats struct {
	TotalFlows       uint64
	ActiveFlows      uint64
	ClassifiedFlows  uint64
	UnknownFlows     uint64
	TotalPackets     uint64
	TotalBytes       uint64
	ProtocolStats    map[Protocol]uint64
	CategoryStats    map[Category]uint64
	LastClassification time.Time
}

package vlan

import (
	"fmt"
	"time"
)

// Config represents VLAN configuration
type Config struct {
	ID               int       // VLAN ID (1-4094)
	ParentInterface  string    // Parent interface name (e.g., "eth0")
	Name             string    // VLAN interface name (e.g., "eth0.100" or custom)
	DisplayName      string    // User-friendly name
	Priority         uint8     // 802.1p priority (0-7)
	MTU              int       // MTU size (default: inherit from parent)
	DownloadSpeed    uint64    // Known/standard download speed (bps)
	UploadSpeed      uint64    // Known/standard upload speed (bps)

	// State
	Enabled          bool
	AutoCreate       bool      // Auto-create on system start

	// Timestamps
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Interface represents a VLAN interface
type Interface struct {
	Config      *Config
	SystemName  string    // Actual system interface name
	ParentIndex int       // Parent interface index
	State       State
	Error       error     // Last error if any
}

// State represents the state of a VLAN interface
type State string

const (
	StateNone      State = "none"       // Not created
	StateCreating  State = "creating"   // Being created
	StateActive    State = "active"     // Active and working
	StateError     State = "error"      // Error state
	StateDeleting  State = "deleting"   // Being deleted
)

// Priority802_1p represents 802.1p priority classes
type Priority802_1p uint8

const (
	PriorityBestEffort       Priority802_1p = 0 // BE - Best Effort
	PriorityBackground       Priority802_1p = 1 // BK - Background
	PriorityExcellentEffort  Priority802_1p = 2 // EE - Excellent Effort
	PriorityCriticalApps     Priority802_1p = 3 // CA - Critical Applications
	PriorityVideo            Priority802_1p = 4 // VI - Video
	PriorityVoice            Priority802_1p = 5 // VO - Voice
	PriorityInternetControl  Priority802_1p = 6 // IC - Internetwork Control
	PriorityNetworkControl   Priority802_1p = 7 // NC - Network Control
)

// String returns the name of the priority
func (p Priority802_1p) String() string {
	names := map[Priority802_1p]string{
		PriorityBestEffort:       "Best Effort",
		PriorityBackground:       "Background",
		PriorityExcellentEffort:  "Excellent Effort",
		PriorityCriticalApps:     "Critical Applications",
		PriorityVideo:            "Video",
		PriorityVoice:            "Voice",
		PriorityInternetControl:  "Internetwork Control",
		PriorityNetworkControl:   "Network Control",
	}
	if name, ok := names[p]; ok {
		return name
	}
	return "Unknown"
}

// ValidateID validates a VLAN ID
func ValidateID(id int) error {
	if id < 1 || id > 4094 {
		return ErrInvalidVLANID
	}
	if id == 1 {
		// VLAN 1 is the default VLAN, generally should not be manually created
		return ErrReservedVLANID
	}
	return nil
}

// ValidatePriority validates 802.1p priority
func ValidatePriority(priority uint8) error {
	if priority > 7 {
		return ErrInvalidPriority
	}
	return nil
}

// GenerateName generates a default VLAN interface name
func GenerateName(parent string, vlanID int) string {
	return fmt.Sprintf("%s.%d", parent, vlanID)
}

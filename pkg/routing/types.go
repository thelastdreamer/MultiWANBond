// Package routing provides policy-based routing capabilities for multi-WAN management
package routing

import (
	"net"
	"time"
)

// RouteType represents the type of route
type RouteType int

const (
	// RouteTypeUnicast is a standard unicast route
	RouteTypeUnicast RouteType = iota

	// RouteTypeLocal is a local route
	RouteTypeLocal

	// RouteTypeBroadcast is a broadcast route
	RouteTypeBroadcast

	// RouteTypeMulticast is a multicast route
	RouteTypeMulticast

	// RouteTypeBlackhole drops packets silently
	RouteTypeBlackhole

	// RouteTypeUnreachable sends ICMP unreachable
	RouteTypeUnreachable

	// RouteTypeProhibit sends ICMP prohibited
	RouteTypeProhibit
)

// String returns string representation of route type
func (t RouteType) String() string {
	switch t {
	case RouteTypeUnicast:
		return "unicast"
	case RouteTypeLocal:
		return "local"
	case RouteTypeBroadcast:
		return "broadcast"
	case RouteTypeMulticast:
		return "multicast"
	case RouteTypeBlackhole:
		return "blackhole"
	case RouteTypeUnreachable:
		return "unreachable"
	case RouteTypeProhibit:
		return "prohibit"
	default:
		return "unknown"
	}
}

// RouteScope represents the scope of a route
type RouteScope int

const (
	// RouteScopeUniverse is global route
	RouteScopeUniverse RouteScope = iota

	// RouteScopeHost is local route
	RouteScopeHost

	// RouteScopeLink is link-local route
	RouteScopeLink

	// RouteScopeSite is site-local route
	RouteScopeSite
)

// String returns string representation of route scope
func (s RouteScope) String() string {
	switch s {
	case RouteScopeUniverse:
		return "global"
	case RouteScopeHost:
		return "host"
	case RouteScopeLink:
		return "link"
	case RouteScopeSite:
		return "site"
	default:
		return "unknown"
	}
}

// Route represents a routing table entry
type Route struct {
	// Destination is the destination network
	Destination *net.IPNet

	// Gateway is the next hop gateway
	Gateway net.IP

	// Interface is the output interface name
	Interface string

	// Type is the route type
	Type RouteType

	// Scope is the route scope
	Scope RouteScope

	// Priority is the route priority/metric
	Priority int

	// Table is the routing table ID
	Table int

	// Source is the preferred source address
	Source net.IP

	// MTU is the maximum transmission unit
	MTU int

	// Added is when this route was added
	Added time.Time
}

// RoutingTable represents a routing table
type RoutingTable struct {
	// ID is the table identifier
	ID int

	// Name is the table name
	Name string

	// WANID is the associated WAN interface ID (0 for main table)
	WANID uint8

	// Routes contains all routes in this table
	Routes []*Route

	// Created is when this table was created
	Created time.Time
}

// PolicyRule represents a policy routing rule
type PolicyRule struct {
	// Priority determines rule evaluation order (lower = higher priority)
	Priority int

	// Table is the routing table to use if rule matches
	Table int

	// SourceNetwork is the source IP/network to match
	SourceNetwork *net.IPNet

	// DestNetwork is the destination IP/network to match
	DestNetwork *net.IPNet

	// InputInterface is the input interface to match
	InputInterface string

	// OutputInterface is the output interface to match
	OutputInterface string

	// Mark is the packet mark to match (fwmark)
	Mark uint32

	// MarkMask is the mask for mark matching
	MarkMask uint32

	// TOS is the Type of Service to match
	TOS uint8

	// Protocol is the IP protocol to match (TCP=6, UDP=17, etc.)
	Protocol uint8

	// SourcePortRange is the source port range [min, max]
	SourcePortRange [2]uint16

	// DestPortRange is the destination port range [min, max]
	DestPortRange [2]uint16

	// Invert inverts the match
	Invert bool

	// Action is the action to take
	Action RuleAction

	// Enabled indicates if rule is active
	Enabled bool

	// Created is when this rule was created
	Created time.Time
}

// RuleAction represents the action for a policy rule
type RuleAction int

const (
	// RuleActionTable routes via specified table
	RuleActionTable RuleAction = iota

	// RuleActionBlackhole drops packet silently
	RuleActionBlackhole

	// RuleActionUnreachable sends ICMP unreachable
	RuleActionUnreachable

	// RuleActionProhibit sends ICMP prohibited
	RuleActionProhibit
)

// String returns string representation of rule action
func (a RuleAction) String() string {
	switch a {
	case RuleActionTable:
		return "table"
	case RuleActionBlackhole:
		return "blackhole"
	case RuleActionUnreachable:
		return "unreachable"
	case RuleActionProhibit:
		return "prohibit"
	default:
		return "unknown"
	}
}

// RoutingPolicy represents a complete routing policy
type RoutingPolicy struct {
	// Name is the policy name
	Name string

	// Description describes the policy
	Description string

	// Rules contains all policy rules
	Rules []*PolicyRule

	// DefaultTable is the default table for unmatched traffic
	DefaultTable int

	// Enabled indicates if policy is active
	Enabled bool

	// Created is when this policy was created
	Created time.Time
}

// RoutingConfig contains routing configuration
type RoutingConfig struct {
	// EnablePolicyRouting enables policy-based routing
	EnablePolicyRouting bool

	// MainTableID is the main routing table ID (254)
	MainTableID int

	// DefaultTableID is the default routing table ID (253)
	DefaultTableID int

	// LocalTableID is the local routing table ID (255)
	LocalTableID int

	// MaxCustomTables is maximum number of custom tables
	MaxCustomTables int

	// TableIDStart is the starting table ID for custom tables
	TableIDStart int

	// EnableSourceRouting enables source-based routing
	EnableSourceRouting bool

	// EnableMarkRouting enables mark-based routing
	EnableMarkRouting bool

	// MarkBase is the base value for fwmark (default 100)
	MarkBase uint32

	// AutoCreateTables automatically creates per-WAN tables
	AutoCreateTables bool

	// SyncInterval is how often to sync routing state
	SyncInterval time.Duration
}

// DefaultRoutingConfig returns default routing configuration
func DefaultRoutingConfig() *RoutingConfig {
	return &RoutingConfig{
		EnablePolicyRouting: true,
		MainTableID:        254,
		DefaultTableID:     253,
		LocalTableID:       255,
		MaxCustomTables:    100,
		TableIDStart:       100,
		EnableSourceRouting: true,
		EnableMarkRouting:   true,
		MarkBase:           100,
		AutoCreateTables:   true,
		SyncInterval:       30 * time.Second,
	}
}

// WANRoutingTable represents a routing table for a specific WAN
type WANRoutingTable struct {
	// WANID is the WAN interface ID
	WANID uint8

	// Table is the routing table
	Table *RoutingTable

	// DefaultRoute is the default route via this WAN
	DefaultRoute *Route

	// SourceIP is the source IP for this WAN
	SourceIP net.IP

	// Gateway is the gateway for this WAN
	Gateway net.IP

	// Interface is the interface name
	Interface string

	// Mark is the fwmark for this WAN
	Mark uint32

	// Active indicates if this WAN table is active
	Active bool
}

// RoutingStats contains routing statistics
type RoutingStats struct {
	// TablesCreated is total tables created
	TablesCreated uint64

	// TablesDeleted is total tables deleted
	TablesDeleted uint64

	// ActiveTables is currently active tables
	ActiveTables uint64

	// RoutesAdded is total routes added
	RoutesAdded uint64

	// RoutesDeleted is total routes deleted
	RoutesDeleted uint64

	// ActiveRoutes is currently active routes
	ActiveRoutes uint64

	// RulesAdded is total rules added
	RulesAdded uint64

	// RulesDeleted is total rules deleted
	RulesDeleted uint64

	// ActiveRules is currently active rules
	ActiveRules uint64

	// LastSync is when routing was last synced
	LastSync time.Time

	// SyncErrors is number of sync errors
	SyncErrors uint64
}

// TableManager defines interface for routing table management
type TableManager interface {
	// CreateTable creates a new routing table
	CreateTable(id int, name string) error

	// DeleteTable deletes a routing table
	DeleteTable(id int) error

	// GetTable retrieves a routing table
	GetTable(id int) (*RoutingTable, error)

	// ListTables lists all routing tables
	ListTables() ([]*RoutingTable, error)

	// AddRoute adds a route to a table
	AddRoute(tableID int, route *Route) error

	// DeleteRoute deletes a route from a table
	DeleteRoute(tableID int, route *Route) error

	// ListRoutes lists all routes in a table
	ListRoutes(tableID int) ([]*Route, error)

	// FlushTable removes all routes from a table
	FlushTable(tableID int) error
}

// RuleManager defines interface for policy rule management
type RuleManager interface {
	// AddRule adds a policy rule
	AddRule(rule *PolicyRule) error

	// DeleteRule deletes a policy rule
	DeleteRule(priority int) error

	// GetRule retrieves a rule by priority
	GetRule(priority int) (*PolicyRule, error)

	// ListRules lists all policy rules
	ListRules() ([]*PolicyRule, error)

	// FlushRules removes all policy rules
	FlushRules() error

	// EnableRule enables a rule
	EnableRule(priority int) error

	// DisableRule disables a rule
	DisableRule(priority int) error
}

// SourceRoutingRule represents a source-based routing rule
type SourceRoutingRule struct {
	// SourceNetwork is the source network
	SourceNetwork *net.IPNet

	// WANID is the WAN to route via
	WANID uint8

	// Table is the routing table ID
	Table int

	// Priority is the rule priority
	Priority int

	// Enabled indicates if active
	Enabled bool
}

// MarkRoutingRule represents a mark-based routing rule
type MarkRoutingRule struct {
	// Mark is the packet mark
	Mark uint32

	// Mask is the mark mask
	Mask uint32

	// WANID is the WAN to route via
	WANID uint8

	// Table is the routing table ID
	Table int

	// Priority is the rule priority
	Priority int

	// Enabled indicates if active
	Enabled bool
}

// ApplicationRule represents an application-specific routing rule
type ApplicationRule struct {
	// Name is the application name
	Name string

	// ProcessName is the process name to match
	ProcessName string

	// UID is the user ID to match
	UID int

	// GID is the group ID to match
	GID int

	// WANID is the WAN to route via
	WANID uint8

	// Mark is the fwmark to apply
	Mark uint32

	// Priority is the rule priority
	Priority int

	// Enabled indicates if active
	Enabled bool
}

// RouteChange represents a routing change event
type RouteChange struct {
	// Type is the change type
	Type RouteChangeType

	// Route is the affected route
	Route *Route

	// Table is the affected table
	Table int

	// Timestamp is when the change occurred
	Timestamp time.Time
}

// RouteChangeType represents the type of route change
type RouteChangeType int

const (
	// RouteChangeAdd indicates route was added
	RouteChangeAdd RouteChangeType = iota

	// RouteChangeDelete indicates route was deleted
	RouteChangeDelete

	// RouteChangeModify indicates route was modified
	RouteChangeModify
)

// String returns string representation of change type
func (t RouteChangeType) String() string {
	switch t {
	case RouteChangeAdd:
		return "add"
	case RouteChangeDelete:
		return "delete"
	case RouteChangeModify:
		return "modify"
	default:
		return "unknown"
	}
}

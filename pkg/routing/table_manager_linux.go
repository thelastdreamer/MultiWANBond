//go:build linux

package routing

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
)

// LinuxTableManager manages routing tables on Linux using netlink
type LinuxTableManager struct {
	mu     sync.RWMutex
	tables map[int]*RoutingTable
}

// NewLinuxTableManager creates a new Linux table manager
func NewLinuxTableManager() *LinuxTableManager {
	return &LinuxTableManager{
		tables: make(map[int]*RoutingTable),
	}
}

// CreateTable creates a new routing table
func (tm *LinuxTableManager) CreateTable(id int, name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tables[id]; exists {
		return fmt.Errorf("table %d already exists", id)
	}

	table := &RoutingTable{
		ID:      id,
		Name:    name,
		Routes:  make([]*Route, 0),
		Created: time.Now(),
	}

	tm.tables[id] = table
	return nil
}

// DeleteTable deletes a routing table
func (tm *LinuxTableManager) DeleteTable(id int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	table, exists := tm.tables[id]
	if !exists {
		return fmt.Errorf("table %d not found", id)
	}

	// Flush all routes from the table first
	if err := tm.flushTableLocked(id); err != nil {
		return fmt.Errorf("failed to flush table: %w", err)
	}

	delete(tm.tables, id)
	_ = table // Avoid unused variable warning

	return nil
}

// GetTable retrieves a routing table
func (tm *LinuxTableManager) GetTable(id int) (*RoutingTable, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	table, exists := tm.tables[id]
	if !exists {
		return nil, fmt.Errorf("table %d not found", id)
	}

	return table, nil
}

// ListTables lists all routing tables
func (tm *LinuxTableManager) ListTables() ([]*RoutingTable, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tables := make([]*RoutingTable, 0, len(tm.tables))
	for _, table := range tm.tables {
		tables = append(tables, table)
	}

	return tables, nil
}

// AddRoute adds a route to a table
func (tm *LinuxTableManager) AddRoute(tableID int, route *Route) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	table, exists := tm.tables[tableID]
	if !exists {
		return fmt.Errorf("table %d not found", tableID)
	}

	// Create netlink route
	nlRoute := &netlink.Route{
		Table:    tableID,
		Scope:    tm.scopeToNetlink(route.Scope),
		Type:     tm.typeToNetlink(route.Type),
		Priority: route.Priority,
		MTU:      route.MTU,
	}

	// Set destination
	if route.Destination != nil {
		nlRoute.Dst = route.Destination
	}

	// Set gateway
	if route.Gateway != nil {
		nlRoute.Gw = route.Gateway
	}

	// Set source
	if route.Source != nil {
		nlRoute.Src = route.Source
	}

	// Set interface
	if route.Interface != "" {
		link, err := netlink.LinkByName(route.Interface)
		if err != nil {
			return fmt.Errorf("failed to find interface %s: %w", route.Interface, err)
		}
		nlRoute.LinkIndex = link.Attrs().Index
	}

	// Add route via netlink
	if err := netlink.RouteAdd(nlRoute); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	// Add to our table tracking
	route.Table = tableID
	route.Added = time.Now()
	table.Routes = append(table.Routes, route)

	return nil
}

// DeleteRoute deletes a route from a table
func (tm *LinuxTableManager) DeleteRoute(tableID int, route *Route) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	table, exists := tm.tables[tableID]
	if !exists {
		return fmt.Errorf("table %d not found", tableID)
	}

	// Create netlink route for deletion
	nlRoute := &netlink.Route{
		Table: tableID,
		Dst:   route.Destination,
		Gw:    route.Gateway,
	}

	if route.Interface != "" {
		link, err := netlink.LinkByName(route.Interface)
		if err == nil {
			nlRoute.LinkIndex = link.Attrs().Index
		}
	}

	// Delete route via netlink
	if err := netlink.RouteDel(nlRoute); err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}

	// Remove from our table tracking
	for i, r := range table.Routes {
		if tm.routesEqual(r, route) {
			table.Routes = append(table.Routes[:i], table.Routes[i+1:]...)
			break
		}
	}

	return nil
}

// ListRoutes lists all routes in a table
func (tm *LinuxTableManager) ListRoutes(tableID int) ([]*Route, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	table, exists := tm.tables[tableID]
	if !exists {
		return nil, fmt.Errorf("table %d not found", tableID)
	}

	// Get routes from kernel
	filter := &netlink.Route{
		Table: tableID,
	}

	nlRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, filter, netlink.RT_FILTER_TABLE)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}

	// Convert to our Route type
	routes := make([]*Route, 0, len(nlRoutes))
	for _, nlRoute := range nlRoutes {
		route := tm.netlinkToRoute(&nlRoute)
		routes = append(routes, route)
	}

	// Update our tracking
	table.Routes = routes

	return routes, nil
}

// FlushTable removes all routes from a table
func (tm *LinuxTableManager) FlushTable(tableID int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.flushTableLocked(tableID)
}

// flushTableLocked flushes a table (caller must hold lock)
func (tm *LinuxTableManager) flushTableLocked(tableID int) error {
	table, exists := tm.tables[tableID]
	if !exists {
		return fmt.Errorf("table %d not found", tableID)
	}

	// Get all routes in the table
	filter := &netlink.Route{
		Table: tableID,
	}

	nlRoutes, err := netlink.RouteListFiltered(netlink.FAMILY_V4, filter, netlink.RT_FILTER_TABLE)
	if err != nil {
		return fmt.Errorf("failed to list routes: %w", err)
	}

	// Delete each route
	for _, nlRoute := range nlRoutes {
		if err := netlink.RouteDel(&nlRoute); err != nil {
			// Continue on error to delete as many as possible
			continue
		}
	}

	// Clear our tracking
	table.Routes = make([]*Route, 0)

	return nil
}

// scopeToNetlink converts RouteScope to netlink scope
func (tm *LinuxTableManager) scopeToNetlink(scope RouteScope) netlink.Scope {
	switch scope {
	case RouteScopeUniverse:
		return netlink.SCOPE_UNIVERSE
	case RouteScopeHost:
		return netlink.SCOPE_HOST
	case RouteScopeLink:
		return netlink.SCOPE_LINK
	case RouteScopeSite:
		return netlink.SCOPE_SITE
	default:
		return netlink.SCOPE_UNIVERSE
	}
}

// typeToNetlink converts RouteType to netlink type
func (tm *LinuxTableManager) typeToNetlink(routeType RouteType) int {
	switch routeType {
	case RouteTypeUnicast:
		return 1 // RTN_UNICAST
	case RouteTypeLocal:
		return 2 // RTN_LOCAL
	case RouteTypeBroadcast:
		return 3 // RTN_BROADCAST
	case RouteTypeMulticast:
		return 5 // RTN_MULTICAST
	case RouteTypeBlackhole:
		return 6 // RTN_BLACKHOLE
	case RouteTypeUnreachable:
		return 7 // RTN_UNREACHABLE
	case RouteTypeProhibit:
		return 8 // RTN_PROHIBIT
	default:
		return 1 // RTN_UNICAST
	}
}

// netlinkToRoute converts netlink.Route to Route
func (tm *LinuxTableManager) netlinkToRoute(nlRoute *netlink.Route) *Route {
	route := &Route{
		Destination: nlRoute.Dst,
		Gateway:     nlRoute.Gw,
		Source:      nlRoute.Src,
		Priority:    nlRoute.Priority,
		Table:       nlRoute.Table,
		MTU:         nlRoute.MTU,
		Scope:       tm.netlinkToScope(nlRoute.Scope),
		Type:        tm.netlinkToType(nlRoute.Type),
	}

	// Get interface name
	if nlRoute.LinkIndex > 0 {
		link, err := netlink.LinkByIndex(nlRoute.LinkIndex)
		if err == nil {
			route.Interface = link.Attrs().Name
		}
	}

	return route
}

// netlinkToScope converts netlink scope to RouteScope
func (tm *LinuxTableManager) netlinkToScope(scope netlink.Scope) RouteScope {
	switch scope {
	case netlink.SCOPE_UNIVERSE:
		return RouteScopeUniverse
	case netlink.SCOPE_HOST:
		return RouteScopeHost
	case netlink.SCOPE_LINK:
		return RouteScopeLink
	case netlink.SCOPE_SITE:
		return RouteScopeSite
	default:
		return RouteScopeUniverse
	}
}

// netlinkToType converts netlink type to RouteType
func (tm *LinuxTableManager) netlinkToType(routeType int) RouteType {
	switch routeType {
	case 1: // RTN_UNICAST
		return RouteTypeUnicast
	case 2: // RTN_LOCAL
		return RouteTypeLocal
	case 3: // RTN_BROADCAST
		return RouteTypeBroadcast
	case 5: // RTN_MULTICAST
		return RouteTypeMulticast
	case 6: // RTN_BLACKHOLE
		return RouteTypeBlackhole
	case 7: // RTN_UNREACHABLE
		return RouteTypeUnreachable
	case 8: // RTN_PROHIBIT
		return RouteTypeProhibit
	default:
		return RouteTypeUnicast
	}
}

// routesEqual checks if two routes are equal
func (tm *LinuxTableManager) routesEqual(r1, r2 *Route) bool {
	// Compare destinations
	if !tm.ipNetsEqual(r1.Destination, r2.Destination) {
		return false
	}

	// Compare gateways
	if !r1.Gateway.Equal(r2.Gateway) {
		return false
	}

	// Compare interfaces
	if r1.Interface != r2.Interface {
		return false
	}

	return true
}

// ipNetsEqual checks if two IPNets are equal
func (tm *LinuxTableManager) ipNetsEqual(n1, n2 *net.IPNet) bool {
	if n1 == nil && n2 == nil {
		return true
	}
	if n1 == nil || n2 == nil {
		return false
	}
	return n1.IP.Equal(n2.IP) && n1.Mask.String() == n2.Mask.String()
}

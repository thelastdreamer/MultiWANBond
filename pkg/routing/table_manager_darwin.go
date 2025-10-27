//go:build darwin

package routing

import (
	"fmt"
)

// DarwinTableManager manages routing tables on macOS
type DarwinTableManager struct {
	tables map[int]*RoutingTable
}

// NewDarwinTableManager creates a new macOS table manager
func NewDarwinTableManager() *DarwinTableManager {
	return &DarwinTableManager{
		tables: make(map[int]*RoutingTable),
	}
}

// CreateTable creates a new routing table
func (tm *DarwinTableManager) CreateTable(id int, name string) error {
	return fmt.Errorf("routing tables not yet implemented on macOS")
}

// DeleteTable deletes a routing table
func (tm *DarwinTableManager) DeleteTable(id int) error {
	return fmt.Errorf("routing tables not yet implemented on macOS")
}

// GetTable retrieves a routing table
func (tm *DarwinTableManager) GetTable(id int) (*RoutingTable, error) {
	return nil, fmt.Errorf("routing tables not yet implemented on macOS")
}

// ListTables lists all routing tables
func (tm *DarwinTableManager) ListTables() ([]*RoutingTable, error) {
	return nil, fmt.Errorf("routing tables not yet implemented on macOS")
}

// AddRoute adds a route to a table
func (tm *DarwinTableManager) AddRoute(tableID int, route *Route) error {
	return fmt.Errorf("routing tables not yet implemented on macOS")
}

// DeleteRoute deletes a route from a table
func (tm *DarwinTableManager) DeleteRoute(tableID int, route *Route) error {
	return fmt.Errorf("routing tables not yet implemented on macOS")
}

// ListRoutes lists all routes in a table
func (tm *DarwinTableManager) ListRoutes(tableID int) ([]*Route, error) {
	return nil, fmt.Errorf("routing tables not yet implemented on macOS")
}

// FlushTable removes all routes from a table
func (tm *DarwinTableManager) FlushTable(tableID int) error {
	return fmt.Errorf("routing tables not yet implemented on macOS")
}

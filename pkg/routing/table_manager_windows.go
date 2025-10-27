//go:build windows

package routing

import (
	"fmt"
)

// WindowsTableManager manages routing tables on Windows
type WindowsTableManager struct {
	tables map[int]*RoutingTable
}

// NewWindowsTableManager creates a new Windows table manager
func NewWindowsTableManager() *WindowsTableManager {
	return &WindowsTableManager{
		tables: make(map[int]*RoutingTable),
	}
}

// CreateTable creates a new routing table
func (tm *WindowsTableManager) CreateTable(id int, name string) error {
	return fmt.Errorf("routing tables not yet implemented on Windows")
}

// DeleteTable deletes a routing table
func (tm *WindowsTableManager) DeleteTable(id int) error {
	return fmt.Errorf("routing tables not yet implemented on Windows")
}

// GetTable retrieves a routing table
func (tm *WindowsTableManager) GetTable(id int) (*RoutingTable, error) {
	return nil, fmt.Errorf("routing tables not yet implemented on Windows")
}

// ListTables lists all routing tables
func (tm *WindowsTableManager) ListTables() ([]*RoutingTable, error) {
	return nil, fmt.Errorf("routing tables not yet implemented on Windows")
}

// AddRoute adds a route to a table
func (tm *WindowsTableManager) AddRoute(tableID int, route *Route) error {
	return fmt.Errorf("routing tables not yet implemented on Windows")
}

// DeleteRoute deletes a route from a table
func (tm *WindowsTableManager) DeleteRoute(tableID int, route *Route) error {
	return fmt.Errorf("routing tables not yet implemented on Windows")
}

// ListRoutes lists all routes in a table
func (tm *WindowsTableManager) ListRoutes(tableID int) ([]*Route, error) {
	return nil, fmt.Errorf("routing tables not yet implemented on Windows")
}

// FlushTable removes all routes from a table
func (tm *WindowsTableManager) FlushTable(tableID int) error {
	return fmt.Errorf("routing tables not yet implemented on Windows")
}

package routing

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Manager coordinates policy-based routing for multi-WAN
type Manager struct {
	config *RoutingConfig
	mu     sync.RWMutex

	// Components
	tableManager TableManager
	ruleManager  RuleManager

	// WAN routing tables
	wanTables map[uint8]*WANRoutingTable

	// Policies
	policies map[string]*RoutingPolicy

	// Stats
	stats *RoutingStats

	// Control
	running bool
	stopCh  chan struct{}
}

// NewManager creates a new routing manager
func NewManager(config *RoutingConfig) *Manager {
	if config == nil {
		config = DefaultRoutingConfig()
	}

	return &Manager{
		config:       config,
		tableManager: NewTableManager(),
		ruleManager:  NewRuleManager(),
		wanTables:    make(map[uint8]*WANRoutingTable),
		policies:     make(map[string]*RoutingPolicy),
		stats:        &RoutingStats{},
		stopCh:       make(chan struct{}),
	}
}

// Start starts the routing manager
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return fmt.Errorf("already running")
	}
	m.running = true
	m.mu.Unlock()

	// Start sync routine
	go m.syncRoutine()

	return nil
}

// Stop stops the routing manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)
	return nil
}

// CreateWANTable creates a routing table for a WAN interface
func (m *Manager) CreateWANTable(wanID uint8, iface string, sourceIP, gateway net.IP) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.wanTables[wanID]; exists {
		return fmt.Errorf("WAN table for WAN %d already exists", wanID)
	}

	// Calculate table ID
	tableID := m.config.TableIDStart + int(wanID)

	// Create routing table
	tableName := fmt.Sprintf("wan%d", wanID)
	if err := m.tableManager.CreateTable(tableID, tableName); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create default route via this WAN
	defaultRoute := &Route{
		Destination: &net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		},
		Gateway:   gateway,
		Interface: iface,
		Source:    sourceIP,
		Type:      RouteTypeUnicast,
		Scope:     RouteScopeUniverse,
		Table:     tableID,
	}

	// Add default route
	if err := m.tableManager.AddRoute(tableID, defaultRoute); err != nil {
		m.tableManager.DeleteTable(tableID)
		return fmt.Errorf("failed to add default route: %w", err)
	}

	// Calculate fwmark
	mark := m.config.MarkBase + uint32(wanID)

	// Create WAN table entry
	wanTable := &WANRoutingTable{
		WANID:        wanID,
		DefaultRoute: defaultRoute,
		SourceIP:     sourceIP,
		Gateway:      gateway,
		Interface:    iface,
		Mark:         mark,
		Active:       true,
	}

	m.wanTables[wanID] = wanTable
	m.stats.TablesCreated++
	m.stats.ActiveTables++
	m.stats.RoutesAdded++
	m.stats.ActiveRoutes++

	return nil
}

// DeleteWANTable deletes a WAN routing table
func (m *Manager) DeleteWANTable(wanID uint8) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wanTable, exists := m.wanTables[wanID]
	if !exists {
		return fmt.Errorf("WAN table for WAN %d not found", wanID)
	}

	tableID := m.config.TableIDStart + int(wanID)

	// Delete the table
	if err := m.tableManager.DeleteTable(tableID); err != nil {
		return fmt.Errorf("failed to delete table: %w", err)
	}

	delete(m.wanTables, wanID)
	m.stats.TablesDeleted++
	m.stats.ActiveTables--

	_ = wanTable // Avoid unused variable warning

	return nil
}

// AddSourceRoutingRule adds a source-based routing rule
func (m *Manager) AddSourceRoutingRule(sourceNet *net.IPNet, wanID uint8) error {
	if !m.config.EnableSourceRouting {
		return fmt.Errorf("source routing is disabled")
	}

	m.mu.RLock()
	wanTable, exists := m.wanTables[wanID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("WAN table for WAN %d not found", wanID)
	}

	tableID := m.config.TableIDStart + int(wanID)

	// Create policy rule
	rule := &PolicyRule{
		Priority:      1000 + int(wanID)*100, // Priority based on WAN ID
		Table:         tableID,
		SourceNetwork: sourceNet,
		Action:        RuleActionTable,
		Enabled:       true,
		Created:       time.Now(),
	}

	// Add rule
	if err := m.ruleManager.AddRule(rule); err != nil {
		return fmt.Errorf("failed to add source routing rule: %w", err)
	}

	m.mu.Lock()
	m.stats.RulesAdded++
	m.stats.ActiveRules++
	m.mu.Unlock()

	_ = wanTable // Avoid unused variable warning

	return nil
}

// AddMarkRoutingRule adds a mark-based routing rule
func (m *Manager) AddMarkRoutingRule(mark, mask uint32, wanID uint8) error {
	if !m.config.EnableMarkRouting {
		return fmt.Errorf("mark routing is disabled")
	}

	m.mu.RLock()
	wanTable, exists := m.wanTables[wanID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("WAN table for WAN %d not found", wanID)
	}

	tableID := m.config.TableIDStart + int(wanID)

	// Create policy rule
	rule := &PolicyRule{
		Priority: 2000 + int(wanID)*100, // Higher priority than source rules
		Table:    tableID,
		Mark:     mark,
		MarkMask: mask,
		Action:   RuleActionTable,
		Enabled:  true,
		Created:  time.Now(),
	}

	// Add rule
	if err := m.ruleManager.AddRule(rule); err != nil {
		return fmt.Errorf("failed to add mark routing rule: %w", err)
	}

	m.mu.Lock()
	m.stats.RulesAdded++
	m.stats.ActiveRules++
	m.mu.Unlock()

	_ = wanTable // Avoid unused variable warning

	return nil
}

// SetDefaultWAN sets the default WAN for unmatched traffic
func (m *Manager) SetDefaultWAN(wanID uint8) error {
	m.mu.RLock()
	_, exists := m.wanTables[wanID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("WAN table for WAN %d not found", wanID)
	}

	tableID := m.config.TableIDStart + int(wanID)

	// Create lowest priority rule to catch all traffic
	rule := &PolicyRule{
		Priority: 32000, // Very low priority
		Table:    tableID,
		Action:   RuleActionTable,
		Enabled:  true,
		Created:  time.Now(),
	}

	// Add rule
	if err := m.ruleManager.AddRule(rule); err != nil {
		return fmt.Errorf("failed to set default WAN: %w", err)
	}

	return nil
}

// GetWANTable returns a WAN routing table
func (m *Manager) GetWANTable(wanID uint8) (*WANRoutingTable, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	table, exists := m.wanTables[wanID]
	return table, exists
}

// ListWANTables returns all WAN routing tables
func (m *Manager) ListWANTables() map[uint8]*WANRoutingTable {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	tables := make(map[uint8]*WANRoutingTable, len(m.wanTables))
	for k, v := range m.wanTables {
		tables[k] = v
	}
	return tables
}

// AddPolicy adds a routing policy
func (m *Manager) AddPolicy(policy *RoutingPolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.policies[policy.Name]; exists {
		return fmt.Errorf("policy %s already exists", policy.Name)
	}

	// Add all rules in the policy
	for _, rule := range policy.Rules {
		if err := m.ruleManager.AddRule(rule); err != nil {
			return fmt.Errorf("failed to add policy rule: %w", err)
		}
		m.stats.RulesAdded++
		m.stats.ActiveRules++
	}

	m.policies[policy.Name] = policy
	return nil
}

// DeletePolicy deletes a routing policy
func (m *Manager) DeletePolicy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy, exists := m.policies[name]
	if !exists {
		return fmt.Errorf("policy %s not found", name)
	}

	// Delete all rules in the policy
	for _, rule := range policy.Rules {
		if err := m.ruleManager.DeleteRule(rule.Priority); err != nil {
			// Continue on error
			continue
		}
		m.stats.RulesDeleted++
		m.stats.ActiveRules--
	}

	delete(m.policies, name)
	return nil
}

// GetStats returns routing statistics
func (m *Manager) GetStats() *RoutingStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy
	stats := *m.stats
	return &stats
}

// syncRoutine periodically syncs routing state
func (m *Manager) syncRoutine() {
	ticker := time.NewTicker(m.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.sync()
		}
	}
}

// sync synchronizes routing state with kernel
func (m *Manager) sync() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// List all rules from kernel
	rules, err := m.ruleManager.ListRules()
	if err != nil {
		m.stats.SyncErrors++
		return fmt.Errorf("failed to list rules: %w", err)
	}

	// Update our active rules count
	m.stats.ActiveRules = uint64(len(rules))

	// List all tables
	tables, err := m.tableManager.ListTables()
	if err != nil {
		m.stats.SyncErrors++
		return fmt.Errorf("failed to list tables: %w", err)
	}

	// Update our active tables count
	m.stats.ActiveTables = uint64(len(tables))

	// Count total routes
	totalRoutes := uint64(0)
	for _, table := range tables {
		routes, err := m.tableManager.ListRoutes(table.ID)
		if err == nil {
			totalRoutes += uint64(len(routes))
		}
	}
	m.stats.ActiveRoutes = totalRoutes

	m.stats.LastSync = time.Now()
	return nil
}

// Flush removes all custom routing rules and tables
func (m *Manager) Flush() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Flush all WAN tables
	for wanID := range m.wanTables {
		tableID := m.config.TableIDStart + int(wanID)
		m.tableManager.DeleteTable(tableID)
	}

	// Flush all rules
	if err := m.ruleManager.FlushRules(); err != nil {
		return fmt.Errorf("failed to flush rules: %w", err)
	}

	// Clear our tracking
	m.wanTables = make(map[uint8]*WANRoutingTable)
	m.policies = make(map[string]*RoutingPolicy)
	m.stats.ActiveTables = 0
	m.stats.ActiveRoutes = 0
	m.stats.ActiveRules = 0

	return nil
}

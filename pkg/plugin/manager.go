package plugin

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// Manager manages plugins in the system
type Manager struct {
	mu             sync.RWMutex
	plugins        map[string]protocol.Plugin
	filters        []protocol.PacketFilter
	metrics        []protocol.MetricsCollector
	alerts         []protocol.AlertManager
	running        bool
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]protocol.Plugin),
		filters: make([]protocol.PacketFilter, 0),
		metrics: make([]protocol.MetricsCollector, 0),
		alerts:  make([]protocol.AlertManager, 0),
	}
}

// Register registers a plugin
func (m *Manager) Register(plugin protocol.Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := plugin.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	m.plugins[name] = plugin

	// Add to specialized lists based on interface
	if filter, ok := plugin.(protocol.PacketFilter); ok {
		m.filters = append(m.filters, filter)
		// Sort filters by priority
		sort.Slice(m.filters, func(i, j int) bool {
			return m.filters[i].Priority() < m.filters[j].Priority()
		})
	}

	if collector, ok := plugin.(protocol.MetricsCollector); ok {
		m.metrics = append(m.metrics, collector)
	}

	if alertMgr, ok := plugin.(protocol.AlertManager); ok {
		m.alerts = append(m.alerts, alertMgr)
	}

	return nil
}

// Unregister unregisters a plugin
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Stop plugin if running
	if m.running {
		if err := plugin.Stop(); err != nil {
			return fmt.Errorf("failed to stop plugin: %w", err)
		}
	}

	// Remove from specialized lists
	if filter, ok := plugin.(protocol.PacketFilter); ok {
		m.removeFilter(filter)
	}

	if collector, ok := plugin.(protocol.MetricsCollector); ok {
		m.removeMetricsCollector(collector)
	}

	if alertMgr, ok := plugin.(protocol.AlertManager); ok {
		m.removeAlertManager(alertMgr)
	}

	delete(m.plugins, name)
	return nil
}

// Get returns a plugin by name
func (m *Manager) Get(name string) (protocol.Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}

	return plugin, nil
}

// List returns all registered plugins
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}

	return names
}

// StartAll starts all registered plugins
func (m *Manager) StartAll(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("plugins already running")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)

	for name, plugin := range m.plugins {
		if err := plugin.Start(m.ctx); err != nil {
			// Stop already started plugins
			m.stopAll()
			return fmt.Errorf("failed to start plugin %s: %w", name, err)
		}
	}

	m.running = true
	return nil
}

// StopAll stops all running plugins
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return fmt.Errorf("plugins not running")
	}

	m.cancel()
	m.stopAll()
	m.running = false

	return nil
}

// stopAll stops all plugins (must be called with lock held)
func (m *Manager) stopAll() {
	for _, plugin := range m.plugins {
		plugin.Stop() // Ignore errors during shutdown
	}
}

// FilterOutgoing applies all packet filters to outgoing packets
func (m *Manager) FilterOutgoing(packet *protocol.Packet) (*protocol.Packet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current := packet
	for _, filter := range m.filters {
		filtered, err := filter.FilterOutgoing(current)
		if err != nil {
			return nil, err
		}
		if filtered == nil {
			// Packet was dropped
			return nil, nil
		}
		current = filtered
	}

	return current, nil
}

// FilterIncoming applies all packet filters to incoming packets
func (m *Manager) FilterIncoming(packet *protocol.Packet) (*protocol.Packet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	current := packet
	for _, filter := range m.filters {
		filtered, err := filter.FilterIncoming(current)
		if err != nil {
			return nil, err
		}
		if filtered == nil {
			// Packet was dropped
			return nil, nil
		}
		current = filtered
	}

	return current, nil
}

// RecordPacket records packet metrics to all collectors
func (m *Manager) RecordPacket(wanID uint8, packet *protocol.Packet, sent bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, collector := range m.metrics {
		collector.RecordPacket(wanID, packet, sent)
	}
}

// RecordMetrics records WAN metrics to all collectors
func (m *Manager) RecordMetrics(wanID uint8, metrics *protocol.WANMetrics) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, collector := range m.metrics {
		collector.RecordMetrics(wanID, metrics)
	}
}

// Alert sends an alert to all alert managers
func (m *Manager) Alert(level protocol.AlertLevel, message string, details map[string]interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, alertMgr := range m.alerts {
		if err := alertMgr.Alert(level, message, details); err != nil {
			// Log error but continue to other alert managers
			continue
		}
	}

	return nil
}

// removeFilter removes a filter from the list
func (m *Manager) removeFilter(filter protocol.PacketFilter) {
	for i, f := range m.filters {
		if f == filter {
			m.filters = append(m.filters[:i], m.filters[i+1:]...)
			break
		}
	}
}

// removeMetricsCollector removes a metrics collector from the list
func (m *Manager) removeMetricsCollector(collector protocol.MetricsCollector) {
	for i, c := range m.metrics {
		if c == collector {
			m.metrics = append(m.metrics[:i], m.metrics[i+1:]...)
			break
		}
	}
}

// removeAlertManager removes an alert manager from the list
func (m *Manager) removeAlertManager(alertMgr protocol.AlertManager) {
	for i, a := range m.alerts {
		if a == alertMgr {
			m.alerts = append(m.alerts[:i], m.alerts[i+1:]...)
			break
		}
	}
}

// PluginLoader loads plugins from external sources (e.g., shared libraries)
type PluginLoader struct {
	mu      sync.RWMutex
	manager *Manager
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(manager *Manager) *PluginLoader {
	return &PluginLoader{
		manager: manager,
	}
}

// LoadFromFile loads a plugin from a file (e.g., .so, .dll)
// This is a placeholder for future implementation using Go's plugin package
func (pl *PluginLoader) LoadFromFile(path string) error {
	// TODO: Implement dynamic plugin loading
	// For now, plugins must be compiled in
	return fmt.Errorf("dynamic plugin loading not yet implemented")
}

// BasePlugin provides common functionality for plugins
type BasePlugin struct {
	name    string
	version string
	config  map[string]interface{}
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version string) *BasePlugin {
	return &BasePlugin{
		name:    name,
		version: version,
		config:  make(map[string]interface{}),
	}
}

// Name returns the plugin name
func (bp *BasePlugin) Name() string {
	return bp.name
}

// Version returns the plugin version
func (bp *BasePlugin) Version() string {
	return bp.version
}

// Init initializes the plugin
func (bp *BasePlugin) Init(config map[string]interface{}) error {
	bp.config = config
	return nil
}

// Start starts the plugin (default implementation does nothing)
func (bp *BasePlugin) Start(ctx context.Context) error {
	return nil
}

// Stop stops the plugin (default implementation does nothing)
func (bp *BasePlugin) Stop() error {
	return nil
}

// GetConfig returns the plugin configuration
func (bp *BasePlugin) GetConfig() map[string]interface{} {
	return bp.config
}

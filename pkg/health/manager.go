package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// Manager coordinates health checking for multiple WANs
type Manager struct {
	mu          sync.RWMutex
	checkers    map[uint8]*SmartChecker
	configs     map[uint8]*CheckConfig
	eventChan   chan HealthEvent
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	running     bool
}

// HealthEvent represents a health status change event
type HealthEvent struct {
	WANID         uint8
	InterfaceName string
	OldStatus     WANStatus
	NewStatus     WANStatus
	Timestamp     time.Time
	Reason        string
	CheckResult   *CheckResult
}

// NewManager creates a new health check manager
func NewManager() *Manager {
	return &Manager{
		checkers:  make(map[uint8]*SmartChecker),
		configs:   make(map[uint8]*CheckConfig),
		eventChan: make(chan HealthEvent, 100),
	}
}

// AddWAN adds a WAN interface to monitor
func (m *Manager) AddWAN(wanID uint8, interfaceName string, config *CheckConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.checkers[wanID]; exists {
		return fmt.Errorf("WAN %d already being monitored", wanID)
	}

	// Create config if not provided
	if config == nil {
		config = DefaultCheckConfig(wanID, interfaceName)
	} else {
		config.WANID = wanID
		config.InterfaceName = interfaceName
	}

	// Create smart checker for this WAN
	checker := NewSmartChecker(config)

	m.checkers[wanID] = checker
	m.configs[wanID] = config

	// If manager is running, start monitoring this WAN immediately
	if m.running {
		m.wg.Add(1)
		go m.monitorWAN(wanID, checker)
	}

	return nil
}

// RemoveWAN removes a WAN interface from monitoring
func (m *Manager) RemoveWAN(wanID uint8) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.checkers[wanID]; !exists {
		return fmt.Errorf("WAN %d not being monitored", wanID)
	}

	delete(m.checkers, wanID)
	delete(m.configs, wanID)

	return nil
}

// Start begins health monitoring for all WANs
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("health manager already running")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.running = true

	// Start monitoring goroutine for each WAN
	for wanID, checker := range m.checkers {
		m.wg.Add(1)
		go m.monitorWAN(wanID, checker)
	}

	return nil
}

// Stop stops health monitoring
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return fmt.Errorf("health manager not running")
	}
	m.cancel()
	m.running = false
	m.mu.Unlock()

	m.wg.Wait()

	return nil
}

// monitorWAN continuously monitors a single WAN interface
func (m *Manager) monitorWAN(wanID uint8, checker *SmartChecker) {
	defer m.wg.Done()

	ticker := time.NewTicker(checker.GetCurrentInterval())
	defer ticker.Stop()

	var lastStatus WANStatus = WANStatusUnknown

	for {
		select {
		case <-m.ctx.Done():
			return

		case <-ticker.C:
			// Perform health check
			result, _ := checker.Check()

			if result != nil {
				// Check for status change
				if result.Status != lastStatus {
					event := HealthEvent{
						WANID:         wanID,
						InterfaceName: checker.config.InterfaceName,
						OldStatus:     lastStatus,
						NewStatus:     result.Status,
						Timestamp:     result.Timestamp,
						Reason:        fmt.Sprintf("Health check result: %v", result.Success),
						CheckResult:   result,
					}

					// Send event (non-blocking)
					select {
					case m.eventChan <- event:
					default:
						// Channel full, skip event
					}

					lastStatus = result.Status
				}
			}

			// Adjust ticker interval if adaptive
			if checker.config.AdaptiveInterval {
				newInterval := checker.GetCurrentInterval()
				ticker.Reset(newInterval)
			}
		}
	}
}

// GetWANHealth returns the health status for a specific WAN
func (m *Manager) GetWANHealth(wanID uint8) (*WANHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	checker, exists := m.checkers[wanID]
	if !exists {
		return nil, fmt.Errorf("WAN %d not being monitored", wanID)
	}

	return checker.GetHealth(), nil
}

// GetAllWANHealth returns health status for all WANs
func (m *Manager) GetAllWANHealth() map[uint8]*WANHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[uint8]*WANHealth)
	for wanID, checker := range m.checkers {
		result[wanID] = checker.GetHealth()
	}

	return result
}

// GetHealthyWANs returns a list of WAN IDs that are currently healthy
func (m *Manager) GetHealthyWANs() []uint8 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthy []uint8
	for wanID, checker := range m.checkers {
		health := checker.GetHealth()
		if health.Status == WANStatusUp || health.Status == WANStatusDegraded {
			healthy = append(healthy, wanID)
		}
	}

	return healthy
}

// GetDownWANs returns a list of WAN IDs that are currently down
func (m *Manager) GetDownWANs() []uint8 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var down []uint8
	for wanID, checker := range m.checkers {
		health := checker.GetHealth()
		if health.Status == WANStatusDown {
			down = append(down, wanID)
		}
	}

	return down
}

// GetEventChannel returns the channel for health events
func (m *Manager) GetEventChannel() <-chan HealthEvent {
	return m.eventChan
}

// UpdateWANConfig updates the configuration for a WAN
func (m *Manager) UpdateWANConfig(wanID uint8, config *CheckConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	checker, exists := m.checkers[wanID];
	if !exists {
		return fmt.Errorf("WAN %d not being monitored", wanID)
	}

	config.WANID = wanID
	config.InterfaceName = checker.config.InterfaceName

	// Update config
	checker.mu.Lock()
	checker.config = config
	checker.mu.Unlock()

	m.configs[wanID] = config

	return nil
}

// GetWANConfig returns the configuration for a WAN
func (m *Manager) GetWANConfig(wanID uint8) (*CheckConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config, exists := m.configs[wanID]
	if !exists {
		return nil, fmt.Errorf("WAN %d not being monitored", wanID)
	}

	// Return a copy
	configCopy := *config
	return &configCopy, nil
}

// IsWANHealthy returns true if the WAN is currently healthy (up or degraded)
func (m *Manager) IsWANHealthy(wanID uint8) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	checker, exists := m.checkers[wanID]
	if !exists {
		return false
	}

	health := checker.GetHealth()
	return health.Status == WANStatusUp || health.Status == WANStatusDegraded
}

// GetBestWAN returns the WAN ID with the best performance metrics
func (m *Manager) GetBestWAN() (uint8, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var bestWAN uint8
	var bestScore float64 = -1
	found := false

	for wanID, checker := range m.checkers {
		health := checker.GetHealth()

		// Only consider healthy WANs
		if health.Status != WANStatusUp && health.Status != WANStatusDegraded {
			continue
		}

		// Calculate score based on uptime, latency, and packet loss
		// Higher is better
		score := health.Uptime * 100.0 // 0-100 points for uptime

		// Subtract latency penalty (normalized to 0-50ms = 0 penalty, 500ms+ = 50 penalty)
		latencyMs := float64(health.AvgLatency.Milliseconds())
		latencyPenalty := (latencyMs / 500.0) * 50.0
		if latencyPenalty > 50 {
			latencyPenalty = 50
		}
		score -= latencyPenalty

		// Subtract packet loss penalty (0-100%)
		score -= health.AvgPacketLoss * 100.0

		if score > bestScore {
			bestScore = score
			bestWAN = wanID
			found = true
		}
	}

	if !found {
		return 0, fmt.Errorf("no healthy WAN interfaces available")
	}

	return bestWAN, nil
}

// GetWANMetrics converts WANHealth to protocol.WANMetrics format
func (m *Manager) GetWANMetrics(wanID uint8) (*protocol.WANMetrics, error) {
	health, err := m.GetWANHealth(wanID)
	if err != nil {
		return nil, err
	}

	metrics := &protocol.WANMetrics{
		Latency:    health.AvgLatency,
		Jitter:     health.AvgJitter,
		PacketLoss: health.AvgPacketLoss,
		Bandwidth:  0, // Not measured by health checker
		LastUpdate: health.LastCheck,
	}

	return metrics, nil
}

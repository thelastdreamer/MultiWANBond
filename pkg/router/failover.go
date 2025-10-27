package router

import (
	"fmt"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// FailoverManager handles automatic failover between WANs based on health
type FailoverManager struct {
	mu               sync.RWMutex
	router           *Router
	wanHealth        map[uint8]bool // WANID -> is healthy
	primaryWAN       uint8          // Current primary WAN
	activeWAN        uint8          // Currently active WAN
	wansByPriority   []uint8        // WANs sorted by priority (0 = highest)
	lastFailover     time.Time
	failoverCount    uint64
	failoverCallback func(oldWAN, newWAN uint8, reason string)
}

// NewFailoverManager creates a new failover manager
func NewFailoverManager(router *Router) *FailoverManager {
	return &FailoverManager{
		router:         router,
		wanHealth:      make(map[uint8]bool),
		wansByPriority: make([]uint8, 0),
	}
}

// SetFailoverCallback sets a callback function that's called when failover occurs
func (fm *FailoverManager) SetFailoverCallback(callback func(oldWAN, newWAN uint8, reason string)) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.failoverCallback = callback
}

// UpdateWANHealth updates the health status of a WAN interface
// Returns true if failover was triggered
func (fm *FailoverManager) UpdateWANHealth(wanID uint8, isHealthy bool) bool {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	oldHealth := fm.wanHealth[wanID]
	fm.wanHealth[wanID] = isHealthy

	// If health status changed, log it
	if oldHealth != isHealthy {
		if isHealthy {
			fmt.Printf("[Failover] WAN %d came back up\n", wanID)
		} else {
			fmt.Printf("[Failover] WAN %d went down\n", wanID)
		}
	}

	// Check if current active WAN is down
	if !isHealthy && wanID == fm.activeWAN {
		// Current WAN is down - trigger failover
		return fm.performFailover()
	}

	// Check if a higher priority WAN came back up
	if isHealthy && fm.shouldFailbackTo(wanID) {
		// Higher priority WAN is back - fail back to it
		return fm.performFailback(wanID)
	}

	return false
}

// performFailover switches to the next available WAN
func (fm *FailoverManager) performFailover() bool {
	oldWAN := fm.activeWAN

	// Find next healthy WAN by priority
	newWAN := fm.findNextHealthyWAN()

	if newWAN == 0 {
		fmt.Printf("[Failover] WARNING: No healthy WANs available!\n")
		return false
	}

	if newWAN == fm.activeWAN {
		// Already on the best available WAN
		return false
	}

	// Perform failover
	fm.activeWAN = newWAN
	fm.lastFailover = time.Now()
	fm.failoverCount++

	reason := fmt.Sprintf("WAN %d failed health check", oldWAN)
	fmt.Printf("[Failover] Switched from WAN %d to WAN %d (reason: %s)\n", oldWAN, newWAN, reason)

	// Call callback if set
	if fm.failoverCallback != nil {
		go fm.failoverCallback(oldWAN, newWAN, reason)
	}

	return true
}

// performFailback switches back to a higher priority WAN
func (fm *FailoverManager) performFailback(higherPriorityWAN uint8) bool {
	oldWAN := fm.activeWAN

	// Only fail back if the higher priority WAN has been stable
	// (avoid flapping)
	const stabilityPeriod = 5 * time.Second
	if time.Since(fm.lastFailover) < stabilityPeriod {
		return false // Too soon after last failover
	}

	fm.activeWAN = higherPriorityWAN
	fm.lastFailover = time.Now()

	reason := fmt.Sprintf("WAN %d (higher priority) came back up", higherPriorityWAN)
	fmt.Printf("[Failover] Failing back from WAN %d to WAN %d (reason: %s)\n", oldWAN, higherPriorityWAN, reason)

	// Call callback if set
	if fm.failoverCallback != nil {
		go fm.failoverCallback(oldWAN, higherPriorityWAN, reason)
	}

	return true
}

// findNextHealthyWAN finds the next available healthy WAN by priority
func (fm *FailoverManager) findNextHealthyWAN() uint8 {
	// Return first healthy WAN by priority
	for _, wanID := range fm.wansByPriority {
		if fm.wanHealth[wanID] {
			return wanID
		}
	}

	return 0 // No healthy WANs
}

// shouldFailbackTo checks if we should fail back to a higher priority WAN
func (fm *FailoverManager) shouldFailbackTo(wanID uint8) bool {
	// Get priorities
	currentPriority := fm.getWANPriority(fm.activeWAN)
	newPriority := fm.getWANPriority(wanID)

	// Fail back if new WAN has higher priority (lower number)
	return newPriority < currentPriority
}

// getWANPriority returns the priority of a WAN
func (fm *FailoverManager) getWANPriority(wanID uint8) int {
	fm.router.mu.RLock()
	defer fm.router.mu.RUnlock()

	if wan, exists := fm.router.wans[wanID]; exists {
		return wan.Config.Priority
	}

	return 999 // Very low priority if WAN not found
}

// UpdateWANsByPriority updates the list of WANs sorted by priority
func (fm *FailoverManager) UpdateWANsByPriority(wans map[uint8]*protocol.WANInterface) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Build list of WAN IDs sorted by priority
	type wanPriority struct {
		ID       uint8
		Priority int
	}

	priorities := make([]wanPriority, 0, len(wans))
	for wanID, wan := range wans {
		priorities = append(priorities, wanPriority{
			ID:       wanID,
			Priority: wan.Config.Priority,
		})

		// Initialize health status if not present
		if _, exists := fm.wanHealth[wanID]; !exists {
			fm.wanHealth[wanID] = true // Assume healthy initially
		}
	}

	// Sort by priority (lower number = higher priority)
	// Simple bubble sort for small arrays
	for i := 0; i < len(priorities); i++ {
		for j := i + 1; j < len(priorities); j++ {
			if priorities[j].Priority < priorities[i].Priority {
				priorities[i], priorities[j] = priorities[j], priorities[i]
			}
		}
	}

	// Extract WAN IDs
	fm.wansByPriority = make([]uint8, len(priorities))
	for i, wp := range priorities {
		fm.wansByPriority[i] = wp.ID
	}

	// Set primary and active WAN to highest priority (if not set)
	if len(fm.wansByPriority) > 0 && fm.primaryWAN == 0 {
		fm.primaryWAN = fm.wansByPriority[0]
		fm.activeWAN = fm.wansByPriority[0]
	}
}

// GetActiveWAN returns the currently active WAN
func (fm *FailoverManager) GetActiveWAN() uint8 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.activeWAN
}

// GetPrimaryWAN returns the primary (highest priority) WAN
func (fm *FailoverManager) GetPrimaryWAN() uint8 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.primaryWAN
}

// GetWANsByPriority returns all WANs in priority order
func (fm *FailoverManager) GetWANsByPriority() []uint8 {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	// Return a copy
	wans := make([]uint8, len(fm.wansByPriority))
	copy(wans, fm.wansByPriority)
	return wans
}

// GetFailoverStats returns failover statistics
func (fm *FailoverManager) GetFailoverStats() (count uint64, lastTime time.Time) {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.failoverCount, fm.lastFailover
}

// IsWANHealthy returns the health status of a WAN
func (fm *FailoverManager) IsWANHealthy(wanID uint8) bool {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	return fm.wanHealth[wanID]
}

// GetHealthyWANCount returns the number of healthy WANs
func (fm *FailoverManager) GetHealthyWANCount() int {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	count := 0
	for _, healthy := range fm.wanHealth {
		if healthy {
			count++
		}
	}
	return count
}

// ForceFailoverTo forces failover to a specific WAN (for testing/manual control)
func (fm *FailoverManager) ForceFailoverTo(wanID uint8) error {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	// Check if WAN exists and is healthy
	if !fm.wanHealth[wanID] {
		return fmt.Errorf("WAN %d is not healthy", wanID)
	}

	oldWAN := fm.activeWAN
	fm.activeWAN = wanID
	fm.lastFailover = time.Now()
	fm.failoverCount++

	reason := "Manual failover"
	fmt.Printf("[Failover] Manually switched from WAN %d to WAN %d\n", oldWAN, wanID)

	// Call callback if set
	if fm.failoverCallback != nil {
		go fm.failoverCallback(oldWAN, wanID, reason)
	}

	return nil
}

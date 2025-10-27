package router

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

// Router implements intelligent packet routing across multiple WANs
type Router struct {
	mu              sync.RWMutex
	wans            map[uint8]*protocol.WANInterface
	mode            protocol.LoadBalanceMode
	currentWAN      uint8 // For round-robin
	flowMap         map[protocol.FlowKey]uint8
	metrics         map[uint8]*protocol.WANMetrics
	bandwidthUsage  map[uint8]uint64
	lastCleanup     time.Time
}

// NewRouter creates a new router
func NewRouter(mode protocol.LoadBalanceMode) *Router {
	return &Router{
		wans:           make(map[uint8]*protocol.WANInterface),
		mode:           mode,
		flowMap:        make(map[protocol.FlowKey]uint8),
		metrics:        make(map[uint8]*protocol.WANMetrics),
		bandwidthUsage: make(map[uint8]uint64),
		lastCleanup:    time.Now(),
	}
}

// AddWAN adds a WAN interface to the router
func (r *Router) AddWAN(wan *protocol.WANInterface) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wans[wan.ID] = wan
	r.bandwidthUsage[wan.ID] = 0
}

// RemoveWAN removes a WAN interface from the router
func (r *Router) RemoveWAN(wanID uint8) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.wans, wanID)
	delete(r.bandwidthUsage, wanID)
}

// Route determines routing for a packet
func (r *Router) Route(packet *protocol.Packet, flowKey *protocol.FlowKey) (*protocol.RoutingDecision, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Get available WANs
	availableWANs := r.getAvailableWANs()
	if len(availableWANs) == 0 {
		return nil, fmt.Errorf("no available WAN interfaces")
	}

	decision := &protocol.RoutingDecision{
		Priority: packet.Priority,
	}

	// Select primary WAN based on mode
	switch r.mode {
	case protocol.LoadBalanceRoundRobin:
		decision.PrimaryWAN = r.routeRoundRobin(availableWANs)

	case protocol.LoadBalanceWeighted:
		decision.PrimaryWAN = r.routeWeighted(availableWANs)

	case protocol.LoadBalanceLeastUsed:
		decision.PrimaryWAN = r.routeLeastUsed(availableWANs)

	case protocol.LoadBalanceLeastLatency:
		decision.PrimaryWAN = r.routeLeastLatency(availableWANs)

	case protocol.LoadBalancePerFlow:
		if flowKey != nil {
			decision.PrimaryWAN = r.routePerFlow(flowKey, availableWANs)
		} else {
			// Fallback to round-robin if no flow key
			decision.PrimaryWAN = r.routeRoundRobin(availableWANs)
		}

	case protocol.LoadBalanceAdaptive:
		decision.PrimaryWAN = r.routeAdaptive(availableWANs, packet)

	default:
		decision.PrimaryWAN = r.routeRoundRobin(availableWANs)
	}

	// Determine if we should use backup WANs
	primaryWAN := r.wans[decision.PrimaryWAN]
	if packet.Priority > 200 || (packet.Flags&protocol.FlagDuplicate) != 0 {
		// High priority or explicitly marked for duplication
		decision.BackupWANs = r.selectBackupWANs(decision.PrimaryWAN, availableWANs, 1)
	}

	// Determine if we should use FEC
	if primaryWAN.Metrics != nil {
		// Use FEC if packet loss is high
		if primaryWAN.Metrics.PacketLoss > 5.0 {
			decision.UseFEC = true
		}
	}

	return decision, nil
}

// getAvailableWANs returns WANs that are up and enabled
func (r *Router) getAvailableWANs() []uint8 {
	available := make([]uint8, 0, len(r.wans))
	for id, wan := range r.wans {
		if wan.Config.Enabled && (wan.State == protocol.WANStateUp || wan.State == protocol.WANStateRecovering) {
			available = append(available, id)
		}
	}
	return available
}

// routeRoundRobin implements round-robin routing
func (r *Router) routeRoundRobin(availableWANs []uint8) uint8 {
	if len(availableWANs) == 0 {
		return 0
	}

	// Find next available WAN
	for i := 0; i < len(availableWANs); i++ {
		r.currentWAN = (r.currentWAN + 1) % uint8(len(availableWANs))
		wanID := availableWANs[r.currentWAN]
		if _, exists := r.wans[wanID]; exists {
			return wanID
		}
	}

	return availableWANs[0]
}

// routeWeighted implements weighted routing based on bandwidth and latency
func (r *Router) routeWeighted(availableWANs []uint8) uint8 {
	if len(availableWANs) == 0 {
		return 0
	}

	type wanScore struct {
		id    uint8
		score float64
	}

	scores := make([]wanScore, 0, len(availableWANs))

	for _, id := range availableWANs {
		wan := r.wans[id]
		metrics := r.metrics[id]

		// Calculate score based on multiple factors
		score := float64(wan.Config.Weight)

		// Adjust based on latency (lower is better)
		if metrics != nil && metrics.AvgLatency > 0 {
			latencyMs := float64(metrics.AvgLatency.Milliseconds())
			score *= 100.0 / (latencyMs + 1)
		}

		// Adjust based on packet loss (lower is better)
		if metrics != nil {
			score *= (100.0 - metrics.PacketLoss) / 100.0
		}

		// Adjust based on bandwidth usage
		if wan.Config.MaxBandwidth > 0 {
			usage := r.bandwidthUsage[id]
			utilization := float64(usage) / float64(wan.Config.MaxBandwidth)
			score *= (1.0 - utilization)
		}

		scores = append(scores, wanScore{id: id, score: score})
	}

	// Select WAN with highest score
	bestWAN := scores[0]
	for _, ws := range scores[1:] {
		if ws.score > bestWAN.score {
			bestWAN = ws
		}
	}

	return bestWAN.id
}

// routeLeastUsed implements routing to least utilized WAN
func (r *Router) routeLeastUsed(availableWANs []uint8) uint8 {
	if len(availableWANs) == 0 {
		return 0
	}

	leastUsed := availableWANs[0]
	minUsage := r.bandwidthUsage[leastUsed]

	for _, id := range availableWANs[1:] {
		usage := r.bandwidthUsage[id]
		if usage < minUsage {
			minUsage = usage
			leastUsed = id
		}
	}

	return leastUsed
}

// routeLeastLatency implements routing to lowest latency WAN
func (r *Router) routeLeastLatency(availableWANs []uint8) uint8 {
	if len(availableWANs) == 0 {
		return 0
	}

	bestWAN := availableWANs[0]
	bestLatency := time.Duration(1 << 62) // Max duration

	for _, id := range availableWANs {
		metrics := r.metrics[id]
		if metrics != nil && metrics.AvgLatency < bestLatency {
			bestLatency = metrics.AvgLatency
			bestWAN = id
		}
	}

	return bestWAN
}

// routePerFlow implements consistent per-flow routing
func (r *Router) routePerFlow(flowKey *protocol.FlowKey, availableWANs []uint8) uint8 {
	// Check if we already have a WAN for this flow
	if wanID, exists := r.flowMap[*flowKey]; exists {
		// Verify WAN is still available
		for _, id := range availableWANs {
			if id == wanID {
				return wanID
			}
		}
	}

	// New flow or previous WAN unavailable - use consistent hashing
	hash := r.hashFlow(flowKey)
	wanID := availableWANs[hash%uint32(len(availableWANs))]
	r.flowMap[*flowKey] = wanID

	// Cleanup old flows periodically
	if time.Since(r.lastCleanup) > 5*time.Minute {
		r.cleanupFlowMap()
	}

	return wanID
}

// routeAdaptive implements adaptive routing based on real-time conditions
func (r *Router) routeAdaptive(availableWANs []uint8, packet *protocol.Packet) uint8 {
	// For high priority packets, use lowest latency
	if packet.Priority > 200 {
		return r.routeLeastLatency(availableWANs)
	}

	// For bulk traffic, use least used
	if packet.Priority < 50 {
		return r.routeLeastUsed(availableWANs)
	}

	// For normal traffic, use weighted routing
	return r.routeWeighted(availableWANs)
}

// selectBackupWANs selects backup WANs for redundancy
func (r *Router) selectBackupWANs(primaryWAN uint8, availableWANs []uint8, count int) []uint8 {
	backups := make([]uint8, 0, count)

	for _, id := range availableWANs {
		if id != primaryWAN && len(backups) < count {
			backups = append(backups, id)
		}
	}

	return backups
}

// hashFlow creates a hash from a flow key
func (r *Router) hashFlow(flowKey *protocol.FlowKey) uint32 {
	h := fnv.New32a()
	h.Write(flowKey.SrcIP)
	h.Write(flowKey.DstIP)
	h.Write([]byte{byte(flowKey.SrcPort >> 8), byte(flowKey.SrcPort)})
	h.Write([]byte{byte(flowKey.DstPort >> 8), byte(flowKey.DstPort)})
	h.Write([]byte{flowKey.Protocol})
	return h.Sum32()
}

// cleanupFlowMap removes old flow mappings
func (r *Router) cleanupFlowMap() {
	// Simple cleanup: clear entire map
	// In production, you'd track flow last-seen times
	r.flowMap = make(map[protocol.FlowKey]uint8)
	r.lastCleanup = time.Now()
}

// UpdateMetrics updates routing decisions based on new metrics
func (r *Router) UpdateMetrics(wanID uint8, metrics *protocol.WANMetrics) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.metrics[wanID] = metrics
}

// RecordBandwidthUsage records bandwidth usage for a WAN
func (r *Router) RecordBandwidthUsage(wanID uint8, bytes uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.bandwidthUsage[wanID] += bytes
}

// ResetBandwidthUsage resets bandwidth usage counters
func (r *Router) ResetBandwidthUsage() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id := range r.bandwidthUsage {
		r.bandwidthUsage[id] = 0
	}
}

// SetMode changes the load balancing mode
func (r *Router) SetMode(mode protocol.LoadBalanceMode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mode = mode
}

// GetMode returns the current load balancing mode
func (r *Router) GetMode() protocol.LoadBalanceMode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.mode
}

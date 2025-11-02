package dpi

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// Classifier manages flow classification and tracking
type Classifier struct {
	config   *DPIConfig
	detector *Detector
	mu       sync.RWMutex

	// Active flows
	flows map[string]*Flow

	// Statistics
	stats *DPIStats

	// Policies
	policies map[string]*ApplicationPolicy

	// Control
	running bool
	stopCh  chan struct{}
}

// NewClassifier creates a new traffic classifier
func NewClassifier(config *DPIConfig) *Classifier {
	if config == nil {
		config = DefaultDPIConfig()
	}

	return &Classifier{
		config:   config,
		detector: NewDetector(config),
		flows:    make(map[string]*Flow),
		stats: &DPIStats{
			ProtocolStats: make(map[Protocol]uint64),
			CategoryStats: make(map[Category]uint64),
		},
		policies: make(map[string]*ApplicationPolicy),
		stopCh:   make(chan struct{}),
	}
}

// Start starts the classifier
func (c *Classifier) Start() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("already running")
	}
	c.running = true
	c.mu.Unlock()

	// Start flow cleanup routine
	go c.cleanupRoutine()

	return nil
}

// Stop stops the classifier
func (c *Classifier) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	c.mu.Unlock()

	close(c.stopCh)
	return nil
}

// ClassifyPacket classifies a packet and updates flow state
func (c *Classifier) ClassifyPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, proto uint8, payload []byte, isUpload bool) (*Classification, *Flow) {
	// Create flow key
	flowKey := makeFlowKey(srcIP, dstIP, srcPort, dstPort, proto)

	c.mu.Lock()

	// Get or create flow
	flow, exists := c.flows[flowKey]
	if !exists {
		// Check max flows limit
		if len(c.flows) >= c.config.MaxFlows {
			c.mu.Unlock()
			return nil, nil
		}

		// Create new flow
		flow = &Flow{
			SrcIP:     srcIP,
			DstIP:     dstIP,
			SrcPort:   srcPort,
			DstPort:   dstPort,
			Proto:     proto,
			FirstSeen: time.Now(),
			Protocol:  ProtocolUnknown,
			Category:  CategoryUnknown,
		}
		c.flows[flowKey] = flow
		c.stats.TotalFlows++
		c.stats.ActiveFlows++
	}

	// Update flow statistics
	flow.LastSeen = time.Now()
	flow.Packets++
	flow.Bytes += uint64(len(payload))

	if isUpload {
		flow.PacketsUp++
		flow.BytesUp += uint64(len(payload))
	} else {
		flow.PacketsDown++
		flow.BytesDown += uint64(len(payload))
	}

	c.stats.TotalPackets++
	c.stats.TotalBytes += uint64(len(payload))

	// Classify if not yet classified or low confidence
	var classification *Classification
	if flow.Protocol == ProtocolUnknown || flow.Confidence < 0.7 {
		// Release lock during classification
		c.mu.Unlock()

		// Detect protocol
		if proto == 6 && dstPort == 443 { // TCP + HTTPS
			classification = c.detector.ClassifyTLS(payload)
		} else {
			classification = c.detector.Classify(payload, srcPort, dstPort)
		}

		c.mu.Lock()

		// Update flow with classification
		if classification.Confidence > flow.Confidence {
			flow.Protocol = classification.Protocol
			flow.Category = classification.Category
			flow.Confidence = classification.Confidence

			// Update stats
			if flow.Protocol != ProtocolUnknown {
				c.stats.ClassifiedFlows++
				c.stats.ProtocolStats[flow.Protocol]++
				c.stats.CategoryStats[flow.Category]++
				c.stats.LastClassification = time.Now()
			} else {
				c.stats.UnknownFlows++
			}
		}
	} else {
		classification = &Classification{
			Protocol:   flow.Protocol,
			Category:   flow.Category,
			Confidence: flow.Confidence,
		}
	}

	c.mu.Unlock()

	return classification, flow
}

// GetFlow retrieves a flow
func (c *Classifier) GetFlow(srcIP, dstIP net.IP, srcPort, dstPort uint16, proto uint8) (*Flow, bool) {
	flowKey := makeFlowKey(srcIP, dstIP, srcPort, dstPort, proto)

	c.mu.RLock()
	defer c.mu.RUnlock()

	flow, exists := c.flows[flowKey]
	return flow, exists
}

// GetFlowsByProtocol returns all flows for a protocol
func (c *Classifier) GetFlowsByProtocol(protocol Protocol) []*Flow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	flows := make([]*Flow, 0)
	for _, flow := range c.flows {
		if flow.Protocol == protocol {
			flows = append(flows, flow)
		}
	}
	return flows
}

// GetFlowsByCategory returns all flows for a category
func (c *Classifier) GetFlowsByCategory(category Category) []*Flow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	flows := make([]*Flow, 0)
	for _, flow := range c.flows {
		if flow.Category == category {
			flows = append(flows, flow)
		}
	}
	return flows
}

// AddPolicy adds an application routing policy
func (c *Classifier) AddPolicy(policy *ApplicationPolicy) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.policies[policy.Name]; exists {
		return fmt.Errorf("policy %s already exists", policy.Name)
	}

	c.policies[policy.Name] = policy
	return nil
}

// GetPolicy retrieves a policy by name
func (c *Classifier) GetPolicy(name string) (*ApplicationPolicy, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	policy, exists := c.policies[name]
	return policy, exists
}

// GetPolicyForProtocol finds the matching policy for a protocol
func (c *Classifier) GetPolicyForProtocol(protocol Protocol) *ApplicationPolicy {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Try exact protocol match
	for _, policy := range c.policies {
		if policy.Enabled && policy.Protocol == protocol {
			return policy
		}
	}

	// Try category match
	category := protocol.GetCategory()
	for _, policy := range c.policies {
		if policy.Enabled && policy.Category == category {
			return policy
		}
	}

	return nil
}

// GetStats returns DPI statistics
func (c *Classifier) GetStats() *DPIStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy
	stats := *c.stats
	stats.ProtocolStats = make(map[Protocol]uint64)
	stats.CategoryStats = make(map[Category]uint64)

	for k, v := range c.stats.ProtocolStats {
		stats.ProtocolStats[k] = v
	}
	for k, v := range c.stats.CategoryStats {
		stats.CategoryStats[k] = v
	}

	return &stats
}

// GetAllFlows returns all active flows
func (c *Classifier) GetAllFlows() []*Flow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	flows := make([]*Flow, 0, len(c.flows))
	for _, flow := range c.flows {
		flows = append(flows, flow)
	}
	return flows
}

// GetTopProtocols returns top N protocols by byte count
func (c *Classifier) GetTopProtocols(n int) []Protocol {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Count bytes per protocol
	protocolBytes := make(map[Protocol]uint64)
	for _, flow := range c.flows {
		if flow.Protocol != ProtocolUnknown {
			protocolBytes[flow.Protocol] += flow.Bytes
		}
	}

	// Sort by bytes (simple bubble sort for small n)
	protocols := make([]Protocol, 0, len(protocolBytes))
	for proto := range protocolBytes {
		protocols = append(protocols, proto)
	}

	for i := 0; i < len(protocols); i++ {
		for j := i + 1; j < len(protocols); j++ {
			if protocolBytes[protocols[j]] > protocolBytes[protocols[i]] {
				protocols[i], protocols[j] = protocols[j], protocols[i]
			}
		}
	}

	if len(protocols) > n {
		protocols = protocols[:n]
	}

	return protocols
}

// cleanupRoutine periodically removes expired flows
func (c *Classifier) cleanupRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.cleanupExpiredFlows()
		}
	}
}

// cleanupExpiredFlows removes flows that have exceeded timeout
func (c *Classifier) cleanupExpiredFlows() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	toDelete := make([]string, 0)

	for key, flow := range c.flows {
		if now.Sub(flow.LastSeen) > c.config.FlowTimeout {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(c.flows, key)
		c.stats.ActiveFlows--
	}
}

// makeFlowKey creates a unique key for a flow
func makeFlowKey(srcIP, dstIP net.IP, srcPort, dstPort uint16, proto uint8) string {
	return fmt.Sprintf("%s:%d->%s:%d/%d", srcIP, srcPort, dstIP, dstPort, proto)
}

// GetActiveFlowCount returns the number of active flows
func (c *Classifier) GetActiveFlowCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.flows)
}

// GetActiveFlows returns all currently active flows
func (c *Classifier) GetActiveFlows() []*Flow {
	c.mu.RLock()
	defer c.mu.RUnlock()

	flows := make([]*Flow, 0, len(c.flows))
	for _, flow := range c.flows {
		// Create a copy of the flow to avoid race conditions
		flowCopy := *flow
		flows = append(flows, &flowCopy)
	}

	return flows
}

// GetProtocolStats returns statistics for a specific protocol
func (c *Classifier) GetProtocolStats(protocol Protocol) (flows, bytes uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, flow := range c.flows {
		if flow.Protocol == protocol {
			flows++
			bytes += flow.Bytes
		}
	}

	return flows, bytes
}

// GetCategoryStats returns statistics for a specific category
func (c *Classifier) GetCategoryStats(category Category) (flows, bytes uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	for _, flow := range c.flows {
		if flow.Category == category {
			flows++
			bytes += flow.Bytes
		}
	}

	return flows, bytes
}

// Reset resets all statistics and flows
func (c *Classifier) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.flows = make(map[string]*Flow)
	c.stats = &DPIStats{
		ProtocolStats: make(map[Protocol]uint64),
		CategoryStats: make(map[Category]uint64),
	}
}

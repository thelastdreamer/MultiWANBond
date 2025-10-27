package health

import (
	"context"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

const (
	// Default health check interval (can detect failure in < 1s)
	DefaultCheckInterval = 200 * time.Millisecond
	// Number of consecutive failures before marking down
	DefaultFailureThreshold = 3
	// Probe timeout
	DefaultProbeTimeout = 150 * time.Millisecond
	// Sample size for calculating moving averages
	SampleSize = 10
)

// Checker implements the HealthChecker interface
type Checker struct {
	mu              sync.RWMutex
	wans            map[uint8]*protocol.WANInterface
	metrics         map[uint8]*protocol.WANMetrics
	samples         map[uint8]*LatencySamples
	eventChan       chan protocol.HealthEvent
	failureCount    map[uint8]int
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	checkInterval   time.Duration
	failureThreshold int
}

// LatencySamples stores recent latency samples for averaging
type LatencySamples struct {
	Latencies  []time.Duration
	Jitters    []time.Duration
	PacketLoss []float64
	Index      int
	Count      int
}

// NewChecker creates a new health checker
func NewChecker() *Checker {
	return &Checker{
		wans:            make(map[uint8]*protocol.WANInterface),
		metrics:         make(map[uint8]*protocol.WANMetrics),
		samples:         make(map[uint8]*LatencySamples),
		eventChan:       make(chan protocol.HealthEvent, 100),
		failureCount:    make(map[uint8]int),
		checkInterval:   DefaultCheckInterval,
		failureThreshold: DefaultFailureThreshold,
	}
}

// Start begins health monitoring
func (c *Checker) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		return fmt.Errorf("health checker already running")
	}

	c.ctx, c.cancel = context.WithCancel(ctx)

	// Start monitoring goroutine for each WAN
	for _, wan := range c.wans {
		c.wg.Add(1)
		go c.monitorWAN(wan)
	}

	return nil
}

// Stop stops health monitoring
func (c *Checker) Stop() error {
	c.mu.Lock()
	if c.cancel == nil {
		c.mu.Unlock()
		return fmt.Errorf("health checker not running")
	}
	c.cancel()
	c.mu.Unlock()

	c.wg.Wait()

	c.mu.Lock()
	c.cancel = nil
	c.mu.Unlock()

	return nil
}

// AddWAN adds a WAN interface to monitor
func (c *Checker) AddWAN(wan *protocol.WANInterface) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.wans[wan.ID] = wan
	c.metrics[wan.ID] = &protocol.WANMetrics{
		LastUpdate: time.Now(),
	}
	c.samples[wan.ID] = &LatencySamples{
		Latencies:  make([]time.Duration, SampleSize),
		Jitters:    make([]time.Duration, SampleSize),
		PacketLoss: make([]float64, SampleSize),
	}
	c.failureCount[wan.ID] = 0

	// If already running, start monitoring this WAN
	if c.cancel != nil {
		c.wg.Add(1)
		go c.monitorWAN(wan)
	}

	return nil
}

// RemoveWAN removes a WAN interface from monitoring
func (c *Checker) RemoveWAN(wanID uint8) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.wans, wanID)
	delete(c.metrics, wanID)
	delete(c.samples, wanID)
	delete(c.failureCount, wanID)

	return nil
}

// monitorWAN continuously monitors a WAN interface
func (c *Checker) monitorWAN(wan *protocol.WANInterface) {
	defer c.wg.Done()

	interval := c.checkInterval
	if wan.Config.HealthCheckInterval > 0 {
		interval = wan.Config.HealthCheckInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			metrics, err := c.CheckWAN(wan)
			if err != nil {
				c.handleCheckFailure(wan)
			} else {
				c.handleCheckSuccess(wan, metrics)
			}
		}
	}
}

// CheckWAN performs a health check on a specific WAN
func (c *Checker) CheckWAN(wan *protocol.WANInterface) (*protocol.WANMetrics, error) {
	if wan.RemoteAddr == nil {
		return nil, fmt.Errorf("no remote address configured")
	}

	// Send probe with timeout
	sendTime := time.Now()

	// Encode packet (simplified for now)
	// TODO: Use proper packet encoding with full protocol.Packet structure
	probeData := []byte{byte(protocol.PacketTypeHeartbeat)}

	// Set deadline for response
	wan.Conn.SetReadDeadline(time.Now().Add(DefaultProbeTimeout))

	// Send probe
	_, err := wan.Conn.WriteToUDP(probeData, wan.RemoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to send probe: %w", err)
	}

	// Wait for response (echo server should respond)
	buf := make([]byte, 1024)
	n, _, err := wan.Conn.ReadFromUDP(buf)
	recvTime := time.Now()

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return nil, fmt.Errorf("probe timeout")
		}
		return nil, fmt.Errorf("failed to receive probe response: %w", err)
	}

	if n == 0 {
		return nil, fmt.Errorf("empty probe response")
	}

	// Calculate metrics
	latency := recvTime.Sub(sendTime)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Get or create metrics
	metrics, exists := c.metrics[wan.ID]
	if !exists {
		metrics = &protocol.WANMetrics{}
		c.metrics[wan.ID] = metrics
	}

	// Calculate jitter (variance from average latency)
	jitter := time.Duration(0)
	if metrics.Latency > 0 {
		diff := latency - metrics.Latency
		if diff < 0 {
			jitter = -diff
		} else {
			jitter = diff
		}
	}

	// Update current metrics
	metrics.Latency = latency
	metrics.Jitter = jitter
	metrics.LastUpdate = time.Now()

	// Update samples for moving average
	samples := c.samples[wan.ID]
	samples.Latencies[samples.Index] = latency
	samples.Jitters[samples.Index] = jitter
	samples.Index = (samples.Index + 1) % SampleSize
	if samples.Count < SampleSize {
		samples.Count++
	}

	// Calculate moving averages
	metrics.AvgLatency = c.calculateAvgDuration(samples.Latencies, samples.Count)
	metrics.AvgJitter = c.calculateAvgDuration(samples.Jitters, samples.Count)

	// Estimate packet loss (based on consecutive failures)
	failCount := c.failureCount[wan.ID]
	totalChecks := samples.Count
	if totalChecks > 0 {
		metrics.PacketLoss = float64(failCount) / float64(totalChecks) * 100.0
	}

	// Update WAN metrics reference
	wan.Metrics = metrics
	wan.LastSeen = time.Now()

	return metrics, nil
}

// handleCheckFailure handles a failed health check
func (c *Checker) handleCheckFailure(wan *protocol.WANInterface) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount[wan.ID]++

	threshold := c.failureThreshold
	if wan.Config.FailureThreshold > 0 {
		threshold = wan.Config.FailureThreshold
	}

	// Check if we should transition to down state
	if c.failureCount[wan.ID] >= threshold {
		oldState := wan.State
		wan.State = protocol.WANStateDown

		if oldState != protocol.WANStateDown {
			// Send health event
			event := protocol.HealthEvent{
				WANID:     wan.ID,
				OldState:  oldState,
				NewState:  protocol.WANStateDown,
				Metrics:   c.metrics[wan.ID],
				Timestamp: time.Now().UnixNano(),
			}

			select {
			case c.eventChan <- event:
			default:
				// Channel full, skip
			}
		}
	} else if wan.State == protocol.WANStateUp {
		// Not down yet, but degraded
		wan.State = protocol.WANStateDegraded
	}
}

// handleCheckSuccess handles a successful health check
func (c *Checker) handleCheckSuccess(wan *protocol.WANInterface, metrics *protocol.WANMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldFailureCount := c.failureCount[wan.ID]
	c.failureCount[wan.ID] = 0

	oldState := wan.State

	// Determine new state based on metrics
	newState := protocol.WANStateUp

	// Check if metrics exceed thresholds
	if wan.Config.MaxLatency > 0 && metrics.AvgLatency > wan.Config.MaxLatency {
		newState = protocol.WANStateDegraded
	}
	if wan.Config.MaxJitter > 0 && metrics.AvgJitter > wan.Config.MaxJitter {
		newState = protocol.WANStateDegraded
	}
	if wan.Config.MaxPacketLoss > 0 && metrics.AvgPacketLoss > wan.Config.MaxPacketLoss {
		newState = protocol.WANStateDegraded
	}

	// If recovering from down state
	if oldState == protocol.WANStateDown && oldFailureCount > 0 {
		newState = protocol.WANStateRecovering
	}

	wan.State = newState

	// Send event if state changed
	if oldState != newState {
		event := protocol.HealthEvent{
			WANID:     wan.ID,
			OldState:  oldState,
			NewState:  newState,
			Metrics:   metrics,
			Timestamp: time.Now().UnixNano(),
		}

		select {
		case c.eventChan <- event:
		default:
			// Channel full, skip
		}
	}
}

// GetMetrics returns current metrics for a WAN
func (c *Checker) GetMetrics(wanID uint8) (*protocol.WANMetrics, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics, exists := c.metrics[wanID]
	if !exists {
		return nil, fmt.Errorf("no metrics for WAN %d", wanID)
	}

	// Return a copy
	metricsCopy := *metrics
	return &metricsCopy, nil
}

// Subscribe returns a channel for health events
func (c *Checker) Subscribe() <-chan protocol.HealthEvent {
	return c.eventChan
}

// calculateAvgDuration calculates average of duration samples
func (c *Checker) calculateAvgDuration(samples []time.Duration, count int) time.Duration {
	if count == 0 {
		return 0
	}

	var sum time.Duration
	for i := 0; i < count; i++ {
		sum += samples[i]
	}

	return sum / time.Duration(count)
}

// calculateStdDev calculates standard deviation of duration samples
func (c *Checker) calculateStdDev(samples []time.Duration, count int, avg time.Duration) time.Duration {
	if count == 0 {
		return 0
	}

	var sumSquares float64
	for i := 0; i < count; i++ {
		diff := float64(samples[i] - avg)
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(count)
	return time.Duration(math.Sqrt(variance))
}

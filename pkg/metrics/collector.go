// Package metrics - Metrics collector
package metrics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Collector collects and manages metrics
type Collector struct {
	config *MetricsConfig

	// Time series storage
	timeSeries map[string]*TimeSeries
	tsMu       sync.RWMutex

	// WAN metrics
	wanMetrics map[uint8]*WANMetrics
	wanMu      sync.RWMutex

	// Flow metrics
	flowMetrics map[string]*FlowMetrics
	flowMu      sync.RWMutex

	// System metrics
	systemMetrics *SystemMetrics
	systemMu      sync.RWMutex

	// Bandwidth quotas
	quotas map[uint8]*BandwidthQuota
	quotaMu sync.RWMutex

	// Alerts
	alerts     []*Alert
	alertsMu   sync.RWMutex
	alertIndex int

	// Collection state
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewCollector creates a new metrics collector
func NewCollector(config *MetricsConfig) *Collector {
	if config == nil {
		config = DefaultMetricsConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Collector{
		config:        config,
		timeSeries:    make(map[string]*TimeSeries),
		wanMetrics:    make(map[uint8]*WANMetrics),
		flowMetrics:   make(map[string]*FlowMetrics),
		systemMetrics: NewSystemMetrics(),
		quotas:        make(map[uint8]*BandwidthQuota),
		alerts:        make([]*Alert, 0),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the metrics collector
func (c *Collector) Start() error {
	// Start collection goroutines
	if c.config.EnableWANMetrics {
		c.wg.Add(1)
		go c.collectWANMetrics()
	}

	if c.config.EnableFlowMetrics {
		c.wg.Add(1)
		go c.collectFlowMetrics()
	}

	if c.config.EnableSystemMetrics {
		c.wg.Add(1)
		go c.collectSystemMetrics()
	}

	// Start pruning goroutine
	c.wg.Add(1)
	go c.pruneOldData()

	// Start alert checking if enabled
	if c.config.EnableAlerts {
		c.wg.Add(1)
		go c.checkAlerts()
	}

	return nil
}

// Stop stops the metrics collector
func (c *Collector) Stop() error {
	c.cancel()
	c.wg.Wait()
	return nil
}

// RecordWANMetric records a metric for a WAN interface
func (c *Collector) RecordWANMetric(wanID uint8, bytesSent, bytesRecv, pktsSent, pktsRecv uint64,
	latency, jitter time.Duration, loss float64) {

	c.wanMu.Lock()
	metrics, exists := c.wanMetrics[wanID]
	if !exists {
		metrics = NewWANMetrics(wanID)
		c.wanMetrics[wanID] = metrics
	}
	c.wanMu.Unlock()

	metrics.Update(bytesSent, bytesRecv, pktsSent, pktsRecv, latency, jitter, loss)

	// Record to time series
	c.recordTimeSeries(fmt.Sprintf("wan_%d_bytes_sent", wanID), MetricTypeCounter,
		map[string]string{"wan_id": fmt.Sprintf("%d", wanID)}, float64(bytesSent))

	c.recordTimeSeries(fmt.Sprintf("wan_%d_bytes_received", wanID), MetricTypeCounter,
		map[string]string{"wan_id": fmt.Sprintf("%d", wanID)}, float64(bytesRecv))

	c.recordTimeSeries(fmt.Sprintf("wan_%d_latency_ms", wanID), MetricTypeGauge,
		map[string]string{"wan_id": fmt.Sprintf("%d", wanID)}, float64(latency.Milliseconds()))

	c.recordTimeSeries(fmt.Sprintf("wan_%d_packet_loss", wanID), MetricTypeGauge,
		map[string]string{"wan_id": fmt.Sprintf("%d", wanID)}, loss)

	// Update bandwidth quota if exists
	c.quotaMu.RLock()
	quota, hasQuota := c.quotas[wanID]
	c.quotaMu.RUnlock()

	if hasQuota {
		totalBytes := bytesSent + bytesRecv
		dailyExceeded, weeklyExceeded, monthlyExceeded := quota.AddUsage(totalBytes)

		// Generate alerts if quotas exceeded
		if dailyExceeded {
			c.AddAlert(NewAlert(
				fmt.Sprintf("quota_daily_wan_%d", wanID),
				"warning",
				"Daily bandwidth quota exceeded",
				fmt.Sprintf("WAN %d has exceeded its daily bandwidth quota", wanID),
				"bandwidth_quota_daily",
				float64(quota.DailyLimit),
				float64(quota.DailyUsage),
			))
		}
		if weeklyExceeded {
			c.AddAlert(NewAlert(
				fmt.Sprintf("quota_weekly_wan_%d", wanID),
				"warning",
				"Weekly bandwidth quota exceeded",
				fmt.Sprintf("WAN %d has exceeded its weekly bandwidth quota", wanID),
				"bandwidth_quota_weekly",
				float64(quota.WeeklyLimit),
				float64(quota.WeeklyUsage),
			))
		}
		if monthlyExceeded {
			c.AddAlert(NewAlert(
				fmt.Sprintf("quota_monthly_wan_%d", wanID),
				"critical",
				"Monthly bandwidth quota exceeded",
				fmt.Sprintf("WAN %d has exceeded its monthly bandwidth quota", wanID),
				"bandwidth_quota_monthly",
				float64(quota.MonthlyLimit),
				float64(quota.MonthlyUsage),
			))
		}
	}
}

// RecordWANBandwidth records bandwidth metrics for a WAN
func (c *Collector) RecordWANBandwidth(wanID uint8, upload, download float64) {
	c.wanMu.RLock()
	metrics, exists := c.wanMetrics[wanID]
	c.wanMu.RUnlock()

	if exists {
		metrics.UpdateBandwidth(upload, download)

		c.recordTimeSeries(fmt.Sprintf("wan_%d_upload_bps", wanID), MetricTypeGauge,
			map[string]string{"wan_id": fmt.Sprintf("%d", wanID)}, upload)

		c.recordTimeSeries(fmt.Sprintf("wan_%d_download_bps", wanID), MetricTypeGauge,
			map[string]string{"wan_id": fmt.Sprintf("%d", wanID)}, download)
	}
}

// RecordFlowMetric records metrics for an application flow
func (c *Collector) RecordFlowMetric(flowID, application, category string, wanID uint8,
	bytesSent, bytesRecv, pktsSent, pktsRecv uint64) {

	c.flowMu.Lock()
	metrics, exists := c.flowMetrics[flowID]
	if !exists {
		metrics = NewFlowMetrics(flowID, application, category, wanID)
		c.flowMetrics[flowID] = metrics
	}
	c.flowMu.Unlock()

	metrics.UpdateTraffic(bytesSent, bytesRecv, pktsSent, pktsRecv)

	// Record application-level time series
	c.recordTimeSeries(fmt.Sprintf("app_%s_bytes", application), MetricTypeCounter,
		map[string]string{"application": application, "category": category}, float64(bytesSent+bytesRecv))
}

// CloseFlow marks a flow as closed
func (c *Collector) CloseFlow(flowID string) {
	c.flowMu.RLock()
	metrics, exists := c.flowMetrics[flowID]
	c.flowMu.RUnlock()

	if exists {
		metrics.Close()

		// Update system metrics
		c.systemMu.Lock()
		c.systemMetrics.TotalFlowsClosed++
		c.systemMetrics.ActiveFlows--
		c.systemMu.Unlock()
	}
}

// RecordSystemMetric records a system-level metric
func (c *Collector) RecordSystemMetric(name string, value float64, labels map[string]string) {
	c.recordTimeSeries(name, MetricTypeGauge, labels, value)
}

// RecordFailover records a failover event
func (c *Collector) RecordFailover(fromWAN, toWAN uint8, reason string) {
	c.systemMu.Lock()
	c.systemMetrics.FailoverCount++
	c.systemMetrics.LastFailover = time.Now()
	c.systemMu.Unlock()

	c.recordTimeSeries("failover_count", MetricTypeCounter,
		map[string]string{
			"from_wan": fmt.Sprintf("%d", fromWAN),
			"to_wan":   fmt.Sprintf("%d", toWAN),
			"reason":   reason,
		}, 1.0)

	// Generate alert
	c.AddAlert(NewAlert(
		fmt.Sprintf("failover_%d_%d_%d", fromWAN, toWAN, time.Now().Unix()),
		"info",
		"WAN Failover",
		fmt.Sprintf("Failed over from WAN %d to WAN %d: %s", fromWAN, toWAN, reason),
		"failover",
		0,
		1,
	))
}

// SetBandwidthQuota sets bandwidth quota for a WAN
func (c *Collector) SetBandwidthQuota(wanID uint8, daily, weekly, monthly uint64) {
	c.quotaMu.Lock()
	defer c.quotaMu.Unlock()

	c.quotas[wanID] = NewBandwidthQuota(wanID, daily, weekly, monthly)
}

// GetBandwidthQuota gets bandwidth quota for a WAN
func (c *Collector) GetBandwidthQuota(wanID uint8) (*BandwidthQuota, bool) {
	c.quotaMu.RLock()
	defer c.quotaMu.RUnlock()

	quota, exists := c.quotas[wanID]
	return quota, exists
}

// GetWANMetrics returns metrics for a specific WAN
func (c *Collector) GetWANMetrics(wanID uint8) (*WANMetrics, bool) {
	c.wanMu.RLock()
	defer c.wanMu.RUnlock()

	metrics, exists := c.wanMetrics[wanID]
	return metrics, exists
}

// GetAllWANMetrics returns metrics for all WANs
func (c *Collector) GetAllWANMetrics() map[uint8]*WANMetrics {
	c.wanMu.RLock()
	defer c.wanMu.RUnlock()

	result := make(map[uint8]*WANMetrics, len(c.wanMetrics))
	for id, metrics := range c.wanMetrics {
		result[id] = metrics
	}
	return result
}

// GetFlowMetrics returns metrics for a specific flow
func (c *Collector) GetFlowMetrics(flowID string) (*FlowMetrics, bool) {
	c.flowMu.RLock()
	defer c.flowMu.RUnlock()

	metrics, exists := c.flowMetrics[flowID]
	return metrics, exists
}

// GetSystemMetrics returns system-wide metrics
func (c *Collector) GetSystemMetrics() *SystemMetrics {
	c.systemMu.RLock()
	defer c.systemMu.RUnlock()

	return c.systemMetrics
}

// GetTimeSeries returns a time series by name
func (c *Collector) GetTimeSeries(name string) (*TimeSeries, bool) {
	c.tsMu.RLock()
	defer c.tsMu.RUnlock()

	ts, exists := c.timeSeries[name]
	return ts, exists
}

// GetAllTimeSeries returns all time series
func (c *Collector) GetAllTimeSeries() map[string]*TimeSeries {
	c.tsMu.RLock()
	defer c.tsMu.RUnlock()

	result := make(map[string]*TimeSeries, len(c.timeSeries))
	for name, ts := range c.timeSeries {
		result[name] = ts
	}
	return result
}

// AddAlert adds a new alert
func (c *Collector) AddAlert(alert *Alert) {
	c.alertsMu.Lock()
	defer c.alertsMu.Unlock()

	c.alerts = append(c.alerts, alert)
}

// GetAlerts returns all alerts
func (c *Collector) GetAlerts() []*Alert {
	c.alertsMu.RLock()
	defer c.alertsMu.RUnlock()

	result := make([]*Alert, len(c.alerts))
	copy(result, c.alerts)
	return result
}

// GetUnresolvedAlerts returns unresolved alerts
func (c *Collector) GetUnresolvedAlerts() []*Alert {
	c.alertsMu.RLock()
	defer c.alertsMu.RUnlock()

	result := make([]*Alert, 0)
	for _, alert := range c.alerts {
		if !alert.Resolved {
			result = append(result, alert)
		}
	}
	return result
}

// ResolveAlert marks an alert as resolved
func (c *Collector) ResolveAlert(alertID string) bool {
	c.alertsMu.Lock()
	defer c.alertsMu.Unlock()

	for _, alert := range c.alerts {
		if alert.ID == alertID && !alert.Resolved {
			alert.Resolve()
			return true
		}
	}
	return false
}

// recordTimeSeries records a data point to a time series
func (c *Collector) recordTimeSeries(name string, metricType MetricType, labels map[string]string, value float64) {
	c.tsMu.Lock()
	ts, exists := c.timeSeries[name]
	if !exists {
		ts = NewTimeSeries(name, metricType, labels)
		c.timeSeries[name] = ts
	}
	c.tsMu.Unlock()

	ts.AddPoint(time.Now(), value)

	// Limit data points
	if len(ts.DataPoints) > c.config.MaxDataPoints {
		ts.Prune(c.config.RetentionPeriod)
	}
}

// collectWANMetrics periodically collects WAN metrics
func (c *Collector) collectWANMetrics() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Update WAN metrics from actual interfaces
			c.wanMu.RLock()
			wanCount := len(c.wanMetrics)
			activeCount := 0
			for _, metrics := range c.wanMetrics {
				if time.Since(metrics.LastUpdate) < 30*time.Second {
					activeCount++
				}
			}
			c.wanMu.RUnlock()

			// Update system metrics
			c.systemMu.Lock()
			c.systemMetrics.TotalWANs = wanCount
			c.systemMetrics.ActiveWANs = activeCount
			c.systemMu.Unlock()
		}
	}
}

// collectFlowMetrics periodically collects flow metrics
func (c *Collector) collectFlowMetrics() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Count active flows
			c.flowMu.RLock()
			activeCount := 0
			for _, flow := range c.flowMetrics {
				if flow.Active {
					activeCount++
				}
			}
			c.flowMu.RUnlock()

			// Update system metrics
			c.systemMu.Lock()
			c.systemMetrics.ActiveFlows = activeCount
			c.systemMu.Unlock()
		}
	}
}

// collectSystemMetrics periodically collects system metrics
func (c *Collector) collectSystemMetrics() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Update uptime
			c.systemMu.Lock()
			c.systemMetrics.UpdateUptime()

			// Get memory stats
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			c.systemMetrics.AllocatedMemory = memStats.Alloc
			c.systemMetrics.UsedMemory = memStats.Sys
			c.systemMu.Unlock()

			// Record system-level time series
			c.recordTimeSeries("system_uptime_seconds", MetricTypeGauge, nil,
				c.systemMetrics.Uptime.Seconds())

			c.recordTimeSeries("system_memory_allocated", MetricTypeGauge, nil,
				float64(memStats.Alloc))

			c.recordTimeSeries("system_goroutines", MetricTypeGauge, nil,
				float64(runtime.NumGoroutine()))
		}
	}
}

// pruneOldData periodically prunes old data from time series
func (c *Collector) pruneOldData() {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.tsMu.RLock()
			for _, ts := range c.timeSeries {
				ts.Prune(c.config.RetentionPeriod)
			}
			c.tsMu.RUnlock()

			// Prune old resolved alerts (keep for 24 hours)
			c.alertsMu.Lock()
			newAlerts := make([]*Alert, 0)
			cutoff := time.Now().Add(-24 * time.Hour)
			for _, alert := range c.alerts {
				if !alert.Resolved || alert.ResolvedAt.After(cutoff) {
					newAlerts = append(newAlerts, alert)
				}
			}
			c.alerts = newAlerts
			c.alertsMu.Unlock()
		}
	}
}

// checkAlerts periodically checks for alert conditions
func (c *Collector) checkAlerts() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// Check WAN health alerts
			c.wanMu.RLock()
			for wanID, metrics := range c.wanMetrics {
				metrics.mu.RLock()
				latency := metrics.Latency
				loss := metrics.PacketLoss
				metrics.mu.RUnlock()

				// High latency alert (> 200ms)
				if latency > 200*time.Millisecond {
					alertID := fmt.Sprintf("high_latency_wan_%d", wanID)
					if !c.alertExists(alertID) {
						c.AddAlert(NewAlert(
							alertID,
							"warning",
							"High Latency",
							fmt.Sprintf("WAN %d latency is %v", wanID, latency),
							"wan_latency",
							200,
							float64(latency.Milliseconds()),
						))
					}
				}

				// High packet loss alert (> 5%)
				if loss > 5.0 {
					alertID := fmt.Sprintf("high_loss_wan_%d", wanID)
					if !c.alertExists(alertID) {
						c.AddAlert(NewAlert(
							alertID,
							"warning",
							"High Packet Loss",
							fmt.Sprintf("WAN %d packet loss is %.2f%%", wanID, loss),
							"wan_packet_loss",
							5.0,
							loss,
						))
					}
				}
			}
			c.wanMu.RUnlock()
		}
	}
}

// alertExists checks if an unresolved alert with the given ID exists
func (c *Collector) alertExists(alertID string) bool {
	c.alertsMu.RLock()
	defer c.alertsMu.RUnlock()

	for _, alert := range c.alerts {
		if alert.ID == alertID && !alert.Resolved {
			return true
		}
	}
	return false
}

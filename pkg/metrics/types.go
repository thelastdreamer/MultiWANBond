// Package metrics provides time-series metrics collection and storage
package metrics

import (
	"sync"
	"time"
)

// MetricType defines the type of metric
type MetricType int

const (
	// MetricTypeCounter is a monotonically increasing counter
	MetricTypeCounter MetricType = iota
	// MetricTypeGauge is a value that can go up or down
	MetricTypeGauge
	// MetricTypeHistogram tracks distribution of values
	MetricTypeHistogram
	// MetricTypeSummary tracks summary statistics
	MetricTypeSummary
)

// String returns the string representation of the metric type
func (mt MetricType) String() string {
	switch mt {
	case MetricTypeCounter:
		return "counter"
	case MetricTypeGauge:
		return "gauge"
	case MetricTypeHistogram:
		return "histogram"
	case MetricTypeSummary:
		return "summary"
	default:
		return "unknown"
	}
}

// AggregationWindow defines time windows for data aggregation
type AggregationWindow int

const (
	// Window1Minute aggregates data over 1 minute
	Window1Minute AggregationWindow = iota
	// Window5Minutes aggregates data over 5 minutes
	Window5Minutes
	// Window15Minutes aggregates data over 15 minutes
	Window15Minutes
	// Window1Hour aggregates data over 1 hour
	Window1Hour
	// Window6Hours aggregates data over 6 hours
	Window6Hours
	// Window1Day aggregates data over 1 day
	Window1Day
	// Window1Week aggregates data over 1 week
	Window1Week
)

// Duration returns the duration of the aggregation window
func (w AggregationWindow) Duration() time.Duration {
	switch w {
	case Window1Minute:
		return 1 * time.Minute
	case Window5Minutes:
		return 5 * time.Minute
	case Window15Minutes:
		return 15 * time.Minute
	case Window1Hour:
		return 1 * time.Hour
	case Window6Hours:
		return 6 * time.Hour
	case Window1Day:
		return 24 * time.Hour
	case Window1Week:
		return 7 * 24 * time.Hour
	default:
		return 1 * time.Minute
	}
}

// String returns the string representation of the window
func (w AggregationWindow) String() string {
	switch w {
	case Window1Minute:
		return "1m"
	case Window5Minutes:
		return "5m"
	case Window15Minutes:
		return "15m"
	case Window1Hour:
		return "1h"
	case Window6Hours:
		return "6h"
	case Window1Day:
		return "1d"
	case Window1Week:
		return "1w"
	default:
		return "unknown"
	}
}

// DataPoint represents a single metric data point
type DataPoint struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// TimeSeries represents a series of data points over time
type TimeSeries struct {
	Name       string
	Type       MetricType
	Labels     map[string]string
	DataPoints []*DataPoint
	mu         sync.RWMutex
}

// NewTimeSeries creates a new time series
func NewTimeSeries(name string, metricType MetricType, labels map[string]string) *TimeSeries {
	if labels == nil {
		labels = make(map[string]string)
	}
	return &TimeSeries{
		Name:       name,
		Type:       metricType,
		Labels:     labels,
		DataPoints: make([]*DataPoint, 0),
	}
}

// AddPoint adds a data point to the time series
func (ts *TimeSeries) AddPoint(timestamp time.Time, value float64) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.DataPoints = append(ts.DataPoints, &DataPoint{
		Timestamp: timestamp,
		Value:     value,
		Labels:    ts.Labels,
	})
}

// GetPoints returns all data points within a time range
func (ts *TimeSeries) GetPoints(start, end time.Time) []*DataPoint {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	result := make([]*DataPoint, 0)
	for _, dp := range ts.DataPoints {
		if dp.Timestamp.After(start) && dp.Timestamp.Before(end) {
			result = append(result, dp)
		}
	}
	return result
}

// Prune removes data points older than the specified duration
func (ts *TimeSeries) Prune(maxAge time.Duration) int {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	newPoints := make([]*DataPoint, 0)
	pruned := 0

	for _, dp := range ts.DataPoints {
		if dp.Timestamp.After(cutoff) {
			newPoints = append(newPoints, dp)
		} else {
			pruned++
		}
	}

	ts.DataPoints = newPoints
	return pruned
}

// Latest returns the most recent data point
func (ts *TimeSeries) Latest() *DataPoint {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if len(ts.DataPoints) == 0 {
		return nil
	}
	return ts.DataPoints[len(ts.DataPoints)-1]
}

// AggregatedData represents aggregated metrics over a time window
type AggregatedData struct {
	Start     time.Time
	End       time.Time
	Window    AggregationWindow
	Count     int
	Sum       float64
	Min       float64
	Max       float64
	Avg       float64
	Median    float64
	P95       float64
	P99       float64
	StdDev    float64
}

// WANMetrics represents metrics for a single WAN interface
type WANMetrics struct {
	WANID uint8

	// Traffic counters
	BytesSent     uint64
	BytesReceived uint64
	PacketsSent   uint64
	PacketsReceived uint64

	// Error counters
	ErrorsSent     uint64
	ErrorsReceived uint64
	DroppedSent    uint64
	DroppedReceived uint64

	// Performance metrics
	Latency       time.Duration
	Jitter        time.Duration
	PacketLoss    float64

	// Bandwidth metrics
	CurrentUpload   float64 // bytes per second
	CurrentDownload float64 // bytes per second
	PeakUpload      float64
	PeakDownload    float64

	// Availability
	Uptime   time.Duration
	Downtime time.Duration

	LastUpdate time.Time
	mu         sync.RWMutex
}

// NewWANMetrics creates a new WAN metrics instance
func NewWANMetrics(wanID uint8) *WANMetrics {
	return &WANMetrics{
		WANID:      wanID,
		LastUpdate: time.Now(),
	}
}

// Update updates the WAN metrics
func (wm *WANMetrics) Update(bytesSent, bytesRecv, pktsSent, pktsRecv uint64, latency, jitter time.Duration, loss float64) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.BytesSent = bytesSent
	wm.BytesReceived = bytesRecv
	wm.PacketsSent = pktsSent
	wm.PacketsReceived = pktsRecv
	wm.Latency = latency
	wm.Jitter = jitter
	wm.PacketLoss = loss
	wm.LastUpdate = time.Now()
}

// UpdateBandwidth updates bandwidth metrics
func (wm *WANMetrics) UpdateBandwidth(upload, download float64) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.CurrentUpload = upload
	wm.CurrentDownload = download

	if upload > wm.PeakUpload {
		wm.PeakUpload = upload
	}
	if download > wm.PeakDownload {
		wm.PeakDownload = download
	}
}

// FlowMetrics represents metrics for application flows
type FlowMetrics struct {
	FlowID      string
	Application string
	Category    string
	WANID       uint8

	BytesSent     uint64
	BytesReceived uint64
	PacketsSent   uint64
	PacketsReceived uint64

	Duration   time.Duration
	StartTime  time.Time
	EndTime    time.Time
	Active     bool

	mu sync.RWMutex
}

// NewFlowMetrics creates a new flow metrics instance
func NewFlowMetrics(flowID, application, category string, wanID uint8) *FlowMetrics {
	return &FlowMetrics{
		FlowID:      flowID,
		Application: application,
		Category:    category,
		WANID:       wanID,
		StartTime:   time.Now(),
		Active:      true,
	}
}

// UpdateTraffic updates flow traffic counters
func (fm *FlowMetrics) UpdateTraffic(bytesSent, bytesRecv, pktsSent, pktsRecv uint64) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.BytesSent = bytesSent
	fm.BytesReceived = bytesRecv
	fm.PacketsSent = pktsSent
	fm.PacketsReceived = pktsRecv
	fm.Duration = time.Since(fm.StartTime)
}

// Close marks the flow as closed
func (fm *FlowMetrics) Close() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.Active = false
	fm.EndTime = time.Now()
	fm.Duration = fm.EndTime.Sub(fm.StartTime)
}

// SystemMetrics represents overall system metrics
type SystemMetrics struct {
	// System information
	Uptime        time.Duration
	StartTime     time.Time

	// Overall traffic
	TotalBytesSent     uint64
	TotalBytesReceived uint64
	TotalPacketsSent   uint64
	TotalPacketsReceived uint64

	// Performance
	CurrentPPS        uint64
	PeakPPS           uint64
	CurrentBandwidth  float64
	PeakBandwidth     float64

	// WANs
	ActiveWANs        int
	TotalWANs         int

	// Flows
	ActiveFlows       int
	TotalFlowsCreated uint64
	TotalFlowsClosed  uint64

	// Failovers
	FailoverCount     uint64
	LastFailover      time.Time

	// Memory usage
	AllocatedMemory   uint64
	UsedMemory        uint64

	LastUpdate time.Time
	mu         sync.RWMutex
}

// NewSystemMetrics creates a new system metrics instance
func NewSystemMetrics() *SystemMetrics {
	now := time.Now()
	return &SystemMetrics{
		StartTime:  now,
		LastUpdate: now,
	}
}

// UpdateUptime updates the system uptime
func (sm *SystemMetrics) UpdateUptime() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.Uptime = time.Since(sm.StartTime)
	sm.LastUpdate = time.Now()
}

// UpdateTraffic updates overall traffic counters
func (sm *SystemMetrics) UpdateTraffic(bytesSent, bytesRecv, pktsSent, pktsRecv uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.TotalBytesSent = bytesSent
	sm.TotalBytesReceived = bytesRecv
	sm.TotalPacketsSent = pktsSent
	sm.TotalPacketsReceived = pktsRecv
	sm.LastUpdate = time.Now()
}

// UpdatePPS updates packets per second metrics
func (sm *SystemMetrics) UpdatePPS(currentPPS uint64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.CurrentPPS = currentPPS
	if currentPPS > sm.PeakPPS {
		sm.PeakPPS = currentPPS
	}
}

// BandwidthQuota represents bandwidth quota and accounting
type BandwidthQuota struct {
	WANID       uint8

	// Quota limits (bytes)
	DailyLimit   uint64
	WeeklyLimit  uint64
	MonthlyLimit uint64

	// Current usage (bytes)
	DailyUsage   uint64
	WeeklyUsage  uint64
	MonthlyUsage uint64

	// Reset times
	DailyReset   time.Time
	WeeklyReset  time.Time
	MonthlyReset time.Time

	// Alert thresholds (percentage)
	AlertThreshold float64

	mu sync.RWMutex
}

// NewBandwidthQuota creates a new bandwidth quota
func NewBandwidthQuota(wanID uint8, daily, weekly, monthly uint64) *BandwidthQuota {
	now := time.Now()
	return &BandwidthQuota{
		WANID:          wanID,
		DailyLimit:     daily,
		WeeklyLimit:    weekly,
		MonthlyLimit:   monthly,
		DailyReset:     now.Add(24 * time.Hour),
		WeeklyReset:    now.Add(7 * 24 * time.Hour),
		MonthlyReset:   now.Add(30 * 24 * time.Hour),
		AlertThreshold: 0.8, // Alert at 80%
	}
}

// AddUsage adds bandwidth usage and checks quotas
func (bq *BandwidthQuota) AddUsage(bytes uint64) (dailyExceeded, weeklyExceeded, monthlyExceeded bool) {
	bq.mu.Lock()
	defer bq.mu.Unlock()

	now := time.Now()

	// Check if we need to reset counters
	if now.After(bq.DailyReset) {
		bq.DailyUsage = 0
		bq.DailyReset = now.Add(24 * time.Hour)
	}
	if now.After(bq.WeeklyReset) {
		bq.WeeklyUsage = 0
		bq.WeeklyReset = now.Add(7 * 24 * time.Hour)
	}
	if now.After(bq.MonthlyReset) {
		bq.MonthlyUsage = 0
		bq.MonthlyReset = now.Add(30 * 24 * time.Hour)
	}

	// Add usage
	bq.DailyUsage += bytes
	bq.WeeklyUsage += bytes
	bq.MonthlyUsage += bytes

	// Check if limits exceeded
	dailyExceeded = bq.DailyLimit > 0 && bq.DailyUsage > bq.DailyLimit
	weeklyExceeded = bq.WeeklyLimit > 0 && bq.WeeklyUsage > bq.WeeklyLimit
	monthlyExceeded = bq.MonthlyLimit > 0 && bq.MonthlyUsage > bq.MonthlyLimit

	return dailyExceeded, weeklyExceeded, monthlyExceeded
}

// GetUsagePercent returns the usage percentage for each quota period
func (bq *BandwidthQuota) GetUsagePercent() (daily, weekly, monthly float64) {
	bq.mu.RLock()
	defer bq.mu.RUnlock()

	if bq.DailyLimit > 0 {
		daily = float64(bq.DailyUsage) / float64(bq.DailyLimit) * 100.0
	}
	if bq.WeeklyLimit > 0 {
		weekly = float64(bq.WeeklyUsage) / float64(bq.WeeklyLimit) * 100.0
	}
	if bq.MonthlyLimit > 0 {
		monthly = float64(bq.MonthlyUsage) / float64(bq.MonthlyLimit) * 100.0
	}

	return daily, weekly, monthly
}

// MetricsConfig contains configuration for metrics collection
type MetricsConfig struct {
	// Collection intervals
	CollectionInterval time.Duration

	// Data retention
	RetentionPeriod time.Duration

	// Aggregation windows to maintain
	AggregationWindows []AggregationWindow

	// Maximum data points per time series
	MaxDataPoints int

	// Enable specific metric types
	EnableWANMetrics    bool
	EnableFlowMetrics   bool
	EnableSystemMetrics bool

	// Export settings
	PrometheusEnabled bool
	PrometheusPort    int

	// Alert settings
	EnableAlerts      bool
	AlertCheckInterval time.Duration
}

// DefaultMetricsConfig returns default metrics configuration
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		CollectionInterval: 10 * time.Second,
		RetentionPeriod:    7 * 24 * time.Hour, // 7 days
		AggregationWindows: []AggregationWindow{
			Window1Minute,
			Window5Minutes,
			Window1Hour,
			Window1Day,
		},
		MaxDataPoints:       10000,
		EnableWANMetrics:    true,
		EnableFlowMetrics:   true,
		EnableSystemMetrics: true,
		PrometheusEnabled:   true,
		PrometheusPort:      9090,
		EnableAlerts:        true,
		AlertCheckInterval:  30 * time.Second,
	}
}

// Alert represents a metrics-based alert
type Alert struct {
	ID          string
	Severity    string // "info", "warning", "critical"
	Title       string
	Description string
	Metric      string
	Threshold   float64
	CurrentValue float64
	Timestamp   time.Time
	Resolved    bool
	ResolvedAt  time.Time
}

// NewAlert creates a new alert
func NewAlert(id, severity, title, description, metric string, threshold, currentValue float64) *Alert {
	return &Alert{
		ID:           id,
		Severity:     severity,
		Title:        title,
		Description:  description,
		Metric:       metric,
		Threshold:    threshold,
		CurrentValue: currentValue,
		Timestamp:    time.Now(),
		Resolved:     false,
	}
}

// Resolve marks the alert as resolved
func (a *Alert) Resolve() {
	a.Resolved = true
	a.ResolvedAt = time.Now()
}

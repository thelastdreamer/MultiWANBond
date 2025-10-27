// Package metrics - Metrics exporter
package metrics

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Exporter exports metrics in various formats
type Exporter struct {
	collector *Collector
}

// NewExporter creates a new metrics exporter
func NewExporter(collector *Collector) *Exporter {
	return &Exporter{
		collector: collector,
	}
}

// ExportPrometheus exports metrics in Prometheus text format
func (e *Exporter) ExportPrometheus() string {
	var sb strings.Builder

	// Header
	sb.WriteString("# MultiWANBond Metrics\n")
	sb.WriteString(fmt.Sprintf("# Generated at %s\n\n", time.Now().Format(time.RFC3339)))

	// System metrics
	systemMetrics := e.collector.GetSystemMetrics()
	if systemMetrics != nil {
		sb.WriteString("# HELP multiwanbond_uptime_seconds System uptime in seconds\n")
		sb.WriteString("# TYPE multiwanbond_uptime_seconds gauge\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_uptime_seconds %.0f\n", systemMetrics.Uptime.Seconds()))

		sb.WriteString("# HELP multiwanbond_total_bytes_sent Total bytes sent across all WANs\n")
		sb.WriteString("# TYPE multiwanbond_total_bytes_sent counter\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_total_bytes_sent %d\n", systemMetrics.TotalBytesSent))

		sb.WriteString("# HELP multiwanbond_total_bytes_received Total bytes received across all WANs\n")
		sb.WriteString("# TYPE multiwanbond_total_bytes_received counter\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_total_bytes_received %d\n", systemMetrics.TotalBytesReceived))

		sb.WriteString("# HELP multiwanbond_active_wans Number of active WAN interfaces\n")
		sb.WriteString("# TYPE multiwanbond_active_wans gauge\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_active_wans %d\n", systemMetrics.ActiveWANs))

		sb.WriteString("# HELP multiwanbond_active_flows Number of active flows\n")
		sb.WriteString("# TYPE multiwanbond_active_flows gauge\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_active_flows %d\n", systemMetrics.ActiveFlows))

		sb.WriteString("# HELP multiwanbond_failover_count Total number of failover events\n")
		sb.WriteString("# TYPE multiwanbond_failover_count counter\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_failover_count %d\n", systemMetrics.FailoverCount))

		sb.WriteString("# HELP multiwanbond_current_pps Current packets per second\n")
		sb.WriteString("# TYPE multiwanbond_current_pps gauge\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_current_pps %d\n", systemMetrics.CurrentPPS))

		sb.WriteString("# HELP multiwanbond_memory_allocated Allocated memory in bytes\n")
		sb.WriteString("# TYPE multiwanbond_memory_allocated gauge\n")
		sb.WriteString(fmt.Sprintf("multiwanbond_memory_allocated %d\n", systemMetrics.AllocatedMemory))

		sb.WriteString("\n")
	}

	// WAN metrics
	wanMetrics := e.collector.GetAllWANMetrics()
	if len(wanMetrics) > 0 {
		sb.WriteString("# HELP multiwanbond_wan_bytes_sent Bytes sent on WAN interface\n")
		sb.WriteString("# TYPE multiwanbond_wan_bytes_sent counter\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_bytes_sent{wan_id=\"%d\"} %d\n", wanID, metrics.BytesSent))
		}

		sb.WriteString("# HELP multiwanbond_wan_bytes_received Bytes received on WAN interface\n")
		sb.WriteString("# TYPE multiwanbond_wan_bytes_received counter\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_bytes_received{wan_id=\"%d\"} %d\n", wanID, metrics.BytesReceived))
		}

		sb.WriteString("# HELP multiwanbond_wan_latency_milliseconds WAN interface latency\n")
		sb.WriteString("# TYPE multiwanbond_wan_latency_milliseconds gauge\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_latency_milliseconds{wan_id=\"%d\"} %.2f\n", wanID, float64(metrics.Latency.Milliseconds())))
		}

		sb.WriteString("# HELP multiwanbond_wan_jitter_milliseconds WAN interface jitter\n")
		sb.WriteString("# TYPE multiwanbond_wan_jitter_milliseconds gauge\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_jitter_milliseconds{wan_id=\"%d\"} %.2f\n", wanID, float64(metrics.Jitter.Milliseconds())))
		}

		sb.WriteString("# HELP multiwanbond_wan_packet_loss_percent WAN interface packet loss percentage\n")
		sb.WriteString("# TYPE multiwanbond_wan_packet_loss_percent gauge\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_packet_loss_percent{wan_id=\"%d\"} %.2f\n", wanID, metrics.PacketLoss))
		}

		sb.WriteString("# HELP multiwanbond_wan_upload_bps WAN interface upload bandwidth (bytes per second)\n")
		sb.WriteString("# TYPE multiwanbond_wan_upload_bps gauge\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_upload_bps{wan_id=\"%d\"} %.2f\n", wanID, metrics.CurrentUpload))
		}

		sb.WriteString("# HELP multiwanbond_wan_download_bps WAN interface download bandwidth (bytes per second)\n")
		sb.WriteString("# TYPE multiwanbond_wan_download_bps gauge\n")
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_download_bps{wan_id=\"%d\"} %.2f\n", wanID, metrics.CurrentDownload))
		}

		sb.WriteString("\n")
	}

	// Bandwidth quotas
	sb.WriteString("# HELP multiwanbond_wan_daily_usage_bytes Daily bandwidth usage\n")
	sb.WriteString("# TYPE multiwanbond_wan_daily_usage_bytes gauge\n")
	for wanID, quota := range e.collector.quotas {
		sb.WriteString(fmt.Sprintf("multiwanbond_wan_daily_usage_bytes{wan_id=\"%d\"} %d\n", wanID, quota.DailyUsage))
	}

	sb.WriteString("# HELP multiwanbond_wan_daily_limit_bytes Daily bandwidth limit\n")
	sb.WriteString("# TYPE multiwanbond_wan_daily_limit_bytes gauge\n")
	for wanID, quota := range e.collector.quotas {
		if quota.DailyLimit > 0 {
			sb.WriteString(fmt.Sprintf("multiwanbond_wan_daily_limit_bytes{wan_id=\"%d\"} %d\n", wanID, quota.DailyLimit))
		}
	}

	sb.WriteString("\n")

	// Alerts
	alerts := e.collector.GetUnresolvedAlerts()
	sb.WriteString("# HELP multiwanbond_unresolved_alerts Number of unresolved alerts\n")
	sb.WriteString("# TYPE multiwanbond_unresolved_alerts gauge\n")
	sb.WriteString(fmt.Sprintf("multiwanbond_unresolved_alerts %d\n", len(alerts)))

	return sb.String()
}

// ExportJSON exports metrics in JSON format
func (e *Exporter) ExportJSON() (string, error) {
	data := make(map[string]interface{})

	// System metrics
	systemMetrics := e.collector.GetSystemMetrics()
	data["system"] = map[string]interface{}{
		"uptime_seconds":      systemMetrics.Uptime.Seconds(),
		"total_bytes_sent":    systemMetrics.TotalBytesSent,
		"total_bytes_received": systemMetrics.TotalBytesReceived,
		"active_wans":         systemMetrics.ActiveWANs,
		"active_flows":        systemMetrics.ActiveFlows,
		"failover_count":      systemMetrics.FailoverCount,
		"current_pps":         systemMetrics.CurrentPPS,
		"memory_allocated":    systemMetrics.AllocatedMemory,
	}

	// WAN metrics
	wanMetricsData := make(map[string]interface{})
	for wanID, metrics := range e.collector.GetAllWANMetrics() {
		wanMetricsData[fmt.Sprintf("wan_%d", wanID)] = map[string]interface{}{
			"bytes_sent":       metrics.BytesSent,
			"bytes_received":   metrics.BytesReceived,
			"packets_sent":     metrics.PacketsSent,
			"packets_received": metrics.PacketsReceived,
			"latency_ms":       metrics.Latency.Milliseconds(),
			"jitter_ms":        metrics.Jitter.Milliseconds(),
			"packet_loss":      metrics.PacketLoss,
			"upload_bps":       metrics.CurrentUpload,
			"download_bps":     metrics.CurrentDownload,
		}
	}
	data["wan_metrics"] = wanMetricsData

	// Bandwidth quotas
	quotasData := make(map[string]interface{})
	for wanID, quota := range e.collector.quotas {
		daily, weekly, monthly := quota.GetUsagePercent()
		quotasData[fmt.Sprintf("wan_%d", wanID)] = map[string]interface{}{
			"daily_usage":     quota.DailyUsage,
			"daily_limit":     quota.DailyLimit,
			"daily_percent":   daily,
			"weekly_usage":    quota.WeeklyUsage,
			"weekly_limit":    quota.WeeklyLimit,
			"weekly_percent":  weekly,
			"monthly_usage":   quota.MonthlyUsage,
			"monthly_limit":   quota.MonthlyLimit,
			"monthly_percent": monthly,
		}
	}
	data["quotas"] = quotasData

	// Alerts
	alertsData := make([]map[string]interface{}, 0)
	for _, alert := range e.collector.GetAlerts() {
		alertsData = append(alertsData, map[string]interface{}{
			"id":            alert.ID,
			"severity":      alert.Severity,
			"title":         alert.Title,
			"description":   alert.Description,
			"metric":        alert.Metric,
			"threshold":     alert.Threshold,
			"current_value": alert.CurrentValue,
			"timestamp":     alert.Timestamp,
			"resolved":      alert.Resolved,
		})
	}
	data["alerts"] = alertsData

	// Time series summary
	timeSeriesData := make(map[string]interface{})
	for name, ts := range e.collector.GetAllTimeSeries() {
		latest := ts.Latest()
		if latest != nil {
			timeSeriesData[name] = map[string]interface{}{
				"type":        ts.Type.String(),
				"data_points": len(ts.DataPoints),
				"latest_value": latest.Value,
				"latest_time":  latest.Timestamp,
			}
		}
	}
	data["time_series"] = timeSeriesData

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// ExportCSV exports time series data in CSV format
func (e *Exporter) ExportCSV(seriesName string) string {
	ts, exists := e.collector.GetTimeSeries(seriesName)
	if !exists {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("timestamp,value\n")

	ts.mu.RLock()
	defer ts.mu.RUnlock()

	for _, dp := range ts.DataPoints {
		sb.WriteString(fmt.Sprintf("%s,%.6f\n", dp.Timestamp.Format(time.RFC3339), dp.Value))
	}

	return sb.String()
}

// ExportInfluxDB exports metrics in InfluxDB line protocol format
func (e *Exporter) ExportInfluxDB() string {
	var sb strings.Builder

	// System metrics
	systemMetrics := e.collector.GetSystemMetrics()
	if systemMetrics != nil {
		sb.WriteString(fmt.Sprintf("system uptime=%.0f,bytes_sent=%d,bytes_received=%d,active_wans=%d,active_flows=%d %d\n",
			systemMetrics.Uptime.Seconds(),
			systemMetrics.TotalBytesSent,
			systemMetrics.TotalBytesReceived,
			systemMetrics.ActiveWANs,
			systemMetrics.ActiveFlows,
			time.Now().UnixNano()))
	}

	// WAN metrics
	for wanID, metrics := range e.collector.GetAllWANMetrics() {
		sb.WriteString(fmt.Sprintf("wan,wan_id=%d bytes_sent=%d,bytes_received=%d,latency_ms=%.2f,packet_loss=%.2f %d\n",
			wanID,
			metrics.BytesSent,
			metrics.BytesReceived,
			float64(metrics.Latency.Milliseconds()),
			metrics.PacketLoss,
			time.Now().UnixNano()))
	}

	return sb.String()
}

// ExportGraphite exports metrics in Graphite plaintext format
func (e *Exporter) ExportGraphite(prefix string) string {
	var sb strings.Builder
	timestamp := time.Now().Unix()

	if prefix == "" {
		prefix = "multiwanbond"
	}

	// System metrics
	systemMetrics := e.collector.GetSystemMetrics()
	if systemMetrics != nil {
		sb.WriteString(fmt.Sprintf("%s.system.uptime %.0f %d\n", prefix, systemMetrics.Uptime.Seconds(), timestamp))
		sb.WriteString(fmt.Sprintf("%s.system.bytes_sent %d %d\n", prefix, systemMetrics.TotalBytesSent, timestamp))
		sb.WriteString(fmt.Sprintf("%s.system.bytes_received %d %d\n", prefix, systemMetrics.TotalBytesReceived, timestamp))
		sb.WriteString(fmt.Sprintf("%s.system.active_wans %d %d\n", prefix, systemMetrics.ActiveWANs, timestamp))
		sb.WriteString(fmt.Sprintf("%s.system.active_flows %d %d\n", prefix, systemMetrics.ActiveFlows, timestamp))
	}

	// WAN metrics
	for wanID, metrics := range e.collector.GetAllWANMetrics() {
		sb.WriteString(fmt.Sprintf("%s.wan.%d.bytes_sent %d %d\n", prefix, wanID, metrics.BytesSent, timestamp))
		sb.WriteString(fmt.Sprintf("%s.wan.%d.bytes_received %d %d\n", prefix, wanID, metrics.BytesReceived, timestamp))
		sb.WriteString(fmt.Sprintf("%s.wan.%d.latency_ms %.2f %d\n", prefix, wanID, float64(metrics.Latency.Milliseconds()), timestamp))
		sb.WriteString(fmt.Sprintf("%s.wan.%d.packet_loss %.2f %d\n", prefix, wanID, metrics.PacketLoss, timestamp))
	}

	return sb.String()
}

// ExportAggregatedJSON exports aggregated metrics in JSON format
func (e *Exporter) ExportAggregatedJSON(window AggregationWindow) (string, error) {
	aggregator := NewAggregator()
	data := make(map[string]interface{})

	// Aggregate each time series
	aggregatedSeries := make(map[string]interface{})
	for name, ts := range e.collector.GetAllTimeSeries() {
		aggregated := aggregator.AggregateTimeSeries(ts, window)
		if aggregated != nil {
			aggregatedSeries[name] = map[string]interface{}{
				"window":  window.String(),
				"count":   aggregated.Count,
				"sum":     aggregated.Sum,
				"min":     aggregated.Min,
				"max":     aggregated.Max,
				"avg":     aggregated.Avg,
				"median":  aggregated.Median,
				"p95":     aggregated.P95,
				"p99":     aggregated.P99,
				"std_dev": aggregated.StdDev,
			}
		}
	}
	data["aggregated_series"] = aggregatedSeries
	data["window"] = window.String()
	data["duration_seconds"] = window.Duration().Seconds()

	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// ExportSummary exports a summary of all metrics
func (e *Exporter) ExportSummary() string {
	var sb strings.Builder

	sb.WriteString("=== MultiWANBond Metrics Summary ===\n\n")

	// System metrics
	systemMetrics := e.collector.GetSystemMetrics()
	if systemMetrics != nil {
		sb.WriteString("System Metrics:\n")
		sb.WriteString(fmt.Sprintf("  Uptime: %v\n", systemMetrics.Uptime))
		sb.WriteString(fmt.Sprintf("  Active WANs: %d/%d\n", systemMetrics.ActiveWANs, systemMetrics.TotalWANs))
		sb.WriteString(fmt.Sprintf("  Active Flows: %d\n", systemMetrics.ActiveFlows))
		sb.WriteString(fmt.Sprintf("  Total Traffic: %d bytes sent, %d bytes received\n",
			systemMetrics.TotalBytesSent, systemMetrics.TotalBytesReceived))
		sb.WriteString(fmt.Sprintf("  Failovers: %d\n", systemMetrics.FailoverCount))
		sb.WriteString(fmt.Sprintf("  Current PPS: %d\n", systemMetrics.CurrentPPS))
		sb.WriteString("\n")
	}

	// WAN metrics
	wanMetrics := e.collector.GetAllWANMetrics()
	if len(wanMetrics) > 0 {
		sb.WriteString(fmt.Sprintf("WAN Metrics (%d WANs):\n", len(wanMetrics)))
		for wanID, metrics := range wanMetrics {
			sb.WriteString(fmt.Sprintf("  WAN %d:\n", wanID))
			sb.WriteString(fmt.Sprintf("    Traffic: %d sent, %d received\n",
				metrics.BytesSent, metrics.BytesReceived))
			sb.WriteString(fmt.Sprintf("    Performance: latency=%v, jitter=%v, loss=%.2f%%\n",
				metrics.Latency, metrics.Jitter, metrics.PacketLoss))
			sb.WriteString(fmt.Sprintf("    Bandwidth: upload=%.2f bps, download=%.2f bps\n",
				metrics.CurrentUpload, metrics.CurrentDownload))
		}
		sb.WriteString("\n")
	}

	// Quotas
	if len(e.collector.quotas) > 0 {
		sb.WriteString("Bandwidth Quotas:\n")
		for wanID, quota := range e.collector.quotas {
			daily, weekly, monthly := quota.GetUsagePercent()
			sb.WriteString(fmt.Sprintf("  WAN %d:\n", wanID))
			sb.WriteString(fmt.Sprintf("    Daily: %.1f%% (%d/%d bytes)\n",
				daily, quota.DailyUsage, quota.DailyLimit))
			sb.WriteString(fmt.Sprintf("    Weekly: %.1f%% (%d/%d bytes)\n",
				weekly, quota.WeeklyUsage, quota.WeeklyLimit))
			sb.WriteString(fmt.Sprintf("    Monthly: %.1f%% (%d/%d bytes)\n",
				monthly, quota.MonthlyUsage, quota.MonthlyLimit))
		}
		sb.WriteString("\n")
	}

	// Alerts
	alerts := e.collector.GetUnresolvedAlerts()
	if len(alerts) > 0 {
		sb.WriteString(fmt.Sprintf("Unresolved Alerts (%d):\n", len(alerts)))
		for _, alert := range alerts {
			sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", alert.Severity, alert.Title, alert.Description))
		}
		sb.WriteString("\n")
	}

	// Time series
	timeSeriesCount := len(e.collector.GetAllTimeSeries())
	sb.WriteString(fmt.Sprintf("Time Series: %d series tracked\n", timeSeriesCount))

	return sb.String()
}

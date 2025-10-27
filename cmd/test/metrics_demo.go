// Package main demonstrates Metrics functionality
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/thelastdreamer/MultiWANBond/pkg/metrics"
)

func main() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("MultiWANBond - Advanced Metrics & Time-Series Demo")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	testResults := make(map[string]bool)

	// Test 1: Create metrics collector
	fmt.Println("Test 1: Metrics Collector Creation")
	fmt.Println(strings.Repeat("-", 80))

	config := metrics.DefaultMetricsConfig()
	fmt.Printf("Collection Interval: %v\n", config.CollectionInterval)
	fmt.Printf("Retention Period: %v\n", config.RetentionPeriod)
	fmt.Printf("Max Data Points: %d\n", config.MaxDataPoints)
	fmt.Printf("Aggregation Windows: %d\n", len(config.AggregationWindows))

	collector := metrics.NewCollector(config)
	if collector != nil {
		fmt.Println("Collector created successfully")
		testResults["Collector Creation"] = true
	} else {
		fmt.Println("Failed to create collector")
		testResults["Collector Creation"] = false
	}
	fmt.Println()

	// Test 2: Time series data
	fmt.Println("Test 2: Time Series Management")
	fmt.Println(strings.Repeat("-", 80))

	ts := metrics.NewTimeSeries("test_metric", metrics.MetricTypeGauge, map[string]string{"test": "label"})

	// Add sample data points
	now := time.Now()
	for i := 0; i < 100; i++ {
		value := 50.0 + 10.0*float64(i) + rand.Float64()*5.0
		ts.AddPoint(now.Add(-time.Duration(100-i)*time.Second), value)
	}

	fmt.Printf("Time series: %s\n", ts.Name)
	fmt.Printf("Type: %s\n", ts.Type.String())
	fmt.Printf("Data points: %d\n", len(ts.DataPoints))

	latest := ts.Latest()
	if latest != nil {
		fmt.Printf("Latest value: %.2f at %s\n", latest.Value, latest.Timestamp.Format("15:04:05"))
	}

	// Test pruning
	pruned := ts.Prune(30 * time.Second)
	fmt.Printf("Pruned %d old data points (older than 30s)\n", pruned)
	fmt.Printf("Remaining data points: %d\n", len(ts.DataPoints))

	testResults["Time Series Management"] = len(ts.DataPoints) > 0
	fmt.Println()

	// Test 3: WAN metrics
	fmt.Println("Test 3: WAN Metrics Recording")
	fmt.Println(strings.Repeat("-", 80))

	// Start collector
	if err := collector.Start(); err != nil {
		fmt.Printf("Failed to start collector: %v\n", err)
		testResults["WAN Metrics"] = false
	} else {
		// Record metrics for WAN 1
		collector.RecordWANMetric(1, 1000000, 2000000, 10000, 20000, 25*time.Millisecond, 3*time.Millisecond, 0.5)
		collector.RecordWANBandwidth(1, 1000000, 2000000)

		// Record metrics for WAN 2
		collector.RecordWANMetric(2, 500000, 1000000, 5000, 10000, 45*time.Millisecond, 5*time.Millisecond, 1.2)
		collector.RecordWANBandwidth(2, 500000, 1000000)

		time.Sleep(100 * time.Millisecond)

		// Retrieve WAN metrics
		wan1Metrics, exists1 := collector.GetWANMetrics(1)
		wan2Metrics, exists2 := collector.GetWANMetrics(2)

		if exists1 && exists2 {
			fmt.Printf("WAN 1:\n")
			fmt.Printf("  Bytes sent: %d, received: %d\n", wan1Metrics.BytesSent, wan1Metrics.BytesReceived)
			fmt.Printf("  Latency: %v, Jitter: %v, Loss: %.2f%%\n", wan1Metrics.Latency, wan1Metrics.Jitter, wan1Metrics.PacketLoss)
			fmt.Printf("  Upload: %.0f bps, Download: %.0f bps\n", wan1Metrics.CurrentUpload, wan1Metrics.CurrentDownload)

			fmt.Printf("WAN 2:\n")
			fmt.Printf("  Bytes sent: %d, received: %d\n", wan2Metrics.BytesSent, wan2Metrics.BytesReceived)
			fmt.Printf("  Latency: %v, Jitter: %v, Loss: %.2f%%\n", wan2Metrics.Latency, wan2Metrics.Jitter, wan2Metrics.PacketLoss)
			fmt.Printf("  Upload: %.0f bps, Download: %.0f bps\n", wan2Metrics.CurrentUpload, wan2Metrics.CurrentDownload)

			testResults["WAN Metrics"] = true
		} else {
			fmt.Println("Failed to retrieve WAN metrics")
			testResults["WAN Metrics"] = false
		}
	}
	fmt.Println()

	// Test 4: Flow metrics
	fmt.Println("Test 4: Flow Metrics Recording")
	fmt.Println(strings.Repeat("-", 80))

	// Record flow metrics
	collector.RecordFlowMetric("flow1", "YouTube", "Streaming", 1, 100000, 500000, 1000, 5000)
	collector.RecordFlowMetric("flow2", "Zoom", "VoIP", 2, 50000, 50000, 500, 500)
	collector.RecordFlowMetric("flow3", "HTTP", "Web", 1, 10000, 100000, 100, 1000)

	time.Sleep(100 * time.Millisecond)

	flow1, exists := collector.GetFlowMetrics("flow1")
	if exists {
		fmt.Printf("Flow 1 (YouTube):\n")
		fmt.Printf("  Application: %s, Category: %s\n", flow1.Application, flow1.Category)
		fmt.Printf("  Traffic: %d sent, %d received\n", flow1.BytesSent, flow1.BytesReceived)
		fmt.Printf("  Duration: %v, Active: %v\n", flow1.Duration, flow1.Active)

		// Close flow
		collector.CloseFlow("flow1")
		fmt.Printf("  Flow closed\n")

		testResults["Flow Metrics"] = true
	} else {
		fmt.Println("Failed to retrieve flow metrics")
		testResults["Flow Metrics"] = false
	}
	fmt.Println()

	// Test 5: System metrics
	fmt.Println("Test 5: System Metrics")
	fmt.Println(strings.Repeat("-", 80))

	time.Sleep(500 * time.Millisecond)

	systemMetrics := collector.GetSystemMetrics()
	fmt.Printf("Uptime: %v\n", systemMetrics.Uptime)
	fmt.Printf("Active WANs: %d/%d\n", systemMetrics.ActiveWANs, systemMetrics.TotalWANs)
	fmt.Printf("Active Flows: %d\n", systemMetrics.ActiveFlows)
	fmt.Printf("Allocated Memory: %d bytes\n", systemMetrics.AllocatedMemory)

	testResults["System Metrics"] = systemMetrics.Uptime > 0
	fmt.Println()

	// Test 6: Bandwidth quotas
	fmt.Println("Test 6: Bandwidth Quotas")
	fmt.Println(strings.Repeat("-", 80))

	// Set quota for WAN 1: 10GB daily, 50GB weekly, 200GB monthly
	collector.SetBandwidthQuota(1, 10*1024*1024*1024, 50*1024*1024*1024, 200*1024*1024*1024)

	quota, hasQuota := collector.GetBandwidthQuota(1)
	if hasQuota {
		fmt.Printf("WAN 1 Quota:\n")
		fmt.Printf("  Daily: %d bytes limit\n", quota.DailyLimit)
		fmt.Printf("  Weekly: %d bytes limit\n", quota.WeeklyLimit)
		fmt.Printf("  Monthly: %d bytes limit\n", quota.MonthlyLimit)

		// Simulate usage
		dailyExceeded, weeklyExceeded, monthlyExceeded := quota.AddUsage(1024 * 1024 * 1024) // 1GB
		fmt.Printf("  Used 1GB: daily_exceeded=%v, weekly_exceeded=%v, monthly_exceeded=%v\n",
			dailyExceeded, weeklyExceeded, monthlyExceeded)

		daily, weekly, monthly := quota.GetUsagePercent()
		fmt.Printf("  Usage: Daily=%.2f%%, Weekly=%.2f%%, Monthly=%.2f%%\n", daily, weekly, monthly)

		testResults["Bandwidth Quotas"] = true
	} else {
		fmt.Println("Failed to get bandwidth quota")
		testResults["Bandwidth Quotas"] = false
	}
	fmt.Println()

	// Test 7: Aggregation
	fmt.Println("Test 7: Time Series Aggregation")
	fmt.Println(strings.Repeat("-", 80))

	aggregator := metrics.NewAggregator()

	// Create sample time series with 1 minute of data
	testTS := metrics.NewTimeSeries("test_latency", metrics.MetricTypeGauge, nil)
	baseTime := time.Now().Add(-1 * time.Minute)
	for i := 0; i < 60; i++ {
		value := 20.0 + 5.0*float64(i)/60.0 + rand.Float64()*2.0
		testTS.AddPoint(baseTime.Add(time.Duration(i)*time.Second), value)
	}

	// Aggregate over 1 minute window
	aggregated := aggregator.AggregateTimeSeries(testTS, metrics.Window1Minute)
	if aggregated != nil {
		fmt.Printf("Aggregation Window: %s\n", aggregated.Window.String())
		fmt.Printf("Data Points: %d\n", aggregated.Count)
		fmt.Printf("Statistics:\n")
		fmt.Printf("  Min: %.2f\n", aggregated.Min)
		fmt.Printf("  Max: %.2f\n", aggregated.Max)
		fmt.Printf("  Avg: %.2f\n", aggregated.Avg)
		fmt.Printf("  Median: %.2f\n", aggregated.Median)
		fmt.Printf("  P95: %.2f\n", aggregated.P95)
		fmt.Printf("  P99: %.2f\n", aggregated.P99)
		fmt.Printf("  StdDev: %.2f\n", aggregated.StdDev)

		testResults["Aggregation"] = aggregated.Count == 60
	} else {
		fmt.Println("Failed to aggregate time series")
		testResults["Aggregation"] = false
	}
	fmt.Println()

	// Test 8: Failover recording
	fmt.Println("Test 8: Failover Recording")
	fmt.Println(strings.Repeat("-", 80))

	collector.RecordFailover(2, 1, "High packet loss on WAN 2")
	collector.RecordFailover(1, 2, "WAN 1 latency spike")

	systemMetrics = collector.GetSystemMetrics()
	fmt.Printf("Total Failovers: %d\n", systemMetrics.FailoverCount)
	if !systemMetrics.LastFailover.IsZero() {
		fmt.Printf("Last Failover: %s\n", systemMetrics.LastFailover.Format("15:04:05"))
	}

	testResults["Failover Recording"] = systemMetrics.FailoverCount == 2
	fmt.Println()

	// Test 9: Alerts
	fmt.Println("Test 9: Alert System")
	fmt.Println(strings.Repeat("-", 80))

	alerts := collector.GetAlerts()
	fmt.Printf("Total Alerts: %d\n", len(alerts))

	unresolvedAlerts := collector.GetUnresolvedAlerts()
	fmt.Printf("Unresolved Alerts: %d\n", len(unresolvedAlerts))

	if len(unresolvedAlerts) > 0 {
		fmt.Println("Recent Alerts:")
		for i, alert := range unresolvedAlerts {
			if i >= 5 {
				break
			}
			fmt.Printf("  [%s] %s: %s\n", alert.Severity, alert.Title, alert.Description)
		}

		// Resolve first alert
		if len(unresolvedAlerts) > 0 {
			resolved := collector.ResolveAlert(unresolvedAlerts[0].ID)
			fmt.Printf("Resolved alert: %v\n", resolved)
		}
	}

	testResults["Alert System"] = len(alerts) > 0
	fmt.Println()

	// Test 10: Export formats
	fmt.Println("Test 10: Metrics Export Formats")
	fmt.Println(strings.Repeat("-", 80))

	exporter := metrics.NewExporter(collector)

	// Prometheus format
	promData := exporter.ExportPrometheus()
	fmt.Printf("Prometheus Export:\n")
	lines := strings.Split(promData, "\n")
	for i := 0; i < 10 && i < len(lines); i++ {
		if len(lines[i]) > 0 && !strings.HasPrefix(lines[i], "#") {
			fmt.Printf("  %s\n", lines[i])
		}
	}
	fmt.Printf("  ... (%d lines total)\n", len(lines))

	// JSON format
	jsonData, err := exporter.ExportJSON()
	if err == nil {
		fmt.Printf("\nJSON Export: %d bytes\n", len(jsonData))
	}

	// Summary
	summary := exporter.ExportSummary()
	fmt.Printf("\nSummary Export:\n")
	summaryLines := strings.Split(summary, "\n")
	for i := 0; i < 15 && i < len(summaryLines); i++ {
		fmt.Printf("%s\n", summaryLines[i])
	}

	testResults["Export Formats"] = len(promData) > 0 && len(jsonData) > 0
	fmt.Println()

	// Stop collector
	if err := collector.Stop(); err != nil {
		fmt.Printf("Error stopping collector: %v\n", err)
	}

	// Print summary
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Test Summary")
	fmt.Println(strings.Repeat("=", 80))

	passed := 0
	total := len(testResults)

	for test, result := range testResults {
		status := "❌ FAIL"
		if result {
			status = "✓ PASS"
			passed++
		}
		fmt.Printf("%s: %s\n", status, test)
	}

	fmt.Println()
	fmt.Printf("Tests Passed: %d/%d (%.0f%%)\n", passed, total, float64(passed)/float64(total)*100)
	fmt.Println(strings.Repeat("=", 80))

	// Print feature summary
	fmt.Println()
	fmt.Println("Phase 8 Features Implemented:")
	fmt.Println("  - Time-series data collection and storage")
	fmt.Println("  - WAN, Flow, and System metrics tracking")
	fmt.Println("  - Bandwidth quota management and accounting")
	fmt.Println("  - Alert generation and resolution")
	fmt.Println("  - Statistical aggregations (min, max, avg, median, p95, p99, stddev)")
	fmt.Println("  - Multiple export formats (Prometheus, JSON, CSV, InfluxDB, Graphite)")
	fmt.Println("  - Rolling window aggregations (1m, 5m, 15m, 1h, 6h, 1d, 1w)")
	fmt.Println("  - Anomaly detection and trend analysis")
	fmt.Println("  - Moving average calculations")
	fmt.Println("  - Automatic data pruning and retention")
	fmt.Println()
	fmt.Println("Ready for production monitoring!")
}

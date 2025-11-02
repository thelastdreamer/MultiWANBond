# MultiWANBond Metrics Guide

**Complete reference for all MultiWANBond Prometheus metrics**

**Version**: 1.2
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Metrics Overview](#metrics-overview)
- [System Metrics](#system-metrics)
- [WAN Metrics](#wan-metrics)
- [Traffic Metrics](#traffic-metrics)
- [Flow Metrics](#flow-metrics)
- [Alert Metrics](#alert-metrics)
- [Using Metrics](#using-metrics)
- [Metric Retention](#metric-retention)

---

## Metrics Overview

MultiWANBond exposes metrics in Prometheus format at the `/metrics` endpoint.

**Endpoint**: `http://localhost:8080/metrics`

**Format**: Prometheus text-based exposition format

**Authentication**: Requires session cookie (same as Web UI)

**Metric Types**:
- **Gauge**: Value that can go up and down (e.g., latency, memory)
- **Counter**: Value that only increases (e.g., bytes transferred)

---

## System Metrics

### multiwanbond_uptime_seconds

**Type**: Gauge

**Description**: System uptime in seconds since MultiWANBond started

**Unit**: seconds

**Example**:
```prometheus
# HELP multiwanbond_uptime_seconds System uptime in seconds
# TYPE multiwanbond_uptime_seconds gauge
multiwanbond_uptime_seconds 86400
```

**Interpretation**:
- 86400 seconds = 24 hours
- Useful for tracking service restarts
- Combined with Grafana, shows uptime duration

**PromQL Examples**:
```promql
# Uptime in hours
multiwanbond_uptime_seconds / 3600

# Uptime in days
multiwanbond_uptime_seconds / 86400

# Alert if uptime < 1 hour (recent restart)
multiwanbond_uptime_seconds < 3600
```

---

### multiwanbond_goroutines

**Type**: Gauge

**Description**: Number of active goroutines in the Go runtime

**Unit**: count

**Example**:
```prometheus
# HELP multiwanbond_goroutines Number of goroutines
# TYPE multiwanbond_goroutines gauge
multiwanbond_goroutines 42
```

**Interpretation**:
- Typical value: 20-100 goroutines
- Increasing trend: May indicate goroutine leak
- Sudden spike: Burst of activity or potential issue

**PromQL Examples**:
```promql
# Alert if goroutines > 1000
multiwanbond_goroutines > 1000

# Rate of goroutine growth
rate(multiwanbond_goroutines[5m])
```

---

### multiwanbond_memory_bytes

**Type**: Gauge

**Description**: Memory usage in bytes

**Labels**:
- `type="alloc"`: Memory allocated and in use
- `type="sys"`: Memory obtained from system

**Unit**: bytes

**Example**:
```prometheus
# HELP multiwanbond_memory_bytes Memory usage in bytes
# TYPE multiwanbond_memory_bytes gauge
multiwanbond_memory_bytes{type="alloc"} 52428800
multiwanbond_memory_bytes{type="sys"} 73400320
```

**Interpretation**:
- `alloc`: Actual memory used by application (~50 MB in example)
- `sys`: Total memory reserved from OS (~70 MB in example)
- `sys` > `alloc`: Normal, OS may reuse freed memory

**PromQL Examples**:
```promql
# Memory usage in MB
multiwanbond_memory_bytes{type="alloc"} / 1024 / 1024

# Alert if memory usage > 1 GB
multiwanbond_memory_bytes{type="alloc"} > 1073741824

# Memory growth rate
rate(multiwanbond_memory_bytes{type="alloc"}[5m])
```

---

## WAN Metrics

### multiwanbond_wan_state

**Type**: Gauge

**Description**: WAN interface state (1 = up, 0 = down)

**Labels**:
- `wan_id`: WAN interface ID (e.g., "1", "2", "3")
- `wan_name`: WAN interface name (e.g., "wan1", "wan2")

**Example**:
```prometheus
# HELP multiwanbond_wan_state WAN state (1=up, 0=down)
# TYPE multiwanbond_wan_state gauge
multiwanbond_wan_state{wan_id="1",wan_name="wan1"} 1
multiwanbond_wan_state{wan_id="2",wan_name="wan2"} 1
multiwanbond_wan_state{wan_id="3",wan_name="wan3"} 0
```

**Interpretation**:
- `1`: WAN is up and healthy
- `0`: WAN is down or failed health checks

**PromQL Examples**:
```promql
# Number of healthy WANs
count(multiwanbond_wan_state == 1)

# Number of down WANs
count(multiwanbond_wan_state == 0)

# Alert if any WAN down
multiwanbond_wan_state == 0

# Alert if <2 WANs available
count(multiwanbond_wan_state == 1) < 2
```

---

### multiwanbond_wan_latency_ms

**Type**: Gauge

**Description**: WAN latency in milliseconds

**Labels**:
- `wan_id`: WAN interface ID
- `wan_name`: WAN interface name

**Unit**: milliseconds

**Example**:
```prometheus
# HELP multiwanbond_wan_latency_ms WAN latency in milliseconds
# TYPE multiwanbond_wan_latency_ms gauge
multiwanbond_wan_latency_ms{wan_id="1",wan_name="wan1"} 5.23
multiwanbond_wan_latency_ms{wan_id="2",wan_name="wan2"} 25.47
multiwanbond_wan_latency_ms{wan_id="3",wan_name="wan3"} 150.82
```

**Interpretation**:
- Low latency: <20ms (fiber, local connections)
- Medium latency: 20-100ms (satellite, long-distance)
- High latency: >100ms (may affect real-time applications)

**PromQL Examples**:
```promql
# Average latency across all WANs
avg(multiwanbond_wan_latency_ms)

# Maximum latency
max(multiwanbond_wan_latency_ms)

# WAN with lowest latency
topk(1, multiwanbond_wan_latency_ms)

# Alert if latency > 100ms
multiwanbond_wan_latency_ms > 100

# Latency spike (change > 50ms in 5 minutes)
delta(multiwanbond_wan_latency_ms[5m]) > 50
```

---

### multiwanbond_wan_jitter_ms

**Type**: Gauge

**Description**: WAN jitter (latency variation) in milliseconds

**Labels**:
- `wan_id`: WAN interface ID
- `wan_name`: WAN interface name

**Unit**: milliseconds

**Example**:
```prometheus
# HELP multiwanbond_wan_jitter_ms WAN jitter in milliseconds
# TYPE multiwanbond_wan_jitter_ms gauge
multiwanbond_wan_jitter_ms{wan_id="1",wan_name="wan1"} 0.82
multiwanbond_wan_jitter_ms{wan_id="2",wan_name="wan2"} 3.45
```

**Interpretation**:
- Low jitter: <10ms (stable connection)
- Medium jitter: 10-50ms (acceptable for most uses)
- High jitter: >50ms (may cause issues for VoIP, gaming)

**PromQL Examples**:
```promql
# Average jitter
avg(multiwanbond_wan_jitter_ms)

# Alert if jitter > 50ms
multiwanbond_wan_jitter_ms > 50
```

---

### multiwanbond_wan_packet_loss

**Type**: Gauge

**Description**: WAN packet loss percentage

**Labels**:
- `wan_id`: WAN interface ID
- `wan_name`: WAN interface name

**Unit**: percentage (0-100)

**Example**:
```prometheus
# HELP multiwanbond_wan_packet_loss WAN packet loss percentage
# TYPE multiwanbond_wan_packet_loss gauge
multiwanbond_wan_packet_loss{wan_id="1",wan_name="wan1"} 0.01
multiwanbond_wan_packet_loss{wan_id="2",wan_name="wan2"} 2.5
multiwanbond_wan_packet_loss{wan_id="3",wan_name="wan3"} 15.0
```

**Interpretation**:
- Excellent: 0-0.1% (nearly lossless)
- Good: 0.1-1% (acceptable)
- Fair: 1-5% (FEC can compensate)
- Poor: >5% (significant quality degradation)

**PromQL Examples**:
```promql
# Average packet loss
avg(multiwanbond_wan_packet_loss)

# Alert if packet loss > 5%
multiwanbond_wan_packet_loss > 5

# WANs with >1% loss
multiwanbond_wan_packet_loss > 1
```

---

## Traffic Metrics

### multiwanbond_traffic_bytes

**Type**: Counter

**Description**: Total traffic in bytes (cumulative)

**Labels**:
- `wan_id`: WAN interface ID
- `wan_name`: WAN interface name
- `direction`: "tx" (upload) or "rx" (download)

**Unit**: bytes

**Example**:
```prometheus
# HELP multiwanbond_traffic_bytes Total traffic in bytes
# TYPE multiwanbond_traffic_bytes counter
multiwanbond_traffic_bytes{wan_id="1",wan_name="wan1",direction="tx"} 5368709120
multiwanbond_traffic_bytes{wan_id="1",wan_name="wan1",direction="rx"} 10737418240
multiwanbond_traffic_bytes{wan_id="2",wan_name="wan2",direction="tx"} 2147483648
multiwanbond_traffic_bytes{wan_id="2",wan_name="wan2",direction="rx"} 4294967296
```

**Interpretation**:
- Cumulative counter (always increasing)
- Use `rate()` to get current throughput
- Reset on service restart

**PromQL Examples**:
```promql
# Upload rate (bytes per second) for WAN 1
rate(multiwanbond_traffic_bytes{wan_id="1",direction="tx"}[5m])

# Download rate in Mbps for WAN 1
rate(multiwanbond_traffic_bytes{wan_id="1",direction="rx"}[5m]) * 8 / 1000000

# Total upload across all WANs (Mbps)
sum(rate(multiwanbond_traffic_bytes{direction="tx"}[5m])) * 8 / 1000000

# Total download across all WANs (Mbps)
sum(rate(multiwanbond_traffic_bytes{direction="rx"}[5m])) * 8 / 1000000

# Traffic in last 24 hours (GB)
increase(multiwanbond_traffic_bytes[24h]) / 1024 / 1024 / 1024

# Traffic distribution (percentage per WAN)
multiwanbond_traffic_bytes / sum(multiwanbond_traffic_bytes) * 100
```

---

### multiwanbond_total_bytes_all

**Type**: Counter

**Description**: Total bytes across all WANs (aggregate)

**Labels**:
- `direction`: "tx" (upload) or "rx" (download)

**Unit**: bytes

**Example**:
```prometheus
# HELP multiwanbond_total_bytes_all Total bytes across all WANs
# TYPE multiwanbond_total_bytes_all counter
multiwanbond_total_bytes_all{direction="tx"} 7516192768
multiwanbond_total_bytes_all{direction="rx"} 15032385536
```

**PromQL Examples**:
```promql
# Total throughput (Mbps)
sum(rate(multiwanbond_total_bytes_all[5m])) * 8 / 1000000

# Upload vs download ratio
multiwanbond_total_bytes_all{direction="tx"} / multiwanbond_total_bytes_all{direction="rx"}
```

---

### multiwanbond_current_mbps

**Type**: Gauge

**Description**: Current throughput in Mbps

**Labels**:
- `direction`: "tx" (upload) or "rx" (download)

**Unit**: Mbps (megabits per second)

**Example**:
```prometheus
# HELP multiwanbond_current_mbps Current throughput in Mbps
# TYPE multiwanbond_current_mbps gauge
multiwanbond_current_mbps{direction="tx"} 45.23
multiwanbond_current_mbps{direction="rx"} 125.67
```

**PromQL Examples**:
```promql
# Average throughput over 1 hour
avg_over_time(multiwanbond_current_mbps[1h])

# Peak throughput in last 24h
max_over_time(multiwanbond_current_mbps[24h])

# Alert if throughput < 10 Mbps (underutilized)
multiwanbond_current_mbps < 10
```

---

## Flow Metrics

### multiwanbond_flows_total

**Type**: Gauge

**Description**: Total number of active network flows

**Example**:
```prometheus
# HELP multiwanbond_flows_total Total number of active flows
# TYPE multiwanbond_flows_total gauge
multiwanbond_flows_total 142
```

**Interpretation**:
- Normal: 10-1000 flows
- High: 1000-5000 flows
- Very high: >5000 flows (may need tuning)

**PromQL Examples**:
```promql
# Average flows over time
avg_over_time(multiwanbond_flows_total[1h])

# Peak flows
max_over_time(multiwanbond_flows_total[24h])

# Alert if flows > 5000
multiwanbond_flows_total > 5000
```

---

## Alert Metrics

### multiwanbond_alerts_total

**Type**: Gauge

**Description**: Total number of active alerts

**Example**:
```prometheus
# HELP multiwanbond_alerts_total Total number of active alerts
# TYPE multiwanbond_alerts_total gauge
multiwanbond_alerts_total 3
```

**Interpretation**:
- 0 alerts: System healthy
- 1-5 alerts: Some issues, investigate
- >5 alerts: Multiple problems, urgent attention needed

**PromQL Examples**:
```promql
# Alert if any alerts present
multiwanbond_alerts_total > 0

# Alert if >5 alerts (critical)
multiwanbond_alerts_total > 5
```

---

## Using Metrics

### Accessing Metrics

**Via cURL**:
```bash
curl http://localhost:8080/metrics
```

**Via Browser**:
```
http://localhost:8080/metrics
```

**With Authentication**:
```bash
curl -H "Cookie: session_id=YOUR_SESSION_ID" http://localhost:8080/metrics
```

### Metric Format

```prometheus
# HELP metric_name Description of the metric
# TYPE metric_name gauge|counter
metric_name{label1="value1",label2="value2"} 123.45
```

**Example**:
```prometheus
# HELP multiwanbond_wan_latency_ms WAN latency in milliseconds
# TYPE multiwanbond_wan_latency_ms gauge
multiwanbond_wan_latency_ms{wan_id="1",wan_name="wan1"} 5.23
```

---

## Metric Retention

### Prometheus Default

- **Default retention**: 15 days
- **Configurable**: Via `--storage.tsdb.retention.time` flag

**Example**:
```bash
prometheus --storage.tsdb.retention.time=30d
```

### Recommended Retention

| Environment | Retention | Reason |
|-------------|-----------|--------|
| **Development** | 7 days | Short-lived data, save disk space |
| **Production** | 30 days | Troubleshooting historical issues |
| **Long-term** | Use remote storage | Thanos, Cortex, or InfluxDB |

### Storage Size Estimates

**Formula**: `Samples per second × Retention days × 24 × 3600 × 2 bytes`

**Example**:
- 100 samples/sec
- 30 days retention
- Storage: ~520 MB

**Actual usage varies based on**:
- Number of WANs
- Scrape interval
- Compression

---

## Complete Metrics List

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `multiwanbond_uptime_seconds` | Gauge | - | System uptime |
| `multiwanbond_goroutines` | Gauge | - | Number of goroutines |
| `multiwanbond_memory_bytes` | Gauge | `type` | Memory usage (alloc, sys) |
| `multiwanbond_wan_state` | Gauge | `wan_id`, `wan_name` | WAN state (1=up, 0=down) |
| `multiwanbond_wan_latency_ms` | Gauge | `wan_id`, `wan_name` | WAN latency |
| `multiwanbond_wan_jitter_ms` | Gauge | `wan_id`, `wan_name` | WAN jitter |
| `multiwanbond_wan_packet_loss` | Gauge | `wan_id`, `wan_name` | Packet loss percentage |
| `multiwanbond_traffic_bytes` | Counter | `wan_id`, `wan_name`, `direction` | Traffic per WAN |
| `multiwanbond_total_bytes_all` | Counter | `direction` | Total traffic all WANs |
| `multiwanbond_current_mbps` | Gauge | `direction` | Current throughput |
| `multiwanbond_flows_total` | Gauge | - | Active flows |
| `multiwanbond_alerts_total` | Gauge | - | Active alerts |

---

## Additional Resources

- [Prometheus Query Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [PromQL Examples](https://prometheus.io/docs/prometheus/latest/querying/examples/)
- [Grafana Setup Guide](GRAFANA_SETUP.md)
- [Performance Guide](PERFORMANCE.md)

---

**Last Updated**: November 2, 2025
**Version**: 1.2
**MultiWANBond Version**: 1.2

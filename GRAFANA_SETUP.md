# MultiWANBond Grafana Setup Guide

**Complete guide for setting up Grafana dashboards with Prometheus metrics**

**Version**: 1.2
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installing Prometheus](#installing-prometheus)
- [Installing Grafana](#installing-grafana)
- [Configuring Prometheus](#configuring-prometheus)
- [Importing Dashboard](#importing-dashboard)
- [Dashboard Overview](#dashboard-overview)
- [Creating Custom Dashboards](#creating-custom-dashboards)
- [Alerting](#alerting)
- [Troubleshooting](#troubleshooting)

---

## Overview

MultiWANBond exposes Prometheus-compatible metrics at `/metrics` endpoint, allowing you to visualize and monitor your multi-WAN setup using Grafana dashboards.

**What You'll Get**:
- Real-time WAN status visualization
- Traffic rate monitoring per WAN
- Latency and packet loss charts
- Traffic distribution pie charts
- System resource monitoring
- Historical data retention
- Customizable alerts

---

## Prerequisites

- MultiWANBond running with metrics enabled (enabled by default)
- Linux server (or Windows with Docker)
- Basic understanding of Prometheus and Grafana
- Network access to MultiWANBond metrics endpoint

---

## Installing Prometheus

### Linux (Ubuntu/Debian)

**1. Create Prometheus user**:
```bash
sudo useradd --no-create-home --shell /bin/false prometheus
```

**2. Download Prometheus**:
```bash
cd /tmp
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar xvf prometheus-2.45.0.linux-amd64.tar.gz
```

**3. Install binaries**:
```bash
sudo mv prometheus-2.45.0.linux-amd64/prometheus /usr/local/bin/
sudo mv prometheus-2.45.0.linux-amd64/promtool /usr/local/bin/
sudo chown prometheus:prometheus /usr/local/bin/prometheus
sudo chown prometheus:prometheus /usr/local/bin/promtool
```

**4. Create directories**:
```bash
sudo mkdir /etc/prometheus
sudo mkdir /var/lib/prometheus
sudo chown prometheus:prometheus /etc/prometheus
sudo chown prometheus:prometheus /var/lib/prometheus
```

**5. Create systemd service** (`/etc/systemd/system/prometheus.service`):
```ini
[Unit]
Description=Prometheus
Wants=network-online.target
After=network-online.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/bin/prometheus \
  --config.file /etc/prometheus/prometheus.yml \
  --storage.tsdb.path /var/lib/prometheus/ \
  --web.console.templates=/etc/prometheus/consoles \
  --web.console.libraries=/etc/prometheus/console_libraries

[Install]
WantedBy=multi-user.target
```

**6. Enable and start**:
```bash
sudo systemctl daemon-reload
sudo systemctl enable prometheus
sudo systemctl start prometheus
```

**7. Verify**:
```bash
sudo systemctl status prometheus
curl http://localhost:9090
```

### Docker

```bash
docker run -d \
  --name prometheus \
  -p 9090:9090 \
  -v /path/to/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus
```

---

## Installing Grafana

### Linux (Ubuntu/Debian)

**1. Add Grafana APT repository**:
```bash
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
```

**2. Install Grafana**:
```bash
sudo apt-get update
sudo apt-get install grafana
```

**3. Enable and start**:
```bash
sudo systemctl daemon-reload
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

**4. Access Grafana**:
- URL: `http://localhost:3000`
- Default username: `admin`
- Default password: `admin` (will be prompted to change)

### Docker

```bash
docker run -d \
  --name=grafana \
  -p 3000:3000 \
  grafana/grafana
```

---

## Configuring Prometheus

### Basic Configuration

Create `/etc/prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 15s  # Scrape every 15 seconds
  evaluation_interval: 15s  # Evaluate rules every 15 seconds

# Scrape MultiWANBond metrics
scrape_configs:
  - job_name: 'multiwanbond'
    static_configs:
      - targets: ['localhost:8080']  # MultiWANBond Web UI port
    metrics_path: '/metrics'
    scrape_interval: 5s  # More frequent scraping for real-time data
```

### Multiple MultiWANBond Instances

```yaml
scrape_configs:
  - job_name: 'multiwanbond'
    static_configs:
      - targets:
        - 'server1:8080'
        - 'server2:8080'
        - 'server3:8080'
        labels:
          environment: 'production'

  - job_name: 'multiwanbond-dev'
    static_configs:
      - targets:
        - 'dev-server:8080'
        labels:
          environment: 'development'
```

### With Authentication

If your MultiWANBond requires authentication:

```yaml
scrape_configs:
  - job_name: 'multiwanbond'
    static_configs:
      - targets: ['server:8080']
    basic_auth:
      username: 'admin'
      password: 'your_password'
```

**Security Note**: Store credentials in a separate file and reference it:

```yaml
scrape_configs:
  - job_name: 'multiwanbond'
    static_configs:
      - targets: ['server:8080']
    basic_auth:
      username: 'admin'
      password_file: '/etc/prometheus/multiwanbond_password.txt'
```

### Reload Configuration

After editing `prometheus.yml`:

```bash
# Send reload signal
sudo killall -HUP prometheus

# Or restart service
sudo systemctl restart prometheus
```

---

## Importing Dashboard

### Method 1: Import JSON File

**1. Download dashboard JSON**:
```bash
wget https://raw.githubusercontent.com/thelastdreamer/MultiWANBond/main/grafana/multiwanbond-dashboard.json
```

**2. In Grafana UI**:
- Navigate to **Dashboards** → **Import**
- Click **Upload JSON file**
- Select `multiwanbond-dashboard.json`
- Select Prometheus data source
- Click **Import**

### Method 2: Copy-Paste JSON

1. Open `grafana/multiwanbond-dashboard.json` in text editor
2. Copy all content
3. In Grafana: **Dashboards** → **Import** → **Import via panel json**
4. Paste JSON content
5. Click **Load**
6. Select Prometheus data source
7. Click **Import**

### Method 3: Manual Dashboard Creation

See [Creating Custom Dashboards](#creating-custom-dashboards) section below.

---

## Dashboard Overview

The MultiWANBond dashboard includes **10 panels**:

### Row 1: Status Overview

**1. System Uptime** (Stat)
- Shows system uptime in seconds/minutes/hours/days
- Helps track service stability

**2. WAN Status** (Stat)
- Shows UP/DOWN status for each WAN
- Green = UP, Red = DOWN
- Quick visual health check

**3. Active Flows** (Stat)
- Current number of active network flows
- Yellow threshold: 1000+ flows
- Red threshold: 5000+ flows

**4. Active Alerts** (Stat)
- Number of unresolved alerts
- Yellow: 1+ alerts
- Red: 5+ alerts

### Row 2: Latency & Packet Loss

**5. WAN Latency** (Time Series)
- Line chart showing latency per WAN over time
- Unit: milliseconds (ms)
- Lower is better

**6. WAN Packet Loss** (Time Series)
- Line chart showing packet loss per WAN
- Unit: percentage (%)
- 0% is ideal

### Row 3: Traffic

**7. Traffic Rate per WAN** (Time Series)
- Upload and download rates for each WAN
- Unit: Bytes per second (Bps)
- Legend shows mean and max

### Row 4: Distribution

**8. Upload Distribution by WAN** (Pie Chart)
- Shows how upload traffic is distributed
- Percentage and bytes per WAN

**9. Download Distribution by WAN** (Pie Chart)
- Shows how download traffic is distributed
- Should match WAN weights roughly

### Row 5: System Resources

**10. Memory Usage** (Time Series)
- Shows memory allocated vs system memory
- Helps detect memory leaks

---

## Creating Custom Dashboards

### Adding a New Panel

**1. Edit Dashboard**:
- Click **Dashboard settings** (gear icon)
- Click **Add panel**

**2. Configure Query**:
- Data source: **Prometheus**
- Metric: Select from dropdown (e.g., `multiwanbond_wan_latency_ms`)
- Legend: `{{wan_name}}`

**3. Example Queries**:

**WAN State**:
```promql
multiwanbond_wan_state
```

**Average Latency Across All WANs**:
```promql
avg(multiwanbond_wan_latency_ms)
```

**Total Upload Traffic**:
```promql
sum(rate(multiwanbond_traffic_bytes{direction="tx"}[5m]))
```

**Packet Loss Above 1%**:
```promql
multiwanbond_wan_packet_loss > 1
```

**4. Customize Visualization**:
- **Panel type**: Time series, Stat, Gauge, Bar chart, Pie chart, Table
- **Thresholds**: Set color thresholds (green/yellow/red)
- **Units**: Select appropriate unit (bytes, bps, ms, %)
- **Legend**: Show/hide, position, values

**5. Save Panel**:
- Click **Apply**
- Click **Save dashboard** (floppy disk icon)

### Useful PromQL Queries

**WAN with Highest Latency**:
```promql
topk(1, multiwanbond_wan_latency_ms)
```

**Traffic Rate (Upload) Last 5 Minutes**:
```promql
rate(multiwanbond_traffic_bytes{direction="tx"}[5m])
```

**Number of Healthy WANs**:
```promql
count(multiwanbond_wan_state == 1)
```

**Total Bandwidth (All WANs)**:
```promql
sum(rate(multiwanbond_traffic_bytes[5m]))
```

---

## Alerting

### Creating Alerts in Grafana

**1. Edit Panel**:
- Select panel you want to alert on
- Click **Alert** tab

**2. Configure Alert**:
- **Alert name**: "High WAN Latency"
- **Evaluate every**: 1m
- **For**: 5m (alert after condition persists for 5 minutes)

**3. Define Conditions**:
```promql
WHEN avg() OF query(A, 5m, now) IS ABOVE 100
```

Example alert: "Alert when average WAN latency > 100ms for 5 minutes"

**4. Add Notification**:
- **Send to**: Select notification channel
- **Message**: Custom alert message

### Example Alerts

**High Latency Alert**:
```yaml
- alert: HighWANLatency
  expr: multiwanbond_wan_latency_ms > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High latency on WAN {{ $labels.wan_name }}"
    description: "Latency is {{ $value }}ms (threshold: 100ms)"
```

**WAN Down Alert**:
```yaml
- alert: WANDown
  expr: multiwanbond_wan_state == 0
  for: 1m
  labels:
    severity: critical
  annotations:
    summary: "WAN {{ $labels.wan_name }} is down"
    description: "WAN has been down for 1 minute"
```

**High Packet Loss**:
```yaml
- alert: HighPacketLoss
  expr: multiwanbond_wan_packet_loss > 5
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High packet loss on {{ $labels.wan_name }}"
    description: "Packet loss is {{ $value }}% (threshold: 5%)"
```

### Notification Channels

**Slack**:
1. Grafana → **Alerting** → **Notification channels** → **Add channel**
2. Type: **Slack**
3. Webhook URL: `https://hooks.slack.com/services/YOUR/WEBHOOK/URL`
4. Test and save

**Email**:
1. Configure SMTP in `/etc/grafana/grafana.ini`:
```ini
[smtp]
enabled = true
host = smtp.gmail.com:587
user = your-email@gmail.com
password = your-app-password
from_address = your-email@gmail.com
```
2. Create email notification channel in Grafana

**Discord**, **PagerDuty**, **Telegram**: Similar setup via notification channels

---

## Troubleshooting

### Metrics Not Appearing

**Check MultiWANBond metrics endpoint**:
```bash
curl http://localhost:8080/metrics
```

**Expected output**:
```
# HELP multiwanbond_uptime_seconds System uptime in seconds
# TYPE multiwanbond_uptime_seconds gauge
multiwanbond_uptime_seconds 3600
...
```

**If empty or error**:
- Verify MultiWANBond is running
- Check Web UI configuration (`EnableMetrics: true`)
- Check firewall (port 8080)

### Prometheus Not Scraping

**Check Prometheus targets**:
```bash
curl http://localhost:9090/targets
```

**Or in Prometheus UI**:
- Navigate to `http://localhost:9090/targets`
- Find `multiwanbond` job
- Check **State** (should be "UP")

**If DOWN**:
- Verify `prometheus.yml` configuration
- Check network connectivity: `curl http://<multiwanbond-host>:8080/metrics`
- Check Prometheus logs: `sudo journalctl -u prometheus -f`

### Grafana Shows "No Data"

**Check data source**:
- Grafana → **Configuration** → **Data Sources** → **Prometheus**
- Verify URL: `http://localhost:9090`
- Click **Save & Test** (should show green checkmark)

**Check dashboard queries**:
- Edit panel
- Check **Query inspector** (bottom of edit page)
- Verify query returns data

**Check time range**:
- Top-right corner of dashboard
- Ensure time range includes data (e.g., "Last 6 hours")

### High Memory Usage (Prometheus)

**Reduce retention period**:
```yaml
# In prometheus.yml or startup flags
--storage.tsdb.retention.time=7d  # Keep data for 7 days
--storage.tsdb.retention.size=10GB  # Or limit by size
```

**Reduce scrape frequency**:
```yaml
scrape_interval: 30s  # Instead of 5s
```

---

## Best Practices

**1. Data Retention**:
- Production: 15-30 days
- Development: 7 days
- High-traffic: Consider remote storage (Thanos, Cortex)

**2. Scrape Intervals**:
- Critical metrics: 5-10 seconds
- Normal metrics: 15-30 seconds
- Low-priority: 1 minute

**3. Alert Fatigue**:
- Set appropriate thresholds
- Use `for:` duration to avoid flapping alerts
- Group related alerts

**4. Dashboard Organization**:
- One dashboard per service/component
- Use variables for dynamic filtering
- Share dashboards via JSON export

**5. Security**:
- Enable Grafana authentication
- Use HTTPS (reverse proxy with Let's Encrypt)
- Restrict Prometheus to internal network
- Use firewall rules

---

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [PromQL Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Grafana Alerting](https://grafana.com/docs/grafana/latest/alerting/)
- [MultiWANBond Metrics Guide](METRICS_GUIDE.md)

---

**Last Updated**: November 2, 2025
**Version**: 1.2
**MultiWANBond Version**: 1.2

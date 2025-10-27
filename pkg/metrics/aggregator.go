// Package metrics - Time-series aggregator
package metrics

import (
	"math"
	"sort"
	"time"
)

// Aggregator performs time-series aggregations
type Aggregator struct {
	// Aggregation cache
	cache map[string]map[AggregationWindow]*AggregatedData
}

// NewAggregator creates a new aggregator
func NewAggregator() *Aggregator {
	return &Aggregator{
		cache: make(map[string]map[AggregationWindow]*AggregatedData),
	}
}

// Aggregate aggregates data points over a time window
func (a *Aggregator) Aggregate(dataPoints []*DataPoint, window AggregationWindow) *AggregatedData {
	if len(dataPoints) == 0 {
		return nil
	}

	// Calculate time range
	end := time.Now()
	start := end.Add(-window.Duration())

	// Filter points within window
	filtered := make([]*DataPoint, 0)
	for _, dp := range dataPoints {
		if dp.Timestamp.After(start) && dp.Timestamp.Before(end) {
			filtered = append(filtered, dp)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Calculate statistics
	return a.calculateStats(filtered, start, end, window)
}

// AggregateTimeSeries aggregates a time series over a window
func (a *Aggregator) AggregateTimeSeries(ts *TimeSeries, window AggregationWindow) *AggregatedData {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	return a.Aggregate(ts.DataPoints, window)
}

// AggregateRange aggregates data points over a specific time range
func (a *Aggregator) AggregateRange(dataPoints []*DataPoint, start, end time.Time) *AggregatedData {
	if len(dataPoints) == 0 {
		return nil
	}

	// Filter points within range
	filtered := make([]*DataPoint, 0)
	for _, dp := range dataPoints {
		if dp.Timestamp.After(start) && dp.Timestamp.Before(end) {
			filtered = append(filtered, dp)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	// Determine window based on duration
	duration := end.Sub(start)
	window := Window1Minute
	if duration > 24*time.Hour {
		window = Window1Day
	} else if duration > 6*time.Hour {
		window = Window6Hours
	} else if duration > 1*time.Hour {
		window = Window1Hour
	} else if duration > 15*time.Minute {
		window = Window15Minutes
	} else if duration > 5*time.Minute {
		window = Window5Minutes
	}

	return a.calculateStats(filtered, start, end, window)
}

// calculateStats calculates statistical measures for data points
func (a *Aggregator) calculateStats(dataPoints []*DataPoint, start, end time.Time, window AggregationWindow) *AggregatedData {
	count := len(dataPoints)
	if count == 0 {
		return nil
	}

	// Extract values
	values := make([]float64, count)
	sum := 0.0
	min := math.MaxFloat64
	max := -math.MaxFloat64

	for i, dp := range dataPoints {
		values[i] = dp.Value
		sum += dp.Value
		if dp.Value < min {
			min = dp.Value
		}
		if dp.Value > max {
			max = dp.Value
		}
	}

	// Calculate average
	avg := sum / float64(count)

	// Calculate median
	sortedValues := make([]float64, count)
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	median := sortedValues[count/2]
	if count%2 == 0 {
		median = (sortedValues[count/2-1] + sortedValues[count/2]) / 2.0
	}

	// Calculate percentiles
	p95Index := int(float64(count) * 0.95)
	if p95Index >= count {
		p95Index = count - 1
	}
	p95 := sortedValues[p95Index]

	p99Index := int(float64(count) * 0.99)
	if p99Index >= count {
		p99Index = count - 1
	}
	p99 := sortedValues[p99Index]

	// Calculate standard deviation
	variance := 0.0
	for _, v := range values {
		diff := v - avg
		variance += diff * diff
	}
	variance /= float64(count)
	stdDev := math.Sqrt(variance)

	return &AggregatedData{
		Start:  start,
		End:    end,
		Window: window,
		Count:  count,
		Sum:    sum,
		Min:    min,
		Max:    max,
		Avg:    avg,
		Median: median,
		P95:    p95,
		P99:    p99,
		StdDev: stdDev,
	}
}

// GetMovingAverage calculates moving average over N points
func (a *Aggregator) GetMovingAverage(dataPoints []*DataPoint, n int) []float64 {
	if len(dataPoints) < n {
		return nil
	}

	result := make([]float64, 0, len(dataPoints)-n+1)
	sum := 0.0

	// Calculate first window
	for i := 0; i < n; i++ {
		sum += dataPoints[i].Value
	}
	result = append(result, sum/float64(n))

	// Sliding window
	for i := n; i < len(dataPoints); i++ {
		sum = sum - dataPoints[i-n].Value + dataPoints[i].Value
		result = append(result, sum/float64(n))
	}

	return result
}

// GetExponentialMovingAverage calculates exponential moving average
func (a *Aggregator) GetExponentialMovingAverage(dataPoints []*DataPoint, alpha float64) []float64 {
	if len(dataPoints) == 0 {
		return nil
	}

	result := make([]float64, len(dataPoints))
	result[0] = dataPoints[0].Value

	for i := 1; i < len(dataPoints); i++ {
		result[i] = alpha*dataPoints[i].Value + (1-alpha)*result[i-1]
	}

	return result
}

// DetectAnomaly detects anomalies using standard deviation
func (a *Aggregator) DetectAnomaly(dataPoints []*DataPoint, threshold float64) []*DataPoint {
	if len(dataPoints) < 2 {
		return nil
	}

	// Calculate mean and standard deviation
	sum := 0.0
	for _, dp := range dataPoints {
		sum += dp.Value
	}
	mean := sum / float64(len(dataPoints))

	variance := 0.0
	for _, dp := range dataPoints {
		diff := dp.Value - mean
		variance += diff * diff
	}
	variance /= float64(len(dataPoints))
	stdDev := math.Sqrt(variance)

	// Find anomalies (values beyond threshold * stdDev)
	anomalies := make([]*DataPoint, 0)
	for _, dp := range dataPoints {
		deviation := math.Abs(dp.Value - mean)
		if deviation > threshold*stdDev {
			anomalies = append(anomalies, dp)
		}
	}

	return anomalies
}

// GetRate calculates rate of change between consecutive points
func (a *Aggregator) GetRate(dataPoints []*DataPoint) []float64 {
	if len(dataPoints) < 2 {
		return nil
	}

	rates := make([]float64, len(dataPoints)-1)
	for i := 1; i < len(dataPoints); i++ {
		timeDiff := dataPoints[i].Timestamp.Sub(dataPoints[i-1].Timestamp).Seconds()
		if timeDiff > 0 {
			valueDiff := dataPoints[i].Value - dataPoints[i-1].Value
			rates[i-1] = valueDiff / timeDiff
		}
	}

	return rates
}

// Downsample reduces data points by sampling at intervals
func (a *Aggregator) Downsample(dataPoints []*DataPoint, interval time.Duration, aggregationType string) []*DataPoint {
	if len(dataPoints) == 0 {
		return nil
	}

	result := make([]*DataPoint, 0)
	currentBucket := make([]*DataPoint, 0)
	var bucketStart time.Time

	for _, dp := range dataPoints {
		if bucketStart.IsZero() {
			bucketStart = dp.Timestamp.Truncate(interval)
		}

		bucketEnd := bucketStart.Add(interval)

		if dp.Timestamp.Before(bucketEnd) {
			currentBucket = append(currentBucket, dp)
		} else {
			// Process current bucket
			if len(currentBucket) > 0 {
				aggregated := a.aggregateBucket(currentBucket, aggregationType)
				result = append(result, &DataPoint{
					Timestamp: bucketStart.Add(interval / 2),
					Value:     aggregated,
					Labels:    currentBucket[0].Labels,
				})
			}

			// Start new bucket
			bucketStart = dp.Timestamp.Truncate(interval)
			currentBucket = []*DataPoint{dp}
		}
	}

	// Process last bucket
	if len(currentBucket) > 0 {
		aggregated := a.aggregateBucket(currentBucket, aggregationType)
		result = append(result, &DataPoint{
			Timestamp: bucketStart.Add(interval / 2),
			Value:     aggregated,
			Labels:    currentBucket[0].Labels,
		})
	}

	return result
}

// aggregateBucket aggregates data points in a bucket
func (a *Aggregator) aggregateBucket(dataPoints []*DataPoint, aggregationType string) float64 {
	if len(dataPoints) == 0 {
		return 0
	}

	switch aggregationType {
	case "avg", "mean":
		sum := 0.0
		for _, dp := range dataPoints {
			sum += dp.Value
		}
		return sum / float64(len(dataPoints))

	case "sum":
		sum := 0.0
		for _, dp := range dataPoints {
			sum += dp.Value
		}
		return sum

	case "min":
		min := math.MaxFloat64
		for _, dp := range dataPoints {
			if dp.Value < min {
				min = dp.Value
			}
		}
		return min

	case "max":
		max := -math.MaxFloat64
		for _, dp := range dataPoints {
			if dp.Value > max {
				max = dp.Value
			}
		}
		return max

	case "median":
		values := make([]float64, len(dataPoints))
		for i, dp := range dataPoints {
			values[i] = dp.Value
		}
		sort.Float64s(values)
		if len(values)%2 == 0 {
			return (values[len(values)/2-1] + values[len(values)/2]) / 2.0
		}
		return values[len(values)/2]

	case "last":
		return dataPoints[len(dataPoints)-1].Value

	case "first":
		return dataPoints[0].Value

	default:
		// Default to average
		sum := 0.0
		for _, dp := range dataPoints {
			sum += dp.Value
		}
		return sum / float64(len(dataPoints))
	}
}

// GetTrend calculates trend direction (positive, negative, stable)
func (a *Aggregator) GetTrend(dataPoints []*DataPoint, sensitivity float64) string {
	if len(dataPoints) < 3 {
		return "stable"
	}

	// Simple linear regression
	n := float64(len(dataPoints))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, dp := range dataPoints {
		x := float64(i)
		y := dp.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	// Calculate slope
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend based on slope and sensitivity
	if math.Abs(slope) < sensitivity {
		return "stable"
	} else if slope > 0 {
		return "increasing"
	} else {
		return "decreasing"
	}
}

// CompareWindows compares metrics between two time windows
func (a *Aggregator) CompareWindows(dataPoints []*DataPoint, window1Start, window1End, window2Start, window2End time.Time) (float64, string) {
	// Get data for both windows
	window1Points := make([]*DataPoint, 0)
	window2Points := make([]*DataPoint, 0)

	for _, dp := range dataPoints {
		if dp.Timestamp.After(window1Start) && dp.Timestamp.Before(window1End) {
			window1Points = append(window1Points, dp)
		}
		if dp.Timestamp.After(window2Start) && dp.Timestamp.Before(window2End) {
			window2Points = append(window2Points, dp)
		}
	}

	if len(window1Points) == 0 || len(window2Points) == 0 {
		return 0, "insufficient_data"
	}

	// Calculate averages
	avg1 := 0.0
	for _, dp := range window1Points {
		avg1 += dp.Value
	}
	avg1 /= float64(len(window1Points))

	avg2 := 0.0
	for _, dp := range window2Points {
		avg2 += dp.Value
	}
	avg2 /= float64(len(window2Points))

	// Calculate percentage change
	if avg1 == 0 {
		return 0, "no_baseline"
	}

	percentChange := ((avg2 - avg1) / avg1) * 100.0

	status := "stable"
	if math.Abs(percentChange) > 10 {
		if percentChange > 0 {
			status = "increased"
		} else {
			status = "decreased"
		}
	}

	return percentChange, status
}

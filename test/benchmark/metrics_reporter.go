package benchmark

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

// LatencyMetrics holds detailed latency statistics
type LatencyMetrics struct {
	Min   time.Duration `json:"min"`
	Max   time.Duration `json:"max"`
	Mean  time.Duration `json:"mean"`
	P50   time.Duration `json:"p50"`
	P90   time.Duration `json:"p90"`
	P95   time.Duration `json:"p95"`
	P99   time.Duration `json:"p99"`
	P999  time.Duration `json:"p999"`
	Count int           `json:"count"`
	Total time.Duration `json:"total"`
}

// ThroughputMetrics holds throughput statistics
type ThroughputMetrics struct {
	RequestsPerSecond float64       `json:"requests_per_second"`
	TotalRequests     int           `json:"total_requests"`
	Duration          time.Duration `json:"duration"`
}

// BenchmarkReport holds comprehensive benchmark results
type BenchmarkReport struct {
	TestName    string            `json:"test_name"`
	Protocol    string            `json:"protocol"`
	Endpoint    string            `json:"endpoint"`
	Latency     LatencyMetrics    `json:"latency"`
	Throughput  ThroughputMetrics `json:"throughput"`
	SuccessRate float64           `json:"success_rate"`
	ErrorCount  int               `json:"error_count"`
	Timestamp   time.Time         `json:"timestamp"`
}

// MetricsCollector collects timing data during benchmarks
type MetricsCollector struct {
	latencies []time.Duration
	startTime time.Time
	errors    int
	total     int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies: make([]time.Duration, 0),
		startTime: time.Now(),
	}
}

// RecordLatency records a single operation latency
func (mc *MetricsCollector) RecordLatency(duration time.Duration) {
	mc.latencies = append(mc.latencies, duration)
	mc.total++
}

// RecordError records an error occurrence
func (mc *MetricsCollector) RecordError() {
	mc.errors++
	mc.total++
}

// CalculateMetrics calculates all metrics from collected data
func (mc *MetricsCollector) CalculateMetrics() (LatencyMetrics, ThroughputMetrics) {
	if len(mc.latencies) == 0 {
		return LatencyMetrics{}, ThroughputMetrics{}
	}

	// Sort latencies for percentile calculations
	sortedLatencies := make([]time.Duration, len(mc.latencies))
	copy(sortedLatencies, mc.latencies)
	sort.Slice(sortedLatencies, func(i, j int) bool {
		return sortedLatencies[i] < sortedLatencies[j]
	})

	// Calculate basic statistics
	var total time.Duration
	min := sortedLatencies[0]
	max := sortedLatencies[len(sortedLatencies)-1]

	for _, latency := range sortedLatencies {
		total += latency
	}

	mean := total / time.Duration(len(sortedLatencies))

	// Calculate percentiles
	p50 := percentile(sortedLatencies, 0.50)
	p90 := percentile(sortedLatencies, 0.90)
	p95 := percentile(sortedLatencies, 0.95)
	p99 := percentile(sortedLatencies, 0.99)
	p999 := percentile(sortedLatencies, 0.999)

	latencyMetrics := LatencyMetrics{
		Min:   min,
		Max:   max,
		Mean:  mean,
		P50:   p50,
		P90:   p90,
		P95:   p95,
		P99:   p99,
		P999:  p999,
		Count: len(sortedLatencies),
		Total: total,
	}

	// Calculate throughput
	duration := time.Since(mc.startTime)
	rps := float64(len(sortedLatencies)) / duration.Seconds()

	throughputMetrics := ThroughputMetrics{
		RequestsPerSecond: rps,
		TotalRequests:     mc.total,
		Duration:          duration,
	}

	return latencyMetrics, throughputMetrics
}

// percentile calculates the percentile value from a sorted slice
func percentile(sortedLatencies []time.Duration, p float64) time.Duration {
	if len(sortedLatencies) == 0 {
		return 0
	}
	if p <= 0 {
		return sortedLatencies[0]
	}
	if p >= 1 {
		return sortedLatencies[len(sortedLatencies)-1]
	}

	index := p * float64(len(sortedLatencies)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedLatencies[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sortedLatencies[lower] + time.Duration(weight*float64(sortedLatencies[upper]-sortedLatencies[lower]))
}

// GenerateReport creates a comprehensive benchmark report
func (mc *MetricsCollector) GenerateReport(testName, protocol, endpoint string) *BenchmarkReport {
	latency, throughput := mc.CalculateMetrics()

	successRate := 0.0
	if mc.total > 0 {
		successRate = float64(mc.total-mc.errors) / float64(mc.total) * 100
	}

	return &BenchmarkReport{
		TestName:    testName,
		Protocol:    protocol,
		Endpoint:    endpoint,
		Latency:     latency,
		Throughput:  throughput,
		SuccessRate: successRate,
		ErrorCount:  mc.errors,
		Timestamp:   time.Now(),
	}
}

// PrintReport prints a formatted benchmark report
func (r *BenchmarkReport) PrintReport() {
	fmt.Printf("\n=== %s (%s) ===\n", r.TestName, r.Protocol)
	fmt.Printf("Endpoint: %s\n", r.Endpoint)
	fmt.Printf("Success Rate: %.2f%% (%d/%d requests)\n", r.SuccessRate, r.Throughput.TotalRequests-r.ErrorCount, r.Throughput.TotalRequests)
	fmt.Printf("Duration: %v\n", r.Throughput.Duration)

	fmt.Printf("\nLatency Metrics:\n")
	fmt.Printf("  Min: %v\n", r.Latency.Min)
	fmt.Printf("  Max: %v\n", r.Latency.Max)
	fmt.Printf("  Mean: %v\n", r.Latency.Mean)
	fmt.Printf("  P50: %v\n", r.Latency.P50)
	fmt.Printf("  P90: %v\n", r.Latency.P90)
	fmt.Printf("  P95: %v\n", r.Latency.P95)
	fmt.Printf("  P99: %v\n", r.Latency.P99)
	fmt.Printf("  P99.9: %v\n", r.Latency.P999)

	fmt.Printf("\nThroughput Metrics:\n")
	fmt.Printf("  Requests/sec: %.2f\n", r.Throughput.RequestsPerSecond)
	fmt.Printf("  Total Requests: %d\n", r.Throughput.TotalRequests)

	if r.ErrorCount > 0 {
		fmt.Printf("\nErrors: %d\n", r.ErrorCount)
	}

	fmt.Printf("\nTimestamp: %s\n", r.Timestamp.Format(time.RFC3339))
	fmt.Println("=====================================")
}

// ToJSON converts the report to JSON format
func (r *BenchmarkReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CompareReports compares two benchmark reports and shows differences
func CompareReports(report1, report2 *BenchmarkReport) {
	fmt.Printf("\n=== Comparison: %s vs %s ===\n", report1.Protocol, report2.Protocol)

	fmt.Printf("Latency Comparison:\n")
	fmt.Printf("  P50: %v vs %v (%.2fx)\n",
		report1.Latency.P50, report2.Latency.P50,
		float64(report2.Latency.P50)/float64(report1.Latency.P50))
	fmt.Printf("  P99: %v vs %v (%.2fx)\n",
		report1.Latency.P99, report2.Latency.P99,
		float64(report2.Latency.P99)/float64(report1.Latency.P99))

	fmt.Printf("\nThroughput Comparison:\n")
	fmt.Printf("  RPS: %.2f vs %.2f (%.2fx)\n",
		report1.Throughput.RequestsPerSecond, report2.Throughput.RequestsPerSecond,
		report2.Throughput.RequestsPerSecond/report1.Throughput.RequestsPerSecond)

	fmt.Printf("\nSuccess Rate: %.2f%% vs %.2f%%\n",
		report1.SuccessRate, report2.SuccessRate)

	fmt.Println("=====================================")
}

// Expected performance targets based on requirements
var ExpectedTargets = map[string]map[string]time.Duration{
	"gRPC": {
		"p50": 1 * time.Millisecond,
		"p99": 5 * time.Millisecond,
	},
	"REST": {
		"p50": 5 * time.Millisecond,
		"p99": 15 * time.Millisecond,
	},
}

// CheckAgainstTargets checks if a report meets performance targets
func (r *BenchmarkReport) CheckAgainstTargets() {
	targets, exists := ExpectedTargets[r.Protocol]
	if !exists {
		fmt.Printf("No performance targets defined for %s\n", r.Protocol)
		return
	}

	fmt.Printf("\n=== Performance Target Check (%s) ===\n", r.Protocol)

	// Check P50 target
	if r.Latency.P50 <= targets["p50"] {
		fmt.Printf("✅ P50: %v (target: ≤%v)\n", r.Latency.P50, targets["p50"])
	} else {
		fmt.Printf("❌ P50: %v (target: ≤%v) - %.2fx slower\n",
			r.Latency.P50, targets["p50"],
			float64(r.Latency.P50)/float64(targets["p50"]))
	}

	// Check P99 target
	if r.Latency.P99 <= targets["p99"] {
		fmt.Printf("✅ P99: %v (target: ≤%v)\n", r.Latency.P99, targets["p99"])
	} else {
		fmt.Printf("❌ P99: %v (target: ≤%v) - %.2fx slower\n",
			r.Latency.P99, targets["p99"],
			float64(r.Latency.P99)/float64(targets["p99"]))
	}

	// Check throughput targets
	expectedThroughput := 50000.0 // gRPC target
	if r.Protocol == "REST" {
		expectedThroughput = 20000.0 // REST target
	}

	if r.Throughput.RequestsPerSecond >= expectedThroughput {
		fmt.Printf("✅ Throughput: %.2f req/s (target: ≥%.0f req/s)\n",
			r.Throughput.RequestsPerSecond, expectedThroughput)
	} else {
		fmt.Printf("❌ Throughput: %.2f req/s (target: ≥%.0f req/s) - %.2fx lower\n",
			r.Throughput.RequestsPerSecond, expectedThroughput,
			expectedThroughput/r.Throughput.RequestsPerSecond)
	}

	fmt.Println("=====================================")
}

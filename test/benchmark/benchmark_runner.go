package benchmark

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// BenchmarkConfig holds configuration for benchmark runs
type BenchmarkConfig struct {
	Duration         time.Duration `json:"duration"`
	Concurrency      int           `json:"concurrency"`
	WarmupDuration   time.Duration `json:"warmup_duration"`
	OutputFormat     string        `json:"output_format"`
	OutputFile       string        `json:"output_file"`
	EnableWarmup     bool          `json:"enable_warmup"`
	CollectMemory    bool          `json:"collect_memory"`
	EnableCPUProfile bool          `json:"enable_cpu_profile"`
}

// DefaultBenchmarkConfig returns default benchmark configuration
func DefaultBenchmarkConfig() *BenchmarkConfig {
	return &BenchmarkConfig{
		Duration:         30 * time.Second,
		Concurrency:      10,
		WarmupDuration:   5 * time.Second,
		OutputFormat:     "table",
		OutputFile:       "",
		EnableWarmup:     true,
		CollectMemory:    false,
		EnableCPUProfile: false,
	}
}

// BenchmarkRunner executes comprehensive benchmarks
type BenchmarkRunner struct {
	config *BenchmarkConfig
}

// NewBenchmarkRunner creates a new benchmark runner
func NewBenchmarkRunner(config *BenchmarkConfig) *BenchmarkRunner {
	if config == nil {
		config = DefaultBenchmarkConfig()
	}
	return &BenchmarkRunner{config: config}
}

// RunAllBenchmarks executes all benchmark tests
func (br *BenchmarkRunner) RunAllBenchmarks() ([]*BenchmarkReport, error) {
	var reports []*BenchmarkReport

	// gRPC Benchmarks
	fmt.Println("Running gRPC Benchmarks...")
	grpcReports := br.runGRPCBenchmarks()
	reports = append(reports, grpcReports...)

	// REST Benchmarks
	fmt.Println("\nRunning REST Benchmarks...")
	restReports := br.runRESTBenchmarks()
	reports = append(reports, restReports...)

	// Generate comparison report
	if len(grpcReports) > 0 && len(restReports) > 0 {
		br.generateComparisonReport(grpcReports, restReports)
	}

	// Save reports if output file specified
	if br.config.OutputFile != "" {
		br.saveReports(reports)
	}

	return reports, nil
}

// runGRPCBenchmarks executes all gRPC benchmark tests
func (br *BenchmarkRunner) runGRPCBenchmarks() []*BenchmarkReport {
	var reports []*BenchmarkReport

	benchmarks := []struct {
		name string
		test func(*testing.T, *MetricsCollector)
	}{
		{"CreateUser", br.runGRPCCreateUserBenchmark},
		{"GetUser", br.runGRPCGetUserBenchmark},
		{"UpdateUser", br.runGRPCUpdateUserBenchmark},
		{"DeleteUser", br.runGRPCDeleteUserBenchmark},
		{"ListUsers", br.runGRPCListUsersBenchmark},
		{"MixedWorkload", br.runGRPCMixedWorkloadBenchmark},
	}

	for _, benchmark := range benchmarks {
		fmt.Printf("  Running gRPC %s...\n", benchmark.name)
		report := br.runSingleBenchmark("gRPC", benchmark.name, benchmark.test)
		if report != nil {
			reports = append(reports, report)
		}
	}

	return reports
}

// runRESTBenchmarks executes all REST benchmark tests
func (br *BenchmarkRunner) runRESTBenchmarks() []*BenchmarkReport {
	var reports []*BenchmarkReport

	benchmarks := []struct {
		name string
		test func(*testing.T, *MetricsCollector)
	}{
		{"CreateUser", br.runRESTCreateUserBenchmark},
		{"GetUser", br.runRESTGetUserBenchmark},
		{"UpdateUser", br.runRESTUpdateUserBenchmark},
		{"DeleteUser", br.runRESTDeleteUserBenchmark},
		{"ListUsers", br.runRESTListUsersBenchmark},
		{"MixedWorkload", br.runRESTMixedWorkloadBenchmark},
	}

	for _, benchmark := range benchmarks {
		fmt.Printf("  Running REST %s...\n", benchmark.name)
		report := br.runSingleBenchmark("REST", benchmark.name, benchmark.test)
		if report != nil {
			reports = append(reports, report)
		}
	}

	return reports
}

// runSingleBenchmark executes a single benchmark test
func (br *BenchmarkRunner) runSingleBenchmark(protocol, testName string, testFunc func(*testing.T, *MetricsCollector)) *BenchmarkReport {
	// Create a mock testing.T for benchmark execution
	mockT := &testing.T{}

	// Initialize metrics collector
	collector := NewMetricsCollector()

	// Setup warmup if enabled
	if br.config.EnableWarmup {
		fmt.Printf("    Warming up (%v)...\n", br.config.WarmupDuration)
		br.runWarmup(testFunc, collector)
		// Reset collector after warmup
		collector = NewMetricsCollector()
	}

	fmt.Printf("    Running benchmark for %v...\n", br.config.Duration)

	// Execute the benchmark
	ctx, cancel := context.WithTimeout(context.Background(), br.config.Duration)
	defer cancel()

	var wg sync.WaitGroup
	results := make(chan *MetricsCollector, br.config.Concurrency)

	// Start concurrent workers
	for i := 0; i < br.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerCollector := NewMetricsCollector()

			for {
				select {
				case <-ctx.Done():
					results <- workerCollector
					return
				default:
					// Execute a single operation
					start := time.Now()
					testFunc(mockT, workerCollector)
					duration := time.Since(start)
					workerCollector.RecordLatency(duration)
				}
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()
	close(results)

	// Aggregate results from all workers
	for workerCollector := range results {
		collector.latencies = append(collector.latencies, workerCollector.latencies...)
		collector.errors += workerCollector.errors
		collector.total += workerCollector.total
	}

	// Generate report
	endpoint := "/v1/users"
	if testName == "ListUsers" {
		endpoint += "?page=1&limit=10"
	}

	report := collector.GenerateReport(testName, protocol, endpoint)

	// Print report and check against targets
	report.PrintReport()
	report.CheckAgainstTargets()

	return report
}

// Warmup methods
func (br *BenchmarkRunner) runWarmup(testFunc func(*testing.T, *MetricsCollector), collector *MetricsCollector) {
	ctx, cancel := context.WithTimeout(context.Background(), br.config.WarmupDuration)
	defer cancel()

	mockT := &testing.T{}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			testFunc(mockT, collector)
		}
	}
}

// gRPC benchmark implementations
// NOTE: These implementations use simulated delays for demonstration purposes.
// In actual benchmarks (grpc_benchmark_test.go and rest_benchmark_test.go),
// real gRPC/REST calls are made to measure actual performance.
func (br *BenchmarkRunner) runGRPCCreateUserBenchmark(t *testing.T, collector *MetricsCollector) {
	// This would be implemented using the actual gRPC benchmark logic
	// For now, simulate the operation
	time.Sleep(100 * time.Microsecond) // Simulate network latency
}

func (br *BenchmarkRunner) runGRPCGetUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(50 * time.Microsecond) // Simulate faster read operation
}

func (br *BenchmarkRunner) runGRPCUpdateUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(120 * time.Microsecond)
}

func (br *BenchmarkRunner) runGRPCDeleteUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(80 * time.Microsecond)
}

func (br *BenchmarkRunner) runGRPCListUsersBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(200 * time.Microsecond) // Simulate list operation
}

func (br *BenchmarkRunner) runGRPCMixedWorkloadBenchmark(t *testing.T, collector *MetricsCollector) {
	// Simulate mixed workload with varying latencies
	operations := []time.Duration{
		100 * time.Microsecond, // Create
		50 * time.Microsecond,  // Get
		120 * time.Microsecond, // Update
		200 * time.Microsecond, // List
	}

	// Cycle through operations
	opIndex := int(time.Now().UnixNano()) % len(operations)
	time.Sleep(operations[opIndex])
}

// REST benchmark implementations
func (br *BenchmarkRunner) runRESTCreateUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(300 * time.Microsecond) // REST is typically slower than gRPC
}

func (br *BenchmarkRunner) runRESTGetUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(250 * time.Microsecond)
}

func (br *BenchmarkRunner) runRESTUpdateUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(320 * time.Microsecond)
}

func (br *BenchmarkRunner) runRESTDeleteUserBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(280 * time.Microsecond)
}

func (br *BenchmarkRunner) runRESTListUsersBenchmark(t *testing.T, collector *MetricsCollector) {
	time.Sleep(400 * time.Microsecond)
}

func (br *BenchmarkRunner) runRESTMixedWorkloadBenchmark(t *testing.T, collector *MetricsCollector) {
	operations := []time.Duration{
		300 * time.Microsecond, // Create
		250 * time.Microsecond, // Get
		320 * time.Microsecond, // Update
		400 * time.Microsecond, // List
	}

	opIndex := int(time.Now().UnixNano()) % len(operations)
	time.Sleep(operations[opIndex])
}

// generateComparisonReport creates a comparison between gRPC and REST
func (br *BenchmarkRunner) generateComparisonReport(grpcReports, restReports []*BenchmarkReport) {
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("           gRPC vs REST PERFORMANCE COMPARISON")
	fmt.Println(strings.Repeat("=", 60))

	// Compare similar operations
	operations := []string{"CreateUser", "GetUser", "UpdateUser", "DeleteUser", "ListUsers", "MixedWorkload"}

	for _, op := range operations {
		var grpcReport, restReport *BenchmarkReport

		// Find corresponding reports
		for _, report := range grpcReports {
			if report.TestName == op {
				grpcReport = report
				break
			}
		}

		for _, report := range restReports {
			if report.TestName == op {
				restReport = report
				break
			}
		}

		if grpcReport != nil && restReport != nil {
			fmt.Printf("\n%s:\n", op)
			fmt.Printf("  Latency P50: gRPC %v vs REST %v (%.2fx difference)\n",
				grpcReport.Latency.P50, restReport.Latency.P50,
				float64(restReport.Latency.P50)/float64(grpcReport.Latency.P50))
			fmt.Printf("  Latency P99: gRPC %v vs REST %v (%.2fx difference)\n",
				grpcReport.Latency.P99, restReport.Latency.P99,
				float64(restReport.Latency.P99)/float64(grpcReport.Latency.P99))
			fmt.Printf("  Throughput: gRPC %.0f req/s vs REST %.0f req/s (%.2fx difference)\n",
				grpcReport.Throughput.RequestsPerSecond, restReport.Throughput.RequestsPerSecond,
				restReport.Throughput.RequestsPerSecond/grpcReport.Throughput.RequestsPerSecond)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
}

// saveReports saves benchmark reports to file
func (br *BenchmarkRunner) saveReports(reports []*BenchmarkReport) {
	var output string

	switch br.config.OutputFormat {
	case "json":
		output = "[\n"
		for i, report := range reports {
			jsonStr, err := report.ToJSON()
			if err != nil {
				fmt.Printf("Error converting report %s to JSON: %v\n", report.TestName, err)
				continue
			}
			output += jsonStr
			if i < len(reports)-1 {
				output += strings.Repeat("-", 80) + "\n"
			}
		}
		output += "\n]"
	default: // table format
		output = br.generateTableFormat(reports)
	}

	err := os.WriteFile(br.config.OutputFile, []byte(output), 0644)
	if err != nil {
		fmt.Printf("Error saving reports to file %s: %v\n", br.config.OutputFile, err)
	} else {
		fmt.Printf("\nBenchmark reports saved to: %s\n", br.config.OutputFile)
	}
}

// generateTableFormat creates a table-formatted report
func (br *BenchmarkRunner) generateTableFormat(reports []*BenchmarkReport) string {
	output := "Benchmark Results Summary\n"
	output += strings.Repeat("=", 80) + "\n"
	output += fmt.Sprintf("%-15s %-8s %-12s %-12s %-12s %-15s %-10s\n",
		"Test", "Protocol", "P50 (ms)", "P99 (ms)", "Throughput", "Success Rate", "Errors")
	output += strings.Repeat("-", 80) + "\n"

	for _, report := range reports {
		output += fmt.Sprintf("%-15s %-8s %-12.2f %-12.2f %-12.0f %-15.2f %-10d\n",
			report.TestName,
			report.Protocol,
			float64(report.Latency.P50.Nanoseconds())/1e6,
			float64(report.Latency.P99.Nanoseconds())/1e6,
			report.Throughput.RequestsPerSecond,
			report.SuccessRate,
			report.ErrorCount)
	}

	return output
}

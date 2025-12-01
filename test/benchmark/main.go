//go:build ignore
// +build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"grpc-user-service/test/benchmark"
)

func main() {
	// Parse command line flags
	duration := flag.Duration("duration", 30*time.Second, "Benchmark duration")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	warmup := flag.Duration("warmup", 5*time.Second, "Warmup duration")
	output := flag.String("output", "table", "Output format (table|json)")
	outputFile := flag.String("file", "", "Output file (optional)")
	noWarmup := flag.Bool("no-warmup", false, "Disable warmup")

	flag.Parse()

	// Create benchmark configuration
	config := &benchmark.BenchmarkConfig{
		Duration:         *duration,
		Concurrency:      *concurrency,
		WarmupDuration:   *warmup,
		OutputFormat:     *output,
		OutputFile:       *outputFile,
		EnableWarmup:     !*noWarmup,
		CollectMemory:    false,
		EnableCPUProfile: false,
	}

	// Print configuration
	fmt.Printf("Performance Benchmark Configuration:\n")
	fmt.Printf("  Duration: %v\n", config.Duration)
	fmt.Printf("  Concurrency: %d\n", config.Concurrency)
	fmt.Printf("  Warmup: %v (enabled: %t)\n", config.WarmupDuration, config.EnableWarmup)
	fmt.Printf("  Output Format: %s\n", config.OutputFormat)
	if config.OutputFile != "" {
		fmt.Printf("  Output File: %s\n", config.OutputFile)
	}
	fmt.Println()

	// Run benchmarks
	runner := benchmark.NewBenchmarkRunner(config)
	reports, err := runner.RunAllBenchmarks()
	if err != nil {
		fmt.Printf("Error running benchmarks: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nBenchmark completed successfully!\n")
	fmt.Printf("Total tests run: %d\n", len(reports))

	// Print summary
	if len(reports) > 0 {
		fmt.Println("\nSummary:")
		grpcCount := 0
		restCount := 0

		for _, report := range reports {
			switch report.Protocol {
			case "gRPC":
				grpcCount++
			case "REST":
				restCount++
			}
		}

		fmt.Printf("  gRPC tests: %d\n", grpcCount)
		fmt.Printf("  REST tests: %d\n", restCount)

		// Calculate averages
		var avgGRPCThroughput, avgRESTThroughput float64
		var grpcTests, restTests int

		for _, report := range reports {
			if report.Protocol == "gRPC" {
				avgGRPCThroughput += report.Throughput.RequestsPerSecond
				grpcTests++
			} else if report.Protocol == "REST" {
				avgRESTThroughput += report.Throughput.RequestsPerSecond
				restTests++
			}
		}

		if grpcTests > 0 {
			avgGRPCThroughput /= float64(grpcTests)
			fmt.Printf("  Average gRPC throughput: %.0f req/s\n", avgGRPCThroughput)
		}

		if restTests > 0 {
			avgRESTThroughput /= float64(restTests)
			fmt.Printf("  Average REST throughput: %.0f req/s\n", avgRESTThroughput)
		}

		if grpcTests > 0 && restTests > 0 {
			ratio := avgGRPCThroughput / avgRESTThroughput
			fmt.Printf("  gRPC/REST throughput ratio: %.2fx\n", ratio)
		}
	}
}

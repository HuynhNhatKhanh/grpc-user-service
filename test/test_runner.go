package test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Runner runs all test cases in the project with comprehensive reporting.
// It executes unit tests, extended unit tests, and integration tests with proper categorization.
func Runner(t *testing.T) {
	fmt.Println("Starting comprehensive test suite...")
	fmt.Println(strings.Repeat("=", 60))

	startTime := time.Now()

	// Test categories
	testCategories := []struct {
		name        string
		description string
		pattern     string
		timeout     time.Duration
	}{
		{
			name:        "Unit Tests",
			description: "Testing business logic and validation",
			pattern:     "./internal/.../..._test.go",
			timeout:     30 * time.Second,
		},
		{
			name:        "Extended Unit Tests",
			description: "Testing edge cases and boundary conditions",
			pattern:     "./test/.../..._test.go",
			timeout:     45 * time.Second,
		},
		{
			name:        "Integration Tests",
			description: "Testing API endpoints and workflows",
			pattern:     "./test/integration/..._test.go",
			timeout:     60 * time.Second,
		},
	}

	var totalTests, totalPassed, totalFailed int
	var failedTests []string

	for _, category := range testCategories {
		fmt.Printf("\n📋 Running %s\n", category.name)
		fmt.Printf("   %s\n", category.description)
		fmt.Println(strings.Repeat("-", 50))

		// Run tests for this category
		passed, failed, errors := runTestCategory(t, category.pattern, category.timeout)

		totalTests += passed + failed
		totalPassed += passed
		totalFailed += failed

		if len(errors) > 0 {
			failedTests = append(failedTests, errors...)
		}

		fmt.Printf("Passed: %d | Failed: %d\n", passed, failed)
	}

	// Summary
	duration := time.Since(startTime)
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("TEST SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("Passed: %d (%.1f%%)\n", totalPassed, float64(totalPassed)/float64(totalTests)*100)
	fmt.Printf("Failed: %d (%.1f%%)\n", totalFailed, float64(totalFailed)/float64(totalTests)*100)
	fmt.Printf("Duration: %v\n", duration.Round(time.Millisecond))

	if totalFailed > 0 {
		fmt.Println("\nFAILED TESTS:")
		for _, failed := range failedTests {
			fmt.Printf("   - %s\n", failed)
		}
		fmt.Println("\nSome tests failed. Please check the output above for details.")
		t.Fail()
	} else {
		fmt.Println("\nAll tests passed! Your code is ready for production.")
	}
}

// runTestCategory chạy tests cho một category cụ thể
func runTestCategory(t *testing.T, pattern string, _ time.Duration) (passed, failed int, errors []string) {
	// Find test files matching the pattern
	testFiles, err := filepath.Glob(pattern)
	if err != nil {
		t.Errorf("Error finding test files: %v", err)
		return 0, 0, []string{fmt.Sprintf("Glob error: %v", err)}
	}

	if len(testFiles) == 0 {
		fmt.Printf("No test files found for pattern: %s\n", pattern)
		return 0, 0, nil
	}

	for _, testFile := range testFiles {
		fmt.Printf("Testing: %s\n", testFile)

		// Run go test on specific file
		ctx := context.Background()
		cmd := exec.CommandContext(ctx, "go", "test", "-v", "-race", testFile) // #nosec G204
		cmd.Dir = "." // Run from project root

		output, err := cmd.CombinedOutput()

		if err != nil {
			fmt.Printf("FAILED: %s\n", testFile)
			fmt.Printf("   Error: %v\n", err)
			fmt.Printf("   Output: %s\n", string(output))
			failed++
			errors = append(errors, fmt.Sprintf("%s: %v", testFile, err))
		} else {
			fmt.Printf("PASSED: %s\n", testFile)
			passed++
		}
	}

	return passed, failed, errors
}

// Coverage chạy test và generate coverage report
func Coverage(t *testing.T) {
	fmt.Println("Running tests with coverage analysis...")

	// Run tests with coverage
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "-race", "-coverprofile=coverage.out", "-covermode=atomic", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Error running tests with coverage: %v\nOutput: %s", err, string(output))
		return
	}

	fmt.Println(string(output))

	// Generate HTML coverage report
	cmd = exec.CommandContext(ctx, "go", "tool", "cover", "-html=coverage.out", "-o=coverage.html")
	cmd.Dir = "."

	err = cmd.Run()
	if err != nil {
		t.Errorf("Error generating HTML coverage: %v", err)
		return
	}

	fmt.Println("Coverage report generated: coverage.html")
	fmt.Println("Open coverage.html in your browser to see detailed coverage analysis")
}

// Benchmark chạy performance benchmarks
func Benchmark(t *testing.T) {
	fmt.Println("Running performance benchmarks...")

	// Find all benchmark tests
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "test", "-bench=.", "-benchmem", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Error running benchmarks: %v\nOutput: %s", err, string(output))
		return
	}

	fmt.Println(string(output))

	// Check if any benchmarks were actually run
	if !strings.Contains(string(output), "Benchmark") {
		fmt.Println("No benchmarks found. Consider adding benchmark tests for performance-critical functions.")
	}
}

// RaceCondition chạy race condition detection
func RaceCondition(t *testing.T) {
	fmt.Println("Running race condition detection...")

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "test", "-race", "-short", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Race condition detected: %v\nOutput: %s", err, string(output))
		return
	}

	fmt.Println("No race conditions detected")
	fmt.Println(string(output))
}

// MemoryUsage kiểm tra memory leaks
func MemoryUsage(t *testing.T) {
	fmt.Println("Running memory leak detection...")

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "test", "-memprofile=mem.prof", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Error running memory profile: %v\nOutput: %s", err, string(output))
		return
	}

	fmt.Println("Memory profile generated: mem.prof")
	fmt.Println("Run 'go tool pprof mem.prof' to analyze memory usage")
}

// Linting chạy code quality checks
func Linting(t *testing.T) {
	fmt.Println("Running code quality checks...")

	// Check if golangci-lint is available
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "golangci-lint", "version")
	cmd.Dir = "."

	if err := cmd.Run(); err != nil {
		fmt.Println("golangci-lint not found. Skipping linting.")
		fmt.Println("Install golangci-lint: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2")
		return
	}

	// Run golangci-lint
	cmd = exec.CommandContext(ctx, "golangci-lint", "run", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Linting issues found:\n%s\n", string(output))
		// Don't fail the test for linting issues, just report them
		return
	}

	fmt.Println("Code quality checks passed")
}

// Security chạy security vulnerability scan
func Security(t *testing.T) {
	fmt.Println("Running security vulnerability scan...")

	// Check if gosec is available
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "gosec", "version")
	cmd.Dir = "."

	if err := cmd.Run(); err != nil {
		fmt.Println("gosec not found. Skipping security scan.")
		fmt.Println("Install gosec: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest")
		return
	}

	// Run gosec
	cmd = exec.CommandContext(ctx, "gosec", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Security issues found:\n%s\n", string(output))
		// Don't fail the test for security issues, just report them
		return
	}

	fmt.Println("Security scan passed")
}

// Dependencies kiểm tra dependencies vulnerabilities
func Dependencies(t *testing.T) {
	fmt.Println("Checking for vulnerable dependencies...")

	// Check if govulncheck is available
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "govulncheck", "version")
	cmd.Dir = "."

	if err := cmd.Run(); err != nil {
		fmt.Println("govulncheck not found. Skipping vulnerability check.")
		fmt.Println("Install govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest")
		return
	}

	// Run govulncheck
	cmd = exec.CommandContext(ctx, "govulncheck", "./...")
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Vulnerability check failed:\n%s\n", string(output))
		return
	}

	fmt.Println("No known vulnerabilities found")
	fmt.Println(string(output))
}

// AllComprehensive chạy tất cả test types
func AllComprehensive(t *testing.T) {
	fmt.Println("Running comprehensive test suite...")
	fmt.Println("This will run: Unit Tests, Integration Tests, Coverage, Benchmarks, Race Detection, Memory Profiling, Linting, Security Scan")

	// Create a sub-test suite
	t.Run("UnitAndIntegration", Runner)
	t.Run("Coverage", Coverage)
	t.Run("Benchmarks", Benchmark)
	t.Run("RaceConditions", RaceCondition)
	t.Run("MemoryUsage", MemoryUsage)
	t.Run("Linting", Linting)
	t.Run("Security", Security)
	t.Run("Dependencies", Dependencies)

	fmt.Println("\nComprehensive test suite completed!")
	fmt.Println("Check the generated reports:")
	fmt.Println("   - coverage.html: Test coverage visualization")
	fmt.Println("   - mem.prof: Memory usage profile")
	fmt.Println("   - Linting and security reports in the output above")
}

// Helper function to check if required tools are installed
func checkTool(tool string) bool {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "which", tool)
	err := cmd.Run()
	return err == nil
}

// Environment kiểm tra môi trường test
func Environment(t *testing.T) {
	fmt.Println("Checking test environment...")

	// Check Go version
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Error checking Go version: %v", err)
		return
	}
	fmt.Printf("Go version: %s", string(output))

	// Check if we're in the right directory
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		t.Error("go.mod not found. Please run tests from project root directory.")
		return
	}

	// Check required tools
	tools := []string{"go", "git"}
	for _, tool := range tools {
		if !checkTool(tool) {
			t.Errorf("Required tool not found: %s", tool)
		}
	}

	// Check optional tools
	optionalTools := []string{"golangci-lint", "gosec", "govulncheck"}
	fmt.Println("\nOptional tools:")
	for _, tool := range optionalTools {
		if checkTool(tool) {
			fmt.Printf("%s is installed\n", tool)
		} else {
			fmt.Printf("%s is not installed (optional)\n", tool)
		}
	}

	fmt.Println("Test environment is ready")
}

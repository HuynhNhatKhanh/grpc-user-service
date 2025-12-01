package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	ginhandler "grpc-user-service/internal/adapter/gin/handler"
	ginrouter "grpc-user-service/internal/adapter/gin/router"
	"grpc-user-service/internal/usecase/user"
	redisclient "grpc-user-service/pkg/redis"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap/zaptest"
)

// Gin Benchmark Server setup
type GinBenchmarkServer struct {
	httpServer  *http.Server
	httpClient  *http.Client
	baseURL     string
	redisClient *redis.Client
}

// Global counter to ensure unique ports for Gin benchmarks
var ginPortCounter int64 = 30000

func setupGinBenchmarkServer(b *testing.B) *GinBenchmarkServer {
	logger := zaptest.NewLogger(b)
	mockRepo := NewMockRepository()
	userUsecase := user.New(mockRepo, nil, logger)

	// Setup Redis client (mock for benchmarking)
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15, // Use separate DB for benchmarks
	})

	redisClientWrapper := &redisclient.Client{
		Client: rdb,
	}

	// Create Gin handler
	ginHandler := ginhandler.NewUserHandler(userUsecase, logger)

	// Setup Gin router
	router := ginrouter.SetupRouter(ginHandler, nil, redisClientWrapper, logger)

	// Get unique port using atomic counter
	port := atomic.AddInt64(&ginPortCounter, 1)
	if port > 35000 {
		port = atomic.AddInt64(&ginPortCounter, -5000) // Reset if too high
	}

	// Start HTTP server
	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.Logf("Gin server error: %v", err)
		}
	}()

	// Setup HTTP client first
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Wait for server to start with extra time
	time.Sleep(1000 * time.Millisecond)

	return &GinBenchmarkServer{
		httpServer:  httpServer,
		httpClient:  httpClient,
		baseURL:     fmt.Sprintf("http://localhost:%d", port),
		redisClient: rdb,
	}
}

func (gs *GinBenchmarkServer) Close() {
	if gs.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		gs.httpServer.Shutdown(ctx)
		cancel()
	}
	if gs.redisClient != nil {
		gs.redisClient.Close()
	}
}

// Helper method to make HTTP requests
func (gs *GinBenchmarkServer) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, gs.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return gs.httpClient.Do(req)
}

// Gin Benchmark Tests

func BenchmarkGin_CreateUser(b *testing.B) {
	gs := setupGinBenchmarkServer(b)
	defer gs.Close()

	var counter int64
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			id := atomic.AddInt64(&counter, 1)
			requestBody := map[string]interface{}{
				"name":  fmt.Sprintf("User_%d", id),
				"email": fmt.Sprintf("user_%d@example.com", id),
			}

			resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkGin_GetUser(b *testing.B) {
	gs := setupGinBenchmarkServer(b)
	defer gs.Close()

	// Pre-create a user for testing
	requestBody := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
	}
	resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	defer resp.Body.Close()

	var createResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		b.Fatalf("Failed to decode create response: %v", err)
	}
	userID := fmt.Sprintf("%.0f", createResp["id"].(float64))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			resp, err := gs.makeRequest("GET", "/v1/users/"+userID, nil)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkGin_UpdateUser(b *testing.B) {
	gs := setupGinBenchmarkServer(b)
	defer gs.Close()

	// Pre-create a user for testing
	requestBody := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
	}
	resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	defer resp.Body.Close()

	var createResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		b.Fatalf("Failed to decode create response: %v", err)
	}
	userID := fmt.Sprintf("%.0f", createResp["id"].(float64))

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			requestBody := map[string]interface{}{
				"name":  fmt.Sprintf("Updated_%d", time.Now().UnixNano()),
				"email": fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
			}

			resp, err := gs.makeRequest("PUT", "/v1/users/"+userID, requestBody)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkGin_DeleteUser(b *testing.B) {
	gs := setupGinBenchmarkServer(b)
	defer gs.Close()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			// Create user first
			requestBody := map[string]interface{}{
				"name":  fmt.Sprintf("User_%d", time.Now().UnixNano()),
				"email": fmt.Sprintf("user_%d@example.com", time.Now().UnixNano()),
			}

			resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
			if err != nil {
				b.Errorf("Create request failed: %v", err)
				continue
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				resp.Body.Close()
				b.Errorf("Create request failed with status: %d", resp.StatusCode)
				continue
			}

			var createResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
				resp.Body.Close()
				b.Errorf("Failed to decode create response: %v", err)
				continue
			}
			resp.Body.Close()

			idVal, ok := createResp["id"].(float64)
			if !ok {
				b.Errorf("Response does not contain valid id: %v", createResp)
				continue
			}
			userID := fmt.Sprintf("%.0f", idVal)

			// Delete the user
			resp, err = gs.makeRequest("DELETE", "/v1/users/"+userID, nil)
			if err != nil {
				b.Errorf("Delete request failed: %v", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkGin_ListUsers(b *testing.B) {
	gs := setupGinBenchmarkServer(b)
	defer gs.Close()

	// Pre-create some users
	for i := 0; i < 50; i++ {
		requestBody := map[string]interface{}{
			"name":  fmt.Sprintf("User_%d", i),
			"email": fmt.Sprintf("user_%d@example.com", i),
		}
		resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
		if err != nil {
			b.Fatalf("Failed to create test user %d: %v", i, err)
		}
		resp.Body.Close()
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			resp, err := gs.makeRequest("GET", "/v1/users?page=1&limit=10", nil)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

// Mixed workload benchmark for Gin
func BenchmarkGin_MixedWorkload(b *testing.B) {
	gs := setupGinBenchmarkServer(b)
	defer gs.Close()

	// Pre-create some users for read operations
	var userIDs []string
	for i := 0; i < 10; i++ {
		requestBody := map[string]interface{}{
			"name":  fmt.Sprintf("User_%d", i),
			"email": fmt.Sprintf("user_%d@example.com", i),
		}
		resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
		if err != nil {
			b.Fatalf("Failed to create test user %d: %v", i, err)
		}

		var createResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
			resp.Body.Close()
			b.Fatalf("Failed to decode create response: %v", err)
		}
		resp.Body.Close()

		userIDs = append(userIDs, fmt.Sprintf("%.0f", createResp["id"].(float64)))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		i := 0
		for p.Next() {
			switch i % 4 {
			case 0: // Create
				requestBody := map[string]interface{}{
					"name":  fmt.Sprintf("MixedUser_%d", time.Now().UnixNano()),
					"email": fmt.Sprintf("mixed_%d@example.com", time.Now().UnixNano()),
				}
				resp, err := gs.makeRequest("POST", "/v1/users", requestBody)
				if err == nil {
					resp.Body.Close()
				}

			case 1: // Get
				if len(userIDs) > 0 {
					userID := userIDs[i%len(userIDs)]
					resp, err := gs.makeRequest("GET", "/v1/users/"+userID, nil)
					if err == nil {
						resp.Body.Close()
					}
				}

			case 2: // Update
				if len(userIDs) > 0 {
					userID := userIDs[i%len(userIDs)]
					requestBody := map[string]interface{}{
						"name":  fmt.Sprintf("Updated_%d", time.Now().UnixNano()),
						"email": fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
					}
					resp, err := gs.makeRequest("PUT", "/v1/users/"+userID, requestBody)
					if err == nil {
						resp.Body.Close()
					}
				}

			case 3: // List
				resp, err := gs.makeRequest("GET", "/v1/users?page=1&limit=10", nil)
				if err == nil {
					resp.Body.Close()
				}
			}

			i++
		}
	})
}

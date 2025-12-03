package benchmark

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	pb "grpc-user-service/api/gen/go/user"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/usecase/user"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// REST Benchmark Server setup
type RESTBenchmarkServer struct {
	httpServer *http.Server
	grpcServer *grpc.Server
	httpClient *http.Client
	baseURL    string
	listener   net.Listener
	conn       *grpc.ClientConn
}

// Global counter to ensure unique ports for REST benchmarks
var restPortCounter int64 = 40000

func setupRESTBenchmarkServer(b *testing.B) *RESTBenchmarkServer {
	logger := zaptest.NewLogger(b)
	mockRepo := NewMockRepository()
	userUsecase := user.New(mockRepo, logger)

	// Get unique port using atomic counter
	grpcPort := atomic.AddInt64(&restPortCounter, 1)
	if grpcPort > 45000 {
		grpcPort = atomic.AddInt64(&restPortCounter, -5000) // Reset if too high
	}
	httpPort := grpcPort + 1000
	if httpPort > 65535 {
		httpPort = grpcPort + 500 // Ensure valid HTTP port
	}

	// Start gRPC server
	grpcServer := grpc.NewServer()
	pb.RegisterUserServiceServer(grpcServer, grpcadapter.NewUserServiceServer(userUsecase, logger))

	//nolint:noctx // Benchmark test server setup requires net.Listen
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", grpcPort))
	if err != nil {
		b.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			b.Logf("gRPC server error: %v", err)
		}
	}()
	mux := runtime.NewServeMux()
	err = pb.RegisterUserServiceHandlerFromEndpoint(
		context.Background(),
		mux,
		fmt.Sprintf("localhost:%d", grpcPort),
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
	)
	if err != nil {
		b.Fatalf("Failed to register gateway: %v", err)
	}

	// Start HTTP server
	httpServer := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Addr:              fmt.Sprintf(":%d", httpPort),
		Handler:           mux,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.Logf("HTTP server error: %v", err)
		}
	}()

	// Wait for servers to start
	time.Sleep(1000 * time.Millisecond)

	// Setup HTTP client
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create gRPC connection for gateway with retries
	var conn *grpc.ClientConn
	var connErr error
	for i := 0; i < 5; i++ {
		conn, connErr = grpc.NewClient(
			fmt.Sprintf("localhost:%d", grpcPort),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if connErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if connErr != nil {
		b.Fatalf("Failed to connect to gRPC server after retries: %v", connErr)
	}

	return &RESTBenchmarkServer{
		httpServer: httpServer,
		grpcServer: grpcServer,
		httpClient: httpClient,
		baseURL:    fmt.Sprintf("http://localhost:%d", httpPort),
		listener:   listener,
		conn:       conn,
	}
}

func (rs *RESTBenchmarkServer) Close() {
	if rs.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = rs.httpServer.Shutdown(ctx)
		cancel()
	}
	if rs.conn != nil {
		_ = rs.conn.Close()
	}
	if rs.grpcServer != nil {
		rs.grpcServer.GracefulStop()
	}
	if rs.listener != nil {
		_ = rs.listener.Close()
	}
}

// Helper method to make HTTP requests
func (rs *RESTBenchmarkServer) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
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

	req, err := http.NewRequestWithContext(context.Background(), method, rs.baseURL+endpoint, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return rs.httpClient.Do(req)
}

// REST Benchmark Tests

func BenchmarkREST_CreateUser(b *testing.B) {
	rs := setupRESTBenchmarkServer(b)
	defer rs.Close()

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

			resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkREST_GetUser(b *testing.B) {
	rs := setupRESTBenchmarkServer(b)
	defer rs.Close()

	// Pre-create a user for testing
	requestBody := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
	}
	resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var createResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		b.Fatalf("Failed to decode create response: %v", err)
	}
	userID, ok := createResp["id"].(string)
	if !ok {
		b.Fatalf("Response does not contain valid id: %v", createResp)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			resp, err := rs.makeRequest("GET", "/v1/users/"+userID, nil)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkREST_UpdateUser(b *testing.B) {
	rs := setupRESTBenchmarkServer(b)
	defer rs.Close()

	// Pre-create a user for testing
	requestBody := map[string]interface{}{
		"name":  "Test User",
		"email": "test@example.com",
	}
	resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var createResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		b.Fatalf("Failed to decode create response: %v", err)
	}
	userID, ok := createResp["id"].(string)
	if !ok {
		b.Fatalf("Response does not contain valid id: %v", createResp)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			requestBody := map[string]interface{}{
				"id":    userID,
				"name":  fmt.Sprintf("Updated_%d", time.Now().UnixNano()),
				"email": fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
			}

			resp, err := rs.makeRequest("PUT", "/v1/users/"+userID, requestBody)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

func BenchmarkREST_DeleteUser(b *testing.B) {
	rs := setupRESTBenchmarkServer(b)
	defer rs.Close()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			// Create user first
			requestBody := map[string]interface{}{
				"name":  fmt.Sprintf("User_%d", time.Now().UnixNano()),
				"email": fmt.Sprintf("user_%d@example.com", time.Now().UnixNano()),
			}

			resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
			if err != nil {
				b.Errorf("Create request failed: %v", err)
				continue
			}

			var createResp map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
				_ = resp.Body.Close()
				b.Errorf("Failed to decode create response: %v", err)
				continue
			}
			_ = resp.Body.Close()

			userID, ok := createResp["id"].(string)
			if !ok {
				b.Errorf("Response does not contain valid id: %v", createResp)
				continue
			}

			// Delete the user
			resp, err = rs.makeRequest("DELETE", "/v1/users/"+userID, nil)
			if err != nil {
				b.Errorf("Delete request failed: %v", err)
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

//nolint:dupl // Benchmark test duplication is acceptable
func BenchmarkREST_ListUsers(b *testing.B) {
	rs := setupRESTBenchmarkServer(b)
	defer rs.Close()

	// Pre-create some users
	for i := 0; i < 50; i++ {
		requestBody := map[string]interface{}{
			"name":  fmt.Sprintf("User_%d", i),
			"email": fmt.Sprintf("user_%d@example.com", i),
		}
		resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
		if err != nil {
			b.Fatalf("Failed to create test user %d: %v", i, err)
		}
		_ = resp.Body.Close()
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			resp, err := rs.makeRequest("GET", "/v1/users?page=1&limit=10", nil)
			if err != nil {
				b.Errorf("Request failed: %v", err)
				continue
			}
			_ = resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				b.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}
	})
}

// Mixed workload benchmark for REST
//
//nolint:gocyclo // Benchmark test complexity is acceptable
func BenchmarkREST_MixedWorkload(b *testing.B) {
	rs := setupRESTBenchmarkServer(b)
	defer rs.Close()

	// Pre-create some users for read operations
	var userIDs []string
	for i := 0; i < 10; i++ {
		requestBody := map[string]interface{}{
			"name":  fmt.Sprintf("User_%d", i),
			"email": fmt.Sprintf("user_%d@example.com", i),
		}
		resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
		if err != nil {
			b.Fatalf("Failed to create test user %d: %v", i, err)
		}

		var createResp map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
			_ = resp.Body.Close()
			b.Fatalf("Failed to decode create response: %v", err)
		}
		_ = resp.Body.Close()

		userID, ok := createResp["id"].(string)
		if !ok {
			b.Fatalf("Response does not contain valid id: %v", createResp)
		}
		userIDs = append(userIDs, userID)
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
				resp, err := rs.makeRequest("POST", "/v1/users", requestBody)
				if err == nil {
					_ = resp.Body.Close()
				}

			case 1: // Get
				if len(userIDs) > 0 {
					userID := userIDs[i%len(userIDs)]
					resp, err := rs.makeRequest("GET", "/v1/users/"+userID, nil)
					if err == nil {
						_ = resp.Body.Close()
					}
				}

			case 2: // Update
				if len(userIDs) > 0 {
					userID := userIDs[i%len(userIDs)]
					requestBody := map[string]interface{}{
						"id":    userID,
						"name":  fmt.Sprintf("Updated_%d", time.Now().UnixNano()),
						"email": fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
					}
					resp, err := rs.makeRequest("PUT", "/v1/users/"+userID, requestBody)
					if err == nil {
						_ = resp.Body.Close()
					}
				}

			case 3: // List
				resp, err := rs.makeRequest("GET", "/v1/users?page=1&limit=10", nil)
				if err == nil {
					_ = resp.Body.Close()
				}
			}

			i++
		}
	})
}

package benchmark

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pb "grpc-user-service/api/gen/go/user"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/usecase/user"

	grpcdomain "grpc-user-service/internal/domain/user"

	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MockRepository for benchmarking
type MockRepository struct {
	users  map[int64]*grpcdomain.User
	nextID int64
	mu     sync.RWMutex
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		users:  make(map[int64]*grpcdomain.User),
		nextID: 1,
	}
}

func (m *MockRepository) Create(ctx context.Context, u *grpcdomain.User) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.nextID++
	u.ID = id
	m.users[id] = u
	return id, nil
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (*grpcdomain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if user, exists := m.users[id]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*grpcdomain.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, nil
}

func (m *MockRepository) Update(ctx context.Context, u *grpcdomain.User) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[u.ID]; exists {
		m.users[u.ID] = u
		return u.ID, nil
	}
	return 0, fmt.Errorf("user not found")
}

func (m *MockRepository) Delete(ctx context.Context, id int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[id]; exists {
		delete(m.users, id)
		return id, nil
	}
	return 0, fmt.Errorf("user not found")
}

func (m *MockRepository) List(ctx context.Context, query string, page, limit int64) ([]grpcdomain.User, int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var users []grpcdomain.User
	for _, user := range m.users {
		users = append(users, *user)
	}

	total := int64(len(users))

	// Simple pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= total {
		return []grpcdomain.User{}, total, nil
	}
	if end > total {
		end = total
	}

	return users[start:end], total, nil
}

// Benchmark setup
type BenchmarkServer struct {
	server   *grpc.Server
	listener net.Listener
	client   pb.UserServiceClient
	conn     *grpc.ClientConn
}

// Global counter to ensure unique ports
var grpcPortCounter int64 = 50000

func setupBenchmarkServer(b *testing.B) *BenchmarkServer {
	logger := zaptest.NewLogger(b)
	mockRepo := NewMockRepository()
	userUsecase := user.New(mockRepo, nil, logger)

	// Get unique port using atomic counter
	port := atomic.AddInt64(&grpcPortCounter, 1)
	if port > 60000 {
		port = atomic.AddInt64(&grpcPortCounter, -10000) // Reset if too high
	}

	// Start gRPC server
	server := grpc.NewServer()
	pb.RegisterUserServiceServer(server, grpcadapter.NewUserServiceServer(userUsecase, logger))

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		b.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		if err := server.Serve(listener); err != nil {
			b.Logf("gRPC server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	// Create client connection with retries
	var conn *grpc.ClientConn
	var connErr error
	for i := 0; i < 5; i++ {
		conn, connErr = grpc.NewClient(
			fmt.Sprintf("127.0.0.1:%d", port),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if connErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if connErr != nil {
		b.Fatalf("Failed to connect after retries: %v", connErr)
	}

	client := pb.NewUserServiceClient(conn)

	return &BenchmarkServer{
		server:   server,
		listener: listener,
		client:   client,
		conn:     conn,
	}
}

func (bs *BenchmarkServer) Close() {
	if bs.conn != nil {
		bs.conn.Close()
	}
	if bs.server != nil {
		bs.server.GracefulStop()
	}
	if bs.listener != nil {
		bs.listener.Close()
	}
}

// gRPC Benchmark Tests

func BenchmarkGRPC_CreateUser(b *testing.B) {
	bs := setupBenchmarkServer(b)
	defer bs.Close()

	var counter int64
	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			ctx := context.Background()
			id := atomic.AddInt64(&counter, 1)
			req := &pb.CreateUserRequest{
				Name:  fmt.Sprintf("User_%d", id),
				Email: fmt.Sprintf("user_%d@example.com", id),
			}

			_, err := bs.client.CreateUser(ctx, req)
			if err != nil {
				b.Errorf("CreateUser failed: %v", err)
			}
		}
	})
}

func BenchmarkGRPC_GetUser(b *testing.B) {
	bs := setupBenchmarkServer(b)
	defer bs.Close()

	// Pre-create a user for testing
	ctx := context.Background()
	createReq := &pb.CreateUserRequest{
		Name:  "Test User",
		Email: "test@example.com",
	}
	resp, err := bs.client.CreateUser(ctx, createReq)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	userID := resp.Id

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			ctx := context.Background()
			req := &pb.GetUserRequest{Id: userID}

			_, err := bs.client.GetUser(ctx, req)
			if err != nil {
				b.Errorf("GetUser failed: %v", err)
			}
		}
	})
}

func BenchmarkGRPC_UpdateUser(b *testing.B) {
	bs := setupBenchmarkServer(b)
	defer bs.Close()

	// Pre-create a user for testing
	ctx := context.Background()
	createReq := &pb.CreateUserRequest{
		Name:  "Test User",
		Email: "test@example.com",
	}
	resp, err := bs.client.CreateUser(ctx, createReq)
	if err != nil {
		b.Fatalf("Failed to create test user: %v", err)
	}
	userID := resp.Id

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			ctx := context.Background()
			req := &pb.UpdateUserRequest{
				Id:    userID,
				Name:  fmt.Sprintf("Updated_%d", time.Now().UnixNano()),
				Email: fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
			}

			_, err := bs.client.UpdateUser(ctx, req)
			if err != nil {
				b.Errorf("UpdateUser failed: %v", err)
			}
		}
	})
}

func BenchmarkGRPC_DeleteUser(b *testing.B) {
	bs := setupBenchmarkServer(b)
	defer bs.Close()

	b.ResetTimer()
	b.ReportAllocs()

	// Create and delete users in the benchmark
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			// Create user first
			ctx := context.Background()
			createReq := &pb.CreateUserRequest{
				Name:  fmt.Sprintf("User_%d", time.Now().UnixNano()),
				Email: fmt.Sprintf("user_%d@example.com", time.Now().UnixNano()),
			}

			resp, err := bs.client.CreateUser(ctx, createReq)
			if err != nil {
				b.Errorf("CreateUser failed: %v", err)
				continue
			}

			// Delete the user
			deleteReq := &pb.DeleteUserRequest{Id: resp.Id}
			_, err = bs.client.DeleteUser(ctx, deleteReq)
			if err != nil {
				b.Errorf("DeleteUser failed: %v", err)
			}
		}
	})
}

func BenchmarkGRPC_ListUsers(b *testing.B) {
	bs := setupBenchmarkServer(b)
	defer bs.Close()

	// Pre-create some users
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		req := &pb.CreateUserRequest{
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user_%d@example.com", i),
		}
		bs.client.CreateUser(ctx, req)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			ctx := context.Background()
			req := &pb.ListUsersRequest{
				Page:  1,
				Limit: 10,
			}

			_, err := bs.client.ListUsers(ctx, req)
			if err != nil {
				b.Errorf("ListUsers failed: %v", err)
			}
		}
	})
}

// Mixed workload benchmark
func BenchmarkGRPC_MixedWorkload(b *testing.B) {
	bs := setupBenchmarkServer(b)
	defer bs.Close()

	// Pre-create some users for read operations
	ctx := context.Background()
	var userIDs []int64
	for i := 0; i < 10; i++ {
		req := &pb.CreateUserRequest{
			Name:  fmt.Sprintf("User_%d", i),
			Email: fmt.Sprintf("user_%d@example.com", i),
		}
		resp, err := bs.client.CreateUser(ctx, req)
		if err != nil {
			b.Fatalf("Failed to create test user: %v", err)
		}
		userIDs = append(userIDs, resp.Id)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(p *testing.PB) {
		i := 0
		for p.Next() {
			ctx := context.Background()

			switch i % 4 {
			case 0: // Create
				req := &pb.CreateUserRequest{
					Name:  fmt.Sprintf("MixedUser_%d", time.Now().UnixNano()),
					Email: fmt.Sprintf("mixed_%d@example.com", time.Now().UnixNano()),
				}
				bs.client.CreateUser(ctx, req)

			case 1: // Get
				if len(userIDs) > 0 {
					userID := userIDs[i%len(userIDs)]
					req := &pb.GetUserRequest{Id: userID}
					bs.client.GetUser(ctx, req)
				}

			case 2: // Update
				if len(userIDs) > 0 {
					userID := userIDs[i%len(userIDs)]
					req := &pb.UpdateUserRequest{
						Id:    userID,
						Name:  fmt.Sprintf("Updated_%d", time.Now().UnixNano()),
						Email: fmt.Sprintf("updated_%d@example.com", time.Now().UnixNano()),
					}
					bs.client.UpdateUser(ctx, req)
				}

			case 3: // List
				req := &pb.ListUsersRequest{
					Page:  1,
					Limit: 10,
				}
				bs.client.ListUsers(ctx, req)
			}

			i++
		}
	})
}

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	pb "grpc-user-service/api/gen/go/user"
	grpcadapter "grpc-user-service/internal/adapter/grpc"
	"grpc-user-service/internal/usecase/user"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	grpcdomain "grpc-user-service/internal/domain/user"
)

// MockRepository is a mock implementation of the Repository interface for integration testing.
// It uses testify/mock to simulate database operations during API testing.
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, u *grpcdomain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (*grpcdomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcdomain.User), args.Error(1)
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*grpcdomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcdomain.User), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, u *grpcdomain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id int64) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, query string, page, limit int64) ([]grpcdomain.User, int64, error) {
	args := m.Called(ctx, query, page, limit)
	return args.Get(0).([]grpcdomain.User), args.Get(1).(int64), args.Error(2)
}

// UserAPIIntegrationTestSuite tests the HTTP API through grpc-gateway
type UserAPIIntegrationTestSuite struct {
	suite.Suite
	httpClient  *http.Client
	baseURL     string
	mockRepo    *MockRepository
	userUsecase user.UserUsecase
}

// SetupSuite starts the actual gRPC server and HTTP gateway for testing
func (suite *UserAPIIntegrationTestSuite) SetupSuite() {
	// Setup mock repository and usecase
	suite.mockRepo = new(MockRepository)
	logger := zaptest.NewLogger(suite.T())
	suite.userUsecase = user.New(suite.mockRepo, logger)

	// Start gRPC server in a goroutine
	go func() {
		grpcServer := grpc.NewServer()
		pb.RegisterUserServiceServer(grpcServer, grpcadapter.NewUserServiceServer(suite.userUsecase, logger))

		lc := net.ListenConfig{}
		lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0") // Use random port
		suite.Require().NoError(err)

		// Store the actual port for the gateway
		port := lis.Addr().(*net.TCPAddr).Port
		suite.baseURL = fmt.Sprintf("http://localhost:%d", port+1000) // HTTP port will be gRPC port + 1000

		go func() {
			if err := grpcServer.Serve(lis); err != nil {
				suite.T().Logf("gRPC server error: %v", err)
			}
		}()

		// Setup HTTP gateway
		mux := runtime.NewServeMux()
		err = pb.RegisterUserServiceHandlerFromEndpoint(
			context.Background(),
			mux,
			fmt.Sprintf("localhost:%d", port),
			[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
		)
		suite.Require().NoError(err)

		// Start HTTP server
		httpServer := &http.Server{
			ReadHeaderTimeout: 10 * time.Second,
			Addr:              fmt.Sprintf(":%d", port+1000),
			Handler:           mux,
		}

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			suite.T().Logf("HTTP server error: %v", err)
		}
	}()

	// Wait for servers to start
	time.Sleep(2 * time.Second)

	// Setup HTTP client
	suite.httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (suite *UserAPIIntegrationTestSuite) SetupTest() {
	suite.mockRepo.ExpectedCalls = nil
	suite.mockRepo.Calls = nil
}

// TearDownSuite cleans up test resources
func (suite *UserAPIIntegrationTestSuite) TearDownSuite() {
	// Cleanup if needed
}

// Helper method to make HTTP requests
func (suite *UserAPIIntegrationTestSuite) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBody, err := json.Marshal(body)
		suite.Require().NoError(err)
		reqBody = bytes.NewBuffer(jsonBody)
	} else {
		reqBody = bytes.NewBuffer(nil) // Create empty buffer for GET/DELETE requests
	}

	req, err := http.NewRequestWithContext(context.Background(), method, suite.baseURL+endpoint, reqBody)
	suite.Require().NoError(err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return suite.httpClient.Do(req)
}

// Test CreateUser API
func (suite *UserAPIIntegrationTestSuite) TestCreateUserAPI() {
	// Mock repository calls
	suite.mockRepo.On("GetByEmail", mock.Anything, "john@example.com").Return(nil, nil)
	suite.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(int64(1), nil)

	// Request payload
	requestBody := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}

	// Make HTTP request
	resp, err := suite.makeRequest("POST", "/v1/users", requestBody)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "1", response["id"])
	suite.mockRepo.AssertExpectations(suite.T())
}

// Test GetUser API
func (suite *UserAPIIntegrationTestSuite) TestGetUserAPI() {
	// Mock repository calls
	mockUser := &grpcdomain.User{
		ID:    1,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	suite.mockRepo.On("GetByID", mock.Anything, int64(1)).Return(mockUser, nil)

	// Make HTTP request
	resp, err := suite.makeRequest("GET", "/v1/users/1", nil)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "1", response["id"])
	assert.Equal(suite.T(), "John Doe", response["name"])
	assert.Equal(suite.T(), "john@example.com", response["email"])
	suite.mockRepo.AssertExpectations(suite.T())
}

// Test UpdateUser API
func (suite *UserAPIIntegrationTestSuite) TestUpdateUserAPI() {
	// Mock repository calls
	suite.mockRepo.On("GetByEmail", mock.Anything, "john.updated@example.com").Return(nil, nil)
	suite.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(int64(1), nil)

	// Request payload
	requestBody := map[string]interface{}{
		"id":    1,
		"name":  "John Updated",
		"email": "john.updated@example.com",
	}

	// Make HTTP request
	resp, err := suite.makeRequest("PUT", "/v1/users/1", requestBody)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "1", response["id"])
	suite.mockRepo.AssertExpectations(suite.T())
}

// Test DeleteUser API
func (suite *UserAPIIntegrationTestSuite) TestDeleteUserAPI() {
	// Mock repository calls
	suite.mockRepo.On("Delete", mock.Anything, int64(1)).Return(int64(1), nil)

	// Make HTTP request
	resp, err := suite.makeRequest("DELETE", "/v1/users/1", nil)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	assert.Equal(suite.T(), "1", response["id"])
	suite.mockRepo.AssertExpectations(suite.T())
}

// Test ListUsers API
func (suite *UserAPIIntegrationTestSuite) TestListUsersAPI() {
	// Mock repository calls
	mockUsers := []grpcdomain.User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
	}
	suite.mockRepo.On("List", mock.Anything, "", int64(1), mock.AnythingOfType("int64")).Return(mockUsers, int64(50), nil)

	// Make HTTP request
	resp, err := suite.makeRequest("GET", "/v1/users?page=1&limit=10", nil)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	// Assertions
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	suite.Require().NoError(err)

	users, ok := response["users"].([]interface{})
	suite.Require().True(ok)
	assert.Equal(suite.T(), 2, len(users))

	// Verify pagination info
	pagination, ok := response["pagination"].(map[string]interface{})
	suite.Require().True(ok)
	assert.Equal(suite.T(), "50", pagination["total"])
	assert.Equal(suite.T(), "1", pagination["page"])
	assert.Equal(suite.T(), "10", pagination["limit"])
	assert.Equal(suite.T(), "5", pagination["totalPages"]) // (50 + 10 - 1) / 10 = 5

	suite.mockRepo.AssertExpectations(suite.T())
}

// Run the test suite
func TestUserAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(UserAPIIntegrationTestSuite))
}

// Test Complete CRUD Workflow
func (suite *UserAPIIntegrationTestSuite) TestCompleteCRUDWorkflow() {
	// 1. Create user
	suite.mockRepo.On("GetByEmail", mock.Anything, "workflow@example.com").Return(nil, nil)
	suite.mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*user.User")).Return(int64(1), nil)

	createReq := map[string]any{
		"name":  "Workflow User",
		"email": "workflow@example.com",
	}
	resp, err := suite.makeRequest("POST", "/v1/users", createReq)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// 2. Get user
	mockUser := &grpcdomain.User{ID: 1, Name: "Workflow User", Email: "workflow@example.com"}
	suite.mockRepo.On("GetByID", mock.Anything, int64(1)).Return(mockUser, nil)

	resp, err = suite.makeRequest("GET", "/v1/users/1", nil)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// 3. Update user
	suite.mockRepo.On("GetByEmail", mock.Anything, "updated@example.com").Return(nil, nil)
	suite.mockRepo.On("Update", mock.Anything, mock.AnythingOfType("*user.User")).Return(int64(1), nil)

	updateReq := map[string]interface{}{
		"id":    1,
		"name":  "Updated User",
		"email": "updated@example.com",
	}
	resp, err = suite.makeRequest("PUT", "/v1/users/1", updateReq)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// 4. Delete user
	suite.mockRepo.On("Delete", mock.Anything, int64(1)).Return(int64(1), nil)

	resp, err = suite.makeRequest("DELETE", "/v1/users/1", nil)
	suite.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	suite.mockRepo.AssertExpectations(suite.T())
}

// Test Error Handling
func (suite *UserAPIIntegrationTestSuite) TestErrorHandling() {
	// Test 1: User not found - should return 404 or 400 depending on implementation
	suite.T().Run("UserNotFound", func(t *testing.T) {
		suite.mockRepo.On("GetByID", mock.Anything, int64(999)).Return(nil, fmt.Errorf("user not found"))

		resp, err := suite.makeRequest("GET", "/v1/users/999", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		// Should return error status (400 for validation error, 404 for not found, or 500 for server error)
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)
		assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 600)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		suite.Require().NoError(err)
		assert.Contains(t, response, "message")
		suite.mockRepo.AssertExpectations(suite.T())
	})

	// Test 2: Invalid user ID (ID = 0) - should return 400
	suite.T().Run("InvalidUserID", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/v1/users/0", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		suite.Require().NoError(err)
		assert.Contains(t, response, "message")
	})

	// Test 3: Negative user ID - should return 400
	suite.T().Run("NegativeUserID", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/v1/users/-1", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		suite.Require().NoError(err)
		assert.Contains(t, response, "message")
	})

	// Test 4: Non-existent endpoint - should return 404
	suite.T().Run("NonExistentEndpoint", func(t *testing.T) {
		resp, err := suite.makeRequest("GET", "/v1/nonexistent", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test 5: Invalid HTTP method - should return 404 or 405
	suite.T().Run("InvalidHTTPMethod", func(t *testing.T) {
		resp, err := suite.makeRequest("PATCH", "/v1/users/1", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		// Should return 404 (Not Found), 405 (Method Not Allowed), or 501 (Not Implemented)
		assert.True(t, resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented, "Expected 404, 405 or 501, got %d", resp.StatusCode)
	})

	// Test 6: Invalid JSON payload - should return 400
	suite.T().Run("InvalidJSON", func(t *testing.T) {
		invalidJSON := `{"name": "John", "email":}`
		req, err := http.NewRequestWithContext(context.Background(), "POST", suite.baseURL+"/v1/users", bytes.NewBufferString(invalidJSON))
		suite.Require().NoError(err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := suite.httpClient.Do(req)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		suite.Require().NoError(err)
		assert.Contains(t, response, "message")
	})

	// Test 7: Empty request body - should return 400
	suite.T().Run("EmptyRequestBody", func(t *testing.T) {
		resp, err := suite.makeRequest("POST", "/v1/users", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		suite.Require().NoError(err)
		assert.Contains(t, response, "message")
	})

	// Test 8: Database error simulation - should return 500
	suite.T().Run("DatabaseError", func(t *testing.T) {
		suite.mockRepo.On("GetByID", mock.Anything, int64(1)).Return(nil, fmt.Errorf("database connection failed"))

		resp, err := suite.makeRequest("GET", "/v1/users/1", nil)
		suite.Require().NoError(err)
		defer func() { _ = resp.Body.Close() }()

		// Database errors typically return 500
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		suite.Require().NoError(err)
		assert.Contains(t, response, "message")
		suite.mockRepo.AssertExpectations(suite.T())
	})
}

// Test Validation Errors
func (suite *UserAPIIntegrationTestSuite) TestValidationErrors() {
	// Test 1: Create user with invalid data
	suite.T().Run("CreateUserValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			requestBody map[string]interface{}
			expectedMsg string
		}{
			{
				name: "Name too short",
				requestBody: map[string]interface{}{
					"name":  "Jo",
					"email": "john@example.com",
				},
				expectedMsg: "Name",
			},
			{
				name: "Invalid email",
				requestBody: map[string]interface{}{
					"name":  "John Doe",
					"email": "invalid-email",
				},
				expectedMsg: "Email",
			},
			{
				name: "Missing name field",
				requestBody: map[string]interface{}{
					"email": "john@example.com",
				},
				expectedMsg: "Name",
			},
			{
				name: "Missing email field",
				requestBody: map[string]interface{}{
					"name": "John Doe",
				},
				expectedMsg: "Email",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := suite.makeRequest("POST", "/v1/users", tc.requestBody)
				suite.Require().NoError(err)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				suite.Require().NoError(err)
				// grpc-gateway returns error message in "message" field
				assert.Contains(t, response, "message")
				assert.Contains(t, response["message"], tc.expectedMsg)
			})
		}
	})

	// Test 2: Update user with invalid data
	suite.T().Run("UpdateUserValidation", func(t *testing.T) {
		testCases := []struct {
			name        string
			requestBody map[string]interface{}
			expectedMsg string
		}{
			{
				name: "Name too short",
				requestBody: map[string]interface{}{
					"id":    1,
					"name":  "Jo",
					"email": "john@example.com",
				},
				expectedMsg: "Name",
			},
			{
				name: "Invalid email",
				requestBody: map[string]interface{}{
					"id":    1,
					"name":  "John Doe",
					"email": "invalid-email",
				},
				expectedMsg: "Email",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := suite.makeRequest("PUT", "/v1/users/1", tc.requestBody)
				suite.Require().NoError(err)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				suite.Require().NoError(err)
				// grpc-gateway returns error message in "message" field
				assert.Contains(t, response, "message")
				assert.Contains(t, response["message"], tc.expectedMsg)
			})
		}
	})

	// Test 3: Business logic errors
	suite.T().Run("BusinessLogicErrors", func(t *testing.T) {
		// Test email already exists
		suite.T().Run("EmailAlreadyExists", func(t *testing.T) {
			existingUser := &grpcdomain.User{ID: 2, Name: "Existing User", Email: "john@example.com"}
			suite.mockRepo.On("GetByEmail", mock.Anything, "john@example.com").Return(existingUser, nil)

			requestBody := map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			}

			resp, err := suite.makeRequest("POST", "/v1/users", requestBody)
			suite.Require().NoError(err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusConflict, resp.StatusCode)

			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			suite.Require().NoError(err)
			// grpc-gateway returns error message in "message" field
			assert.Contains(t, response, "message")
			assert.Contains(t, response["message"], "email already exists")
			suite.mockRepo.AssertExpectations(suite.T())
		})
	})
}

// Test Concurrency
func (suite *UserAPIIntegrationTestSuite) TestConcurrency() {
	// Setup mock for concurrent requests
	suite.mockRepo.On("GetByID", mock.Anything, mock.AnythingOfType("int64")).Return(&grpcdomain.User{
		ID:    1,
		Name:  "Concurrent User",
		Email: "concurrent@example.com",
	}, nil).Maybe()

	// Make multiple concurrent requests
	done := make(chan bool, 5)
	for range 5 {
		go func() {
			resp, err := suite.makeRequest("GET", "/v1/users/1", nil)
			suite.Require().NoError(err)
			_ = resp.Body.Close()
			done <- true
		}()
	}

	// Wait for all requests to complete
	for range 5 {
		<-done
	}
}

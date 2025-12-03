package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	usecase "grpc-user-service/internal/usecase/user"
	pkgerrors "grpc-user-service/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// MockUserUsecase is a mock implementation of user.Usecase
type MockUserUsecase struct {
	mock.Mock
}

func (m *MockUserUsecase) CreateUser(ctx context.Context, req usecase.CreateUserRequest) (*usecase.CreateUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.CreateUserResponse), args.Error(1)
}

func (m *MockUserUsecase) GetUser(ctx context.Context, req usecase.GetUserRequest) (*usecase.GetUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.GetUserResponse), args.Error(1)
}

func (m *MockUserUsecase) UpdateUser(ctx context.Context, req usecase.UpdateUserRequest) (*usecase.UpdateUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.UpdateUserResponse), args.Error(1)
}

func (m *MockUserUsecase) DeleteUser(ctx context.Context, req usecase.DeleteUserRequest) (*usecase.DeleteUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.DeleteUserResponse), args.Error(1)
}

func (m *MockUserUsecase) ListUsers(ctx context.Context, req usecase.ListUsersRequest) (*usecase.ListUsersResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*usecase.ListUsersResponse), args.Error(1)
}

func setupTest(t *testing.T) (*gin.Engine, *UserHandler, *MockUserUsecase) {
	gin.SetMode(gin.TestMode)
	mockUsecase := new(MockUserUsecase)
	logger := zaptest.NewLogger(t)
	handler := NewUserHandler(mockUsecase, logger)

	r := gin.New()
	return r, handler, mockUsecase
}

func TestCreateUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.POST("/users", handler.CreateUser)

		reqBody := CreateUserRequest{
			Name:  "John Doe",
			Email: "john@example.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		expectedResponse := &usecase.CreateUserResponse{
			ID: 1,
		}

		mockUsecase.On("CreateUser", mock.Anything, mock.MatchedBy(func(req usecase.CreateUserRequest) bool {
			return req.Name == reqBody.Name && req.Email == reqBody.Email
		})).Return(expectedResponse, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse.ID, resp.ID)
	})

	t.Run("Invalid Request Body", func(t *testing.T) {
		r, handler, _ := setupTest(t)
		r.POST("/users", handler.CreateUser)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/users", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Validation Error", func(t *testing.T) {
		r, handler, _ := setupTest(t)
		r.POST("/users", handler.CreateUser)

		reqBody := CreateUserRequest{
			Name:  "", // Invalid
			Email: "invalid-email",
		}
		jsonBody, _ := json.Marshal(reqBody)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Usecase Error", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.POST("/users", handler.CreateUser)

		reqBody := CreateUserRequest{
			Name:  "John Doe",
			Email: "john@example.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		mockUsecase.On("CreateUser", mock.Anything, mock.Anything).Return(nil, errors.New("internal error"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/users", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.GET("/users/:id", handler.GetUser)

		expectedResponse := &usecase.GetUserResponse{
			ID:    1,
			Name:  "John Doe",
			Email: "john@example.com",
		}

		mockUsecase.On("GetUser", mock.Anything, usecase.GetUserRequest{ID: 1}).Return(expectedResponse, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp UserResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, expectedResponse.ID, resp.ID)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		r, handler, _ := setupTest(t)
		r.GET("/users/:id", handler.GetUser)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/abc", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Not Found", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.GET("/users/:id", handler.GetUser)

		mockUsecase.On("GetUser", mock.Anything, usecase.GetUserRequest{ID: 1}).Return(nil, pkgerrors.NewNotFoundError("user", "user not found"))

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.PUT("/users/:id", handler.UpdateUser)

		reqBody := UpdateUserRequest{
			Name:  "John Updated",
			Email: "john.updated@example.com",
		}
		jsonBody, _ := json.Marshal(reqBody)

		expectedResponse := &usecase.UpdateUserResponse{
			ID: 1,
		}

		mockUsecase.On("UpdateUser", mock.Anything, mock.MatchedBy(func(req usecase.UpdateUserRequest) bool {
			return req.ID == 1 && req.Name == reqBody.Name && req.Email == reqBody.Email
		})).Return(expectedResponse, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/users/1", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		r, handler, _ := setupTest(t)
		r.PUT("/users/:id", handler.UpdateUser)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("PUT", "/users/abc", bytes.NewBufferString("{}"))
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteUser(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.DELETE("/users/:id", handler.DeleteUser)

		mockUsecase.On("DeleteUser", mock.Anything, usecase.DeleteUserRequest{ID: 1}).Return(&usecase.DeleteUserResponse{ID: 1}, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("DELETE", "/users/1", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestListUsers(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		r, handler, mockUsecase := setupTest(t)
		r.GET("/users", handler.ListUsers)

		expectedResponse := &usecase.ListUsersResponse{
			Users: []usecase.User{
				{ID: 1, Name: "User 1"},
				{ID: 2, Name: "User 2"},
			},
			Pagination: &usecase.Pagination{
				Total: 2,
				Page:  1,
				Limit: 10,
			},
		}

		mockUsecase.On("ListUsers", mock.Anything, mock.MatchedBy(func(req usecase.ListUsersRequest) bool {
			return req.Page == 1 && req.Limit == 10
		})).Return(expectedResponse, nil)

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/users?page=1&limit=10", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp ListUsersResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Users, 2)
		assert.Equal(t, int64(2), resp.Pagination.Total)
	})
}

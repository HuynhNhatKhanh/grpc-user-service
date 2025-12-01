package handler

import (
	"net/http"
	"strconv"

	"grpc-user-service/internal/usecase/user"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	uc  *user.Usecase
	log *zap.Logger
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(uc *user.Usecase, log *zap.Logger) *UserHandler {
	return &UserHandler{
		uc:  uc,
		log: log,
	}
}

// CreateUserRequest represents the HTTP request body for creating a user
type CreateUserRequest struct {
	Name  string `json:"name" binding:"required,min=3,max=100"`
	Email string `json:"email" binding:"required,email"`
}

// UpdateUserRequest represents the HTTP request body for updating a user
type UpdateUserRequest struct {
	Name  string `json:"name" binding:"omitempty,min=3,max=100"`
	Email string `json:"email" binding:"omitempty,email"`
}

// UserResponse represents the HTTP response for user data
type UserResponse struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ListUsersResponse represents the HTTP response for listing users
type ListUsersResponse struct {
	Users      []UserResponse `json:"users"`
	Pagination *Pagination    `json:"pagination,omitempty"`
}

// Pagination represents pagination information
type Pagination struct {
	Total      int64 `json:"total"`
	Page       int64 `json:"page"`
	Limit      int64 `json:"limit"`
	TotalPages int64 `json:"total_pages"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// CreateUser handles POST /v1/users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("Invalid create user request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	h.log.Info("Gin CreateUser request", zap.String("name", req.Name), zap.String("email", req.Email))

	ucReq := user.CreateUserRequest{
		Name:  req.Name,
		Email: req.Email,
	}

	resp, err := h.uc.CreateUser(c.Request.Context(), ucReq)
	if err != nil {
		h.log.Error("Gin CreateUser failed", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": resp.ID,
	})
}

// GetUser handles GET /v1/users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Warn("Invalid user ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	h.log.Info("Gin GetUser request", zap.Int64("id", id))

	ucReq := user.GetUserRequest{ID: id}
	resp, err := h.uc.GetUser(c.Request.Context(), ucReq)
	if err != nil {
		h.log.Error("Gin GetUser failed", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, UserResponse{
		ID:    resp.ID,
		Name:  resp.Name,
		Email: resp.Email,
	})
}

// UpdateUser handles PUT /v1/users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Warn("Invalid user ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("Invalid update user request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "validation_error",
			Message: err.Error(),
		})
		return
	}

	h.log.Info("Gin UpdateUser request", zap.Int64("id", id), zap.String("name", req.Name), zap.String("email", req.Email))

	ucReq := user.UpdateUserRequest{
		ID:    id,
		Name:  req.Name,
		Email: req.Email,
	}

	resp, err := h.uc.UpdateUser(c.Request.Context(), ucReq)
	if err != nil {
		h.log.Error("Gin UpdateUser failed", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": resp.ID,
	})
}

// DeleteUser handles DELETE /v1/users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		h.log.Warn("Invalid user ID", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "User ID must be a valid number",
		})
		return
	}

	h.log.Info("Gin DeleteUser request", zap.Int64("id", id))

	ucReq := user.DeleteUserRequest{ID: id}
	resp, err := h.uc.DeleteUser(c.Request.Context(), ucReq)
	if err != nil {
		h.log.Error("Gin DeleteUser failed", zap.Error(err))
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": resp.ID,
	})
}

// ListUsers handles GET /v1/users
func (h *UserHandler) ListUsers(c *gin.Context) {
	query := c.DefaultQuery("query", "")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	h.log.Info("Gin ListUsers request", zap.String("query", query), zap.Int64("page", page), zap.Int64("limit", limit))

	ucReq := user.ListUsersRequest{
		Query: query,
		Page:  page,
		Limit: limit,
	}

	resp, err := h.uc.ListUsers(c.Request.Context(), ucReq)
	if err != nil {
		h.log.Error("Gin ListUsers failed", zap.Error(err))
		h.handleError(c, err)
		return
	}

	users := make([]UserResponse, len(resp.Users))
	for i, u := range resp.Users {
		users[i] = UserResponse{
			ID:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		}
	}

	var pagination *Pagination
	if resp.Pagination != nil {
		pagination = &Pagination{
			Total:      resp.Pagination.Total,
			Page:       resp.Pagination.Page,
			Limit:      resp.Pagination.Limit,
			TotalPages: resp.Pagination.TotalPages,
		}
	}

	c.JSON(http.StatusOK, ListUsersResponse{
		Users:      users,
		Pagination: pagination,
	})
}

// handleError converts usecase errors to appropriate HTTP responses
func (h *UserHandler) handleError(c *gin.Context, err error) {
	// Check for custom error types from pkg/errors
	type grpcStatuser interface {
		GRPCStatus() any
	}

	if _, ok := err.(grpcStatuser); ok {
		// Handle specific error types
		errMsg := err.Error()
		switch {
		case contains(errMsg, "not found"):
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: errMsg,
			})
		case contains(errMsg, "already exists"):
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "already_exists",
				Message: errMsg,
			})
		case contains(errMsg, "invalid"):
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_input",
				Message: errMsg,
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "An internal error occurred",
			})
		}
		return
	}

	// Default error response
	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error:   "internal_error",
		Message: "An internal error occurred",
	})
}

// contains checks if a string contains a substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

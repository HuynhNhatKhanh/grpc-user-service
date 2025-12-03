package test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	grpcdomain "grpc-user-service/internal/domain/user"
	grpcuser "grpc-user-service/internal/usecase/user"
)

// ComprehensiveMockRepository is a mock implementation of the Repository interface.
// It uses testify/mock for creating mock objects in unit tests.
type ComprehensiveMockRepository struct {
	mock.Mock
}

func (m *ComprehensiveMockRepository) Create(ctx context.Context, u *grpcdomain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *ComprehensiveMockRepository) GetByID(ctx context.Context, id int64) (*grpcdomain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcdomain.User), args.Error(1)
}

func (m *ComprehensiveMockRepository) GetByEmail(ctx context.Context, email string) (*grpcdomain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*grpcdomain.User), args.Error(1)
}

func (m *ComprehensiveMockRepository) Update(ctx context.Context, u *grpcdomain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *ComprehensiveMockRepository) Delete(ctx context.Context, id int64) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *ComprehensiveMockRepository) List(ctx context.Context, query string, page, limit int64) ([]grpcdomain.User, int64, error) {
	args := m.Called(ctx, query, page, limit)
	return args.Get(0).([]grpcdomain.User), args.Get(1).(int64), args.Error(2)
}

// setupComprehensiveTestUsecase creates a new usecase instance with a mock repository for testing.
// It returns both the usecase and the mock repository for test setup and verification.
func setupComprehensiveTestUsecase(t *testing.T) (grpcuser.Usecase, *ComprehensiveMockRepository) {
	mockRepo := new(ComprehensiveMockRepository)
	logger := zaptest.NewLogger(t)
	uc := grpcuser.New(mockRepo, logger)
	return uc, mockRepo
}

// ==================== CREATE USER TESTS ====================

// TestCreateUser_Success tests successful user creation with valid input.
// It verifies that the usecase properly validates input and calls repository methods.
func TestCreateUser_Success(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Mock GetByEmail returns nil (email not found)
	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, nil)
	// Mock Create returns success
	mockRepo.On("Create", ctx, mock.MatchedBy(func(u *grpcdomain.User) bool {
		return u.Name == req.Name && u.Email == req.Email
	})).Return(int64(1), nil)

	resp, err := uc.CreateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

// TestCreateUser_ValidationError_NameRequired tests user creation validation failure.
// It verifies that the usecase returns an error when the name field is empty.
func TestCreateUser_ValidationError_NameRequired(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "", // Empty name
		Email: "john@example.com",
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Name is required")
}

func TestCreateUser_ValidationError_NameTooShort(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "Jo", // Too short (<3)
		Email: "john@example.com",
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Name must be at least 3 characters")
}

func TestCreateUser_ValidationError_EmailRequired(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "John Doe",
		Email: "", // Empty email
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Email is required")
}

func TestCreateUser_ValidationError_EmailInvalid(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "John Doe",
		Email: "invalid-email", // Invalid email format
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Email must be a valid email")
}

func TestCreateUser_ValidationError_MultipleErrors(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "Jo",      // Too short
		Email: "invalid", // Invalid email
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Name must be at least 3 characters")
	assert.Contains(t, err.Error(), "Email must be a valid email")
}

func TestCreateUser_SemanticValidation_EmailAlreadyExists(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	existingUser := &grpcdomain.User{ID: 2, Name: "Existing User", Email: "john@example.com"}

	// Mock GetByEmail returns existing user
	mockRepo.On("GetByEmail", ctx, req.Email).Return(existingUser, nil)

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "email already exists")

	mockRepo.AssertExpectations(t)
}

// ==================== UPDATE USER TESTS ====================

func TestUpdateUser_Success(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.UpdateUserRequest{
		ID:    1,
		Name:  "John Updated",
		Email: "john.updated@example.com",
	}

	// Mock GetByEmail returns nil (email not found)
	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, nil)
	// Mock Update returns success
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *grpcdomain.User) bool {
		return u.ID == req.ID && u.Name == req.Name && u.Email == req.Email
	})).Return(int64(1), nil)

	resp, err := uc.UpdateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestUpdateUser_PartialUpdate_NameOnly(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.UpdateUserRequest{
		ID:   1,
		Name: "John Updated",
		// Email empty - should not trigger email validation
	}

	// Mock Update returns success
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *grpcdomain.User) bool {
		return u.ID == req.ID && u.Name == req.Name
	})).Return(int64(1), nil)

	resp, err := uc.UpdateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestUpdateUser_ValidationError_NameTooShort(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.UpdateUserRequest{
		ID:    1,
		Name:  "Jo", // Too short
		Email: "john@example.com",
	}

	resp, err := uc.UpdateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Name must be at least 3 characters")
}

func TestUpdateUser_ValidationError_EmailInvalid(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.UpdateUserRequest{
		ID:    1,
		Name:  "John Doe",
		Email: "invalid-email",
	}

	resp, err := uc.UpdateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Email must be a valid email")
}

func TestUpdateUser_SemanticValidation_EmailAlreadyExists(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.UpdateUserRequest{
		ID:    1,
		Name:  "John Updated",
		Email: "john@example.com",
	}

	existingUser := &grpcdomain.User{ID: 2, Name: "Existing User", Email: "john@example.com"}

	// Mock GetByEmail returns existing user with different ID
	mockRepo.On("GetByEmail", ctx, req.Email).Return(existingUser, nil)

	resp, err := uc.UpdateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "email already exists")

	mockRepo.AssertExpectations(t)
}

// ==================== DELETE USER TESTS ====================

func TestDeleteUser_Success(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.DeleteUserRequest{ID: 1}

	// Mock Delete returns success
	mockRepo.On("Delete", ctx, req.ID).Return(int64(1), nil)

	resp, err := uc.DeleteUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestDeleteUser_InvalidID(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.DeleteUserRequest{ID: 0} // Invalid ID

	resp, err := uc.DeleteUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid user id")
}

// ==================== GET USER TESTS ====================

func TestGetUser_Success(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.GetUserRequest{ID: 1}
	expectedUser := &grpcdomain.User{ID: 1, Name: "John Doe", Email: "john@example.com"}

	// Mock GetByID returns user
	mockRepo.On("GetByID", ctx, req.ID).Return(expectedUser, nil)

	resp, err := uc.GetUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, expectedUser.ID, resp.ID)
	assert.Equal(t, expectedUser.Name, resp.Name)
	assert.Equal(t, expectedUser.Email, resp.Email)

	mockRepo.AssertExpectations(t)
}

func TestGetUser_InvalidID(t *testing.T) {
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.GetUserRequest{ID: 0} // Invalid ID

	resp, err := uc.GetUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid user id")
}

// ==================== LIST USERS TESTS ====================

func TestListUsers_Success(t *testing.T) {
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.ListUsersRequest{
		Query: "john",
		Page:  1,
		Limit: 10,
	}

	expectedUsers := []grpcdomain.User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "John Smith", Email: "smith@example.com"},
	}

	// Mock List returns users and total count
	mockRepo.On("List", ctx, req.Query, req.Page, req.Limit).Return(expectedUsers, int64(30), nil)

	resp, err := uc.ListUsers(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, expectedUsers[0].ID, resp.Users[0].ID)
	assert.Equal(t, expectedUsers[0].Name, resp.Users[0].Name)
	assert.Equal(t, expectedUsers[0].Email, resp.Users[0].Email)

	// Verify pagination info
	assert.NotNil(t, resp.Pagination)
	assert.Equal(t, int64(30), resp.Pagination.Total)
	assert.Equal(t, int64(1), resp.Pagination.Page)
	assert.Equal(t, int64(10), resp.Pagination.Limit)
	assert.Equal(t, int64(3), resp.Pagination.TotalPages) // (30 + 10 - 1) / 10 = 3

	mockRepo.AssertExpectations(t)
}

// ==================== VALIDATION HELPER TESTS ====================

func TestFormatValidationError(t *testing.T) {
	validate := validator.New()

	type TestStruct struct {
		Name  string `validate:"required,min=3"`
		Email string `validate:"required,email"`
	}

	// Test multiple validation errors
	_ = validate.Struct(&TestStruct{})

	// Since formatValidationError is private, we test validation error handling indirectly
	// by creating a user with validation errors and checking the response
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "", // Invalid - empty name
		Email: "", // Invalid - empty email
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "Name is required")
	assert.Contains(t, err.Error(), "Email is required")
}

func TestFormatValidationError_SingleError(t *testing.T) {
	// Since formatValidationError is private, we test validation error handling indirectly
	// by creating a user with single validation error and checking the response
	uc, _ := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "",                 // Invalid - empty name only
		Email: "test@example.com", // Valid email
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "validation failed")
	assert.Contains(t, err.Error(), "Name is required")
	assert.NotContains(t, err.Error(), "Email")
}

func TestFormatValidationError_NonValidationError(t *testing.T) {
	// Since formatValidationError is private, we test non-validation error handling indirectly
	// by testing repository error scenario
	uc, mockRepo := setupComprehensiveTestUsecase(t)
	ctx := context.Background()

	req := grpcuser.CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Mock repository error (not validation error)
	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, errors.New("database error"))

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "failed to validate email uniqueness")
	assert.NotContains(t, err.Error(), "validation failed")
}

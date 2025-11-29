package user

import (
	"context"
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	domain "grpc-user-service/internal/domain/user"
)

// MockRepository là mock implementation của Repository interface
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, u *domain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockRepository) Update(ctx context.Context, u *domain.User) (int64, error) {
	args := m.Called(ctx, u)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) Delete(ctx context.Context, id int64) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRepository) List(ctx context.Context, query string, page, limit int64) ([]domain.User, error) {
	args := m.Called(ctx, query, page, limit)
	return args.Get(0).([]domain.User), args.Error(1)
}

// Test helper để tạo usecase với mock repo
func setupTestUsecase(t *testing.T) (*Usecase, *MockRepository) {
	mockRepo := new(MockRepository)
	logger := zaptest.NewLogger(t)
	uc := New(mockRepo, logger)
	return uc, mockRepo
}

// ==================== CREATE USER TESTS ====================

func TestCreateUser_Success(t *testing.T) {
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// Mock GetByEmail returns nil (email not found)
	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, nil)
	// Mock Create returns success
	mockRepo.On("Create", ctx, mock.MatchedBy(func(u *domain.User) bool {
		return u.Name == req.Name && u.Email == req.Email
	})).Return(int64(1), nil)

	resp, err := uc.CreateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestCreateUser_ValidationError_NameRequired(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "", // Empty name
		Email: "john@example.com",
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Name is required")
}

func TestCreateUser_ValidationError_NameTooShort(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "Jo", // Too short (<3)
		Email: "john@example.com",
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Name must be at least 3 characters")
}

func TestCreateUser_ValidationError_EmailRequired(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "John Doe",
		Email: "", // Empty email
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Email is required")
}

func TestCreateUser_ValidationError_EmailInvalid(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "John Doe",
		Email: "invalid-email", // Invalid email format
	}

	resp, err := uc.CreateUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "Email must be a valid email")
}

func TestCreateUser_ValidationError_MultipleErrors(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
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
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := CreateUserRequest{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	existingUser := &domain.User{ID: 2, Name: "Existing User", Email: "john@example.com"}

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
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := UpdateUserRequest{
		ID:    1,
		Name:  "John Updated",
		Email: "john.updated@example.com",
	}

	// Mock GetByEmail returns nil (email not found)
	mockRepo.On("GetByEmail", ctx, req.Email).Return(nil, nil)
	// Mock Update returns success
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *domain.User) bool {
		return u.ID == req.ID && u.Name == req.Name && u.Email == req.Email
	})).Return(int64(1), nil)

	resp, err := uc.UpdateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestUpdateUser_PartialUpdate_NameOnly(t *testing.T) {
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := UpdateUserRequest{
		ID:   1,
		Name: "John Updated",
		// Email empty - should not trigger email validation
	}

	// Mock Update returns success
	mockRepo.On("Update", ctx, mock.MatchedBy(func(u *domain.User) bool {
		return u.ID == req.ID && u.Name == req.Name
	})).Return(int64(1), nil)

	resp, err := uc.UpdateUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestUpdateUser_ValidationError_NameTooShort(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := UpdateUserRequest{
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
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := UpdateUserRequest{
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
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := UpdateUserRequest{
		ID:    1,
		Name:  "John Updated",
		Email: "john@example.com",
	}

	existingUser := &domain.User{ID: 2, Name: "Existing User", Email: "john@example.com"}

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
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := DeleteUserRequest{ID: 1}

	// Mock Delete returns success
	mockRepo.On("Delete", ctx, req.ID).Return(int64(1), nil)

	resp, err := uc.DeleteUser(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1), resp.ID)

	mockRepo.AssertExpectations(t)
}

func TestDeleteUser_InvalidID(t *testing.T) {
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := DeleteUserRequest{ID: 0} // Invalid ID

	resp, err := uc.DeleteUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid user id")
}

// ==================== GET USER TESTS ====================

func TestGetUser_Success(t *testing.T) {
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := GetUserRequest{ID: 1}
	expectedUser := &domain.User{ID: 1, Name: "John Doe", Email: "john@example.com"}

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
	uc, _ := setupTestUsecase(t)
	ctx := context.Background()

	req := GetUserRequest{ID: 0} // Invalid ID

	resp, err := uc.GetUser(ctx, req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid user id")
}

// ==================== LIST USERS TESTS ====================

func TestListUsers_Success(t *testing.T) {
	uc, mockRepo := setupTestUsecase(t)
	ctx := context.Background()

	req := ListUsersRequest{
		Query: "john",
		Page:  1,
		Limit: 10,
	}

	expectedUsers := []domain.User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "John Smith", Email: "smith@example.com"},
	}

	// Mock List returns users
	mockRepo.On("List", ctx, req.Query, req.Page, req.Limit).Return(expectedUsers, nil)

	resp, err := uc.ListUsers(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Users, 2)
	assert.Equal(t, expectedUsers[0].ID, resp.Users[0].ID)
	assert.Equal(t, expectedUsers[0].Name, resp.Users[0].Name)
	assert.Equal(t, expectedUsers[0].Email, resp.Users[0].Email)

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
	err := validate.Struct(&TestStruct{})
	formatted := formatValidationError(err)

	assert.Error(t, formatted)
	assert.Contains(t, formatted.Error(), "validation failed")
	assert.Contains(t, formatted.Error(), "Name is required")
	assert.Contains(t, formatted.Error(), "Email is required")
}

func TestFormatValidationError_SingleError(t *testing.T) {
	validate := validator.New()

	type TestStruct struct {
		Name  string `validate:"required,min=3"`
		Email string
	}

	// Test single validation error
	err := validate.Struct(&TestStruct{Email: "test@example.com"})
	formatted := formatValidationError(err)

	assert.Error(t, formatted)
	assert.Contains(t, formatted.Error(), "Name is required")
	assert.NotContains(t, formatted.Error(), "Email")
}

func TestFormatValidationError_NonValidationError(t *testing.T) {
	originalErr := errors.New("some other error")
	formatted := formatValidationError(originalErr)

	assert.Equal(t, originalErr, formatted)
}

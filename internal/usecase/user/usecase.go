package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"grpc-user-service/internal/adapter/cache"
	domain "grpc-user-service/internal/domain/user"

	"github.com/go-playground/validator/v10"
)

// Repository defines the interface for user data access operations.
// It abstracts the data layer, allowing different implementations
// (e.g., PostgreSQL, MongoDB) to be used interchangeably.
type Repository interface {
	Create(ctx context.Context, u *domain.User) (int64, error)                        // Create a new user
	GetByID(ctx context.Context, id int64) (*domain.User, error)                      // Retrieve user by ID
	GetByEmail(ctx context.Context, email string) (*domain.User, error)               // Retrieve user by email
	Update(ctx context.Context, u *domain.User) (int64, error)                        // Update existing user
	Delete(ctx context.Context, id int64) (int64, error)                              // Delete user by ID
	List(ctx context.Context, query string, page, limit int64) ([]domain.User, error) // List users with pagination and search
}

// Usecase implements the business logic for user management operations.
// It provides a clean separation between the transport layer and data layer.
type Usecase struct {
	repo     Repository          // Repository for data access
	cache    cache.UserCache     // Cache for user data
	log      *zap.Logger         // Logger for structured logging
	validate *validator.Validate // Validator for request validation
}

// New creates a new instance of Usecase with the provided repository, cache, and logger.
// If cache is nil, caching will be disabled.
func New(r Repository, c cache.UserCache, log *zap.Logger) *Usecase {
	return &Usecase{repo: r, cache: c, log: log, validate: validator.New()}
}

// formatValidationError converts validator.ValidationErrors into a human-readable error message.
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, e := range validationErrors {
			switch e.Tag() {
			case "required":
				messages = append(messages, fmt.Sprintf("%s is required", e.Field()))
			case "email":
				messages = append(messages, fmt.Sprintf("%s must be a valid email", e.Field()))
			case "min":
				messages = append(messages, fmt.Sprintf("%s must be at least %s characters", e.Field(), e.Param()))
			case "max":
				messages = append(messages, fmt.Sprintf("%s must be at most %s characters", e.Field(), e.Param()))
			default:
				messages = append(messages, fmt.Sprintf("%s is invalid", e.Field()))
			}
		}
		return fmt.Errorf("validation failed: %s", strings.Join(messages, ", "))
	}
	return err
}

// CreateUser creates a new user after validating the request and checking email uniqueness.
func (uc *Usecase) CreateUser(ctx context.Context, in CreateUserRequest) (*CreateUserResponse, error) {
	uc.log.Info("creating user", zap.String("name", in.Name), zap.String("email", in.Email))

	if err := uc.validate.Struct(in); err != nil {
		uc.log.Warn("validate failed", zap.Error(err))
		return nil, formatValidationError(err)
	}

	existingUser, err := uc.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		uc.log.Error("failed to check existing email", zap.String("email", in.Email), zap.Error(err))
		return nil, errors.New("failed to validate email uniqueness")
	}
	if existingUser != nil {
		uc.log.Warn("email already exists", zap.String("email", in.Email))
		return nil, errors.New("email already exists")
	}

	// Business logic: create user
	id, err := uc.repo.Create(ctx, &domain.User{
		Name:  in.Name,
		Email: in.Email,
	})
	if err != nil {
		uc.log.Error("failed to create user", zap.Error(err))
		return nil, err
	}
	return &CreateUserResponse{ID: id}, nil
}

// UpdateUser updates an existing user after validating the request and checking email uniqueness.
// It invalidates the cache after successful update.
func (uc *Usecase) UpdateUser(ctx context.Context, in UpdateUserRequest) (*UpdateUserResponse, error) {
	uc.log.Info("updating user", zap.Int64("id", in.ID), zap.String("name", in.Name), zap.String("email", in.Email))

	if err := uc.validate.Struct(in); err != nil {
		uc.log.Warn("validate failed", zap.Error(err))
		return nil, formatValidationError(err)
	}

	if in.Email != "" {
		existingUser, err := uc.repo.GetByEmail(ctx, in.Email)
		if err != nil {
			uc.log.Error("failed to check existing email", zap.String("email", in.Email), zap.Error(err))
			return nil, errors.New("failed to validate email uniqueness")
		}
		if existingUser != nil && existingUser.ID != in.ID {
			uc.log.Warn("email already exists", zap.String("email", in.Email), zap.Int64("existing_id", existingUser.ID))
			return nil, errors.New("email already exists")
		}
	}

	// Business logic: update user
	id, err := uc.repo.Update(ctx, &domain.User{
		ID:    in.ID,
		Name:  in.Name,
		Email: in.Email,
	})
	if err != nil {
		uc.log.Error("failed to update user", zap.Int64("id", in.ID), zap.Error(err))
		return nil, err
	}

	// Invalidate cache after successful update
	if uc.cache != nil {
		if err := uc.cache.Delete(ctx, in.ID); err != nil {
			uc.log.Warn("failed to invalidate cache after update", zap.Int64("id", in.ID), zap.Error(err))
		}
	}

	return &UpdateUserResponse{ID: id}, nil
}

// DeleteUser deletes a user after validating the user ID.
// It invalidates the cache after successful deletion.
func (uc *Usecase) DeleteUser(ctx context.Context, in DeleteUserRequest) (*DeleteUserResponse, error) {
	uc.log.Info("deleting user", zap.Int64("id", in.ID))

	if in.ID <= 0 {
		uc.log.Warn("delete user validation failed", zap.Int64("id", in.ID), zap.String("reason", "invalid id"))
		return nil, errors.New("invalid user id")
	}

	id, err := uc.repo.Delete(ctx, in.ID)
	if err != nil {
		uc.log.Error("failed to delete user", zap.Int64("id", in.ID), zap.Error(err))
		return nil, err
	}

	// Invalidate cache after successful deletion
	if uc.cache != nil {
		if err := uc.cache.Delete(ctx, in.ID); err != nil {
			uc.log.Warn("failed to invalidate cache after delete", zap.Int64("id", in.ID), zap.Error(err))
		}
	}

	return &DeleteUserResponse{ID: id}, nil
}

// GetUser retrieves a user by ID after validating the request.
// It uses cache-aside pattern: check cache first, then database if cache miss.
func (uc *Usecase) GetUser(ctx context.Context, in GetUserRequest) (*GetUserResponse, error) {
	if in.ID <= 0 {
		uc.log.Warn("get user validation failed", zap.Int64("id", in.ID), zap.String("reason", "invalid id"))
		return nil, errors.New("invalid user id")
	}

	// Try to get from cache first
	if uc.cache != nil {
		cachedUser, err := uc.cache.Get(ctx, in.ID)
		if err != nil {
			uc.log.Warn("cache get error, falling back to database", zap.Int64("id", in.ID), zap.Error(err))
		} else if cachedUser != nil {
			uc.log.Debug("user retrieved from cache", zap.Int64("id", in.ID))
			return &GetUserResponse{
				ID:    cachedUser.ID,
				Name:  cachedUser.Name,
				Email: cachedUser.Email,
			}, nil
		}
	}

	// Cache miss or cache disabled - get from database
	u, err := uc.repo.GetByID(ctx, in.ID)
	if err != nil {
		uc.log.Error("failed to get user", zap.Int64("id", in.ID), zap.Error(err))
		return nil, err
	}

	// Store in cache for future requests
	if uc.cache != nil {
		if err := uc.cache.Set(ctx, u); err != nil {
			uc.log.Warn("failed to cache user", zap.Int64("id", in.ID), zap.Error(err))
		}
	}

	return &GetUserResponse{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

// ListUsers retrieves a paginated list of users with optional search functionality.
func (uc *Usecase) ListUsers(ctx context.Context, in ListUsersRequest) (*ListUsersResponse, error) {
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.Limit <= 0 {
		in.Limit = 10
	}
	if in.Limit > 100 {
		in.Limit = 100
	}

	uc.log.Info("listing users", zap.String("query", in.Query), zap.Int64("page", in.Page), zap.Int64("limit", in.Limit))

	domainUsers, err := uc.repo.List(ctx, in.Query, in.Page, in.Limit)
	if err != nil {
		// Handle validation errors from repository layer
		if strings.Contains(err.Error(), "invalid search query") {
			uc.log.Warn("invalid search query in usecase", zap.String("query", in.Query), zap.Error(err))
			return nil, fmt.Errorf("invalid search query: %s", strings.TrimPrefix(err.Error(), "invalid search query: "))
		}
		uc.log.Error("failed to list users", zap.String("query", in.Query), zap.Int64("page", in.Page), zap.Int64("limit", in.Limit), zap.Error(err))
		return nil, err
	}

	users := make([]User, len(domainUsers))
	for i, du := range domainUsers {
		users[i] = User{
			ID:    du.ID,
			Name:  du.Name,
			Email: du.Email,
		}
	}

	return &ListUsersResponse{
		Users: users,
	}, nil
}

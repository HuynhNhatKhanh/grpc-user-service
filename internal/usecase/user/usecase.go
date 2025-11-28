package user

import (
	"context"
	"errors"

	"go.uber.org/zap"

	domain "grpc-user-service/internal/domain/user"
)

// Repository defines the interface for user data access
type Repository interface {
	Create(ctx context.Context, u *domain.User) (int64, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
}

// Usecase implements the user business logic
type Usecase struct {
	repo Repository
	log  *zap.Logger
}

// New creates a new user usecase
func New(r Repository, log *zap.Logger) *Usecase {
	return &Usecase{repo: r, log: log}
}

// CreateUser creates a new user with validation
func (uc *Usecase) CreateUser(ctx context.Context, name, email string) (int64, error) {
	uc.log.Info("creating user", zap.String("name", name), zap.String("email", email))

	// Validate input
	if name == "" {
		uc.log.Warn("create user validation failed", zap.String("reason", "name required"))
		return 0, errors.New("name is required")
	}
	if email == "" {
		uc.log.Warn("create user validation failed", zap.String("reason", "email required"))
		return 0, errors.New("email is required")
	}

	// Business logic: create user
	id, err := uc.repo.Create(ctx, &domain.User{
		Name:  name,
		Email: email,
	})
	if err != nil {
		uc.log.Error("failed to create user", zap.Error(err))
		return 0, err
	}
	return id, nil
}

// GetUser retrieves a user by ID
func (uc *Usecase) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	// Validate input
	if id <= 0 {
		uc.log.Warn("get user validation failed", zap.Int64("id", id), zap.String("reason", "invalid id"))
		return nil, errors.New("invalid user id")
	}

	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.log.Error("failed to get user", zap.Int64("id", id), zap.Error(err))
		return nil, err
	}
	return u, nil
}

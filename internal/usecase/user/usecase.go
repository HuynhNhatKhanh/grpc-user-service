package user

import (
	"context"
	"errors"
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
}

// New creates a new user usecase
func New(r Repository) *Usecase {
	return &Usecase{repo: r}
}

// CreateUser creates a new user with validation
func (uc *Usecase) CreateUser(ctx context.Context, name, email string) (int64, error) {
	// Validate input
	if name == "" {
		return 0, errors.New("name is required")
	}
	if email == "" {
		return 0, errors.New("email is required")
	}

	// Business logic: create user
	return uc.repo.Create(ctx, &domain.User{
		Name:  name,
		Email: email,
	})
}

// GetUser retrieves a user by ID
func (uc *Usecase) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	// Validate input
	if id <= 0 {
		return nil, errors.New("invalid user id")
	}

	return uc.repo.GetByID(ctx, id)
}

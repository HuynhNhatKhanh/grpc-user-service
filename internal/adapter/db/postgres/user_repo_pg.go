package postgres

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"grpc-user-service/internal/domain/user"
)

// UserRepoPG implements Repository interface using GORM
type UserRepoPG struct {
	db *gorm.DB
}

// NewUserRepoPG creates a new PostgreSQL user repository
func NewUserRepoPG(db *gorm.DB) *UserRepoPG {
	return &UserRepoPG{db: db}
}

// UserSchema represents the database schema for users
type UserSchema struct {
	ID    int64  `gorm:"primaryKey;autoIncrement"`
	Name  string `gorm:"not null"`
	Email string `gorm:"not null;unique"`
}

// TableName specifies the table name
func (UserSchema) TableName() string {
	return "users"
}

// Create inserts a new user into the database
func (r *UserRepoPG) Create(ctx context.Context, u *user.User) (int64, error) {
	if u == nil {
		return 0, errors.New("user cannot be nil")
	}

	model := UserSchema{
		Name:  u.Name,
		Email: u.Email,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	return model.ID, nil
}

// GetByID retrieves a user by ID from the database
func (r *UserRepoPG) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var model UserSchema
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: id=%d", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user.User{
		ID:    model.ID,
		Name:  model.Name,
		Email: model.Email,
	}, nil
}

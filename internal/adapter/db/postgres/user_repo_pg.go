package postgres

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"grpc-user-service/internal/domain/user"
)

// UserRepoPG implements Repository interface using GORM
type UserRepoPG struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewUserRepoPG creates a new PostgreSQL user repository
func NewUserRepoPG(db *gorm.DB, log *zap.Logger) *UserRepoPG {
	return &UserRepoPG{db: db, log: log}
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
		r.log.Error("failed to create user in db", zap.Error(err), zap.String("email", u.Email))
		return 0, fmt.Errorf("failed to create user: %w", err)
	}

	r.log.Info("user created in db", zap.Int64("id", model.ID))
	return model.ID, nil
}

// GetByID retrieves a user by ID from the database
func (r *UserRepoPG) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var model UserSchema
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("user not found", zap.Int64("id", id))
			return nil, fmt.Errorf("user not found: id=%d", id)
		}
		r.log.Error("failed to get user from db", zap.Error(err), zap.Int64("id", id))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user.User{
		ID:    model.ID,
		Name:  model.Name,
		Email: model.Email,
	}, nil
}

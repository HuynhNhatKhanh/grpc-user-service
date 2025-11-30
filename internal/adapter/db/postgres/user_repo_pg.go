package postgres

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"grpc-user-service/internal/domain/user"
	pkgerrors "grpc-user-service/pkg/errors"
	"grpc-user-service/pkg/security"
)

// UserRepoPG implements the Repository interface using PostgreSQL and GORM.
type UserRepoPG struct {
	db  *gorm.DB    // GORM database connection
	log *zap.Logger // Structured logger for database operations
}

// NewUserRepoPG creates a new instance of UserRepoPG.
func NewUserRepoPG(db *gorm.DB, log *zap.Logger) *UserRepoPG {
	return &UserRepoPG{db: db, log: log}
}

// UserSchema represents the database schema for the users table.
type UserSchema struct {
	ID    int64  `gorm:"primaryKey;autoIncrement"` // Unique identifier with auto-increment
	Name  string `gorm:"not null"`                 // User's full name (required)
	Email string `gorm:"not null;unique"`          // User's unique email address (required, unique)
}

// TableName specifies the table name for the UserSchema model.
func (UserSchema) TableName() string {
	return "users"
}

// Create inserts a new user into the database.
func (r *UserRepoPG) Create(ctx context.Context, u *user.User) (int64, error) {
	if u == nil {
		return 0, pkgerrors.NewValidationError("user", "user cannot be nil")
	}

	model := UserSchema{
		Name:  u.Name,
		Email: u.Email,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		r.log.Error("failed to create user in db", zap.Error(err), zap.String("email", u.Email))
		return 0, pkgerrors.NewInternalError("failed to create user", err)
	}

	r.log.Info("user created in db", zap.Int64("id", model.ID))
	return model.ID, nil
}

// Update updates an existing user in the database.
func (r *UserRepoPG) Update(ctx context.Context, u *user.User) (int64, error) {
	if u == nil {
		return 0, pkgerrors.NewValidationError("user", "user cannot be nil")
	}

	model := UserSchema{
		ID:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		r.log.Error("failed to update user in db", zap.Error(err), zap.Int64("id", u.ID))
		return 0, pkgerrors.NewInternalError("failed to update user", err)
	}

	r.log.Info("user updated in db", zap.Int64("id", model.ID))
	return model.ID, nil
}

// Delete removes a user from the database by ID.
func (r *UserRepoPG) Delete(ctx context.Context, id int64) (int64, error) {
	if id <= 0 {
		return 0, pkgerrors.NewValidationError("id", "invalid user id")
	}

	if err := r.db.WithContext(ctx).Delete(&UserSchema{}, id).Error; err != nil {
		r.log.Error("failed to delete user in db", zap.Error(err), zap.Int64("id", id))
		return 0, pkgerrors.NewInternalError("failed to delete user", err)
	}

	r.log.Info("user deleted in db", zap.Int64("id", id))
	return id, nil
}

// GetByID retrieves a user from the database by their unique ID.
func (r *UserRepoPG) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var model UserSchema
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Warn("user not found", zap.Int64("id", id))
			return nil, pkgerrors.NewNotFoundError("user", fmt.Sprintf("user not found: id=%d", id))
		}
		r.log.Error("failed to get user from db", zap.Error(err), zap.Int64("id", id))
		return nil, pkgerrors.NewInternalError("failed to get user", err)
	}

	return &user.User{
		ID:    model.ID,
		Name:  model.Name,
		Email: model.Email,
	}, nil
}

// GetByEmail retrieves a user from the database by their email address.
func (r *UserRepoPG) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var model UserSchema
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.log.Debug("user not found by email", zap.String("email", email))
			return nil, nil // Return nil for not found case (no error)
		}
		r.log.Error("failed to get user by email from db", zap.Error(err), zap.String("email", email))
		return nil, pkgerrors.NewInternalError("failed to get user by email", err)
	}

	return &user.User{
		ID:    model.ID,
		Name:  model.Name,
		Email: model.Email,
	}, nil
}

// List retrieves users from the database with pagination and search functionality.
func (r *UserRepoPG) List(ctx context.Context, query string, page, limit int64) ([]user.User, int64, error) {
	// Validate and sanitize search query
	validatedQuery, err := security.ValidateSearchQuery(query)
	if err != nil {
		r.log.Warn("invalid search query", zap.String("query", query), zap.Error(err))
		return nil, 0, pkgerrors.NewValidationError("query", err.Error())
	}

	var models []UserSchema

	// Build query with proper escaping
	dbQuery := r.db.WithContext(ctx)
	if validatedQuery != "" {
		// Sanitize for LIKE operation and escape wildcards
		sanitizedQuery := security.SanitizeSearchString(validatedQuery)
		searchPattern := "%" + sanitizedQuery + "%"

		if r.db.Dialector.Name() == "postgres" {
			dbQuery = dbQuery.Where("name ILIKE ? OR email ILIKE ?", searchPattern, searchPattern)
		} else {
			// Fallback for SQLite (tests) - Ensure case-insensitive search and escape character
			// Note: We use raw SQL for LOWER() to be compatible with GORM
			dbQuery = dbQuery.Where("LOWER(name) LIKE LOWER(?) ESCAPE '\\' OR LOWER(email) LIKE LOWER(?) ESCAPE '\\'", searchPattern, searchPattern)
		}
	}

	// Count total records
	var total int64
	countQuery := dbQuery
	if err := countQuery.Model(&UserSchema{}).Count(&total).Error; err != nil {
		r.log.Error("failed to count users from db", zap.Error(err), zap.String("query", validatedQuery))
		return nil, 0, pkgerrors.NewInternalError("failed to count users", err)
	}

	// Get paginated results
	if err := dbQuery.Offset(int((page - 1) * limit)).Limit(int(limit)).Find(&models).Error; err != nil {
		r.log.Error("failed to list users from db", zap.Error(err), zap.String("query", validatedQuery), zap.Int64("page", page), zap.Int64("limit", limit))
		return nil, 0, pkgerrors.NewInternalError("failed to list users", err)
	}

	users := make([]user.User, len(models))
	for i, model := range models {
		users[i] = user.User{
			ID:    model.ID,
			Name:  model.Name,
			Email: model.Email,
		}
	}

	return users, total, nil
}

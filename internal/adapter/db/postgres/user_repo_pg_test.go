package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"grpc-user-service/internal/domain/user"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate the schema
	err = db.AutoMigrate(&UserSchema{})
	require.NoError(t, err)

	return db
}

func TestUserRepoPG_List_SQLInjectionProtection(t *testing.T) {
	db := setupTestDB(t)
	logger := zaptest.NewLogger(t)
	repo := NewUserRepoPG(db, logger)

	// Insert test data
	testUsers := []user.User{
		{ID: 1, Name: "John Doe", Email: "john@example.com"},
		{ID: 2, Name: "Jane Smith", Email: "jane@example.com"},
		{ID: 3, Name: "Admin User", Email: "admin@example.com"},
	}

	for _, u := range testUsers {
		_, err := repo.Create(context.Background(), &u)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		query       string
		expectError bool
		errorMsg    string
		expectCount int
	}{
		{
			name:        "valid search query",
			query:       "john",
			expectError: false,
			expectCount: 1, // Should find "John Doe"
		},
		{
			name:        "empty search query",
			query:       "",
			expectError: false,
			expectCount: 3, // Should find all users
		},
		{
			name:        "SQL injection attempt - UNION",
			query:       "john UNION SELECT * FROM users",
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "SQL injection attempt - OR condition",
			query:       "john OR 1=1",
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "SQL injection attempt - DROP",
			query:       "john; DROP TABLE users",
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "SQL injection attempt - comment",
			query:       "john --",
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "XSS attempt",
			query:       "<script>alert('xss')</script>",
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "query too long",
			query:       string(make([]rune, 101)), // Max is 100
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "invalid characters",
			query:       "john&doe",
			expectError: true,
			errorMsg:    "invalid search query",
		},
		{
			name:        "valid email search",
			query:       "example.com",
			expectError: false,
			expectCount: 3, // Should find all users with example.com
		},
		{
			name:        "valid special characters",
			query:       "john.doe+test@example.com",
			expectError: false,
			expectCount: 0, // No match but should not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			users, err := repo.List(ctx, tt.query, 1, 10)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, users)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, users)
				assert.Equal(t, tt.expectCount, len(users))
			}
		})
	}
}

func TestUserRepoPG_List_WildcardEscaping(t *testing.T) {
	db := setupTestDB(t)
	logger := zaptest.NewLogger(t)
	repo := NewUserRepoPG(db, logger)

	// Insert test data with special characters
	testUsers := []user.User{
		{ID: 1, Name: "John%Test", Email: "john%test@example.com"},
		{ID: 2, Name: "Jane_Test", Email: "jane_test@example.com"},
		{ID: 3, Name: "Admin", Email: "admin@example.com"},
	}

	for _, u := range testUsers {
		_, err := repo.Create(context.Background(), &u)
		require.NoError(t, err)
	}

	// Test that wildcards are properly escaped
	tests := []struct {
		name        string
		query       string
		expectCount int
		description string
	}{
		{
			name:        "search for percent literal",
			query:       "John%Test",
			expectCount: 1,
			description: "Should find exact match with % character",
		},
		{
			name:        "search for underscore literal",
			query:       "Jane_Test",
			expectCount: 1,
			description: "Should find exact match with _ character",
		},
		{
			name:        "search with percent in query",
			query:       "john%",
			expectCount: 1,
			description: "Should escape % and search for literal %",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			users, err := repo.List(ctx, tt.query, 1, 10)

			require.NoError(t, err)
			assert.NotNil(t, users)
			assert.Equal(t, tt.expectCount, len(users), tt.description)
		})
	}
}

func TestUserRepoPG_List_CaseInsensitiveSearch(t *testing.T) {
	db := setupTestDB(t)
	logger := zaptest.NewLogger(t)
	repo := NewUserRepoPG(db, logger)

	// Insert test data
	testUsers := []user.User{
		{ID: 1, Name: "John Doe", Email: "JOHN@EXAMPLE.COM"},
		{ID: 2, Name: "jane smith", Email: "jane@example.com"},
		{ID: 3, Name: "ADMIN User", Email: "admin@example.com"},
	}

	for _, u := range testUsers {
		_, err := repo.Create(context.Background(), &u)
		require.NoError(t, err)
	}

	tests := []struct {
		name        string
		query       string
		expectCount int
	}{
		{
			name:        "lowercase search",
			query:       "john",
			expectCount: 1, // Should find "John Doe" and "JOHN@EXAMPLE.COM"
		},
		{
			name:        "uppercase search",
			query:       "JOHN",
			expectCount: 1, // Should find "John Doe" and "JOHN@EXAMPLE.COM"
		},
		{
			name:        "mixed case search",
			query:       "Admin",
			expectCount: 2, // Should find "ADMIN User" and "admin@example.com"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			users, err := repo.List(ctx, tt.query, 1, 10)

			require.NoError(t, err)
			assert.NotNil(t, users)
			assert.Equal(t, tt.expectCount, len(users))
		})
	}
}

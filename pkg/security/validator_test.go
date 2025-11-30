package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSearchQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		expectError bool
		errorMsg    string
		expected    string
	}{
		{
			name:     "valid empty query",
			query:    "",
			expected: "",
		},
		{
			name:     "valid simple query",
			query:    "john",
			expected: "john",
		},
		{
			name:     "valid query with spaces",
			query:    "john doe",
			expected: "john doe",
		},
		{
			name:     "valid email-like query",
			query:    "john@example.com",
			expected: "john@example.com",
		},
		{
			name:     "valid query with allowed punctuation",
			query:    "john-doe_123",
			expected: "john-doe_123",
		},
		{
			name:        "query too long",
			query:       string(make([]rune, MaxSearchQueryLength+1)),
			expectError: true,
			errorMsg:    "search query too long",
		},
		{
			name:        "SQL injection attempt - UNION",
			query:       "john UNION SELECT * FROM users",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:        "SQL injection attempt - OR condition",
			query:       "john OR 1=1",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:        "SQL injection attempt - comment",
			query:       "john --",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:        "SQL injection attempt - DROP",
			query:       "john; DROP TABLE users",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:        "XSS attempt - script",
			query:       "<script>alert('xss')</script>",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:        "invalid characters",
			query:       "john&doe",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:        "invalid characters - semicolon",
			query:       "john;doe",
			expectError: true,
			errorMsg:    "search query contains invalid characters",
		},
		{
			name:     "valid query with leading/trailing spaces",
			query:    "  john doe  ",
			expected: "john doe",
		},
		{
			name:     "valid query with special allowed chars",
			query:    "john.doe+test@example.com",
			expected: "john.doe+test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateSearchQuery(tt.query)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeSearchString(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "empty string",
			query:    "",
			expected: "",
		},
		{
			name:     "normal string",
			query:    "john",
			expected: "john",
		},
		{
			name:     "string with percent wildcard",
			query:    "john%",
			expected: "john\\%",
		},
		{
			name:     "string with underscore wildcard",
			query:    "john_doe",
			expected: "john\\_doe",
		},
		{
			name:     "string with multiple wildcards",
			query:    "%john_%",
			expected: "\\%john\\_\\%",
		},
		{
			name:     "complex string",
			query:    "test@example.com",
			expected: "test@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeSearchString(tt.query)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidSearchChar(t *testing.T) {
	tests := []struct {
		name     string
		char     rune
		expected bool
	}{
		{
			name:     "lowercase letter",
			char:     'a',
			expected: true,
		},
		{
			name:     "uppercase letter",
			char:     'Z',
			expected: true,
		},
		{
			name:     "digit",
			char:     '5',
			expected: true,
		},
		{
			name:     "space",
			char:     ' ',
			expected: true,
		},
		{
			name:     "hyphen",
			char:     '-',
			expected: true,
		},
		{
			name:     "underscore",
			char:     '_',
			expected: true,
		},
		{
			name:     "dot",
			char:     '.',
			expected: true,
		},
		{
			name:     "at symbol",
			char:     '@',
			expected: true,
		},
		{
			name:     "plus",
			char:     '+',
			expected: true,
		},
		{
			name:     "hash",
			char:     '#',
			expected: true,
		},
		{
			name:     "asterisk",
			char:     '*',
			expected: true,
		},
		{
			name:     "semicolon - invalid",
			char:     ';',
			expected: false,
		},
		{
			name:     "ampersand - invalid",
			char:     '&',
			expected: false,
		},
		{
			name:     "less than - invalid",
			char:     '<',
			expected: false,
		},
		{
			name:     "greater than - invalid",
			char:     '>',
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSearchChar(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMaxSearchQueryLength(t *testing.T) {
	// Test that the constant is set to a reasonable value
	assert.Equal(t, 100, MaxSearchQueryLength)
}

// BenchmarkValidateSearchQuery benchmarks the validation function
func BenchmarkValidateSearchQuery(b *testing.B) {
	query := "john doe example"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ValidateSearchQuery(query)
	}
}

// BenchmarkSanitizeSearchString benchmarks the sanitization function
func BenchmarkSanitizeSearchString(b *testing.B) {
	query := "john%doe_example"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = SanitizeSearchString(query)
	}
}

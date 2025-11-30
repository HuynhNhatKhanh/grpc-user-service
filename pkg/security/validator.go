package security

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

const (
	// MaxSearchQueryLength defines the maximum allowed length for search queries
	MaxSearchQueryLength = 100
)

// dangerousPatterns contains regex patterns that could indicate SQL injection attempts
var dangerousPatterns = []*regexp.Regexp{
	// SQL injection patterns
	regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|create|alter|exec|execute)`),
	regexp.MustCompile(`(?i)(or|and)\s+\d+\s*=\s*\d+`),
	regexp.MustCompile(`(?i)(or|and)\s+['"].*['"]\s*=\s*['"].*['"]`),
	regexp.MustCompile(`(?i)(--|#|/\*|\*/)`),
	regexp.MustCompile(`(?i)(waitfor|delay|benchmark|sleep)`),
	
	// XSS patterns (if used for web display)
	regexp.MustCompile(`(?i)(<script|</script|javascript:|vbscript:|onload=|onerror=)`),
}

// ValidateSearchQuery validates and sanitizes a search query to prevent SQL injection
func ValidateSearchQuery(query string) (string, error) {
	if query == "" {
		return "", nil
	}

	// Check length
	if len(query) > MaxSearchQueryLength {
		return "", errors.New("search query too long")
	}

	// Trim whitespace
	query = strings.TrimSpace(query)

	// Check for dangerous patterns
	lowerQuery := strings.ToLower(query)
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(lowerQuery) {
			return "", errors.New("search query contains invalid characters")
		}
	}

	// Additional character validation - allow only safe characters
	for _, char := range query {
		if !isValidSearchChar(char) {
			return "", errors.New("search query contains invalid characters")
		}
	}

	return query, nil
}

// isValidSearchChar checks if a character is safe for search queries
func isValidSearchChar(char rune) bool {
	// Allow letters, numbers, spaces, and common punctuation
	return unicode.IsLetter(char) || unicode.IsNumber(char) ||
		char == ' ' || char == '-' || char == '_' || char == '.' ||
		char == '@' || char == '+' || char == '#' || char == '*'
}

// SanitizeSearchString prepares a query string for LIKE operations
func SanitizeSearchString(query string) string {
	if query == "" {
		return ""
	}
	
	// Escape wildcards and other special characters
	query = strings.ReplaceAll(query, "%", "\\%")
	query = strings.ReplaceAll(query, "_", "\\_")
	
	return query
}

package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Common application errors
var (
	ErrNotFound         = NewNotFoundError("resource", "resource not found")
	ErrAlreadyExists    = NewAlreadyExistsError("resource", "resource already exists")
	ErrInvalidArgument  = NewValidationError("", "invalid argument")
	ErrInternal         = NewInternalError("internal server error", nil)
	ErrUnauthorized     = NewInternalError("unauthorized", nil)
	ErrPermissionDenied = NewInternalError("permission denied", nil)
)

// ValidationError represents a validation failure with field-level details
type ValidationError struct {
	Field   string
	Message string
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation failed: %s - %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation failed: %s", e.Message)
}

// GRPCStatus returns the gRPC status for this error
func (e *ValidationError) GRPCStatus() *status.Status {
	return status.New(codes.InvalidArgument, e.Error())
}

// NotFoundError represents a resource not found error
type NotFoundError struct {
	Resource string
	Message  string
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(resource, message string) *NotFoundError {
	return &NotFoundError{
		Resource: resource,
		Message:  message,
	}
}

// Error implements the error interface
func (e *NotFoundError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// GRPCStatus returns the gRPC status for this error
func (e *NotFoundError) GRPCStatus() *status.Status {
	return status.New(codes.NotFound, e.Error())
}

// AlreadyExistsError represents a resource already exists error
type AlreadyExistsError struct {
	Resource string
	Message  string
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(resource, message string) *AlreadyExistsError {
	return &AlreadyExistsError{
		Resource: resource,
		Message:  message,
	}
}

// Error implements the error interface
func (e *AlreadyExistsError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s already exists", e.Resource)
}

// GRPCStatus returns the gRPC status for this error
func (e *AlreadyExistsError) GRPCStatus() *status.Status {
	return status.New(codes.AlreadyExists, e.Error())
}

// InternalError represents an internal server error with context
type InternalError struct {
	Message string
	Err     error
}

// NewInternalError creates a new internal error
func NewInternalError(message string, err error) *InternalError {
	return &InternalError{
		Message: message,
		Err:     err,
	}
}

// Error implements the error interface
func (e *InternalError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *InternalError) Unwrap() error {
	return e.Err
}

// GRPCStatus returns the gRPC status for this error
func (e *InternalError) GRPCStatus() *status.Status {
	return status.New(codes.Internal, e.Message)
}

// GRPCStatuser interface for errors that can provide gRPC status
type GRPCStatuser interface {
	GRPCStatus() *status.Status
}

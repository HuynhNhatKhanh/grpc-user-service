package user

// CreateUserRequest represents the request payload for creating a new user.
type CreateUserRequest struct {
	Name  string `validate:"required,min=3,max=100"`
	Email string `validate:"required,email"`
}

// CreateUserResponse represents the response payload after creating a user.
type CreateUserResponse struct {
	ID int64
}

// UpdateUserRequest represents the request payload for updating an existing user.
type UpdateUserRequest struct {
	ID    int64  `validate:"required"`
	Name  string `validate:"omitempty,min=3,max=100"`
	Email string `validate:"omitempty,email"`
}

// UpdateUserResponse represents the response payload after updating a user.
type UpdateUserResponse struct {
	ID int64
}

// DeleteUserRequest represents the request payload for deleting a user.
type DeleteUserRequest struct {
	ID int64
}

// DeleteUserResponse represents the response payload after deleting a user.
type DeleteUserResponse struct {
	ID int64
}

// GetUserRequest represents the request payload for retrieving a user.
type GetUserRequest struct {
	ID int64
}

// GetUserResponse represents the response payload for user details.
type GetUserResponse struct {
	ID    int64
	Name  string
	Email string
}

// ListUsersRequest represents the request payload for listing users.
// It supports pagination and search functionality.
type ListUsersRequest struct {
	Query string
	Page  int64
	Limit int64
}

// ListUsersResponse represents the response payload for user listing.
type ListUsersResponse struct {
	Users      []User
	Pagination *Pagination
}

// Pagination represents pagination information for list responses.
type Pagination struct {
	Total      int64
	Page       int64
	Limit      int64
	TotalPages int64
}

// User represents a user DTO (Data Transfer Object) for API responses.
type User struct {
	ID    int64
	Name  string
	Email string
}

package user

import "context"

// Usecase defines the interface for user business logic operations.
type Usecase interface {
	CreateUser(ctx context.Context, in CreateUserRequest) (*CreateUserResponse, error)
	UpdateUser(ctx context.Context, in UpdateUserRequest) (*UpdateUserResponse, error)
	DeleteUser(ctx context.Context, in DeleteUserRequest) (*DeleteUserResponse, error)
	GetUser(ctx context.Context, in GetUserRequest) (*GetUserResponse, error)
	ListUsers(ctx context.Context, in ListUsersRequest) (*ListUsersResponse, error)
}

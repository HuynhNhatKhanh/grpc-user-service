package grpc

import (
	"context"

	pb "grpc-user-service/api/gen/go/user"
	"grpc-user-service/internal/usecase/user"
)

// UserServiceServer implements the gRPC user service
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	uc *user.Usecase
}

// NewUserServiceServer creates a new gRPC user service server
func NewUserServiceServer(uc *user.Usecase) *UserServiceServer {
	return &UserServiceServer{uc: uc}
}

// GetUser handles gRPC GetUser request
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	u, err := s.uc.GetUser(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.GetUserResponse{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

// CreateUser handles gRPC CreateUser request
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	id, err := s.uc.CreateUser(ctx, req.Name, req.Email)
	if err != nil {
		return nil, err
	}

	return &pb.CreateUserResponse{
		Id: id,
	}, nil
}

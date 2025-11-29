package grpc

import (
	"context"

	"go.uber.org/zap"

	pb "grpc-user-service/api/gen/go/user"
	"grpc-user-service/internal/usecase/user"
)

// UserServiceServer implements the gRPC user service
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer
	uc  *user.Usecase
	log *zap.Logger
}

// NewUserServiceServer creates a new gRPC user service server
func NewUserServiceServer(uc *user.Usecase, log *zap.Logger) *UserServiceServer {
	return &UserServiceServer{uc: uc, log: log}
}

// CreateUser handles gRPC CreateUser request
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	s.log.Info("gRPC CreateUser request", zap.String("name", req.Name), zap.String("email", req.Email))
	id, err := s.uc.CreateUser(ctx, req.Name, req.Email)
	if err != nil {
		s.log.Error("gRPC CreateUser failed", zap.Error(err))
		return nil, err
	}

	return &pb.CreateUserResponse{
		Id: id,
	}, nil
}

// UpdateUser handles gRPC UpdateUser request
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	s.log.Info("gRPC UpdateUser request", zap.Int64("id", req.Id), zap.String("name", req.Name), zap.String("email", req.Email))
	id, err := s.uc.UpdateUser(ctx, req.Id, req.Name, req.Email)
	if err != nil {
		s.log.Error("gRPC UpdateUser failed", zap.Error(err))
		return nil, err
	}

	return &pb.UpdateUserResponse{
		Id: id,
	}, nil
}

// DeleteUser handles gRPC DeleteUser request
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	s.log.Info("gRPC DeleteUser request", zap.Int64("id", req.Id))
	id, err := s.uc.DeleteUser(ctx, req.Id)
	if err != nil {
		s.log.Error("gRPC DeleteUser failed", zap.Error(err))
		return nil, err
	}

	return &pb.DeleteUserResponse{
		Id: id,
	}, nil
}

// GetUser handles gRPC GetUser request
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.log.Info("gRPC GetUser request", zap.Int64("id", req.Id))
	u, err := s.uc.GetUser(ctx, req.Id)
	if err != nil {
		s.log.Error("gRPC GetUser failed", zap.Error(err))
		return nil, err
	}

	return &pb.GetUserResponse{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

// ListUsers handles gRPC ListUsers request
func (s *UserServiceServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.log.Info("gRPC ListUsers request", zap.String("query", req.Query), zap.Int64("page", req.Page), zap.Int64("limit", req.Limit))
	users, err := s.uc.ListUsers(ctx, req.Query, req.Page, req.Limit)
	if err != nil {
		s.log.Error("gRPC ListUsers failed", zap.Error(err))
		return nil, err
	}
	usersResponse := make([]*pb.GetUserResponse, len(users))
	for i, u := range users {
		usersResponse[i] = &pb.GetUserResponse{
			Id:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		}
	}

	return &pb.ListUsersResponse{
		Users: usersResponse,
	}, nil
}

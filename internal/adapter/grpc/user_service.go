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

package grpc

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "grpc-user-service/api/gen/go/user"
	"grpc-user-service/internal/usecase/user"
)

// UserServiceServer implements the gRPC user service interface.
type UserServiceServer struct {
	pb.UnimplementedUserServiceServer               // Embedded for forward compatibility
	uc                                *user.Usecase // User business logic handler
	log                               *zap.Logger   // Structured logger
}

// NewUserServiceServer creates a new instance of UserServiceServer.
func NewUserServiceServer(uc *user.Usecase, log *zap.Logger) *UserServiceServer {
	return &UserServiceServer{uc: uc, log: log}
}

// mapError converts domain errors to gRPC status errors
func mapError(err error) error {
	if err == nil {
		return nil
	}

	// Check if error implements GRPCStatuser interface (custom pkg/errors types)
	type grpcStatuser interface {
		GRPCStatus() *status.Status
	}

	// Use type assertion to check if error has GRPCStatus method
	if grpcErr, ok := err.(grpcStatuser); ok {
		return grpcErr.GRPCStatus().Err()
	}

	// Default to internal error for any unhandled errors
	return status.Error(codes.Internal, err.Error())
}

// CreateUser handles the gRPC CreateUser request.
func (s *UserServiceServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	s.log.Info("gRPC CreateUser request", zap.String("name", req.Name), zap.String("email", req.Email))
	ucRequest := user.CreateUserRequest{
		Name:  req.GetName(),
		Email: req.GetEmail(),
	}
	id, err := s.uc.CreateUser(ctx, ucRequest)
	if err != nil {
		s.log.Error("gRPC CreateUser failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &pb.CreateUserResponse{
		Id: id.ID,
	}, nil
}

// UpdateUser handles the gRPC UpdateUser request.
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.UpdateUserResponse, error) {
	s.log.Info("gRPC UpdateUser request", zap.Int64("id", req.Id), zap.String("name", req.Name), zap.String("email", req.Email))
	ucRequest := user.UpdateUserRequest{
		ID:    req.Id,
		Name:  req.GetName(),
		Email: req.GetEmail(),
	}
	id, err := s.uc.UpdateUser(ctx, ucRequest)
	if err != nil {
		s.log.Error("gRPC UpdateUser failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &pb.UpdateUserResponse{
		Id: id.ID,
	}, nil
}

// DeleteUser handles the gRPC DeleteUser request.
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	s.log.Info("gRPC DeleteUser request", zap.Int64("id", req.Id))
	ucRequest := user.DeleteUserRequest{
		ID: req.Id,
	}
	id, err := s.uc.DeleteUser(ctx, ucRequest)
	if err != nil {
		s.log.Error("gRPC DeleteUser failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &pb.DeleteUserResponse{
		Id: id.ID,
	}, nil
}

// GetUser handles the gRPC GetUser request.
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.log.Info("gRPC GetUser request", zap.Int64("id", req.Id))
	ucRequest := user.GetUserRequest{
		ID: req.Id,
	}
	u, err := s.uc.GetUser(ctx, ucRequest)
	if err != nil {
		s.log.Error("gRPC GetUser failed", zap.Error(err))
		return nil, mapError(err)
	}

	return &pb.GetUserResponse{
		Id:    u.ID,
		Name:  u.Name,
		Email: u.Email,
	}, nil
}

// ListUsers handles the gRPC ListUsers request.
func (s *UserServiceServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	s.log.Info("gRPC ListUsers request", zap.String("query", req.Query), zap.Int64("page", req.Page), zap.Int64("limit", req.Limit))
	ucRequest := user.ListUsersRequest{
		Query: req.Query,
		Page:  req.Page,
		Limit: req.Limit,
	}
	usersResponse, err := s.uc.ListUsers(ctx, ucRequest)
	if err != nil {
		s.log.Error("gRPC ListUsers failed", zap.Error(err))
		return nil, mapError(err)
	}

	pbUsers := make([]*pb.GetUserResponse, len(usersResponse.Users))
	for i, u := range usersResponse.Users {
		pbUsers[i] = &pb.GetUserResponse{
			Id:    u.ID,
			Name:  u.Name,
			Email: u.Email,
		}
	}

	var pbPagination *pb.Pagination
	if usersResponse.Pagination != nil {
		pbPagination = &pb.Pagination{
			Total:      usersResponse.Pagination.Total,
			Page:       usersResponse.Pagination.Page,
			Limit:      usersResponse.Pagination.Limit,
			TotalPages: usersResponse.Pagination.TotalPages,
		}
	}

	return &pb.ListUsersResponse{
		Users:      pbUsers,
		Pagination: pbPagination,
	}, nil
}

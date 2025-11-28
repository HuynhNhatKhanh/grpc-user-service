package logger

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// RequestIDInterceptor is a gRPC interceptor that adds a request ID to the context
func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		// Generate a new request ID
		requestID := uuid.New().String()

		// Add request ID to context
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Call the handler with the new context
		return handler(ctx, req)
	}
}

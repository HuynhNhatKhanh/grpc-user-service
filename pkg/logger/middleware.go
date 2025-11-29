package logger

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

// RequestIDInterceptor creates a gRPC unary server interceptor that adds a unique request ID to the context.
// The request ID is generated using UUID v4 and added to the context for traceability.
// This enables request correlation across logs and helps with debugging distributed systems.
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

# Build stage
FROM golang:1.25.4-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files first (for better caching)
COPY go.mod go.sum ./
RUN go mod download

# Install buf and protoc plugins
RUN go install github.com/bufbuild/buf/cmd/buf@latest && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest

# Copy all source code
COPY . .

# Generate proto files (this creates api/gen/go/user/*.pb.go)
RUN buf dep update && buf generate

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/grpc-user-service cmd/api/main.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/bin/grpc-user-service .

# Copy config file if exists
COPY --from=builder /app/app.env* ./

# Copy migrations
COPY --from=builder /app/deployments/migrations ./deployments/migrations

# Expose ports
EXPOSE 50051 8080

# Run the application
CMD ["./grpc-user-service"]

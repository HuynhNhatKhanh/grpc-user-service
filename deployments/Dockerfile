# Build stage
FROM golang:1.25.4-alpine AS builder

RUN apk add --no-cache git make build-base wget

WORKDIR /app

# Go mod
COPY go.mod go.sum ./
RUN go mod download

# Install buf (binary) 
RUN wget https://github.com/bufbuild/buf/releases/latest/download/buf-Linux-x86_64 \
    -O /usr/local/bin/buf && \
    chmod +x /usr/local/bin/buf

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest

COPY . .

# Generate proto
RUN buf dep update && buf generate

# Build app
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/grpc-user-service cmd/api/main.go

# Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /root/

COPY --from=builder /app/bin/grpc-user-service .
COPY --from=builder /app/app.env* ./
COPY --from=builder /app/deployments/migrations ./deployments/migrations

EXPOSE 50051 8080
CMD ["./grpc-user-service"]

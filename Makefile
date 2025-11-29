SHELL := bash

.PHONY: help install-tools proto clean-proto regen-proto buf-dep-update buf-mod-update lint-proto lint format test run build version docker-build docker-up docker-down docker-logs clean migrate-up migrate-down migrate-force migrate-create

# Default target - show help
help:
	@echo "Available targets:"
	@echo "  install-tools      - Install protoc plugins and development tools"
	@echo "  proto              - Generate protobuf and gRPC code using buf"
	@echo "  clean-proto        - Clean generated proto files"
	@echo "  regen-proto        - Clean and regenerate all proto files"
	@echo "  buf-dep-update     - Update buf dependencies"
	@echo "  buf-mod-update     - Update buf module"
	@echo "  lint-proto         - Lint proto files using buf"
	@echo "  lint               - Run golangci-lint"
	@echo "  format             - Format Go code"
	@echo "  test               - Run tests"
	@echo "  run                - Run the application locally"
	@echo "  build              - Build the application binary"
	@echo "  migrate-up         - Apply all pending migrations"
	@echo "  migrate-down       - Rollback last migration"
	@echo "  migrate-create     - Create new migration file (name=xxx)"
	@echo "  docker-build       - Build Docker image"
	@echo "  docker-up          - Start services with docker-compose"
	@echo "  docker-down        - Stop services with docker-compose"
	@echo "  docker-logs        - Show docker-compose logs"
	@echo "  clean              - Clean all generated files and binaries"
	@echo "  local              - Run the application locally"

# Install protoc plugins and tools
install-tools:
	@echo "Installing protoc plugins..."
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "Installing buf..."
	go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "All tools installed successfully"

# Generate protobuf and gRPC code using buf
proto:
	buf generate

# Clean generated files
clean-proto:
	rm -rf api/gen/go

# Regenerate all proto files (clean + generate)
regen-proto: clean-proto proto

# Update buf dependencies
buf-dep-update:
	buf dep update

# Update buf module
buf-mod-update:
	buf mod update

# Lint proto files using buf
lint-proto:
	buf lint

# Format proto files using buf
format-proto:
	buf format -w

# Check for breaking changes (buf)
breaking:
	buf breaking --against .git#branch=main

# Go code quality commands
lint:
	golangci-lint run ./...

# Go code formatter
format:
	gofmt -s -w .

# Run tests
test:
	go test -v -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Run the application locally
run:
	go run cmd/api/main.go

# Build the application binary
build:
	go build -o bin/grpc-user-service cmd/api/main.go

version:
	@echo "1.0.0"

# Download Go modules
deps:
	go mod download
	go mod tidy

# Migration commands
migrate-up:
	@echo "Applying migrations..."
	@bash -c 'if [ -z "$$DB_DSN" ]; then export DB_DSN="postgres://postgres:postgres@localhost:5432/grpc_user_service?sslmode=disable"; fi; migrate -path deployments/migrations -database "$$DB_DSN" up'
	@echo "Migrations applied"

migrate-up-docker:
	@echo "Applying migrations using Docker..."
	docker run --rm -v $(CURDIR)/deployments/migrations:/migrations --network host migrate/migrate -path=/migrations/ -database "postgres://postgres:postgres@host.docker.internal:5432/grpc_user_service?sslmode=disable" up
	@echo "Migrations applied successfully"

migrate-down:
	@echo "Rolling back last migration..."
	@bash -c 'if [ -z "$$DB_DSN" ]; then export DB_DSN="postgres://postgres:postgres@localhost:5432/grpc_user_service?sslmode=disable"; fi; migrate -path deployments/migrations -database "$$DB_DSN" down 1'
	@echo "Migration rolled back"

migrate-force:
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make migrate-force VERSION=1"; \
		exit 1; \
	fi
	@echo "Forcing migration version to $(VERSION)..."
	@bash -c 'if [ -z "$$DB_DSN" ]; then export DB_DSN="postgres://postgres:postgres@localhost:5432/grpc_user_service?sslmode=disable"; fi; migrate -path deployments/migrations -database "$$DB_DSN" force $(VERSION)'
	@echo "Migration version set to $(VERSION)"

migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "Error: name is required. Usage: make migrate-create name=add_users_table"; \
		exit 1; \
	fi
	@echo "Creating migration: $(name)..."
	@TIMESTAMP=$$(date +%s); \
	touch deployments/migrations/$${TIMESTAMP}_$(name).up.sql; \
	touch deployments/migrations/$${TIMESTAMP}_$(name).down.sql
	@echo "Created migration files:"
	@ls -la deployments/migrations/*$(name)*

# Docker commands
docker-build:
	docker build -t grpc-user-service:latest -f deployments/Dockerfile .

docker-up:
	docker-compose -f deployments/docker-compose.yml up -d

docker-bu:
	docker-compose -f deployments/docker-compose.yml up -d --build

docker-down:
	docker-compose -f deployments/docker-compose.yml down

docker-logs:
	docker-compose -f deployments/docker-compose.yml logs -f

docker-restart: docker-down docker-up

# Clean all generated files and binaries
clean:
	rm -rf api/gen/go
	rm -rf bin
	rm -f coverage.out coverage.html
	rm -f buf.lock

# Run local application	
local:
	go run cmd/api/main.go

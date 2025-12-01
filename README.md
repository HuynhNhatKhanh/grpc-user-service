# gRPC User Service

A production-ready microservice built with **Go**, **Protocol Buffers**, **gRPC**, **Gin**, and **Clean Architecture** principles. This project demonstrates best practices for building scalable, maintainable, and testable backend services with **three API delivery mechanisms**: gRPC, gRPC-Gateway REST, and Gin REST API.

## 🎯 Project Goals

This project is designed to **showcase**:

- **Protocol Buffers** for API contract definition and code generation
- **Performance testing** across different API protocols (gRPC vs REST)
- **Clean Architecture** with shared business logic across multiple transport layers
- **Production-ready** features (caching, rate limiting, logging, graceful shutdown)

### Key Features

- **Clean Architecture** - Clear separation between business logic and infrastructure with DI container
- **Three API Protocols** - gRPC, gRPC-Gateway REST, and Gin REST API (all sharing same business logic)
- **Config Validation** - Comprehensive validation at startup (40+ rules) for fail-fast error detection
- **Graceful Shutdown** - Configurable timeout (1-300s) for different environments
- **gRPC + gRPC-Gateway** - Native gRPC with automatic REST API generation
- **Gin REST API** - High-performance HTTP API with middleware support
- **Redis Caching** - Cache-aside pattern with automatic invalidation
- **Rate Limiting** - gRPC interceptor-based rate limiting with Redis
- **Dependency Inversion** - Business logic independent of frameworks and databases
- **Type-safe** - Leveraging Go's strong typing and Protocol Buffers
- **Production-ready** - Structured logging, error handling, and panic recovery
- **Testable** - Interface-based design for easy mocking and testing

## 📋 Protocol Buffers (Protocol Buffers)

### API Contract Definition

The service uses **Protocol Buffers** to define the API contract in `api/proto/user.proto`:

```protobuf
syntax = "proto3";
package user;

option go_package = "grpc-user-service/api/gen/go/user";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}

message User {
  int64 id = 1;
  string name = 2;
  string email = 3;
}
```

### Code Generation

**Generated Go code** from protobuf:

```bash
# Generate all protobuf code
make proto-gen

# Generated files:
api/gen/go/user/
├── user.pb.go          # Go structs for messages
├── user_grpc.pb.go     # gRPC service interfaces
└── user.pb.gw.go       # gRPC-Gateway HTTP handlers
```

### Benefits of Protocol Buffers

- **Language-agnostic** - Same contract works for Go, Java, Python, etc.
- **Type-safe** - Compile-time type checking for all API messages
- **Code generation** - Automatic generation of client/server code
- **Versioning** - Built-in support for API evolution
- **Performance** - Binary serialization more efficient than JSON

## ⚡ Performance Testing

### Benchmark Overview

This project includes **comprehensive performance testing** comparing **three API protocols**:

- **gRPC** - Binary protocol with HTTP/2
- **Gin REST API** - HTTP/1.1 with JSON
- **gRPC-Gateway REST** - HTTP/1.1 JSON via gRPC translation

### Test Results (Mac mini M4)

**Performance Leaderboard (CreateUser operation):**

| Protocol         | Latency (ns/op) | Throughput (ops/sec) | Memory (B/op) | Efficiency |
| ---------------- | --------------- | -------------------- | ------------- | ---------- |
| **gRPC**         | 101,635         | 9,838                | 13,035        | 🥇 Best    |
| **Gin REST**     | 401,614         | 2,488                | 43,108        | 🥈 Good    |
| **REST Gateway** | 442,655         | 2,259                | 56,006        | 🥉 OK      |

**Key Insights:**

- gRPC is **4x faster** than REST APIs
- gRPC uses **3x less memory** than REST APIs
- All protocols share **identical business logic** (Clean Architecture)
- Performance difference purely from transport layer

### Run Performance Tests

```bash
# Quick benchmark comparison
make benchmark

# Individual protocol tests
make benchmark-grpc     # gRPC only
make benchmark-gin      # Gin REST only
make benchmark-rest     # gRPC-Gateway only

# Advanced profiling
make benchmark-cpu      # CPU profiling
make benchmark-mem      # Memory profiling
```

---

## 🏗️ Clean Architecture

This project follows **Clean Architecture** principles with clear dependency rules:

```
┌─────────────────────────────────────────────────────────────┐
│                     Frameworks & Drivers                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              adapter/ (Infrastructure)                 │ │
│  │  • grpc/      - gRPC server implementation            │ │
│  │  • http/      - HTTP handlers & middleware            │ │
│  │  • gin/       - Gin REST API handlers & middleware    │ │
│  │  • db/        - Database implementations (Postgres)   │ │
│  │  • cache/     - Cache implementations (Redis)         │ │
│  └────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │ depends on ↓
┌───────────────────────────┴─────────────────────────────────┐
│                    Application Business Rules                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              usecase/ (Business Logic)                 │ │
│  │  • Defines interfaces (repositories, services)        │ │
│  │  • Implements business rules & validation             │ │
│  │  • Independent of frameworks & databases              │ │
│  └────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │ depends on ↓
┌───────────────────────────┴─────────────────────────────────┐
│                    Enterprise Business Rules                 │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              domain/ (Entities)                        │ │
│  │  • Pure Go structs                                     │ │
│  │  • No external dependencies                            │ │
│  │  • Core business models                                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Dependency Rule

**Dependencies point inward**: Outer layers can depend on inner layers, but inner layers never depend on outer layers.

```
adapter → usecase → domain
  ↓         ↓         ↓
gRPC    Business   Pure
HTTP    Logic      Models
Gin     Rules
DB
Cache
```

## 📁 Project Structure

```
grpc-user-service/
├── api/                          # API definitions
│   ├── proto/                    # Protocol Buffer definitions
│   │   └── user/
│   │       └── user.proto        # User service API contract
│   └── gen/                      # Generated code
│       └── go/
│           └── user/
│               ├── user.pb.go    # gRPC stubs
│               └── user_grpc.pb.go
│
├── cmd/                          # Application entrypoints
│   └── api/
│       ├── main.go               # Main server application
│       ├── app/
│       │   └── app.go            # Application lifecycle management
│       ├── di/                   # NEW: Dependency Injection
│       │   └── container.go      # DI container for all dependencies
│       ├── infrastructure/       # NEW: Infrastructure setup
│       │   ├── database.go       # Database initialization
│       │   └── cache.go          # Redis initialization
│       └── server/
│           ├── server.go         # Server lifecycle
│           ├── grpc.go           # gRPC setup
│           ├── http.go           # HTTP gateway setup
│           └── signal.go         # Signal handling
│
├── internal/                     # Private application code
│   ├── domain/                   # 🟢 Enterprise Business Rules
│   │   └── user/
│   │       ├── entity.go         # User entity (pure Go)
│   │       └── pagination.go     # Pagination models
│   │
│   ├── usecase/                  # 🟡 Application Business Rules
│   │   └── user/
│   │       ├── interface.go      # UserUsecase interface definition
│   │       ├── usecase.go        # Business logic & repository interfaces
│   │       └── dto.go            # Data transfer objects
│   │
│   ├── adapter/                  # 🔴 Interface Adapters
│   │   ├── grpc/                 # gRPC transport layer
│   │   │   ├── user_service.go   # gRPC → Usecase adapter
│   │   │   └── middleware/       # gRPC middleware (rate limiting)
│   │   ├── gin/                  # Gin REST API transport layer
│   │   │   ├── handler/          # Gin HTTP handlers
│   │   │   ├── middleware/       # Gin middleware (logger, recovery, rate limiting)
│   │   │   └── router/           # Gin router configuration
│   │   ├── db/                   # Database implementations
│   │   │   └── postgres/         # PostgreSQL repository
│   │   └── cache/                # Cache implementations
│   │       └── user_cache.go     # Redis cache
│   │
│   └── config/                   # Configuration with validation
│       └── config.go             # Config loading & validation
│   │   ├── http/                 # HTTP handlers
│   │   ├── db/                   # Database implementations
│   │   │   └── postgres/         # PostgreSQL repository
│   │   └── cache/                # Cache implementations
│   │       └── redis/            # Redis cache
│   │
│   └── server/                   # Server setup & lifecycle
│       ├── grpc.go               # gRPC server
│       └── gateway.go            # gRPC-Gateway (REST)
│
├── deployments/                  # Deployment configurations
│   ├── Dockerfile
│   ├── docker-compose.yml
│   └── migrations/               # Database migrations
│
├── test/                         # Test files
│   ├── benchmark/                # Performance benchmarks
│   └── integration/              # Integration tests
│
├── pkg/                          # Shared packages
│   ├── errors/                   # Error handling
│   ├── logger/                   # Structured logging
│   ├── redis/                    # Redis client wrapper
│   └── security/                 # Validation utilities
│
├── scripts/                      # Build & utility scripts
├── buf.yaml
├── buf.gen.yaml
├── .golangci.yml
└── go.mod
```

## 🔄 Data Flow

### Example: GetUser Request

**Three API Entry Points, Same Business Logic:**

#### gRPC Path:

```
1. gRPC Client Request
   ↓
2. adapter/grpc/UserServiceServer
   • Receives: pb.GetUserRequest{Id: 123}
   • Extracts: id := req.Id
   • Calls: usecase.GetUser(ctx, id)
```

#### gRPC-Gateway REST Path:

```
1. HTTP Client: GET /v1/users/123
   ↓
2. gRPC-Gateway converts HTTP → gRPC
   ↓
3. adapter/grpc/UserServiceServer (same as above)
```

#### Gin REST API Path:

```
1. HTTP Client: GET /v1/users/123
   ↓
2. adapter/gin/handler/UserHandler
   • Receives: gin.Context with param "id"
   • Extracts: id := c.Param("id")
   • Calls: usecase.GetUser(ctx, id)
```

**Shared Business Logic Flow:**

```
3. usecase/user/UserUsecase (interface)
   • Receives: id int64
   • Validates: id > 0
   • Calls: repo.GetByID(ctx, id)
   ↓
4. adapter/db/postgres/UserRepository
   • Queries database
   • Returns: *domain.User
   ↓
5. usecase/user/UserUsecase implementation
   • Returns: *domain.User
   ↓
6. Response (varies by protocol):
   • gRPC: pb.GetUserResponse
   • REST: JSON response
   • Gin: JSON response
   ↓
7. Client receives response
```

## 🎨 Layer Responsibilities

### 1. **domain/** - Enterprise Business Rules

**Pure Go entities with zero dependencies**

```go
// Good: Pure business model
type User struct {
    ID    int64
    Name  string
    Email string
}

// Bad: Don't import infrastructure
import "google.golang.org/grpc"  // NO!
import "github.com/lib/pq"       // NO!
```

**Characteristics:**

- No imports from outer layers
- No framework dependencies
- Pure business logic
- Reusable across different applications

---

### 2. **usecase/** - Application Business Rules

**Business logic independent of delivery mechanism**

```go
// UserUsecase interface defines business operations
type UserUsecase interface {
    CreateUser(ctx context.Context, in CreateUserRequest) (*CreateUserResponse, error)
    GetUser(ctx context.Context, in GetUserRequest) (*GetUserResponse, error)
    UpdateUser(ctx context.Context, in UpdateUserRequest) (*UpdateUserResponse, error)
    DeleteUser(ctx context.Context, in DeleteUserRequest) (*DeleteUserResponse, error)
    ListUsers(ctx context.Context, in ListUsersRequest) (*ListUsersResponse, error)
}

// Repository interface for data access (Dependency Inversion)
type Repository interface {
    GetByID(ctx context.Context, id int64) (*user.User, error)
    Create(ctx context.Context, u *user.User) (int64, error)
}

// Business logic implementation
func (uc *Usecase) GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
    // Validation
    if req.ID <= 0 {
        return nil, errors.New("invalid user id")
    }

    // Business logic
    u, err := uc.repo.GetByID(ctx, req.ID)
    if err != nil {
        return nil, err
    }

    return &GetUserResponse{
        ID:    u.ID,
        Name:  u.Name,
        Email: u.Email,
    }, nil
}
```

**Characteristics:**

- Defines repository interfaces (not implementations)
- Doesn't know about gRPC, HTTP, or databases
- Contains validation and business rules
- Depends only on domain models
- Easy to test with mocks

---

### 3. **adapter/** - Interface Adapters

**Converts data between external systems and use cases**

#### adapter/grpc - gRPC Transport Layer

```go
// Converts protobuf ↔ domain models
func (s *UserServiceServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    // Extract domain data from protobuf
    userResp, err := s.uc.GetUser(ctx, user.GetUserRequest{ID: req.Id})
    if err != nil {
        return nil, err
    }

    // Convert domain model to protobuf
    return &pb.GetUserResponse{
        Id:    userResp.ID,
        Name:  userResp.Name,
        Email: userResp.Email,
    }, nil
}
```

#### adapter/gin - Gin REST API Transport Layer

```go
// Converts HTTP requests ↔ domain models
func (h *UserHandler) GetUser(c *gin.Context) {
    // Extract and validate ID from URL parameter
    id, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
        return
    }

    // Call business logic
    userResp, err := h.uc.GetUser(c.Request.Context(), user.GetUserRequest{ID: id})
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
        return
    }

    // Convert domain model to JSON response
    c.JSON(http.StatusOK, gin.H{
        "id":    userResp.ID,
        "name":  userResp.Name,
        "email": userResp.Email,
    })
}
```

**Gin Middleware Stack:**

```go
// Applied in router/router.go
router.Use(middleware.Recovery(log))      // Panic recovery
router.Use(middleware.Logger(log))         // Request logging
router.Use(middleware.RateLimiter(...))    // Rate limiting
```

#### adapter/db - Database Implementations

```go
// Implements repository interface
type UserRepoPG struct {
    db *pgx.Pool
}

func (r *UserRepoPG) GetByID(ctx context.Context, id int64) (*user.User, error) {
    // PostgreSQL-specific implementation
    var u user.User
    err := r.db.QueryRow(ctx, "SELECT id, name, email FROM users WHERE id = $1", id).
        Scan(&u.ID, &u.Name, &u.Email)
    return &u, err
}
```

**Characteristics:**

- Implements interfaces defined by use cases
- Handles external system specifics (gRPC, HTTP, SQL)
- Converts between external formats and domain models
- Can be swapped without changing business logic

---

### 4. **server/** - Server Initialization

**Application composition and lifecycle management**

```go
func NewGRPCServer(uc user.UserUsecase) *grpc.Server {
    grpcServer := grpc.NewServer()
    pb.RegisterUserServiceServer(grpcServer, grpcadapter.NewUserServiceServer(uc))
    return grpcServer
}
```

**Characteristics:**

- Dependency injection
- Server configuration
- Graceful shutdown
- Health checks

## 🔑 Key Design Principles

### 1. **Dependency Inversion Principle (DIP)**

**Why usecase defines interfaces, not implementations?**

```go
// BAD: Usecase depends on concrete implementation
type UserUsecase struct {
    repo *postgres.UserRepoPG  // Coupled to PostgreSQL!
}

// GOOD: Usecase depends on abstraction
type UserUsecase interface {
    GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error)
    CreateUser(ctx context.Context, req CreateUserRequest) (*CreateUserResponse, error)
}

type Repository interface {
    GetByID(ctx context.Context, id int64) (*user.User, error)
    Create(ctx context.Context, u *user.User) (int64, error)
}

type Usecase struct {
    repo Repository  // Can be Postgres, MySQL, MongoDB, Mock!
}
```

**Benefits:**

- Easy to test (inject mocks)
- Easy to swap implementations
- Business logic independent of infrastructure
- Inner layers don't depend on outer layers

---

### 2. **Interface Segregation**

**Small, focused interfaces**

```go
// Good: Focused interface
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*user.User, error)
    Create(ctx context.Context, u *user.User) (int64, error)
}

// Bad: God interface
type Repository interface {
    GetUser(...)
    CreateUser(...)
    GetProduct(...)
    CreateProduct(...)
    GetOrder(...)
    // ... 50 more methods
}
```

---

### 3. **Single Responsibility**

Each layer has one reason to change:

- **domain**: Business rules change
- **usecase**: Application logic changes
- **adapter/grpc**: gRPC protocol changes
- **adapter/db**: Database schema changes

## 🔧 Production Features

### Config Validation

Comprehensive validation at startup prevents runtime errors:

```go
// Validates 40+ rules including:
- Required fields (DB_HOST, DB_USER, etc.)
- Valid port numbers (1-65535)
- Positive values for pool sizes
- Log level: debug/info/warn/error
- Shutdown timeout: 1-300 seconds
```

**Example Error Messages:**

```bash
$ export GRPC_PORT=99999
$ go run cmd/api/main.go
Error: config validation failed: GRPC_PORT is invalid: port must be between 1 and 65535, got 99999

$ unset DB_HOST
$ go run cmd/api/main.go
Error: config validation failed: DB_HOST is required
```

### Graceful Shutdown

Configurable timeout for different environments:

```env
# Development: fast iteration
SHUTDOWN_TIMEOUT_SECONDS=10

# Staging
SHUTDOWN_TIMEOUT_SECONDS=30

# Production: graceful drain
SHUTDOWN_TIMEOUT_SECONDS=60
```

**Shutdown Sequence:**

1. Receive SIGTERM/SIGINT
2. Stop accepting new requests
3. Wait for in-flight requests (up to timeout)
4. Close HTTP server
5. Stop gRPC server gracefully
6. Close database connections
7. Close Redis connections
8. Sync logger

### Dependency Injection

Centralized DI container for clean dependency management:

```go
// cmd/api/di/container.go
type Container struct {
    Config      *config.Config
    Logger      *zap.Logger
    DB          *gorm.DB
    RedisClient *redisclient.Client
    UserUC      user.UserUsecase
    RateLimiter *middleware.RateLimiter
}
```

**Benefits:**

- Single source of truth for dependencies
- Easy to test with mocks
- Clean resource cleanup
- Fail-fast on invalid config

---

## 🚀 Quick Start

**For reviewers - Try these commands to see the project in action:**

```bash
# 1. Start all services (PostgreSQL + Redis + API)
cd deployments
docker-compose up -d

# 2. Test all three API protocols
# gRPC
gprcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser

# REST (gRPC-Gateway)
curl http://localhost:8080/v1/users/1

# Gin REST API
curl http://localhost:9090/v1/users/1

# 3. Run performance comparison
make benchmark

# 4. View logs
docker-compose logs -f grpc-user-service
```

---

### Prerequisites

- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- PostgreSQL 15+
- Redis 7+ (for caching and rate limiting)

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/grpc-user-service.git
cd grpc-user-service

# Install dependencies
go mod download

# Generate protobuf code
make proto-gen

# Run the service
go run cmd/api/main.go
```

### Running with Docker Compose

**Quick Start** - Start all services (PostgreSQL + Redis + Migrations + API):

```bash
cd deployments
docker-compose up -d
```

This automatically:

- Starts PostgreSQL database
- Starts Redis cache
- Runs database migrations
- Starts gRPC User Service

**View logs:**

```bash
docker-compose logs -f grpc-user-service
```

**Stop services:**

```bash
docker-compose down
```

**Reset everything (including data):**

```bash
docker-compose down -v
```

**Services available:**

- gRPC: `localhost:50051`
- REST API (gRPC-Gateway): `http://localhost:8080`
- Gin REST API: `http://localhost:9090`
- PostgreSQL: `localhost:5432`
- Redis: `localhost:6379`

## 📡 API Usage

### Quick Protocol Test

**Test Protocol Buffers code generation:**

```bash
# View generated protobuf code
ls -la api/gen/go/user/
cat api/gen/go/user/user.pb.go

# Test protobuf message creation
go run -c 'package main

import (
    "fmt"
    "grpc-user-service/api/gen/go/user"
)

func main() {
    u := &user.User{
        Id:    1,
        Name:  "Test User",
        Email: "test@example.com",
    }
    fmt.Printf("Protobuf message: %+v\n", u)
}'
```

### gRPC

```bash
# Using grpcurl
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser

grpcurl -plaintext -d '{"name": "John Doe", "email": "john@example.com"}' \
  localhost:50051 user.UserService/CreateUser
```

### REST (via gRPC-Gateway)

```bash
# Get user
curl http://localhost:8080/v1/users/1

# Create user
curl -X POST http://localhost:8080/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'
```

### Gin REST API

```bash
# Get user
curl http://localhost:9090/v1/users/1

# Create user
curl -X POST http://localhost:9090/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# List users (with pagination)
curl "http://localhost:9090/v1/users?page=1&limit=10"

# Update user
curl -X PUT http://localhost:9090/v1/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "John Updated", "email": "john.updated@example.com"}'

# Delete user
curl -X DELETE http://localhost:9090/v1/users/1
```

## 📝 Logging

Production-ready structured logging with **Zap** featuring environment-based configuration, request tracking, and GORM query logging.

### Features

- **Environment-based configuration** - Different settings for development/production
- **Log rotation** - Automatic file rotation with compression (Lumberjack)
- **Sampling** - Reduce log volume in production
- **Request ID tracking** - Trace requests across the entire flow
- **GORM query logging** - All SQL queries with performance metrics
- **Slow query detection** - Configurable threshold for slow queries
- **Default fields** - Service name, version, environment auto-added
- **Structured fields** - JSON format for log aggregators

### Configuration

All logging behavior is controlled via environment variables:

```env
# Logger Configuration
LOG_LEVEL=debug                    # debug, info, warn, error
LOG_FORMAT=console                 # console or json
LOG_OUTPUT_PATH=stdout             # stdout, stderr, or file path
LOG_SLOW_QUERY_SECONDS=0.2        # Slow query threshold (200ms)
LOG_ENABLE_SAMPLING=false         # Enable sampling (recommended for production)
SERVICE_NAME=grpc-user-service
SERVICE_VERSION=1.0.0
```

### Log Levels

| Level   | Development | Production | Use Case                    |
| ------- | ----------- | ---------- | --------------------------- |
| `debug` | ✅ Default  | ❌         | All queries, verbose output |
| `info`  | ✅          | ✅ Default | Normal operations           |
| `warn`  | ✅          | ✅         | Slow queries, deprecations  |
| `error` | ✅          | ✅         | Errors only                 |

### Example Output

**Development (Console format):**

```
2025-11-28T16:40:47.938+0700    info    api/main.go:68  starting application
    {"service": "grpc-user-service", "version": "1.0.0", "environment": "development"}

2025-11-28T16:40:48.123+0700    info    pkg/logger/gorm.go:134  gorm query
    {"service": "grpc-user-service", "request_id": "550e8400-e29b-41d4-a716-446655440000",
     "sql": "SELECT * FROM users WHERE id = $1", "rows": 1, "elapsed": "15.2ms"}

2025-11-28T16:40:48.456+0700    warn    pkg/logger/gorm.go:130  gorm slow query
    {"service": "grpc-user-service", "request_id": "550e8400-e29b-41d4-a716-446655440001",
     "sql": "SELECT * FROM users JOIN orders...", "elapsed": "250ms", "threshold": "200ms"}
```

**Production (JSON format):**

```json
{
  "level": "info",
  "timestamp": "2025-11-28T16:40:47.938+0700",
  "caller": "api/main.go:68",
  "message": "starting application",
  "service": "grpc-user-service",
  "version": "1.0.0",
  "environment": "production"
}
```

### Request ID Tracking

Every gRPC request automatically gets a unique request ID:

```go
// Automatic via middleware
grpc.NewServer(
    grpc.UnaryInterceptor(logger.RequestIDInterceptor()),
)
```

All logs related to the same request share the same `request_id`, making it easy to trace the entire request flow.

### GORM Query Logging

**All database queries are logged with:**

- SQL statement (truncated if > 1000 chars)
- Rows affected
- Execution time (ms)
- Request ID (if available)
- Slow query warnings

**Configuration:**

```env
LOG_LEVEL=info                # See all queries
LOG_SLOW_QUERY_SECONDS=0.1   # Warn if query > 100ms
```

### Log Rotation

For file output, logs automatically rotate:

```env
LOG_OUTPUT_PATH=/var/log/grpc-user-service.log
```

- **Max size**: 100MB per file
- **Max backups**: 3 files
- **Max age**: 28 days
- **Compression**: Automatic `.gz` compression

### Production Best Practices

```env
# Production settings
APP_ENV=production
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT_PATH=/var/log/app.log
LOG_ENABLE_SAMPLING=true
```

**Why JSON format?** Easy integration with:

- ELK Stack (Elasticsearch, Logstash, Kibana)
- Datadog
- Splunk
- CloudWatch Logs

**Why sampling?** Reduces log volume by ~90% while keeping first 100 entries/second and 1/10 thereafter.

## 🧪 Testing

```bash
# Run unit tests
go test ./internal/usecase/...

# Run integration tests
go test ./tests/integration/...

# Run with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 📊 Detailed Benchmark Results

**Complete performance testing framework with detailed metrics collection.**

_For quick overview, see the [Performance Testing](#-performance-testing) section above._

### Test Hardware

- **Model**: Mac mini (2024)
- **Chip**: Apple M4 (10 cores: 4P + 6E)
- **Memory**: 16 GB
- **OS**: macOS

### Detailed Results

Results using in-memory mock repository (Mac mini M4):

#### gRPC Performance

| Operation     | Iterations | ns/op   | B/op    | allocs/op |
| ------------- | ---------- | ------- | ------- | --------- |
| CreateUser    | 20,294     | 101,635 | 13,035  | 211       |
| GetUser       | 38,239     | 45,293  | 7,150   | 116       |
| UpdateUser    | 25,776     | 69,595  | 10,107  | 162       |
| DeleteUser    | 31,639     | 50,849  | 7,367   | 126       |
| ListUsers     | 14,755     | 137,324 | 29,804  | 334       |
| MixedWorkload | 23,960     | 66,016  | 140,398 | 222       |

#### Gin Performance

| Operation     | Iterations | ns/op   | B/op    | allocs/op |
| ------------- | ---------- | ------- | ------- | --------- |
| CreateUser    | 3,004      | 401,614 | 43,108  | 289       |
| GetUser       | 4,894      | 269,096 | 27,503  | 194       |
| UpdateUser    | 2,888      | 417,279 | 53,288  | 292       |
| DeleteUser    | 3,438      | 347,963 | 29,699  | 204       |
| ListUsers     | 2,289      | 552,127 | 110,250 | 505       |
| MixedWorkload | 1,509      | 835,735 | 77,921  | 286       |

#### REST (gRPC-Gateway) Performance

| Operation     | Iterations | ns/op   | B/op    | allocs/op |
| ------------- | ---------- | ------- | ------- | --------- |
| CreateUser    | 2,533      | 442,655 | 56,006  | 344       |
| GetUser       | 4,212      | 286,571 | 36,940  | 241       |
| UpdateUser    | 2,472      | 477,212 | 66,187  | 347       |
| DeleteUser    | 3,343      | 357,497 | 39,137  | 251       |
| ListUsers     | 1,963      | 638,634 | 127,117 | 559       |
| MixedWorkload | 1,315      | 979,219 | 93,698  | 535       |

#### Performance Comparison

- **Latency**: gRPC is **3-4x faster** than Gin and REST
- **Throughput**: gRPC handles **4-5x more requests** than Gin and REST
- **Memory**: gRPC uses significantly less memory and allocations
- **Consistency**: All protocols maintain 100% success rate under load

#### Metrics Explanation

| Metric         | Description                                                                |
| -------------- | -------------------------------------------------------------------------- |
| **Iterations** | Total number of times the operation was executed during the benchmark      |
| **ns/op**      | Nanoseconds per operation (lower is better). Represents latency.           |
| **B/op**       | Bytes allocated per operation (lower is better). Memory usage.             |
| **allocs/op**  | Number of memory allocations per operation (lower is better). GC pressure. |

> **Note**: These results use in-memory mock repository. Real database operations will have higher latencies depending on database performance and network conditions.

### Running Benchmarks

```bash
# Run all benchmarks
make benchmark

# Run gRPC benchmarks only
make benchmark-grpc

# Run Gin benchmarks only
make benchmark-gin

# Run REST (gRPC-Gateway) benchmarks only
make benchmark-rest

# Run benchmarks with CPU profiling
make benchmark-cpu

# Run benchmarks with memory profiling
make benchmark-mem

# Run benchmarks with custom configuration
make benchmark-config
```

## 🛠️ Development

### Generate Protobuf Code

```bash
make proto-gen
```

### Run Linters

```bash
golangci-lint run
```

### Database Migrations

```bash
# Create new migration
migrate create -ext sql -dir deployments/migrations -seq create_users_table

# Run migrations
migrate -path deployments/migrations -database "postgres://user:pass@localhost:5432/dbname?sslmode=disable" up
```

## 🚀 Redis Cache & Rate Limiting

### Redis Caching

Implements **cache-aside pattern** for GetUser queries with automatic cache invalidation:

**Features:**

- Cache hit: ~1-2ms (vs 10-50ms database query)
- TTL: 5 minutes (configurable)
- Automatic invalidation on Update/Delete
- JSON serialization
- Comprehensive logging (cache hit/miss)

**Configuration:**

```env
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
REDIS_CACHE_TTL_SECONDS=300
REDIS_MAX_RETRIES=3
REDIS_POOL_SIZE=10
REDIS_MIN_IDLE_CONN=5
```

**Usage:**

```bash
# Start Redis
docker run -d -p 6379:6379 redis:alpine

# First request (cache miss)
curl http://localhost:8080/v1/users/1

# Second request (cache hit - faster!)
curl http://localhost:8080/v1/users/1

# Verify cache
redis-cli KEYS "user:*"
redis-cli GET "user:1"
```

### Rate Limiting

Protects APIs using **gRPC interceptor** with Redis sliding window algorithm:

**Features:**

- Per-method, per-IP rate limiting
- Atomic increment with Lua script
- Fail-open strategy (allows requests if Redis fails)
- Configurable limits and windows

**Configuration:**

```env
RATE_LIMIT_REQUESTS_PER_SECOND=10.0
RATE_LIMIT_WINDOW_SECONDS=1
RATE_LIMIT_ENABLED=true
```

**Testing:**

```bash
# Send 15 requests rapidly
for i in {1..15}; do curl http://localhost:8080/v1/users/1; done
# First 10 succeed, remaining return ResourceExhausted error

# Verify rate limit keys
redis-cli KEYS "ratelimit:*"
```

## 📚 Project Roadmap

### ✅ Completed

- [x] Clean Architecture foundation with DI container
- [x] Dependency Injection layer (`cmd/api/di/`)
- [x] Infrastructure layer (`cmd/api/infrastructure/`)
- [x] Config validation (40+ rules, fail-fast)
- [x] Graceful shutdown with configurable timeout
- [x] gRPC + gRPC-Gateway
- [x] **Gin REST API** with middleware stack
- [x] **Three API protocols** sharing same business logic
- [x] Structured logging (Zap with production features)
- [x] Redis caching layer
- [x] Rate limiting (gRPC interceptor)
- [x] PostgreSQL repository implementation
- [x] Unit tests (34/34 passing)
- [x] Lint compliance (0 issues)
- [x] Panic recovery in app and server goroutines
- [x] Context-aware shutdown with timeout
- [x] Performance benchmarks for all three protocols
- [x] Health check endpoints

### 🚧 In Progress

- [ ] Metrics endpoint (Prometheus)
- [ ] API documentation (Swagger/OpenAPI)

### 📋 Planned

- [ ] Distributed tracing (OpenTelemetry)
- [ ] Integration tests
- [ ] Load testing (k6)
- [ ] CI/CD pipeline (GitHub Actions)
- [ ] Docker Compose setup improvements
- [ ] Kubernetes manifests
- [ ] API versioning
- [ ] Circuit breaker pattern

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) by Robert C. Martin
- [gRPC-Go](https://github.com/grpc/grpc-go)
- [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Protocol Buffers](https://protobuf.dev/)

---

**Built with ❤️ using Go, gRPC, Gin, and Clean Architecture principles**

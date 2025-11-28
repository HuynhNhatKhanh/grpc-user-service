# gRPC User Service

A production-ready microservice built with **Go**, **gRPC**, and **Clean Architecture** principles. This project demonstrates best practices for building scalable, maintainable, and testable backend services.

## 🎯 Project Overview

This is a user management microservice that provides both **gRPC** and **REST** APIs through gRPC-Gateway. The service is designed following **Clean Architecture** and **SOLID principles**, ensuring clear separation of concerns and high testability.

### Key Features

- **Clean Architecture** - Clear separation between business logic and infrastructure
- **gRPC + gRPC-Gateway** - Native gRPC with automatic REST API generation
- **Dependency Inversion** - Business logic independent of frameworks and databases
- **Type-safe** - Leveraging Go's strong typing and Protocol Buffers
- **Production-ready** - Structured logging, error handling, and graceful shutdown
- **Testable** - Interface-based design for easy mocking and testing

## 🏗️ Architecture

This project follows **Clean Architecture** principles with clear dependency rules:

```
┌─────────────────────────────────────────────────────────────┐
│                     Frameworks & Drivers                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              adapter/ (Infrastructure)                 │ │
│  │  • grpc/      - gRPC server implementation            │ │
│  │  • http/      - HTTP handlers & middleware            │ │
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
DB      Rules
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
│       └── main.go               # Main server application
│
├── internal/                     # Private application code
│   ├── domain/                   # 🟢 Enterprise Business Rules
│   │   └── user/
│   │       ├── entity.go         # User entity (pure Go)
│   │       └── value_object.go   # Value objects
│   │
│   ├── usecase/                  # 🟡 Application Business Rules
│   │   └── user/
│   │       └── usecase.go        # Business logic & interfaces
│   │
│   ├── adapter/                  # 🔴 Interface Adapters
│   │   ├── grpc/                 # gRPC transport layer
│   │   │   └── user_service.go   # gRPC → Usecase adapter
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
├── scripts/                      # Build & utility scripts
├── tests/                        # Integration & E2E tests
├── buf.yaml
├── buf.gen.yaml
├── .golangci.yml
└── go.mod
```

## 🔄 Data Flow

### Example: GetUser Request

```
1. Client Request (gRPC or REST)
   ↓
2. adapter/grpc/UserServiceServer
   • Receives: pb.GetUserRequest{Id: 123}
   • Extracts: id := req.Id
   • Calls: usecase.GetUser(ctx, id)
   ↓
3. usecase/user/UserUsecase
   • Receives: id int64
   • Validates: id > 0
   • Calls: repo.GetByID(ctx, id)
   ↓
4. adapter/db/postgres/UserRepository
   • Queries database
   • Returns: *domain.User
   ↓
5. usecase/user/UserUsecase
   • Returns: *domain.User
   ↓
6. adapter/grpc/UserServiceServer
   • Converts: domain.User → pb.GetUserResponse
   • Returns: pb.GetUserResponse
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
// Defines interfaces (Dependency Inversion)
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*user.User, error)
    Create(ctx context.Context, u *user.User) (int64, error)
}

// Business logic with validation
func (uc *UserUsecase) GetUser(ctx context.Context, id int64) (*user.User, error) {
    // Validation
    if id <= 0 {
        return nil, errors.New("invalid user id")
    }

    // Business logic
    return uc.repo.GetByID(ctx, id)
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
    user, err := s.uc.GetUser(ctx, req.Id)
    if err != nil {
        return nil, err
    }

    // Convert domain model to protobuf
    return &pb.GetUserResponse{
        Id:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    }, nil
}
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
func NewGRPCServer(uc *user.UserUsecase) *grpc.Server {
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
type UserRepository interface {
    GetByID(ctx context.Context, id int64) (*user.User, error)
}

type UserUsecase struct {
    repo UserRepository  // Can be Postgres, MySQL, MongoDB, Mock!
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

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Protocol Buffers compiler (`protoc`)
- PostgreSQL 15+
- Redis 7+ (optional, for caching)

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

### Running with Docker

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down
```

## 📡 API Usage

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

## 📊 Performance Benchmarks

Expected performance metrics:

| Metric        | gRPC       | REST (gRPC-Gateway) |
| ------------- | ---------- | ------------------- |
| Latency (p50) | ~1-2ms     | ~5-7ms              |
| Latency (p99) | ~5ms       | ~15ms               |
| Throughput    | ~50k req/s | ~20k req/s          |

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

## 📚 Project Roadmap

- [x] Clean Architecture foundation
- [x] gRPC + gRPC-Gateway
- [x] Structured logging (Zap with production features)
- [ ] Redis caching layer
- [ ] PostgreSQL repository implementation
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Metrics (Prometheus)
- [ ] Docker Compose setup
- [ ] Database migrations
- [ ] Integration tests
- [ ] Load testing (k6)
- [ ] CI/CD pipeline

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) by Robert C. Martin
- [gRPC-Go](https://github.com/grpc/grpc-go)
- [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
- [Protocol Buffers](https://protobuf.dev/)

---

**Built with ❤️ using Go and Clean Architecture principles**

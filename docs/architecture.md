# Architecture Documentation

## 🏗️ Clean Architecture

This project follows **Clean Architecture** principles with clear dependency rules:

![Clean Architecture Layers](https://mermaid.ink/img/Z3JhcGggVEQKICAgIHN1YmdyYXBoIEZyYW1ld29ya3NfRHJpdmVycyBbRnJhbWV3b3JrcyAmIERyaXZlcnNdCiAgICAgICAgZGlyZWN0aW9uIFRCCiAgICAgICAgQWRhcHRlclthZGFwdGVyLyBJbmZyYXN0cnVjdHVyZV0KICAgICAgICBHUlBDW2dycGMvXQogICAgICAgIEhUVFBbaHR0cC9dCiAgICAgICAgR2luW2dpbi9dCiAgICAgICAgUmVwb1tyZXBvc2l0b3J5L10KICAgICAgICBDYWNoZVtjYWNoZS9dCiAgICBlbmQKCiAgICBzdWJncmFwaCBBcHBfQnVzaW5lc3NfUnVsZXMgW0FwcGxpY2F0aW9uIEJ1c2luZXNzIFJ1bGVzXQogICAgICAgIFVzZWNhc2VbdXNlY2FzZS8gQnVzaW5lc3MgTG9naWNdCiAgICBlbmQKCiAgICBzdWJncmFwaCBFbnRlcnByaXNlX0J1c2luZXNzX1J1bGVzIFtFbnRlcnByaXNlIEJ1c2luZXNzIFJ1bGVzXQogICAgICAgIERvbWFpbltkb21haW4vIEVudGl0aWVzXQogICAgZW5kCgogICAgQWRhcHRlciAtLT4gVXNlY2FzZQogICAgVXNlY2FzZSAtLT4gRG9tYWlu)

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
│   │   ├── repository/           # Repository implementations
│   │   │   ├── postgres/         # PostgreSQL implementation
│   │   │   │   └── user.go       # DB operations
│   │   │   └── cached/           # Cached implementation
│   │   │       └── user.go       # Cache-Aside logic
│   │   └── cache/                # Cache client wrappers
│   │       └── user_cache.go     # Redis cache interface
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

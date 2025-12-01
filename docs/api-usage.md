# API Usage & Protocol Buffers

## 📋 Protocol Buffers

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

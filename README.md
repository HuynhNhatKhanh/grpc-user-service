# gRPC User Service

A production-ready microservice built with **Go**, **Protocol Buffers**, **gRPC**, **Gin**, and **Clean Architecture**. This project demonstrates best practices for building scalable, maintainable, and testable backend services with **three API delivery mechanisms**: gRPC, gRPC-Gateway REST, and Gin REST API.

## 💡 Why I built this project

I built this project to:

- **Master Clean Architecture**: Implementing a strict separation of concerns where business logic is completely isolated from frameworks and drivers.
- **Compare gRPC vs REST**: conducting real-world performance benchmarks to understand the trade-offs between binary (gRPC) and text-based (JSON) protocols.
- **Demonstrate Microservice Best Practices**: Showcasing production-ready features like Graceful Shutdown, Configuration Validation, Caching, and Rate Limiting that are often missing in junior/mid-level portfolios.

## 🛠️ Tech Stack

| Category           | Technology      | Version   | Usage                                     |
| ------------------ | --------------- | --------- | ----------------------------------------- |
| **Language**       | Go              | 1.21+     | Core application logic                    |
| **API Protocol**   | gRPC / Protobuf | v3        | API contract & high-performance transport |
| **HTTP Framework** | Gin             | v1.9      | High-performance REST API                 |
| **Database**       | PostgreSQL      | 16-alpine | Primary data store                        |
| **Caching**        | Redis           | 7-alpine  | Caching & Rate Limiting                   |
| **Migration**      | golang-migrate  | latest    | Database schema management                |
| **Logging**        | Zap             | v1        | Structured, leveled logging               |

## 🏗️ Architecture Overview

This project follows **Clean Architecture** principles. The business logic (`usecase`) is central and independent of external frameworks (`adapter`).

![Architecture Overview](https://mermaid.ink/img/Z3JhcGggVEQKICAgIHN1YmdyYXBoICJFeHRlcm5hbCBJbnRlcmZhY2VzIChBZGFwdGVycykiCiAgICAgICAgZ1JQQ1tnUlBDIFNlcnZlcl0KICAgICAgICBSRVNUW2dSUEMgR2F0ZXdheV0KICAgICAgICBHaW5bR2luIFJFU1QgQVBJXQogICAgZW5kCgogICAgc3ViZ3JhcGggIkNvcmUgQnVzaW5lc3MgTG9naWMiCiAgICAgICAgVXNlY2FzZVtVc2VyIFVzZWNhc2VdCiAgICBlbmQKCiAgICBzdWJncmFwaCAiSW5mcmFzdHJ1Y3R1cmUiCiAgICAgICAgUG9zdGdyZXNbKFBvc3RncmVTUUwpXQogICAgICAgIFJlZGlzWyhSZWRpcyBDYWNoZSldCiAgICBlbmQKCiAgICBnUlBDIC0tPiBVc2VjYXNlCiAgICBSRVNUIC0tPiBVc2VjYXNlCiAgICBHaW4gLS0-IFVzZWNhc2UKICAgIFVzZWNhc2UgLS0-IFBvc3RncmVzCiAgICBVc2VjYXNlIC0tPiBSZWRpcw==)

👉 **[Read detailed Architecture Documentation](docs/architecture.md)**

## ⚡ Performance Highlights

**gRPC is significantly faster and more efficient than REST.**

| Protocol | Latency (ns/op) | Throughput (ops/sec) | Efficiency |
| -------- | --------------- | -------------------- | ---------- |
| **gRPC** | **101,635**     | **9,838**            | 🥇 Best    |
| **Gin**  | 401,614         | 2,488                | 🥈 Good    |
| **REST** | 442,655         | 2,259                | 🥉 OK      |

👉 **[View Full Benchmark Results](docs/performance-benchmarks.md)**

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose

### Run with Docker Compose

```bash
# 1. Clone the repo
git clone https://github.com/huynhnhatkhanh/grpc-user-service
cd grpc-user-service

# 2. Start everything (DB, Redis, API, Migrations)
cd deployments
docker-compose up -d

# 3. View logs
docker-compose logs -f grpc-user-service
```

### Test the API

```bash
# gRPC (using grpcurl)
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser

# REST (gRPC Gateway)
curl http://localhost:8080/v1/users/1

# Gin REST
curl http://localhost:9090/v1/users/1
```

👉 **[See Detailed API Usage](docs/api-usage.md)** | **[Deployment Guide](docs/deployment.md)**

## 📂 Project Structure

```
grpc-user-service/
├── api/                  # Protobuf definitions & generated code
├── cmd/                  # Main entry points & DI container
├── internal/
│   ├── domain/           # Pure business entities (Enterprise Rules)
│   ├── usecase/          # Business logic (Application Rules)
│   └── adapter/          # gRPC, Gin, SQL implementations
├── deployments/          # Docker & Migrations
└── docs/                 # Detailed documentation
```

## 📝 Key Features

- **Clean Architecture**: Strict layer separation.
- **Multi-Protocol**: gRPC, REST Gateway, and Gin sharing the same logic.
- **Production Ready**:
  - **Config Validation**: Fail-fast on invalid config.
  - **Graceful Shutdown**: No dropped requests during deploy.
  - **Structured Logging**: Zap logger with request tracing.
- **Performance**:
  - **Redis Caching**: Cache-aside pattern.
  - **Rate Limiting**: Token Bucket algorithm (smooth rate limiting with burst support).

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

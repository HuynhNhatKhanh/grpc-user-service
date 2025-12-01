# Deployment & Development Guide

## 🐳 Running with Docker Compose

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

# Docker Compose Quick Start

## Start all services (with migrations)

```bash
cd deployments
docker compose up -d
```

This will start:

- PostgreSQL database
- Redis cache
- Run migrations automatically
- gRPC User Service

## View logs

```bash
docker compose logs -f grpc-user-service
```

## Stop services

```bash
docker compose down
```

## Reset everything (including data)

```bash
docker compose down -v
```

## Run migrations manually

```bash
docker compose run --rm migrate -path=/migrations -database="postgres://postgres:postgres@postgres:5432/grpc_user_service?sslmode=disable" up
```

## Check service health

```bash
# Check all services
docker compose ps

# Test gRPC endpoint
grpcurl -plaintext localhost:50051 list

# Test REST endpoint
curl http://localhost:8080/v1/users?page=1&limit=10
```

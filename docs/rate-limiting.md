# Rate Limiting

## Overview

Rate limiting is a critical security and stability feature that controls the number of requests a client can make to the API within a specified time period. This protects the service from:

- **DDoS attacks**: Prevents malicious actors from overwhelming the server
- **Resource exhaustion**: Ensures fair resource distribution among clients
- **Cost control**: Limits API usage for cost-sensitive operations
- **Quality of Service**: Maintains consistent performance for all users

This project implements rate limiting using the **Token Bucket** algorithm, which provides smooth and flexible rate control.

---

## Token Bucket Algorithm

### Concept

The Token Bucket algorithm is a widely-used rate limiting technique that allows for controlled burst traffic while maintaining an average rate limit.

**How it works:**

1. **Bucket**: A virtual container that holds tokens
2. **Capacity**: Maximum number of tokens the bucket can hold
3. **Refill Rate**: Tokens are added to the bucket at a constant rate (e.g., 10 tokens/second)
4. **Consumption**: Each request consumes 1 token from the bucket
5. **Decision**: Request is allowed if tokens are available, otherwise denied

### Visual Representation

```
┌─────────────────────────────┐
│   Token Bucket              │
│                             │
│   Capacity: 20 tokens       │
│   Refill: 10 tokens/sec     │
│                             │
│   Current: ████████░░░░     │  ← 12 tokens available
│                             │
└─────────────────────────────┘
         ▲           │
         │           │
    Refill rate    Consumed by
    (constant)     requests
```

### Example Scenario

**Configuration:**

- Refill Rate: `10 tokens/second`
- Burst Capacity: `20 tokens`

**Timeline:**

| Time | Action      | Tokens Before | Tokens After | Result            |
| ---- | ----------- | ------------- | ------------ | ----------------- |
| 0.0s | Start       | -             | 20           | (bucket full)     |
| 0.1s | 5 requests  | 20            | 15           | ✅ Allowed        |
| 0.2s | 10 requests | 16            | 6            | ✅ Allowed        |
| 0.3s | 10 requests | 7             | 0            | ❌ 3 denied       |
| 1.0s | Wait        | 0             | 10           | (tokens refilled) |
| 1.1s | 10 requests | 10            | 0            | ✅ Allowed        |

---

## Algorithm Comparison

### 1. Fixed Window Counter

**How it works:** Count requests in fixed time windows (e.g., per minute).

**Pros:**

- Simple to implement
- Low memory usage

**Cons:**

- ❌ **Window boundary burst problem**: Allows 2x rate at window edges
- ❌ Unfair to clients with bad timing

**Example problem:**

```
Window 1: [---- 10 req ---] | Window 2: [--- 10 req ----]
Time:     0s          0.9s  1s  1.1s         2s

❌ Problem: 20 requests in 0.2 seconds (from 0.9s to 1.1s)
```

### 2. Sliding Window Log

**How it works:** Track timestamp of each request, count in sliding windows.

**Pros:**

- Precise rate limiting
- No boundary issues

**Cons:**

- ❌ High memory usage (stores all request timestamps)
- ❌ Expensive to compute (needs to scan all timestamps)

### 3. Token Bucket (Our Choice)

**How it works:** Refill tokens at constant rate, consume on requests.

**Pros:**

- ✅ **Smooth rate limiting** over time
- ✅ **Allows controlled bursts** (up to capacity)
- ✅ Memory efficient (only stores 2 values: tokens, last_refill)
- ✅ No window boundary issues
- ✅ Fair to all clients

**Cons:**

- Slightly more complex than fixed window
- Requires accurate timestamps

### Comparison Table

| Feature        | Fixed Window | Sliding Window | Token Bucket          |
| -------------- | ------------ | -------------- | --------------------- |
| Memory         | Low          | High           | Low                   |
| Accuracy       | Low          | High           | Medium-High           |
| Burst Control  | ❌ No        | ✅ Yes         | ✅ Yes (configurable) |
| Boundary Issue | ❌ Yes       | ✅ No          | ✅ No                 |
| Implementation | Simple       | Complex        | Medium                |
| **Our Rating** | ⭐⭐         | ⭐⭐⭐         | ⭐⭐⭐⭐⭐            |

---

## Implementation Details

### Architecture

```
┌──────────────┐
│  gRPC/REST   │  ← Client request
│   Request    │
└──────┬───────┘
       │
       ▼
┌──────────────────┐
│  Rate Limiter    │  ← Middleware
│   Middleware     │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│     Redis        │  ← Token Bucket state
│  Lua Script      │     { last_refill, tokens }
└──────┬───────────┘
       │
       ▼
    Allow/Deny
```

### Redis Data Structure

Each client's token bucket is stored as a Redis hash:

```
Key: ratelimit:tb:{method}:{path}:{client_ip}
Value: {
  "last_refill": 1701587436.5,  // timestamp in seconds
  "tokens": 15.7                 // current tokens available
}
TTL: 60 seconds
```

### Lua Script (Atomic Operation)

```lua
-- Get or initialize bucket
local bucket = redis.call('HMGET', key, 'last_refill', 'tokens')
local last_refill = tonumber(bucket[1]) or now
local tokens = tonumber(bucket[2]) or capacity

-- Calculate tokens to add based on elapsed time
local elapsed = math.max(0, now - last_refill)
local tokens_to_add = elapsed * rate
tokens = math.min(capacity, tokens + tokens_to_add)

-- Try to consume 1 token
if tokens >= 1 then
    tokens = tokens - 1
    redis.call('HMSET', key, 'last_refill', now, 'tokens', tokens)
    return 1  -- Allow
else
    redis.call('HMSET', key, 'last_refill', now, 'tokens', tokens)
    return 0  -- Deny
end
```

**Why Lua script?**

- ✅ **Atomicity**: All operations execute as a single transaction
- ✅ **Performance**: Reduces network round-trips
- ✅ **Consistency**: No race conditions

---

## Configuration

### Environment Variables

```env
# Rate Limiting Configuration
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SECOND=10.0
RATE_LIMIT_BURST_CAPACITY=20
```

### Configuration struct

```go
type RateLimitConfig struct {
    RequestsPerSecond float64  // Token refill rate (e.g., 10.0)
    BurstCapacity     int      // Max tokens in bucket (e.g., 20)
    Enabled           bool     // Enable/disable rate limiting
}
```

### Default Values

| Parameter           | Default | Description                       |
| ------------------- | ------- | --------------------------------- |
| `RequestsPerSecond` | `10.0`  | Steady-state rate (tokens/second) |
| `BurstCapacity`     | `20`    | Maximum burst size (2x the rate)  |
| `Enabled`           | `true`  | Rate limiting on/off              |

### Tuning Guidelines

**For Public APIs:**

```env
RATE_LIMIT_REQUESTS_PER_SECOND=10
RATE_LIMIT_BURST_CAPACITY=20
```

**For Internal Services:**

```env
RATE_LIMIT_REQUESTS_PER_SECOND=100
RATE_LIMIT_BURST_CAPACITY=200
```

**For Premium Clients:**

```env
RATE_LIMIT_REQUESTS_PER_SECOND=50
RATE_LIMIT_BURST_CAPACITY=100
```

---

## Usage Examples

### gRPC Rate Limiting

Rate limiting is automatically applied to all gRPC endpoints:

```bash
# Within limit
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser
# Response: { "id": 1, "name": "..." }

# Exceed limit (after 20+ requests)
grpcurl -plaintext -d '{"id": 1}' localhost:50051 user.UserService/GetUser
# Error: rpc error: code = ResourceExhausted
#        desc = rate limit exceeded: 10.00 requests/second (burst capacity: 20)
```

### REST API Rate Limiting (gRPC-Gateway)

```bash
# Within limit
curl http://localhost:8080/v1/users/1
# Response: {"id": 1, "name": "..."}

# Exceed limit
curl http://localhost:8080/v1/users/1
# Response: {
#   "error": "rate_limit_exceeded",
#   "message": "Rate limit exceeded: 10.00 requests/second (burst capacity: 20)"
# }
# Status: 429 Too Many Requests
```

### Gin REST API Rate Limiting

```bash
# Within limit
curl http://localhost:9090/v1/users/1
# Response: {"id": 1, "name": "..."}

# Exceed limit
curl http://localhost:9090/v1/users/1
# Response: {
#   "error": "rate_limit_exceeded",
#   "message": "Rate limit exceeded: 10.00 requests/second (burst capacity: 20)"
# }
# Status: 429 Too Many Requests
```

---

## Testing Rate Limiting

### Manual Test with curl

```bash
# Send 25 requests rapidly
for i in {1..25}; do
  curl -s -w "\n%{http_code}\n" http://localhost:9090/v1/users/1
done

# Expected output:
# First 20 requests: 200 OK (using burst capacity)
# Next 5 requests: 429 Too Many Requests (rate limited)
```

### Load Testing with hey

```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test with 100 requests, 10 concurrent
hey -n 100 -c 10 http://localhost:9090/v1/users/1

# Output will show:
# - Total requests: 100
# - Successful (200): ~20-30 (depending on timing)
# - Rate limited (429): ~70-80
```

### Unit Tests

Run existing unit tests:

```bash
# Test rate limiter middleware
go test ./internal/adapter/grpc/middleware/... -v

# Test specific scenario
go test ./internal/adapter/grpc/middleware/... -run TestRateLimiter_ExceedLimit -v
```

---

## Client IP Detection

The rate limiter identifies clients by IP address in the following order:

1. **X-Forwarded-For header** (for requests through proxy/gateway)
2. **X-Real-IP header** (alternative proxy header)
3. **Peer address** (direct connection)

### Behind Load Balancer/Proxy

If your service is behind a load balancer or reverse proxy:

```nginx
# Nginx configuration
location /api {
    proxy_pass http://grpc-service:50051;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Real-IP $remote_addr;
}
```

---

## Monitoring and Observability

### Logs

Rate limit events are logged with structured logging:

```json
{
  "level": "warn",
  "msg": "rate limit exceeded",
  "client_ip": "192.168.1.100",
  "method": "/user.UserService/GetUser",
  "rate": 10.0,
  "burst_capacity": 20
}
```

### Redis Metrics

Monitor Redis for rate limiting performance:

```bash
# Check active rate limit keys
redis-cli --scan --pattern "ratelimit:tb:*" | wc -l

# View specific bucket state
redis-cli HGETALL "ratelimit:tb:/user.UserService/GetUser:192.168.1.1"
# Output:
# 1) "last_refill"
# 2) "1701587436.5"
# 3) "tokens"
# 4) "8.3"
```

---

## Production Considerations

### 1. Distributed Systems

For multiple service instances, Token Bucket works seamlessly because:

- ✅ State is stored in shared Redis
- ✅ Lua script ensures atomicity
- ✅ No coordination needed between instances

### 2. Clock Skew

Token Bucket relies on timestamps, so ensure:

- Use NTP for time synchronization
- Redis TIME command is used (single source of truth)

### 3. Redis Availability

**Fail-open strategy:**

- If Redis is unavailable, requests are **allowed** (fail-open)
- This prevents complete service outage due to Redis failure
- Trade-off: Brief period without rate limiting

```go
if err != nil {
    // Log error but allow request
    log.Warn("rate limiter redis error, allowing request")
    return handler(ctx, req)
}
```

### 4. Memory Usage

Each active client uses ~100 bytes in Redis:

- Key: ~60 bytes
- Value (hash): ~40 bytes
- 10,000 active clients ≈ 1 MB

---

## Troubleshooting

### Issue: Rate limits too aggressive

**Solution:** Increase burst capacity or rate

```env
RATE_LIMIT_REQUESTS_PER_SECOND=20.0  # Increase from 10
RATE_LIMIT_BURST_CAPACITY=40         # Increase from 20
```

### Issue: Clients getting rate limited unfairly

**Cause:** Multiple clients behind same NAT/proxy share the same IP

**Solution:** Implement per-user rate limiting (requires authentication)

### Issue: Redis memory growing

**Cause:** TTL not expiring old buckets

**Check:**

```bash
redis-cli INFO memory
redis-cli --scan --pattern "ratelimit:tb:*" | head -10 | xargs -I {} redis-cli TTL {}
```

**Solution:** Ensure EXPIRE is set correctly (60s TTL)

---

## Further Reading

- [Token Bucket Algorithm - Wikipedia](https://en.wikipedia.org/wiki/Token_bucket)
- [Rate Limiting Strategies - IETF Draft](https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/)
- [Redis Lua Scripting](https://redis.io/docs/manual/programmability/eval-intro/)

---

## Summary

✅ **Token Bucket** provides smooth, fair rate limiting with burst support  
✅ **Redis + Lua** ensures atomic, distributed rate limiting  
✅ **Configurable** via environment variables  
✅ **Production-ready** with fail-open strategy and monitoring

For implementation details, see:

- [rate_limit.go](file:///Users/khanh/Documents/golang/grpc-user-service/internal/adapter/grpc/middleware/rate_limit.go) (gRPC)
- [rate_limiter.go](file:///Users/khanh/Documents/golang/grpc-user-service/internal/adapter/gin/middleware/rate_limiter.go) (Gin)

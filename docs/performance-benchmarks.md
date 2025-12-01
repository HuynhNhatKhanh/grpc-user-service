# Performance Benchmarks

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

## 📊 Detailed Benchmark Results

**Complete performance testing framework with detailed metrics collection.**

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

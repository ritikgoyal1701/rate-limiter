# Rate-Limited API Service (Go + Redis)

Production-oriented Go API that enforces per-user rate limits using a Redis-backed **Sliding Window Log** with an **atomic Lua script**.

## Features

- `POST /request` with payload acceptance and rate limiting (`5 requests / 60 seconds / user`)
- `GET /stats?user_id=<id>` for per-user window metrics
- Correct under high concurrency and multi-instance deployments
- Redis is the single source of truth (no in-memory locks for limiter logic)
- Fail-open strategy when Redis is unavailable
- Request ID propagation and structured logging
- Containerized setup with Docker Compose

## Project Structure

```text
root/
  cmd/server/main.go
  internal/
    limiter/
      limiter.go
      script.lua
    handler/
      request.go
      stats.go
    server/
      router.go
  deployment/
    Dockerfile
  docker-compose.yml
  go.mod
  README.md
  .env
```

## Run With Docker Compose

```bash
docker-compose up --build
```

Service endpoints:

- API: `http://localhost:8080`
- Redis: `localhost:6379`

## API Usage

### 1) Send Request (rate-limited)

```bash
curl -i -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-123",
    "payload": {"action":"checkout","amount":42}
  }'
```

Responses:

- `200 OK` when allowed
- `429 Too Many Requests` when the per-user limit is exceeded

### 2) Fetch Per-User Stats

```bash
curl -s "http://localhost:8080/stats?user_id=user-123" | jq
```

Example response:

```json
{
  "user_id": "user-123",
  "current_window_count": 3,
  "remaining_requests": 2,
  "reset_time_seconds": 38
}
```

## Rate Limiting Design

### Sliding Window Log (Redis ZSET)

- Key: `rate_limit:{user_id}` (implemented as `rate_limit:<user_id>`)
- Score: request timestamp in milliseconds
- Member: unique request ID (`timestamp-uuid`)

### Lua Script Atomic Steps

1. Remove expired entries older than 60 seconds.
2. Count current valid requests in the sorted set.
3. If count < 5:
   - Insert current request.
   - Refresh key TTL to window size.
   - Return allowed status and remaining quota.
4. Else:
   - Return denied status.

Because all operations happen inside one Lua script call (`EVALSHA` through `redis.NewScript`), logic is atomic and race-condition free across goroutines and distributed app instances.

## Design Decisions

- **Atomic Lua script** avoids race conditions from multi-call Redis patterns.
- **Redis-backed state only** ensures correctness across multiple service replicas.
- **Fail-open behavior** keeps upstream availability during Redis outages.
- **Explicit timeouts** on HTTP server improve resilience under load.
- **Simple clean architecture** keeps limiter, handlers, and routing concerns separate.

## Tradeoffs

- Sliding window log is precise but stores one entry per accepted request, increasing memory usage versus token bucket/fixed window counters.
- Fail-open prioritizes availability over strict enforcement during Redis incidents.
- `GET /stats` is per-user (query param) rather than global aggregation to keep reads efficient and bounded.

## Limitations

- No authentication/authorization layer.
- No persistent metrics backend (only in-process counters for allowed/429 totals).
- No backpressure queue; excess traffic is rejected immediately.
- Single Redis endpoint configuration in this example.

## Future Improvements

- Redis Cluster/Sentinel support with automatic failover and topology-aware clients.
- Observability: Prometheus metrics, distributed tracing, and RED dashboards.
- Queueing or deferred processing (instead of direct rejection) for burst smoothing.
- Add contract tests and load tests (`k6`/`vegeta`) for throughput validation.

## Local Development (without Docker, optional)

```bash
go mod tidy
go run ./cmd/server
```

Make sure Redis is running and `REDIS_ADDR` points to it.

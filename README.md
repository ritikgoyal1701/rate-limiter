# Rate-Limited API Service (Go + Redis)

Go HTTP service that enforces per-user limits with a Redis-backed **Sliding Window Log** implemented via an **atomic Lua script**.

## Features

- `POST /request` rate-limits per `user_id` (`5 requests / 60 seconds` by default)
- `GET /stats?user_id=<id>` returns live window usage for a user
- Atomic Redis script (`ZSET` + Lua) for correctness under concurrency
- Stateless app instances; Redis is the source of truth for limiter state
- `X-Request-Id` propagation middleware (generated if absent)
- Structured JSON logs via `slog`
- Dockerized runtime with `docker-compose`

## Project Structure

```text
.
├── constants/
│   └── constant.go
├── deployment/
│   └── Dockerfile
├── internal/
│   ├── handler/
│   │   ├── adapter/
│   │   │   └── adapter.go
│   │   ├── requests/
│   │   │   └── request.go
│   │   ├── response/
│   │   │   ├── response.go
│   │   │   └── stats.go
│   │   ├── controller.go
│   │   └── stats.go
│   └── limiter/
│       ├── limiter.go
│       └── script.lua
├── router/
│   └── router.go
├── docker-compose.yml
├── go.mod
├── main.go
└── README.md
```

## Configuration

Environment variables (`.env`):

- `APP_ADDR` (default `:8080`)
- `REDIS_ADDR` (default `localhost:6379`, Docker compose sets it to `redis:6379`)
- `REDIS_PASSWORD` (optional)
- `REDIS_DB` (default `0`)

Rate-limit defaults are defined in `constants/constant.go`:

- limit: `5`
- window: `60s`

## Run

### With Docker Compose

```bash
docker compose up --build
```

Endpoints:

- API: `http://localhost:8080`
- Redis: `localhost:6379`

### Locally (without Docker)

Prerequisites:

- Go `1.23+`
- Redis running and reachable via `REDIS_ADDR`

```bash
go run .
```

## API Usage

### `POST /request`

Body:

```json
{
  "user_id": "user-123",
  "payload": {
    "action": "checkout",
    "amount": 42
  }
}
```

Example:

```bash
curl -i -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user-123","payload":{"action":"checkout","amount":42}}'
```

Response schema:

```json
{
  "allowed": true,
  "status": "allowed",
  "current_window_count": 1,
  "remaining_requests": 4,
  "reset_time_seconds": 60
}
```

Status behavior:

- `200 OK` when request is allowed
- `429 Too Many Requests` when limit is exceeded (`status: "rate_limit_exceeded"`)
- `200 OK` fail-open when Redis is unavailable (`status: "allowed_fail_open"`)
- `400 Bad Request` for invalid JSON or missing `user_id`
- `405 Method Not Allowed` for non-`POST`

### `GET /stats?user_id=<id>`

Example:

```bash
curl -s "http://localhost:8080/stats?user_id=user-123" | jq
```

Response schema:

```json
{
  "user_id": "user-123",
  "current_window_count": 3,
  "remaining_requests": 2,
  "reset_time_seconds": 38
}
```

Status behavior:

- `200 OK` on success
- `400 Bad Request` when `user_id` query param is missing
- `503 Service Unavailable` when Redis is unavailable
- `405 Method Not Allowed` for non-`GET`

## How Rate Limiting Works

Redis key pattern:

- `rate_limit:{<user_id>}` (`{}` keeps hash tags consistent for Redis Cluster slotting)

Lua flow (`internal/limiter/script.lua`):

1. Remove expired entries (`ZREMRANGEBYSCORE`)
2. Count current window entries (`ZCARD`)
3. Optionally add current request (`ZADD`) when under limit
4. Set key expiry (`PEXPIRE`) to window length
5. Return `allowed`, `current_window_count`, `remaining_requests`, `reset_time_seconds`

This is executed atomically through `redis.NewScript(...).Run(...)`, avoiding race conditions across goroutines and multiple app instances.

## Notes and Tradeoffs

- Sliding Window Log is accurate but stores one record per accepted request.
- Request payload is accepted but not persisted/processed by this service.
- Stats are per-user only (no global aggregation endpoint).
- Service logs Redis startup ping failures and still starts; `/request` is fail-open while `/stats` returns `503` if Redis is down.

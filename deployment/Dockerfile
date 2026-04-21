FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/rate-limiter ./cmd/server

FROM alpine:3.20

RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /bin/rate-limiter /usr/local/bin/rate-limiter
COPY .env ./

USER app
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/rate-limiter"]

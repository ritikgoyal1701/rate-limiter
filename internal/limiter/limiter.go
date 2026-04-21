package limiter

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultLimit       = 5
	defaultWindow      = 60 * time.Second
	rateLimitKeyPrefix = "rate_limit:"
)

//go:embed script.lua
var slidingWindowScript string

type Decision struct {
	Allowed            bool  `json:"allowed"`
	CurrentWindowCount int64 `json:"current_window_count"`
	RemainingRequests  int64 `json:"remaining_requests"`
	ResetTimeSeconds   int64 `json:"reset_time_seconds"`
}

type Limiter struct {
	client   redis.UniversalClient
	script   *redis.Script
	limit    int64
	windowMS int64
}

func New(client redis.UniversalClient, limit int64, window time.Duration) *Limiter {
	if limit <= 0 {
		limit = defaultLimit
	}
	if window <= 0 {
		window = defaultWindow
	}

	return &Limiter{
		client:   client,
		script:   redis.NewScript(slidingWindowScript),
		limit:    limit,
		windowMS: window.Milliseconds(),
	}
}

func (l *Limiter) Allow(ctx context.Context, userID string) (Decision, error) {
	nowMS := time.Now().UnixMilli()
	requestID := fmt.Sprintf("%d-%s", nowMS, uuid.NewString())
	return l.eval(ctx, userID, nowMS, requestID, true)
}

func (l *Limiter) Stats(ctx context.Context, userID string) (Decision, error) {
	nowMS := time.Now().UnixMilli()
	return l.eval(ctx, userID, nowMS, "", false)
}

func (l *Limiter) eval(ctx context.Context, userID string, nowMS int64, requestID string, shouldAdd bool) (Decision, error) {
	key := fmt.Sprintf("%s{%s}", rateLimitKeyPrefix, userID)
	flag := 0
	if shouldAdd {
		flag = 1
	}
	values, err := l.script.Run(ctx, l.client, []string{key}, nowMS, l.windowMS, l.limit, requestID, flag).Result()
	if err != nil {
		return Decision{}, err
	}

	resultSlice, ok := values.([]interface{})
	if !ok || len(resultSlice) != 4 {
		return Decision{}, fmt.Errorf("unexpected script response: %T", values)
	}

	allowedVal, err := toInt64(resultSlice[0])
	if err != nil {
		return Decision{}, fmt.Errorf("parse allowed flag: %w", err)
	}
	currentCount, err := toInt64(resultSlice[1])
	if err != nil {
		return Decision{}, fmt.Errorf("parse current count: %w", err)
	}
	remaining, err := toInt64(resultSlice[2])
	if err != nil {
		return Decision{}, fmt.Errorf("parse remaining: %w", err)
	}
	resetSeconds, err := toInt64(resultSlice[3])
	if err != nil {
		return Decision{}, fmt.Errorf("parse reset seconds: %w", err)
	}

	return Decision{
		Allowed:            allowedVal == 1,
		CurrentWindowCount: currentCount,
		RemainingRequests:  remaining,
		ResetTimeSeconds:   resetSeconds,
	}, nil
}

func toInt64(v interface{}) (int64, error) {
	switch t := v.(type) {
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}

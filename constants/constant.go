package constants

import "time"

const (
	AllowedFailOpen = "allowed_fail_open"
	Allowed         = "allowed"
	RateLimitExceeded = "rate_limit_exceeded"
    RateLimit = 5
    RateLimitWindow = 60 * time.Second
    ContentType = "application/json"
)
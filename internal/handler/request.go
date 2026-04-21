package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"

	"rate-limiter/internal/limiter"
)

type Metrics struct {
	AllowedCount atomic.Uint64
	LimitedCount atomic.Uint64
}

type RequestHandler struct {
	limiter *limiter.Limiter
	logger  *slog.Logger
	metrics *Metrics
}

type requestInput struct {
	UserID  string      `json:"user_id"`
	Payload interface{} `json:"payload"`
}

type requestResponse struct {
	Allowed            bool   `json:"allowed"`
	Status             string `json:"status"`
	CurrentWindowCount int64  `json:"current_window_count"`
	RemainingRequests  int64  `json:"remaining_requests"`
	ResetTimeSeconds   int64  `json:"reset_time_seconds"`
}

func NewRequestHandler(l *limiter.Limiter, logger *slog.Logger, metrics *Metrics) *RequestHandler {
	return &RequestHandler{
		limiter: l,
		logger:  logger,
		metrics: metrics,
	}
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var input requestInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if input.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id is required"})
		return
	}

	decision, err := h.limiter.Allow(r.Context(), input.UserID)
	if err != nil {
		// Fail-open strategy: allow traffic when Redis is unavailable.
		h.metrics.AllowedCount.Add(1)
		h.logger.Error("rate limiter unavailable, failing open", "err", err, "user_id", input.UserID)
		writeJSON(w, http.StatusOK, requestResponse{
			Allowed: true,
			Status:  "allowed_fail_open",
		})
		return
	}

	if !decision.Allowed {
		h.metrics.LimitedCount.Add(1)
		writeJSON(w, http.StatusTooManyRequests, requestResponse{
			Allowed:            false,
			Status:             "rate_limit_exceeded",
			CurrentWindowCount: decision.CurrentWindowCount,
			RemainingRequests:  decision.RemainingRequests,
			ResetTimeSeconds:   decision.ResetTimeSeconds,
		})
		return
	}

	h.metrics.AllowedCount.Add(1)
	writeJSON(w, http.StatusOK, requestResponse{
		Allowed:            true,
		Status:             "allowed",
		CurrentWindowCount: decision.CurrentWindowCount,
		RemainingRequests:  decision.RemainingRequests,
		ResetTimeSeconds:   decision.ResetTimeSeconds,
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

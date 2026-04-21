package handler

import (
	"log/slog"
	"net/http"

	"rate-limiter/internal/handler/adapter"
	"rate-limiter/internal/limiter"
)

type StatsHandler struct {
	limiter *limiter.Limiter
	logger  *slog.Logger
}

func NewStatsHandler(l *limiter.Limiter, logger *slog.Logger) *StatsHandler {
	return &StatsHandler{
		limiter: l,
		logger:  logger,
	}
}

func (h *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user_id query parameter is required"})
		return
	}

	decision, err := h.limiter.Stats(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to fetch stats", "err", err, "user_id", userID)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "stats unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, adapter.GetStatsResponse(userID, decision.CurrentWindowCount, decision.RemainingRequests, decision.ResetTimeSeconds))
}

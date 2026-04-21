package response

type StatsResponse struct {
	UserID             string `json:"user_id"`
	CurrentWindowCount int64  `json:"current_window_count"`
	RemainingRequests  int64  `json:"remaining_requests"`
	ResetTimeSeconds   int64  `json:"reset_time_seconds"`
}
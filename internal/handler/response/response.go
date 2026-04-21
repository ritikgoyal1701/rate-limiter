package response

type RequestResponse struct {
	Allowed            bool   `json:"allowed"`
	Status             string `json:"status"`
	CurrentWindowCount int64  `json:"current_window_count"`
	RemainingRequests  int64  `json:"remaining_requests"`
	ResetTimeSeconds   int64  `json:"reset_time_seconds"`
}

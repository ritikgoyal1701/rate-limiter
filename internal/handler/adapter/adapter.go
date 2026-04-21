package adapter

import "rate-limiter/internal/handler/response"

func GetResponse(
    allowed bool, 
    currentWindowCount int64, 
    remainingRequests int64, 
    resetTimeSeconds int64, 
    status string,
) *response.RequestResponse {
    return &response.RequestResponse{
        Allowed:            allowed,
        Status:             status,
        CurrentWindowCount: currentWindowCount,
        RemainingRequests:  remainingRequests,
        ResetTimeSeconds:   resetTimeSeconds,
    }
}

func GetStatsResponse(
    userID string, 
    currentWindowCount int64, 
    remainingRequests int64, 
    resetTimeSeconds int64,
) *response.StatsResponse {
    return &response.StatsResponse{
        UserID:             userID,
        CurrentWindowCount: currentWindowCount,
        RemainingRequests:  remainingRequests,
        ResetTimeSeconds:   resetTimeSeconds,
    }
}
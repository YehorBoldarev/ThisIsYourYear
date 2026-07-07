package http

import (
	"time"

	"kibou/thisisyouryear/internal/service"
)

// TopicTallyResponse is how many topics the user has and how many are complete.
type TopicTallyResponse struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
}

// CountdownResponse carries the next New-Year instant (UTC) and the whole
// seconds remaining. The client localizes the target to the browser timezone.
type CountdownResponse struct {
	Target           time.Time `json:"target"`
	SecondsRemaining int64     `json:"secondsRemaining"`
}

// DashboardResponse is the public projection of the year-level summary.
type DashboardResponse struct {
	Year      int                `json:"year"`
	Message   string             `json:"message"`
	Progress  ProgressResponse   `json:"progress"`
	Topics    TopicTallyResponse `json:"topics"`
	Countdown CountdownResponse  `json:"countdown"`
}

func dashboardResponse(v *service.DashboardView) DashboardResponse {
	return DashboardResponse{
		Year:    v.Year,
		Message: v.Message,
		Progress: ProgressResponse{
			Total:     v.Progress.Total,
			Completed: v.Progress.Completed,
			Ratio:     v.Progress.Ratio,
		},
		Topics: TopicTallyResponse{
			Total:     v.TopicsTotal,
			Completed: v.TopicsCompleted,
		},
		Countdown: CountdownResponse{
			Target:           v.CountdownTarget,
			SecondsRemaining: v.SecondsRemaining,
		},
	}
}

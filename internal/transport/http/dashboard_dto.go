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

// CountdownResponse carries the countdown instant (UTC) and the whole seconds
// remaining: the start of the year being planned during the preparation stage,
// otherwise the next New Year — local midnight in the user's timezone.
type CountdownResponse struct {
	Target           time.Time `json:"target"`
	SecondsRemaining int64     `json:"secondsRemaining"`
}

// DashboardResponse is the public projection of the year-level summary.
type DashboardResponse struct {
	Year int `json:"year"`
	// Phase is "preparation" before Year has begun, otherwise "year".
	Phase     string             `json:"phase"`
	Message   string             `json:"message"`
	Progress  ProgressResponse   `json:"progress"`
	Topics    TopicTallyResponse `json:"topics"`
	Countdown CountdownResponse  `json:"countdown"`
}

func dashboardResponse(v *service.DashboardView) DashboardResponse {
	return DashboardResponse{
		Year:    v.Year,
		Phase:   string(v.Phase),
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

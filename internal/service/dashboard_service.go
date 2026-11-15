package service

import (
	"context"
	"time"

	"kibou/thisisyouryear/internal/domain"
)

// DashboardTopics is the collaborator the dashboard needs: the caller's topics
// with their live progress. *TopicService satisfies it, so the dashboard reuses
// the exact (batched, non-drifting) progress path rather than re-querying goals.
type DashboardTopics interface {
	List(ctx context.Context, userID string, year *int, includeArchived bool) ([]TopicView, error)
}

// DashboardView is the aggregate landing summary for a user's year.
type DashboardView struct {
	Year             int
	Phase            domain.Phase
	Message          string
	Progress         domain.Progress
	TopicsTotal      int
	TopicsCompleted  int
	CountdownTarget  time.Time
	SecondsRemaining int64
}

// DashboardService composes the topic views into one year-level summary. It owns
// no persistence and no clock: the season (resolved per request) says which
// year to summarise and when "now" is.
type DashboardService struct {
	topics DashboardTopics
}

// NewDashboardService wires a DashboardService.
func NewDashboardService(topics DashboardTopics) *DashboardService {
	return &DashboardService{topics: topics}
}

// Get builds the dashboard for the season's year (resolved per request from the
// user's timezone and the preparation-stage setting): aggregate progress and
// topic tallies over its non-archived topics, a positive message, and the
// countdown — to the year's start while preparing, else to the next New Year.
func (s *DashboardService) Get(ctx context.Context, userID string, season domain.Season) (*DashboardView, error) {
	year := season.Year
	views, err := s.topics.List(ctx, userID, &year, false)
	if err != nil {
		return nil, err
	}

	progresses := make([]domain.Progress, len(views))
	completedTopics := 0
	for i := range views {
		p := views[i].Progress
		progresses[i] = p
		// A topic counts as complete only if it has goals and all are done.
		if p.Total > 0 && p.Ratio >= 1.0 {
			completedTopics++
		}
	}

	overall := domain.AggregateProgress(progresses)
	message := domain.DashboardMessage(overall, len(views))
	if season.Preparing {
		message = domain.PreparationMessage(year, len(views))
	}
	target := season.CountdownTarget().UTC()

	return &DashboardView{
		Year:             year,
		Phase:            season.Phase(),
		Message:          message,
		Progress:         overall,
		TopicsTotal:      len(views),
		TopicsCompleted:  completedTopics,
		CountdownTarget:  target,
		SecondsRemaining: season.SecondsRemaining(),
	}, nil
}

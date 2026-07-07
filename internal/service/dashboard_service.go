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
	Message          string
	Progress         domain.Progress
	TopicsTotal      int
	TopicsCompleted  int
	CountdownTarget  time.Time
	SecondsRemaining int64
}

// DashboardService composes the topic views into one year-level summary. It owns
// no persistence of its own.
type DashboardService struct {
	topics DashboardTopics
	now    func() time.Time
}

// NewDashboardService wires a DashboardService. The clock is injected so the
// New-Year countdown is deterministic in tests; it defaults to time.Now.
func NewDashboardService(topics DashboardTopics, now func() time.Time) *DashboardService {
	if now == nil {
		now = time.Now
	}
	return &DashboardService{topics: topics, now: now}
}

// Get builds the dashboard for the caller's current-year, non-archived topics:
// aggregate progress, topic tallies, a positive message, and the New-Year
// countdown.
func (s *DashboardService) Get(ctx context.Context, userID string) (*DashboardView, error) {
	views, err := s.topics.List(ctx, userID, nil, false)
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
	now := s.now().UTC()
	target, seconds := domain.Countdown(now)

	return &DashboardView{
		Year:             domain.CurrentYear(),
		Message:          domain.DashboardMessage(overall, len(views)),
		Progress:         overall,
		TopicsTotal:      len(views),
		TopicsCompleted:  completedTopics,
		CountdownTarget:  target,
		SecondsRemaining: seconds,
	}, nil
}

package http

import (
	"time"

	"kibou/thisisyouryear/internal/domain"
)

// CreateGoalRequest is the body of POST /topics/{id}/goals.
type CreateGoalRequest struct {
	Title string `json:"title" validate:"required,max=200"`
	Notes string `json:"notes" validate:"max=5000"`
}

// UpdateGoalRequest is the body of PATCH /goals/{id}. Pointer fields make the
// update partial; `done` toggles achievement state, `order` reorders.
type UpdateGoalRequest struct {
	Title *string `json:"title" validate:"omitempty,min=1,max=200"`
	Notes *string `json:"notes" validate:"omitempty,max=5000"`
	Done  *bool   `json:"done"`
	Order *int    `json:"order" validate:"omitempty,min=0"`
}

func (r UpdateGoalRequest) toDomain() domain.GoalUpdate {
	return domain.GoalUpdate{
		Title: r.Title,
		Notes: r.Notes,
		Done:  r.Done,
		Order: r.Order,
	}
}

// GoalResponse is the public projection of a goal.
type GoalResponse struct {
	ID          string     `json:"id"`
	TopicID     string     `json:"topicId"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes"`
	Status      string     `json:"status"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Order       int        `json:"order"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

func goalResponse(g *domain.Goal) GoalResponse {
	return GoalResponse{
		ID:          g.ID,
		TopicID:     g.TopicID,
		Title:       g.Title,
		Notes:       g.Notes,
		Status:      string(g.Status),
		CompletedAt: g.CompletedAt,
		Order:       g.Order,
		CreatedAt:   g.CreatedAt,
		UpdatedAt:   g.UpdatedAt,
	}
}

func goalResponses(goals []domain.Goal) []GoalResponse {
	out := make([]GoalResponse, 0, len(goals))
	for i := range goals {
		out = append(out, goalResponse(&goals[i]))
	}
	return out
}

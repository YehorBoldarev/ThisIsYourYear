package http

import (
	"time"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// CreateTopicRequest is the body of POST /topics.
type CreateTopicRequest struct {
	Title       string `json:"title" validate:"required,max=120"`
	Description string `json:"description" validate:"max=2000"`
	// Year is optional; when 0 the domain defaults it to the current year.
	Year  int    `json:"year" validate:"omitempty,min=2000,max=2100"`
	Color string `json:"color" validate:"omitempty,hexcolor"`
	Icon  string `json:"icon" validate:"max=64"`
}

// UpdateTopicRequest is the body of PATCH /topics/{id}. Pointer fields make the
// update partial: only supplied fields change.
type UpdateTopicRequest struct {
	Title       *string `json:"title" validate:"omitempty,min=1,max=120"`
	Description *string `json:"description" validate:"omitempty,max=2000"`
	Year        *int    `json:"year" validate:"omitempty,min=2000,max=2100"`
	Color       *string `json:"color" validate:"omitempty,hexcolor"`
	Icon        *string `json:"icon" validate:"omitempty,max=64"`
	Archived    *bool   `json:"archived"`
}

func (r UpdateTopicRequest) toDomain() domain.TopicUpdate {
	return domain.TopicUpdate{
		Title:       r.Title,
		Description: r.Description,
		Year:        r.Year,
		Color:       r.Color,
		Icon:        r.Icon,
		Archived:    r.Archived,
	}
}

// ProgressResponse is the computed completion of a topic.
type ProgressResponse struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Ratio     float64 `json:"ratio"`
}

// TopicResponse is the public projection of a topic.
type TopicResponse struct {
	ID          string           `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Year        int              `json:"year"`
	Color       string           `json:"color"`
	Icon        string           `json:"icon"`
	Archived    bool             `json:"archived"`
	Progress    ProgressResponse `json:"progress"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

func topicResponse(v *service.TopicView) TopicResponse {
	t := v.Topic
	return TopicResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Year:        t.Year,
		Color:       t.Color,
		Icon:        t.Icon,
		Archived:    t.IsArchived(),
		Progress: ProgressResponse{
			Total:     v.Progress.Total,
			Completed: v.Progress.Completed,
			Ratio:     v.Progress.Ratio,
		},
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func topicResponses(views []service.TopicView) []TopicResponse {
	out := make([]TopicResponse, 0, len(views))
	for i := range views {
		out = append(out, topicResponse(&views[i]))
	}
	return out
}

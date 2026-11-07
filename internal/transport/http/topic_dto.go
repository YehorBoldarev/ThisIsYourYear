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
	Year     int    `json:"year" validate:"omitempty,min=2000,max=2100"`
	Color    string `json:"color" validate:"omitempty,hexcolor"`
	Icon     string `json:"icon" validate:"max=64"`
	Category string `json:"category" validate:"omitempty,max=60"`
	// TargetDate is an optional deadline in DD-MM-YYYY form.
	TargetDate string `json:"targetDate" validate:"omitempty,datetime=02-01-2006"`
}

// toInput maps the validated request to the service input, parsing the date.
func (r CreateTopicRequest) toInput() service.CreateTopicInput {
	return service.CreateTopicInput{
		Title:       r.Title,
		Description: r.Description,
		Year:        r.Year,
		Color:       r.Color,
		Icon:        r.Icon,
		Category:    r.Category,
		TargetDate:  parseDate(r.TargetDate),
	}
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
	Category    *string `json:"category" validate:"omitempty,max=60"`
	// TargetDate: omit to leave unchanged, "" to clear, DD-MM-YYYY to set.
	TargetDate *optionalDate `json:"targetDate,omitempty"`
}

func (r UpdateTopicRequest) toDomain() domain.TopicUpdate {
	return domain.TopicUpdate{
		Title:       r.Title,
		Description: r.Description,
		Year:        r.Year,
		Color:       r.Color,
		Icon:        r.Icon,
		Archived:    r.Archived,
		Category:    r.Category,
		TargetDate:  r.TargetDate.fieldUpdate(),
	}
}

// ProgressResponse is the computed completion of a topic.
type ProgressResponse struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Ratio     float64 `json:"ratio"`
}

// MotivationResponse is the encouraging copy for a topic's current progress
// band. Band is a stable identifier clients key presentation off.
type MotivationResponse struct {
	Band    string `json:"band"`
	Label   string `json:"label"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func motivationResponse(m domain.Motivation) MotivationResponse {
	return MotivationResponse{
		Band:    string(m.Band),
		Label:   m.Label,
		Title:   m.Title,
		Message: m.Message,
	}
}

// TopicResponse is the public projection of a topic.
type TopicResponse struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Year        int                `json:"year"`
	Color       string             `json:"color"`
	Icon        string             `json:"icon"`
	Category    string             `json:"category"`
	TargetDate  string             `json:"targetDate"`
	Archived    bool               `json:"archived"`
	Progress    ProgressResponse   `json:"progress"`
	Motivation  MotivationResponse `json:"motivation"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
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
		Category:    t.Category,
		TargetDate:  formatDate(t.TargetDate),
		Archived:    t.IsArchived(),
		Progress: ProgressResponse{
			Total:     v.Progress.Total,
			Completed: v.Progress.Completed,
			Ratio:     v.Progress.Ratio,
		},
		Motivation: motivationResponse(v.Motivation),
		CreatedAt:  t.CreatedAt,
		UpdatedAt:  t.UpdatedAt,
	}
}

func topicResponses(views []service.TopicView) []TopicResponse {
	out := make([]TopicResponse, 0, len(views))
	for i := range views {
		out = append(out, topicResponse(&views[i]))
	}
	return out
}

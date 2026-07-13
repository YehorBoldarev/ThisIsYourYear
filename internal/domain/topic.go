package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Topic field limits, shared by entity construction and partial updates.
const (
	TopicTitleMaxLen       = 120
	TopicDescriptionMaxLen = 2000
	TopicIconMaxLen        = 64
	TopicCategoryMaxLen    = 60

	minTopicYear = 2000
	maxTopicYear = 2100
)

// hexColorRe accepts #RGB or #RRGGBB, matching the transport layer's `hexcolor`
// validation so both layers agree on what a valid color is.
var hexColorRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// Topic is a user's high-level yearly goal. It is a pure domain entity; the
// persistence layer maps it to its own document model.
type Topic struct {
	ID          string
	UserID      string
	Title       string
	Description string
	Year        int
	Color       string
	Icon        string
	Category    string
	TargetDate  *time.Time // optional deadline, date-only (UTC midnight)
	ArchivedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsArchived reports whether the topic has been archived.
func (t *Topic) IsArchived() bool { return t.ArchivedAt != nil }

// NewTopic constructs a valid Topic owned by userID. Text fields are trimmed,
// year defaults to the current UTC year when zero, and all invariants are
// enforced; violations are reported as ErrValidation. The returned topic has no
// ID or timestamps until it is persisted.
func NewTopic(userID, title, description string, year int, color, icon, category string, targetDate *time.Time) (*Topic, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	color = strings.TrimSpace(color)
	icon = strings.TrimSpace(icon)
	category = strings.TrimSpace(category)

	if userID == "" {
		return nil, fmt.Errorf("%w: userID is required", ErrValidation)
	}
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	if err := validateOptionalText("description", description, TopicDescriptionMaxLen); err != nil {
		return nil, err
	}
	if err := validateOptionalText("icon", icon, TopicIconMaxLen); err != nil {
		return nil, err
	}
	if err := validateOptionalText("category", category, TopicCategoryMaxLen); err != nil {
		return nil, err
	}
	if err := validateColor(color); err != nil {
		return nil, err
	}
	if year == 0 {
		year = CurrentYear()
	}
	if err := ValidateYear(year); err != nil {
		return nil, err
	}

	return &Topic{
		UserID:      userID,
		Title:       title,
		Description: description,
		Year:        year,
		Color:       color,
		Icon:        icon,
		Category:    category,
		TargetDate:  dateOnlyUTC(targetDate),
	}, nil
}

// TopicUpdate is a partial update to a topic: a nil field is left unchanged. It
// is the domain-level description of a PATCH, keeping storage (bson) and
// transport (DTO) concerns out of the service.
type TopicUpdate struct {
	Title       *string
	Description *string
	Year        *int
	Color       *string
	Icon        *string
	Archived    *bool
	Category    *string
	TargetDate  FieldUpdate[time.Time]
}

// IsEmpty reports whether the update would change nothing.
func (u TopicUpdate) IsEmpty() bool {
	return u.Title == nil && u.Description == nil && u.Year == nil &&
		u.Color == nil && u.Icon == nil && u.Archived == nil &&
		u.Category == nil && !u.TargetDate.Present
}

// Validate checks every supplied field against the same rules as NewTopic.
func (u TopicUpdate) Validate() error {
	if u.Title != nil {
		if err := validateTitle(strings.TrimSpace(*u.Title)); err != nil {
			return err
		}
	}
	if u.Description != nil {
		if err := validateOptionalText("description", strings.TrimSpace(*u.Description), TopicDescriptionMaxLen); err != nil {
			return err
		}
	}
	if u.Icon != nil {
		if err := validateOptionalText("icon", strings.TrimSpace(*u.Icon), TopicIconMaxLen); err != nil {
			return err
		}
	}
	if u.Color != nil {
		if err := validateColor(strings.TrimSpace(*u.Color)); err != nil {
			return err
		}
	}
	if u.Category != nil {
		if err := validateOptionalText("category", strings.TrimSpace(*u.Category), TopicCategoryMaxLen); err != nil {
			return err
		}
	}
	if u.Year != nil {
		if err := ValidateYear(*u.Year); err != nil {
			return err
		}
	}
	return nil
}

// NormalizedTargetDate returns the target-date update normalized to date-only
// UTC when a value is being set, so storage and construction agree on the shape.
func (u TopicUpdate) NormalizedTargetDate() FieldUpdate[time.Time] {
	if u.TargetDate.Present && u.TargetDate.Value != nil {
		return FieldUpdate[time.Time]{Present: true, Value: dateOnlyUTC(u.TargetDate.Value)}
	}
	return u.TargetDate
}

// ValidateYear checks a year is within the accepted range.
func ValidateYear(year int) error {
	if year < minTopicYear || year > maxTopicYear {
		return fmt.Errorf("%w: year must be between %d and %d", ErrValidation, minTopicYear, maxTopicYear)
	}
	return nil
}

// CurrentYear returns the current UTC year, used as the default topic year and
// the default list scope.
func CurrentYear() int { return time.Now().UTC().Year() }

func validateTitle(title string) error {
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if len([]rune(title)) > TopicTitleMaxLen {
		return fmt.Errorf("%w: title must be at most %d characters", ErrValidation, TopicTitleMaxLen)
	}
	return nil
}

func validateOptionalText(field, value string, maxLen int) error {
	if len([]rune(value)) > maxLen {
		return fmt.Errorf("%w: %s must be at most %d characters", ErrValidation, field, maxLen)
	}
	return nil
}

func validateColor(color string) error {
	if color != "" && !hexColorRe.MatchString(color) {
		return fmt.Errorf("%w: color must be a hex value like #RRGGBB", ErrValidation)
	}
	return nil
}

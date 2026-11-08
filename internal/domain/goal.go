package domain

import (
	"fmt"
	"strings"
	"time"
)

// Goal field limits.
const (
	GoalTitleMaxLen = 200
	GoalNotesMaxLen = 5000
)

// GoalStatus is the lifecycle state of a goal.
type GoalStatus string

// The two goal states. The values are stable wire identifiers.
const (
	GoalStatusTodo GoalStatus = "todo"
	GoalStatusDone GoalStatus = "done"
)

// Goal is a concrete, achievable step under a Topic. A goal with status "done"
// (and a completedAt) is what the product calls an achievement.
type Goal struct {
	ID            string
	TopicID       string
	UserID        string
	Title         string
	Notes         string
	Status        GoalStatus
	CompletedAt   *time.Time
	Order         int
	DueDate       *time.Time // optional deadline, date-only (UTC midnight)
	IsAchievement bool       // when done, celebrate this milestone as an achievement (★)
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// IsAchieved reports whether the goal has been completed.
func (g *Goal) IsAchieved() bool { return g.Status == GoalStatusDone }

// NewGoal constructs a valid, not-yet-completed goal under a topic. Order is
// assigned by the caller (the service appends it to the topic's list). Text is
// trimmed and validated; violations are ErrValidation.
func NewGoal(userID, topicID, title, notes string, isAchievement bool, dueDate *time.Time) (*Goal, error) {
	title = strings.TrimSpace(title)
	notes = strings.TrimSpace(notes)

	if userID == "" {
		return nil, fmt.Errorf("%w: userID is required", ErrValidation)
	}
	if topicID == "" {
		return nil, fmt.Errorf("%w: topicID is required", ErrValidation)
	}
	if err := validateGoalTitle(title); err != nil {
		return nil, err
	}
	if err := validateOptionalText("notes", notes, GoalNotesMaxLen); err != nil {
		return nil, err
	}

	return &Goal{
		UserID:        userID,
		TopicID:       topicID,
		Title:         title,
		Notes:         notes,
		Status:        GoalStatusTodo,
		DueDate:       dateOnlyUTC(dueDate),
		IsAchievement: isAchievement,
	}, nil
}

// GoalUpdate is a partial update to a goal: a nil field is left unchanged. Done
// is the completion lever — true marks the goal an achievement (stamping
// completedAt), false reverts it to todo (clearing completedAt).
type GoalUpdate struct {
	Title         *string
	Notes         *string
	Done          *bool
	Order         *int
	DueDate       FieldUpdate[time.Time]
	IsAchievement *bool
}

// IsEmpty reports whether the update would change nothing.
func (u GoalUpdate) IsEmpty() bool {
	return u.Title == nil && u.Notes == nil && u.Done == nil && u.Order == nil &&
		!u.DueDate.Present && u.IsAchievement == nil
}

// NormalizedDueDate returns the due-date update normalized to date-only UTC when
// a value is being set.
func (u GoalUpdate) NormalizedDueDate() FieldUpdate[time.Time] {
	if u.DueDate.Present && u.DueDate.Value != nil {
		return FieldUpdate[time.Time]{Present: true, Value: dateOnlyUTC(u.DueDate.Value)}
	}
	return u.DueDate
}

// Validate checks every supplied field against the same rules as NewGoal.
func (u GoalUpdate) Validate() error {
	if u.Title != nil {
		if err := validateGoalTitle(strings.TrimSpace(*u.Title)); err != nil {
			return err
		}
	}
	if u.Notes != nil {
		if err := validateOptionalText("notes", strings.TrimSpace(*u.Notes), GoalNotesMaxLen); err != nil {
			return err
		}
	}
	if u.Order != nil && *u.Order < 0 {
		return fmt.Errorf("%w: order must not be negative", ErrValidation)
	}
	return nil
}

func validateGoalTitle(title string) error {
	if title == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}
	if len([]rune(title)) > GoalTitleMaxLen {
		return fmt.Errorf("%w: title must be at most %d characters", ErrValidation, GoalTitleMaxLen)
	}
	return nil
}

package domain

import "time"

// FieldUpdate expresses an optional, clearable field in a partial update. It
// distinguishes three states that a plain pointer cannot:
//
//   - Present == false        -> the field was omitted; leave it unchanged.
//   - Present == true, Value nil -> the field was supplied as empty; clear it.
//   - Present == true, Value set  -> set the field to *Value.
//
// The zero value is the "leave unchanged" state, so callers only need the
// SetField / ClearField constructors for the other two.
type FieldUpdate[T any] struct {
	Present bool
	Value   *T
}

// SetField marks the field present with a new value.
func SetField[T any](v T) FieldUpdate[T] { return FieldUpdate[T]{Present: true, Value: &v} }

// ClearField marks the field present with no value (clear it).
func ClearField[T any]() FieldUpdate[T] { return FieldUpdate[T]{Present: true} }

// dateOnlyUTC normalizes an instant to date-only at UTC midnight, so target/due
// dates carry no time-of-day and never drift across timezones. A nil input
// stays nil (unset).
func dateOnlyUTC(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	y, m, d := t.Date()
	normalized := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &normalized
}

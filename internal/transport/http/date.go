package http

import (
	"encoding/json"
	"time"

	"kibou/thisisyouryear/internal/domain"
)

// dateLayout is the API's calendar-date format, DD-MM-YYYY. Target and due dates
// are exchanged as date-only strings (no time-of-day, no timezone) to avoid the
// day-shift an instant would risk. It matches the `datetime=02-01-2006`
// validator tag on the create-request DTOs.
const dateLayout = "02-01-2006"

// parseDate turns a create-request date string into a UTC date, treating "" as
// unset. It assumes the value already passed the `datetime` validator, so a
// parse error is not expected; it degrades to nil rather than panicking.
func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return nil
	}
	t = t.UTC()
	return &t
}

// formatDate renders a stored date as DD-MM-YYYY, or "" when unset.
func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dateLayout)
}

// optionalDate is a PATCH date field. It is used as a *pointer* in request DTOs
// (`*optionalDate` with `omitempty`), which gives three states the JSON contract
// needs:
//
//   - key omitted (nil pointer) -> leave unchanged.
//   - empty string ""           -> clear the field (value nil).
//   - "DD-MM-YYYY"              -> set the field.
//
// (JSON null decodes a pointer to nil without calling UnmarshalJSON, so null is
// treated the same as omitted: leave unchanged. Clearing uses "".) The format is
// validated on decode, because the struct validator's `omitempty` cannot skip a
// non-nil pointer to "".
type optionalDate struct {
	value *time.Time // nil means "clear"
}

// UnmarshalJSON parses "" as clear and a DD-MM-YYYY string as a set. A malformed
// date is a validation error (surfaced as a 422 by the request decoder).
func (d *optionalDate) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return &ValidationError{msg: "date must be a string in DD-MM-YYYY format"}
	}
	if s == "" {
		d.value = nil
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return &ValidationError{msg: "date must be in DD-MM-YYYY format"}
	}
	t = t.UTC()
	d.value = &t
	return nil
}

// MarshalJSON renders a set date as DD-MM-YYYY and a clear as "" (used by tests
// building request bodies; production clients send JSON directly).
func (d optionalDate) MarshalJSON() ([]byte, error) {
	if d.value == nil {
		return []byte(`""`), nil
	}
	return json.Marshal(d.value.Format(dateLayout))
}

// fieldUpdate maps the pointer state onto the domain's partial-update field: a
// nil receiver (omitted key) leaves the value unchanged.
func (d *optionalDate) fieldUpdate() domain.FieldUpdate[time.Time] {
	if d == nil {
		return domain.FieldUpdate[time.Time]{}
	}
	if d.value == nil {
		return domain.ClearField[time.Time]()
	}
	return domain.SetField(*d.value)
}

package domain

import "time"

// Phase says whether the year the app is about has started yet. The values are
// stable wire identifiers.
type Phase string

const (
	// PhasePreparation means the year hasn't begun; users plan it while a
	// countdown runs to its first moment.
	PhasePreparation Phase = "preparation"
	// PhaseYear means the year is under way (the normal flow).
	PhaseYear Phase = "year"
)

// Season is the year the app is about for one user at one moment, and the
// boundaries of that year in the user's timezone.
type Season struct {
	Year      int
	Preparing bool
	// Now is the instant the season was resolved for; countdowns are measured
	// from it so one request uses one clock reading throughout.
	Now time.Time
	// StartsAt and EndsAt are local midnight on 1 January of Year and of Year+1.
	StartsAt time.Time
	EndsAt   time.Time
}

// SeasonAt decides which year the app is about. Normally that is the current
// year in loc. While preparationYear lies in the future (a pre-launch
// "preparation stage", e.g. 2027 during late 2026), it is preparationYear
// instead. The stage ends by itself at local midnight on 1 January of that
// year, after which preparationYear has no effect. A zero preparationYear
// disables the stage; a nil loc means UTC.
func SeasonAt(now time.Time, loc *time.Location, preparationYear int) Season {
	if loc == nil {
		loc = time.UTC
	}
	year := now.In(loc).Year()
	preparing := preparationYear > year
	if preparing {
		year = preparationYear
	}
	return Season{
		Year:      year,
		Preparing: preparing,
		Now:       now,
		StartsAt:  time.Date(year, time.January, 1, 0, 0, 0, 0, loc),
		EndsAt:    time.Date(year+1, time.January, 1, 0, 0, 0, 0, loc),
	}
}

// Phase reports whether the season's year has started.
func (s Season) Phase() Phase {
	if s.Preparing {
		return PhasePreparation
	}
	return PhaseYear
}

// CountdownTarget is the moment the dashboard counts down to: the start of the
// year while preparing for it, otherwise the next New Year.
func (s Season) CountdownTarget() time.Time {
	if s.Preparing {
		return s.StartsAt
	}
	return s.EndsAt
}

// SecondsRemaining is the whole seconds from Now until CountdownTarget, floored
// at zero.
func (s Season) SecondsRemaining() int64 {
	return SecondsUntil(s.CountdownTarget(), s.Now)
}

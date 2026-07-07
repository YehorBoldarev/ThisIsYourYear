package domain

import "time"

// dashboardEmptyMessage is shown when the user has no topics yet — a distinct,
// motivating empty state (the per-topic "set your first goal" copy is wrong here
// because there is no topic to add goals to). Kept beside the topic motivation
// copy so the app speaks in one voice.
const dashboardEmptyMessage = "A blank page is a fresh start. Create your first topic and shape your year."

// AggregateProgress sums goal counts across many topics into one overall
// Progress. It aggregates raw counts (not the mean of per-topic ratios), so
// e.g. 1/10 and 9/10 combine to 10/20 = 50%, which is the true overall
// completion. The result reuses ComputeProgress, inheriting its edge guards.
func AggregateProgress(items []Progress) Progress {
	total, completed := 0, 0
	for _, p := range items {
		total += p.Total
		completed += p.Completed
	}
	return ComputeProgress(total, completed)
}

// DashboardMessage picks the positive message for the whole year. With no topics
// it returns the empty-state line; otherwise it reuses the per-topic Motivate
// bands, now read as an overall status, so the copy has a single source of truth.
func DashboardMessage(overall Progress, topicCount int) string {
	if topicCount == 0 {
		return dashboardEmptyMessage
	}
	return Motivate(overall)
}

// NextNewYear returns the instant of the next New Year — Jan 1, 00:00:00 of the
// year after now — in now's own location.
func NextNewYear(now time.Time) time.Time {
	return time.Date(now.Year()+1, time.January, 1, 0, 0, 0, 0, now.Location())
}

// Countdown returns the next New Year instant and the whole seconds remaining
// until it, floored at zero. It is a pure function of now, so it is trivially
// testable and deterministic.
func Countdown(now time.Time) (target time.Time, secondsRemaining int64) {
	target = NextNewYear(now)
	secondsRemaining = max(int64(target.Sub(now).Seconds()), 0)
	return target, secondsRemaining
}

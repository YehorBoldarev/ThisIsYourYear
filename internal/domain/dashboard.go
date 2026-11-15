package domain

import (
	"fmt"
	"time"
)

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
	return Motivate(overall).Message
}

// PreparationMessage is the dashboard line during the preparation stage, when
// the year being planned hasn't started yet. Progress bands don't apply before
// the year begins, so the copy is about planning. Kept beside the other
// dashboard copy so the app speaks in one voice.
func PreparationMessage(year, topicCount int) string {
	if topicCount == 0 {
		return fmt.Sprintf("Your %d is a blank page. Plan the first goal before the year begins.", year)
	}
	return fmt.Sprintf("Every goal you plan now is a head start. %d is almost here.", year)
}

// SecondsUntil returns the whole seconds from now until target, floored at
// zero (a target in the past is "now").
func SecondsUntil(target, now time.Time) int64 {
	return max(int64(target.Sub(now).Seconds()), 0)
}

package domain

// Progress is the completion state of a set of goals: how many are done out of
// the total, and the resulting ratio in [0,1]. It is always computed, never
// stored, so it cannot drift from the underlying goals.
type Progress struct {
	Total     int     `json:"total"`
	Completed int     `json:"completed"`
	Ratio     float64 `json:"ratio"`
}

// ComputeProgress builds a Progress from raw counts, guarding the edges: counts
// are clamped to be non-negative and completed never exceeds total, and a total
// of zero yields a 0.0 ratio (rather than NaN). A topic with no goals is 0%.
func ComputeProgress(total, completed int) Progress {
	if total < 0 {
		total = 0
	}
	if completed < 0 {
		completed = 0
	}
	if completed > total {
		completed = total
	}
	ratio := 0.0
	if total > 0 {
		ratio = float64(completed) / float64(total)
	}
	return Progress{Total: total, Completed: completed, Ratio: ratio}
}

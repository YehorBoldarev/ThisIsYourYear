package domain

// MotivationBand is a coarse stage of a topic's completion, used to pick an
// encouraging message. Bands partition every possible Progress into exactly one
// stage (see MotivationBandOf for the exact, non-overlapping boundaries).
type MotivationBand int

const (
	// BandNoGoals: the topic has no goals yet (total == 0).
	BandNoGoals MotivationBand = iota
	// BandStarted: goals exist but none are done (ratio == 0, total > 0).
	BandStarted
	// BandQuarter: up to a quarter done (0 < ratio <= 0.25).
	BandQuarter
	// BandHalf: up to half done (0.25 < ratio <= 0.5).
	BandHalf
	// BandMost: past half but not finished (0.5 < ratio < 1.0).
	BandMost
	// BandComplete: every goal done (ratio == 1.0).
	BandComplete
)

// motivationMessages holds the single message shipped for each band. The copy
// lives in this one table so wording is a one-file change; the "encouraging &
// warm" voice is a product decision (see plans/06-motivation-engine.md §7).
var motivationMessages = map[MotivationBand]string{
	BandNoGoals:  "Set your first goal — every big year starts with one step.",
	BandStarted:  "The hardest part is starting, and you're already here. Let's go.",
	BandQuarter:  "You're on the board. Keep the momentum going.",
	BandHalf:     "Real progress — you're building something. Keep at it.",
	BandMost:     "More than halfway. The finish line is in sight.",
	BandComplete: "Every goal complete — this one's in the books. 🎉",
}

// MotivationBandOf classifies a Progress into exactly one band. Boundaries are
// upper-closed (e.g. exactly 0.25 is BandQuarter, exactly 0.5 is BandHalf), so
// every ratio in [0,1] maps to one band with no gaps or overlaps.
func MotivationBandOf(p Progress) MotivationBand {
	switch {
	case p.Total == 0:
		return BandNoGoals
	case p.Ratio == 0:
		return BandStarted
	case p.Ratio >= 1.0:
		return BandComplete
	case p.Ratio <= 0.25:
		return BandQuarter
	case p.Ratio <= 0.5:
		return BandHalf
	default:
		return BandMost
	}
}

// Motivate returns the encouraging message for a topic's current progress. It is
// a pure function of Progress — deterministic and free of side effects — so the
// dashboard can reuse it unchanged.
func Motivate(p Progress) string {
	return motivationMessages[MotivationBandOf(p)]
}

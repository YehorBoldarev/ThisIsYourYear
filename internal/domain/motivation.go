package domain

// MotivationBand is a coarse stage of a topic's completion, used to pick an
// encouraging message. Bands partition every possible Progress into exactly one
// stage (see MotivationBandOf for the exact, non-overlapping boundaries). The
// values are stable wire identifiers: clients key presentation (e.g. colour) off
// them, so they must not change.
type MotivationBand string

// The four bands from the design's motivational-message state reference
// (design_handoff_this_is_your_year/README.md, screen 7).
const (
	// BandNotStarted: no goals yet, or none done (ratio == 0).
	BandNotStarted MotivationBand = "not-started"
	// BandInProgress: some goals done, below the almost-there threshold.
	BandInProgress MotivationBand = "in-progress"
	// BandAlmostThere: at or past the threshold but not finished.
	BandAlmostThere MotivationBand = "almost-there"
	// BandAchieved: every goal done (ratio == 1.0).
	BandAchieved MotivationBand = "achieved"
)

// almostThereThreshold is the ratio at which a topic counts as "almost there"
// (the ~80% the design mock uses).
const almostThereThreshold = 0.8

// Motivation is the encouraging copy for one band: a short state label (e.g.
// "Almost there"), a title (e.g. "In sight"), and the message itself.
type Motivation struct {
	Band    MotivationBand
	Label   string
	Title   string
	Message string
}

// motivations holds the copy shipped for each band. It lives in this one table
// so wording is a one-file change; the text is the design's canonical copy.
var motivations = map[MotivationBand]Motivation{
	BandNotStarted: {
		Band:    BandNotStarted,
		Label:   "Just started",
		Title:   "The blank page",
		Message: "A blank page isn’t empty — it’s full of everything that could be. Make the first mark today.",
	},
	BandInProgress: {
		Band:    BandInProgress,
		Label:   "In progress",
		Title:   "Building momentum",
		Message: "Momentum is quiet. Keep showing up, and the small days compound into a remarkable year.",
	},
	BandAlmostThere: {
		Band:    BandAlmostThere,
		Label:   "Almost there",
		Title:   "In sight",
		Message: "You can see it from here. Don’t trade the finish line for comfort — finish what you started.",
	},
	BandAchieved: {
		Band:    BandAchieved,
		Label:   "Complete",
		Title:   "A promise kept",
		Message: "You kept a promise to yourself. This is exactly what that feels like. Take a moment to savour it.",
	},
}

// MotivationBandOf classifies a Progress into exactly one band. The
// almost-there boundary is lower-closed (exactly 0.8 is BandAlmostThere), so
// every ratio in [0,1] maps to one band with no gaps or overlaps.
func MotivationBandOf(p Progress) MotivationBand {
	switch {
	case p.Total == 0 || p.Ratio <= 0:
		return BandNotStarted
	case p.Ratio >= 1.0:
		return BandAchieved
	case p.Ratio >= almostThereThreshold:
		return BandAlmostThere
	default:
		return BandInProgress
	}
}

// Motivate returns the encouraging copy for a topic's current progress. It is
// a pure function of Progress — deterministic and free of side effects — so the
// dashboard can reuse it unchanged.
func Motivate(p Progress) Motivation {
	return motivations[MotivationBandOf(p)]
}

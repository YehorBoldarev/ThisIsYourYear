package domain

import (
	"testing"
	"time"
)

func TestAggregateProgress(t *testing.T) {
	cases := []struct {
		name          string
		items         []Progress
		wantTotal     int
		wantCompleted int
		wantRatio     float64
	}{
		{"none", nil, 0, 0, 0},
		{"single", []Progress{ComputeProgress(4, 1)}, 4, 1, 0.25},
		{
			"sums counts not ratios",
			[]Progress{ComputeProgress(10, 1), ComputeProgress(10, 9)},
			20, 10, 0.5,
		},
		{
			"empty topics contribute nothing",
			[]Progress{ComputeProgress(0, 0), ComputeProgress(2, 2)},
			2, 2, 1.0,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AggregateProgress(tc.items)
			if got.Total != tc.wantTotal || got.Completed != tc.wantCompleted {
				t.Fatalf("AggregateProgress = {%d,%d}, want {%d,%d}",
					got.Total, got.Completed, tc.wantTotal, tc.wantCompleted)
			}
			if diff := got.Ratio - tc.wantRatio; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("ratio = %v, want %v", got.Ratio, tc.wantRatio)
			}
		})
	}
}

func TestDashboardMessage(t *testing.T) {
	if got, want := DashboardMessage(ComputeProgress(0, 0), 0), dashboardEmptyMessage; got != want {
		t.Fatalf("no topics message = %q, want empty-state %q", got, want)
	}
	// With topics, it reuses the per-topic bands.
	if got, want := DashboardMessage(ComputeProgress(2, 2), 1), Motivate(ComputeProgress(2, 2)); got != want {
		t.Fatalf("has-topics message = %q, want Motivate band %q", got, want)
	}
	// Topics but no goals is NOT the empty state.
	if got := DashboardMessage(ComputeProgress(0, 0), 3); got == dashboardEmptyMessage {
		t.Fatalf("topics-with-no-goals must not use the empty-state message")
	}
}

func TestCountdown(t *testing.T) {
	now := time.Date(2026, time.July, 7, 12, 0, 0, 0, time.UTC)
	target, secs := Countdown(now)

	want := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	if !target.Equal(want) {
		t.Fatalf("target = %v, want %v", target, want)
	}
	if secs <= 0 {
		t.Fatalf("secondsRemaining = %d, want > 0", secs)
	}
	if wantSecs := int64(want.Sub(now).Seconds()); secs != wantSecs {
		t.Fatalf("secondsRemaining = %d, want %d", secs, wantSecs)
	}

	// A late-December "now" still targets the coming Jan 1 with a small positive remainder.
	lateDec := time.Date(2026, time.December, 31, 23, 59, 0, 0, time.UTC)
	target2, secs2 := Countdown(lateDec)
	if !target2.Equal(want) {
		t.Fatalf("late-Dec target = %v, want %v", target2, want)
	}
	if secs2 != 60 {
		t.Fatalf("late-Dec secondsRemaining = %d, want 60", secs2)
	}
}

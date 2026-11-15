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
	if got, want := DashboardMessage(ComputeProgress(2, 2), 1), Motivate(ComputeProgress(2, 2)).Message; got != want {
		t.Fatalf("has-topics message = %q, want Motivate band %q", got, want)
	}
	// Topics but no goals is NOT the empty state.
	if got := DashboardMessage(ComputeProgress(0, 0), 3); got == dashboardEmptyMessage {
		t.Fatalf("topics-with-no-goals must not use the empty-state message")
	}
}

func TestPreparationMessage(t *testing.T) {
	if got, want := PreparationMessage(2027, 0), "Your 2027 is a blank page. Plan the first goal before the year begins."; got != want {
		t.Fatalf("no topics = %q, want %q", got, want)
	}
	if got, want := PreparationMessage(2027, 3), "Every goal you plan now is a head start. 2027 is almost here."; got != want {
		t.Fatalf("with topics = %q, want %q", got, want)
	}
}

func TestSecondsUntil(t *testing.T) {
	target := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	if got := SecondsUntil(target, target.Add(-90*time.Second)); got != 90 {
		t.Fatalf("90s before = %d, want 90", got)
	}
	if got := SecondsUntil(target, target.Add(time.Hour)); got != 0 {
		t.Fatalf("after the target = %d, want 0 (floored)", got)
	}
}

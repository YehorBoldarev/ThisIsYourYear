package domain

import "testing"

func TestMotivationBandOf_Boundaries(t *testing.T) {
	cases := []struct {
		name     string
		progress Progress
		want     MotivationBand
	}{
		{"no goals", ComputeProgress(0, 0), BandNotStarted},
		{"goals but none done", ComputeProgress(4, 0), BandNotStarted},
		{"just above zero", ComputeProgress(100, 1), BandInProgress},
		{"half", ComputeProgress(2, 1), BandInProgress},
		{"just below the threshold", ComputeProgress(100, 79), BandInProgress},
		{"exactly the threshold", ComputeProgress(10, 8), BandAlmostThere},
		{"almost complete", ComputeProgress(10000, 9999), BandAlmostThere},
		{"complete", ComputeProgress(3, 3), BandAchieved},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MotivationBandOf(tc.progress); got != tc.want {
				t.Fatalf("MotivationBandOf(%+v) = %q, want %q", tc.progress, got, tc.want)
			}
		})
	}
}

func TestMotivate_EveryBandHasCopy(t *testing.T) {
	bands := []MotivationBand{BandNotStarted, BandInProgress, BandAlmostThere, BandAchieved}
	seen := map[string]bool{}
	for _, b := range bands {
		m, ok := motivations[b]
		if !ok {
			t.Fatalf("band %q has no copy", b)
		}
		if m.Band != b {
			t.Fatalf("band %q copy is keyed as %q", b, m.Band)
		}
		if m.Label == "" || m.Title == "" || m.Message == "" {
			t.Fatalf("band %q has empty copy: %+v", b, m)
		}
		if seen[m.Message] {
			t.Fatalf("band %q reuses a message: %q", b, m.Message)
		}
		seen[m.Message] = true
	}
}

func TestMotivate_MatchesBand(t *testing.T) {
	if got := Motivate(ComputeProgress(4, 4)); got != motivations[BandAchieved] {
		t.Fatalf("Motivate(complete) = %+v, want achieved copy", got)
	}
	if got := Motivate(ComputeProgress(0, 0)); got != motivations[BandNotStarted] {
		t.Fatalf("Motivate(no goals) = %+v, want not-started copy", got)
	}
}

package domain

import "testing"

func TestMotivationBandOf_Boundaries(t *testing.T) {
	cases := []struct {
		name     string
		progress Progress
		want     MotivationBand
	}{
		{"no goals", ComputeProgress(0, 0), BandNoGoals},
		{"goals but none done", ComputeProgress(4, 0), BandStarted},
		{"just above zero", ComputeProgress(100, 1), BandQuarter},
		{"exactly a quarter", ComputeProgress(4, 1), BandQuarter},
		{"just above a quarter", ComputeProgress(100, 26), BandHalf},
		{"exactly a half", ComputeProgress(2, 1), BandHalf},
		{"just above a half", ComputeProgress(100, 51), BandMost},
		{"almost complete", ComputeProgress(10000, 9999), BandMost},
		{"complete", ComputeProgress(3, 3), BandComplete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MotivationBandOf(tc.progress); got != tc.want {
				t.Fatalf("MotivationBandOf(%+v) = %d, want %d", tc.progress, got, tc.want)
			}
		})
	}
}

func TestMotivate_EveryBandHasMessage(t *testing.T) {
	bands := []MotivationBand{
		BandNoGoals, BandStarted, BandQuarter, BandHalf, BandMost, BandComplete,
	}
	seen := map[string]bool{}
	for _, b := range bands {
		msg, ok := motivationMessages[b]
		if !ok || msg == "" {
			t.Fatalf("band %d has no message", b)
		}
		if seen[msg] {
			t.Fatalf("band %d reuses a message: %q", b, msg)
		}
		seen[msg] = true
	}
}

func TestMotivate_MatchesBand(t *testing.T) {
	p := ComputeProgress(4, 4)
	if got, want := Motivate(p), motivationMessages[BandComplete]; got != want {
		t.Fatalf("Motivate(complete) = %q, want %q", got, want)
	}
	if got, want := Motivate(ComputeProgress(0, 0)), motivationMessages[BandNoGoals]; got != want {
		t.Fatalf("Motivate(no goals) = %q, want %q", got, want)
	}
}

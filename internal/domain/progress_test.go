package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeProgress(t *testing.T) {
	cases := map[string]struct {
		total, completed    int
		wantTotal, wantDone int
		wantRatio           float64
	}{
		"no goals is zero, not NaN":  {0, 0, 0, 0, 0.0},
		"none done":                  {4, 0, 4, 0, 0.0},
		"half done":                  {4, 2, 4, 2, 0.5},
		"all done":                   {3, 3, 3, 3, 1.0},
		"completed clamped to total": {2, 5, 2, 2, 1.0},
		"negatives clamped":          {-1, -3, 0, 0, 0.0},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got := ComputeProgress(c.total, c.completed)
			require.Equal(t, c.wantTotal, got.Total)
			require.Equal(t, c.wantDone, got.Completed)
			require.InDelta(t, c.wantRatio, got.Ratio, 1e-9)
		})
	}
}

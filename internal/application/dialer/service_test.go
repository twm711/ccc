package dialer

import "testing"

func TestCalcAbandonRate(t *testing.T) {
	cases := []struct {
		name              string
		abandoned, total  int
		want              float64
	}{
		{"zero total returns zero (no division by zero panic)", 0, 0, 0},
		{"all abandoned", 5, 5, 100},
		{"none abandoned", 0, 100, 0},
		{"half abandoned", 50, 100, 50},
		{"one in three", 1, 3, 33.33333333333333},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := calcAbandonRate(c.abandoned, c.total)
			if got != c.want {
				t.Errorf("calcAbandonRate(%d, %d) = %v, want %v", c.abandoned, c.total, got, c.want)
			}
		})
	}
}

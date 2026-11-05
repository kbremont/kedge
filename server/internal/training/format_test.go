package training

import "testing"

// TestFormatWeight keeps a weight that can be loaded, such as a 1.25 kg plate,
// whole, and everything else to one decimal.
func TestFormatWeight(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{100, "100"},
		{62.5, "62.5"},
		{1.25, "1.25"},
		{61.25, "61.25"},
		{0.75, "0.75"},
		{262.333333, "262.3"},
		{224.4, "224.4"},
	}
	for _, c := range cases {
		if got := FormatWeight(c.in); got != c.want {
			t.Errorf("FormatWeight(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

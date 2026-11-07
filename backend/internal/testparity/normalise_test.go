package testparity

import "testing"

// A measured duration differs from run to run on a real database (0 ms or 1 ms
// for the same request), so it must not take part in the comparison.
func TestNormalise_ReplacesMeasuredLatency(t *testing.T) {
	for _, ms := range []float64{0, 1, 7} {
		got := normalise(map[string]any{"db": "ok", "db_latency_ms": ms}).(map[string]any)
		if got["db_latency_ms"] != "<ms>" {
			t.Errorf("db_latency_ms %v normalised to %v, want <ms>", ms, got["db_latency_ms"])
		}
		if got["db"] != "ok" {
			t.Errorf("other fields must be untouched, got db=%v", got["db"])
		}
	}
}

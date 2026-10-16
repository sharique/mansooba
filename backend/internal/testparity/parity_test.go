package testparity

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

var update = flag.Bool("update", false, "rewrite golden/baseline.json from this run")

const goldenPath = "golden/baseline.json"

// TestParity runs the script against PARITY_BASE_URL (an already running
// server, used to record the baseline from the unmodified build) or, when it
// is unset, against an in-process server (see inprocess_test.go) and compares
// the normalised results with the golden file.
func TestParity(t *testing.T) {
	base := os.Getenv("PARITY_BASE_URL")
	if base == "" {
		base = inProcessBase(t)
	}
	got, err := Run(base, Script())
	if err != nil {
		t.Fatal(err)
	}

	if *update {
		b, err := json.MarshalIndent(got, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, append(b, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d steps to %s", len(got), goldenPath)
		return
	}

	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("no golden file (record one with -update against the unmodified build): %v", err)
	}
	var want []Result
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	// Round-trip the live results through JSON so both sides use the same types.
	gb, _ := json.Marshal(got)
	var gotNorm []Result
	_ = json.Unmarshal(gb, &gotNorm)

	if len(gotNorm) != len(want) {
		t.Fatalf("step count differs: got %d, golden %d", len(gotNorm), len(want))
	}
	failed := 0
	for i := range want {
		if want[i].Name != gotNorm[i].Name {
			t.Fatalf("step %d: name %q != golden %q", i, gotNorm[i].Name, want[i].Name)
		}
		if !reflect.DeepEqual(want[i], gotNorm[i]) {
			failed++
			wb, _ := json.MarshalIndent(want[i], "", "  ")
			gb, _ := json.MarshalIndent(gotNorm[i], "", "  ")
			t.Errorf("step %q differs\n--- golden\n%s\n--- got\n%s", want[i].Name, wb, gb)
		}
	}
	if failed == 0 {
		fmt.Printf("parity: %d steps identical\n", len(want))
	}
}

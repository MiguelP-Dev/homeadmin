package assets

import "testing"

// Version must be a pure function of the embedded asset bytes: recomputing
// yields the same hash, so URLs stay stable within a build.
func TestComputeVersion_Deterministic(t *testing.T) {
	first := computeVersion()
	second := computeVersion()
	if first == "" {
		t.Fatal("computeVersion returned empty string")
	}
	if first != second {
		t.Errorf("computeVersion not deterministic: %q vs %q", first, second)
	}
	if len(first) != 12 {
		t.Errorf("computeVersion length = %d, want 12", len(first))
	}
}

func TestFS_RootsAtStaticContents(t *testing.T) {
	f, err := FS().Open("css/output.css")
	if err != nil {
		t.Fatalf("FS().Open(css/output.css): %v", err)
	}
	defer f.Close()
}

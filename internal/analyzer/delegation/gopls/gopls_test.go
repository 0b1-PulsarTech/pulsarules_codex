package gopls

import (
	"os/exec"
	"testing"
)

func TestAvailable(t *testing.T) {
	t.Parallel()

	r := NewRunner()

	_, lookErr := exec.LookPath(binary)
	want := lookErr == nil

	if got := r.Available(); got != want {
		t.Fatalf(
			"Available() = %v, want %v (exec.LookPath(%q) err = %v)",
			got,
			want,
			binary,
			lookErr,
		)
	}
}

func TestRun_NoGopls(t *testing.T) {
	t.Parallel()

	r := &Runner{}
	findings := r.Run()
	if findings == nil {
		return // OK - no gopls to report
	}
	for _, f := range findings {
		if f.AnalyzerID != "gopls" {
			t.Errorf("unexpected analyzer ID %q", f.AnalyzerID)
		}
	}
}

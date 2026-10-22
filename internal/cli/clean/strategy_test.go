package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"
)

func TestStrategies_InspectNeverWritesAndRewriteDoes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		how       sweepStrategy
		wantVerb  string
		wantWrite bool
	}{
		{"inspecting", inspecting(), "found", false},
		{"rewriting", rewriting(), "left", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "note.md")
			if err := os.WriteFile(path, []byte("a\u200Bb\n"), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
			if _, err := testCase.how.apply(cleaner.New(filepath.Dir(path)), path); err != nil {
				t.Fatalf("apply: %v", err)
			}
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if rewritten := !strings.ContainsRune(
				string(body),
				'\u200B',
			); rewritten != testCase.wantWrite {
				t.Errorf("rewritten = %v, want %v", rewritten, testCase.wantWrite)
			}
			if testCase.how.verb != testCase.wantVerb {
				t.Errorf("verb = %q, want %q", testCase.how.verb, testCase.wantVerb)
			}
		})
	}
}

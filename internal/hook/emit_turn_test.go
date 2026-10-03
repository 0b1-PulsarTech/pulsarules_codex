package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestDispatchStop_CapsTheUncommittedListing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		files      int
		wantListed int
		wantMore   string
	}{
		{"one file", 1, 1, ""},
		{"exactly the cap", 50, 50, ""},
		{"one over the cap", 51, 50, "... and 1 more"},
		{"far over the cap", 120, 50, "... and 70 more"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			gitInit(t, dir)
			seedFiles(t, dir, testCase.files)
			disp, out := dispatchCapture(Deps{
				ProjectDir: dir,
				Governance: stubGovernance("\nblock\n", 1),
			})
			payload := newSessionPayload(t)

			_ = disp.Dispatch("stop", payload)
			listing := stopListing(t, out.String())
			checkListing(t, listing, testCase.wantListed, testCase.wantMore)

			out.Reset()
			_ = disp.Dispatch("stop", payload)
			if strings.TrimSpace(out.String()) != "" {
				t.Errorf("identical change set should dedup to empty, got %q", out.String())
			}
		})
	}
}

func seedFiles(t *testing.T, dir string, count int) {
	t.Helper()
	for fileNo := range count {
		name := filepath.Join(dir, fmt.Sprintf("f%03d.txt", fileNo))
		if err := os.WriteFile(name, []byte("x\n"), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
}

func stopListing(t *testing.T, raw string) string {
	t.Helper()
	_, listing, found := strings.Cut(extractContext(t, raw), "Uncommitted changes:\n")
	if !found {
		t.Fatalf("no listing in %q", raw)
	}
	return listing
}

func checkListing(t *testing.T, listing string, wantListed int, wantMore string) {
	t.Helper()
	lines := strings.Split(listing, "\n")
	if wantMore != "" {
		if last := lines[len(lines)-1]; last != wantMore {
			t.Errorf("last line = %q, want %q", last, wantMore)
		}
		lines = lines[:len(lines)-1]
	}
	if len(lines) != wantListed {
		t.Errorf("listed %d entries, want %d", len(lines), wantListed)
	}
	if wantMore == "" && strings.Contains(listing, "more") {
		t.Errorf("unexpected truncation line in %q", listing)
	}
}

func TestCappedStatus_GitOrder(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		untracked int
		wantLast  string
	}{
		{"over the cap", 60, "... and 11 more"},
		{"just under the cap", 49, "?? f048.txt"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			changes := []vcs.Change{{Path: "zz.go", Staging: ' ', Worktree: 'M'}}
			for fileNo := range testCase.untracked {
				name := fmt.Sprintf("f%03d.txt", fileNo)
				changes = append(changes, vcs.Change{Path: name, Staging: '?', Worktree: '?'})
			}

			lines := strings.Split(cappedStatus(vcs.Status{Changes: changes}), "\n")

			if lines[0] != " M zz.go" {
				t.Errorf("first line = %q, want the tracked change", lines[0])
			}
			if last := lines[len(lines)-1]; last != testCase.wantLast {
				t.Errorf("last line = %q, want %q", last, testCase.wantLast)
			}
		})
	}
}

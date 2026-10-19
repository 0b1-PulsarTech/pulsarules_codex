package hook

import (
	"bytes"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge/templates"
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

func TestDispatchStop(t *testing.T) {
	t.Parallel()

	t.Run("clean tree stays silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitInit(t, dir)
		disp, out := dispatchCapture(Deps{ProjectDir: dir, Governance: stubGovernance("", 0)})
		_ = disp.Dispatch("stop", newSessionPayload(t))
		if strings.TrimSpace(out.String()) != "" {
			t.Errorf("expected no output on clean tree, got %q", out.String())
		}
	})

	t.Run("dirty tree with no findings stays silent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		gitInit(t, dir)
		if err := os.WriteFile(
			filepath.Join(dir, "clean.go"), []byte("package x\n"), 0o600,
		); err != nil {
			t.Fatalf("write: %v", err)
		}
		disp, out := dispatchCapture(Deps{ProjectDir: dir, Governance: stubGovernance("", 0)})
		_ = disp.Dispatch("stop", newSessionPayload(t))
		if strings.TrimSpace(out.String()) != "" {
			t.Errorf(
				"expected no output for a dirty tree with no governance findings, got %q",
				out.String(),
			)
		}
	})

	t.Run(
		"a finding emits then dedups",
		func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			gitInit(t, dir)
			emDash := "\u2014"
			content := "package x\n\n// bad note " + emDash + " trips typographic-markers\n"
			if err := os.WriteFile(
				filepath.Join(dir, "violation.go"),
				[]byte(content),
				0o600,
			); err != nil {
				t.Fatalf("write: %v", err)
			}
			msg := "\nGovernance checks:\n  [error] typographic-markers: (violation.go:3)\n"
			disp, out := dispatchCapture(Deps{ProjectDir: dir, Governance: stubGovernance(msg, 1)})
			payload := newSessionPayload(t)

			_ = disp.Dispatch("stop", payload)
			ctx := extractContext(t, out.String())
			for _, want := range []string{
				"Governance checks:", "typographic-markers", "go vet", "violation.go",
			} {
				if !strings.Contains(ctx, want) {
					t.Errorf("stop output missing %q:\n%s", want, ctx)
				}
			}
			withoutHeading := strings.ReplaceAll(strings.ToLower(ctx), "uncommitted", "")
			if strings.Contains(withoutHeading, "commit") {
				t.Errorf("stop output must not instruct a commit:\n%s", ctx)
			}

			out.Reset()
			_ = disp.Dispatch("stop", payload)
			if strings.TrimSpace(out.String()) != "" {
				t.Errorf("identical change set should dedup to empty, got %q", out.String())
			}
		},
	)
}

func TestDispatchStop_CorruptIndex(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, ".git", "index"), []byte("not an index"), 0o600,
	); err != nil {
		t.Fatalf("corrupt index: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, out := dispatchCapture(Deps{ProjectDir: dir, Logger: logger})

	if err := disp.Dispatch("stop", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected no hook output, got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "read worktree status") {
		t.Errorf("log missing worktree status failure:\n%s", logBuf.String())
	}
}

func TestDispatchStop_MissingStopTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, "dirty.go"),
		[]byte("package x\n"),
		0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	disp, out := dispatchCapture(Deps{
		ProjectDir: dir,
		Logger:     logger,
		Templates:  templates.New(fstest.MapFS{}),
		Governance: stubGovernance("\nGovernance checks:\n  - stub finding\n", 1),
	})

	if err := disp.Dispatch("stop", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected no hook output, got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "read hooks/stop.txt") {
		t.Errorf("log missing stop.txt read failure:\n%s", logBuf.String())
	}
}

func TestStopChecklist(t *testing.T) {
	t.Parallel()

	const goBullets = "  - gofmt -l clean; go vet ./... and go test -race ./... green\n" +
		"  - no doc comments; why-only comments of at most three lines"
	testCases := []struct {
		name string
		exts []string
		fsys fs.FS
		want string
	}{
		{name: "empty status"},
		{
			name: "go and sql in fixed order", exts: []string{".sql", ".go"},
			want: goBullets + "\n  - migrations reversible and reviewed; no raw SQL in handlers",
		},
		{
			name: "yaml then md", exts: []string{".md", ".yaml"},
			want: "  - validate passes: go run ./cmd/pulsarules_cli validate\n" +
				"  - every Markdown file ends with a trailing newline",
		},
		{name: "unknown extension", exts: []string{".rs"}},
		{name: "missing template", exts: []string{".go"}, fsys: fstest.MapFS{}, want: "!"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			fsys := testCase.fsys
			if fsys == nil {
				fsys = os.DirFS("../../knowledge/templates")
			}
			disp := NewDispatcher(Deps{Templates: templates.New(fsys)})
			var status vcs.Status
			for _, ext := range testCase.exts {
				status.Changes = append(status.Changes, vcs.Change{Extension: ext})
			}
			got, err := disp.stopChecklist(status)
			if (err != nil) != (testCase.want == "!") || (err == nil && got != testCase.want) {
				t.Errorf("stopChecklist() = %q, %v; want %q", got, err, testCase.want)
			}
		})
	}
}

func TestDispatchStop_MissingChecklistTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInit(t, dir)
	if err := os.WriteFile(
		filepath.Join(dir, "dirty.go"),
		[]byte("package x\n"),
		0o600,
	); err != nil {
		t.Fatalf("write: %v", err)
	}
	assets := hookTemplates()
	delete(assets, "hooks/stop-checklist.txt.tmpl")
	var logBuf bytes.Buffer
	disp, out := dispatchCapture(Deps{
		ProjectDir: dir,
		Logger:     slog.New(slog.NewTextHandler(&logBuf, nil)),
		Templates:  templates.New(assets),
		Governance: stubGovernance("\nGovernance checks:\n  - stub finding\n", 1),
	})

	if err := disp.Dispatch("stop", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	ctx := extractContext(t, out.String())
	for _, want := range []string{"Governance checks:", "stub finding", "dirty.go"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("stop output missing %q:\n%s", want, ctx)
		}
	}
	if strings.Contains(ctx, "  - gofmt") {
		t.Errorf("checklist bullets emitted without a template:\n%s", ctx)
	}
	if !strings.Contains(logBuf.String(), "stop checklist unavailable") {
		t.Errorf("log missing checklist failure:\n%s", logBuf.String())
	}
}

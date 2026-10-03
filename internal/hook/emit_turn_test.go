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
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
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

func TestUserPromptNudge_ShortAndKeepsPhrases(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	body, err := fs.ReadFile(templates, "hooks/user-prompt.txt")
	if err != nil {
		t.Fatalf("read user-prompt.txt: %v", err)
	}
	text := strings.TrimSpace(string(body))

	const nudgeByteCeiling = 300
	if got := len(text); got > nudgeByteCeiling {
		t.Errorf(
			"user-prompt.txt is %d bytes, want <= %d",
			got,
			nudgeByteCeiling,
		)
	}
	if strings.ContainsRune(text, '\u2014') {
		t.Error("user-prompt.txt contains an em dash (U+2014)")
	}

	testCases := []struct {
		name string
		want string
	}{
		{"routing question kept", "Have you invoked the project-router skill"},
		{"go/sql/config trigger kept", "Go/SQL/config"},
		{"skip consequence kept", "Skipping the router skips the rules"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(text, testCase.want) {
				t.Errorf("user-prompt.txt missing %q:\n%s", testCase.want, text)
			}
		})
	}

	notWantedCases := []struct {
		name   string
		phrase string
	}{
		{"contract.txt's baseline skill list", "code-minimalism"},
		{"contract.txt/pre-edit.txt already state the test shape", "testCase"},
		{"pre-edit.txt's minimalism ladder", "ladder"},
		{"pre-edit.txt's stdlib rung", "slices/maps/cmp"},
		{"pre-edit.txt's named-domain-type rule", "minor-unit int64"},
	}
	for _, notWantedCase := range notWantedCases {
		t.Run(notWantedCase.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(text, notWantedCase.phrase) {
				t.Errorf(
					"user-prompt.txt duplicates contract.txt/pre-edit.txt phrase %q",
					notWantedCase.phrase,
				)
			}
		})
	}
}

func TestDispatchUserPrompt_EveryDispatch(t *testing.T) {
	t.Parallel()

	payload := newSessionPayload(t)
	disp, out := dispatchCapture(Deps{})

	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Errorf("second call did not fire: %q", out.String())
	}
}

func TestDispatchPreSearch(t *testing.T) {
	t.Parallel()

	const goGlob = `"pattern":"foo","glob":"*.go"`
	ready := preSearchFixture{hasGoMod: true, hasGoplsSkill: true}
	testCases := []struct {
		name      string
		toolName  string
		toolInput string
		fixture   preSearchFixture
		wantFire  bool
	}{
		{
			name: "grep on go glob fires", toolName: "Grep", toolInput: goGlob,
			fixture: ready, wantFire: true,
		},
		{
			name: "bash grep command fires", toolName: "Bash",
			toolInput: `"command":"grep -rn foo ."`, fixture: ready, wantFire: true,
		},
		{
			name: "grep on markdown stays silent", toolName: "Grep",
			toolInput: `"pattern":"foo","glob":"*.md"`, fixture: ready,
		},
		{
			name: "no go.mod stays silent", toolName: "Grep", toolInput: goGlob,
			fixture: preSearchFixture{hasGoplsSkill: true},
		},
		{
			name: "gopls-navigation not installed stays silent", toolName: "Grep",
			toolInput: goGlob, fixture: preSearchFixture{hasGoMod: true},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir, skillsDir := preSearchProjectDir(t, testCase.fixture)
			disp, out := dispatchCapture(Deps{ProjectDir: dir, SkillsDir: skillsDir})
			payload := fmt.Sprintf(
				`{"tool_name":%q,"tool_input":{%s},"session_id":%q}`,
				testCase.toolName, testCase.toolInput, uniqueSessionID(t),
			)
			_ = disp.Dispatch("pre-search", []byte(payload))
			if !testCase.wantFire {
				if strings.TrimSpace(out.String()) != "" {
					t.Errorf("expected no output, got %q", out.String())
				}
				return
			}
			ctx := extractContext(t, out.String())
			if !strings.Contains(ctx, "pre-search gopls reminder") {
				t.Errorf("missing reminder text:\n%s", ctx)
			}
		})
	}
}

type preSearchFixture struct {
	hasGoMod      bool
	hasGoplsSkill bool
}

func preSearchProjectDir(t *testing.T, fixture preSearchFixture) (dir, skillsDir string) {
	t.Helper()
	dir = t.TempDir()
	if fixture.hasGoMod {
		if err := os.WriteFile(
			filepath.Join(dir, "go.mod"),
			[]byte("module x\n"),
			0o600,
		); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}
	}
	skillsDir = filepath.Join(dir, ".claude", "skills")
	if fixture.hasGoplsSkill {
		installSkillFixture(t, skillsDir, "gopls-navigation")
	}
	return dir, skillsDir
}

func TestDispatchPreSearchOncePerSession(t *testing.T) {
	t.Parallel()

	dir, skillsDir := preSearchProjectDir(t, preSearchFixture{hasGoMod: true, hasGoplsSkill: true})
	id := uniqueSessionID(t)
	payload := fmt.Appendf(
		nil,
		`{"tool_name":"Grep","tool_input":{"pattern":"foo","glob":"*.go"},"session_id":%q}`,
		id,
	)
	disp, out := dispatchCapture(Deps{ProjectDir: dir, SkillsDir: skillsDir})

	_ = disp.Dispatch("pre-search", payload)
	if !strings.Contains(extractContext(t, out.String()), "pre-search gopls reminder") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("pre-search", payload)
	if strings.Contains(out.String(), "pre-search gopls reminder") {
		t.Errorf("second call fired again: %q", out.String())
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

	t.Run("dirty tree with a finding emits the block then dedups", func(t *testing.T) {
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
		disp, out := dispatchCapture(Deps{ProjectDir: dir})
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
	})
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
		Templates:  fstest.MapFS{},
		Governance: stubGovernance("\nGovernance checks:\n  - stub finding\n", 1),
	})

	if err := disp.Dispatch("stop", newSessionPayload(t)); err != nil {
		t.Fatalf("Dispatch must never block, got error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("expected no hook output, got %q", out.String())
	}
	if !strings.Contains(logBuf.String(), "read stop.txt") {
		t.Errorf("log missing stop.txt read failure:\n%s", logBuf.String())
	}
}

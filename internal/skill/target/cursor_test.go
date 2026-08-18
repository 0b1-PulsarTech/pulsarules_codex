package target

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
)

func TestCursorTargetName(t *testing.T) {
	t.Parallel()
	if got := (cursorTarget{}).Name(); got != "cursor" {
		t.Fatalf("Name() = %q, want cursor", got)
	}
}

// TestCursorTargetPresent covers detecting a .cursor/rules dir and an
// untouched project - the signal uninstall's target auto-detection relies on.
func TestCursorTargetPresent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		makeRules   bool
		wantPresent bool
	}{
		{name: ".cursor/rules present", makeRules: true, wantPresent: true},
		{name: "untouched project", wantPresent: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if testCase.makeRules {
				dir := filepath.Join(base, ".cursor", "rules")
				if err := os.MkdirAll(dir, 0o750); err != nil {
					t.Fatalf("mkdir %q: %v", dir, err)
				}
			}
			if got := (cursorTarget{}).Present(base); got != testCase.wantPresent {
				t.Errorf("Present(%q) = %v, want %v", base, got, testCase.wantPresent)
			}
		})
	}
}

// TestCursorTargetInstall asserts Install writes one .mdc rule per selected
// skill plus the always-on pointer rule, and that the pointer carries
// alwaysApply: true while the skill rule does not.
func TestCursorTargetInstall(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})

	report, err := cursorTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	rulePath := filepath.Join(base, ".cursor", "rules", "go-style.mdc")
	ruleBody, readErr := os.ReadFile(rulePath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("missing %q: %v", rulePath, readErr)
	}
	if !strings.Contains(string(ruleBody), "alwaysApply: false") {
		t.Errorf("go-style.mdc should carry alwaysApply: false, got:\n%s", ruleBody)
	}
	if !strings.Contains(string(ruleBody), marker.Installed) {
		t.Errorf("go-style.mdc missing the ownership marker, got:\n%s", ruleBody)
	}

	pointerPath := filepath.Join(base, ".cursor", "rules", "pulsarules-contract.mdc")
	pointerBody, readErr := os.ReadFile(pointerPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("missing %q: %v", pointerPath, readErr)
	}
	if !strings.Contains(string(pointerBody), "alwaysApply: true") {
		t.Errorf("pointer rule should carry alwaysApply: true, got:\n%s", pointerBody)
	}

	if len(report.Notes) < 2 {
		t.Errorf("expected at least 2 notes (pointer + go-style), got %v", report.Notes)
	}
}

// TestCursorTargetInstall_UnknownSkillWarns asserts an unknown skill id is
// skipped with a warning rather than failing the whole install.
func TestCursorTargetInstall_UnknownSkillWarns(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style", "does-not-exist"})

	report, err := cursorTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !slices.ContainsFunc(report.Warnings, func(w string) bool {
		return strings.Contains(w, "does-not-exist")
	}) {
		t.Errorf("Warnings missing the unknown skill id: %v", report.Warnings)
	}
}

// TestCursorTargetInstall_ForeignRuleSurvives asserts a pre-existing rule
// file this tool never wrote (no ownership marker) is kept, not overwritten,
// with a warning naming it - the same discipline agentswire.WriteAgents
// applies to a foreign AGENTS.md.
func TestCursorTargetInstall_ForeignRuleSurvives(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	rulePath := filepath.Join(base, ".cursor", "rules", "go-style.mdc")
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	foreign := "# my own go-style rule\nDo not touch.\n"
	if err := os.WriteFile(rulePath, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign rule: %v", err)
	}

	ctx := newTestContext(t, base, []string{"go-style"})
	report, err := cursorTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	got, readErr := os.ReadFile(rulePath) //nolint:gosec // test fixture.
	if readErr != nil || string(got) != foreign {
		t.Fatalf("foreign rule changed: err=%v got=%q want=%q", readErr, got, foreign)
	}
	if !slices.ContainsFunc(report.Warnings, func(w string) bool {
		return strings.Contains(w, "kept existing user-authored")
	}) {
		t.Errorf("Warnings missing the kept-foreign-file note: %v", report.Warnings)
	}
}

// TestCursorTargetUninstall covers the full round trip (Install then
// Uninstall) and idempotency (a second Uninstall is not an error).
func TestCursorTargetUninstall(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (cursorTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	rulesDir := filepath.Join(base, ".cursor", "rules")
	if _, statErr := os.Stat(rulesDir); statErr != nil {
		t.Fatalf("Install did not write %q: %v", rulesDir, statErr)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	report, err := (cursorTarget{}).Uninstall(uctx)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(rulesDir); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed once emptied, stat err = %v", rulesDir, statErr)
	}
	cursorDir := filepath.Join(base, ".cursor")
	if _, statErr := os.Stat(cursorDir); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be reaped once empty, stat err = %v", cursorDir, statErr)
	}
	if len(report.Notes) < 2 {
		t.Errorf("expected at least 2 removal notes, got %v", report.Notes)
	}

	if _, err = (cursorTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("second Uninstall: %v", err)
	}
}

// TestCursorTargetUninstall_KeepSkills asserts --keep-skills leaves every
// rendered rule in place.
func TestCursorTargetUninstall_KeepSkills(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (cursorTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	rulePath := filepath.Join(base, ".cursor", "rules", "go-style.mdc")

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry(), KeepSkills: true}
	if _, err := (cursorTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(rulePath); statErr != nil {
		t.Errorf("--keep-skills should have kept %q, stat err = %v", rulePath, statErr)
	}
}

// TestCursorTargetUninstall_ForeignRuleSurvives asserts a rule file this
// tool never wrote survives Uninstall untouched.
func TestCursorTargetUninstall_ForeignRuleSurvives(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	rulePath := filepath.Join(base, ".cursor", "rules", "foreign.mdc")
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	foreign := "# my own rule\n"
	if err := os.WriteFile(rulePath, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign rule: %v", err)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err := (cursorTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	got, err := os.ReadFile(rulePath) //nolint:gosec // test fixture.
	if err != nil || string(got) != foreign {
		t.Fatalf("foreign rule changed: err=%v got=%q want=%q", err, got, foreign)
	}
}

// TestCursorTargetUninstall_UserFileInCursorDirSurvives asserts a .cursor
// directory holding something of the user's outside of rules/ is never
// reaped - fsx.RemoveEmptyDir only ever deletes an actually-empty directory.
func TestCursorTargetUninstall_UserFileInCursorDirSurvives(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (cursorTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	userFile := filepath.Join(base, ".cursor", "mcp.json")
	if err := os.WriteFile(userFile, []byte("{}"), 0o600); err != nil {
		t.Fatalf("seed user file: %v", err)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err := (cursorTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	if _, statErr := os.Stat(userFile); statErr != nil {
		t.Errorf("expected %q to survive, stat err = %v", userFile, statErr)
	}
	cursorDir := filepath.Join(base, ".cursor")
	if _, statErr := os.Stat(cursorDir); statErr != nil {
		t.Errorf("expected %q to survive (holds a user file), stat err = %v", cursorDir, statErr)
	}
}

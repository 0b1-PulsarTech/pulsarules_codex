package target

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/hook/install"
)

func TestAgentsTargetName(t *testing.T) {
	t.Parallel()
	if got := (agentsTarget{}).Name(); got != "agents" {
		t.Fatalf("Name() = %q, want agents", got)
	}
}

// TestAgentsTargetPresent covers detecting a root AGENTS.md and an untouched
// project - the signal uninstall's target auto-detection relies on.
func TestAgentsTargetPresent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		makeAgents bool
		want       bool
	}{
		{name: "AGENTS.md present", makeAgents: true, want: true},
		{name: "untouched project", want: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			if testCase.makeAgents {
				path := filepath.Join(base, "AGENTS.md")
				if err := os.WriteFile(path, []byte("# AGENTS.md\n"), 0o600); err != nil {
					t.Fatalf("seed AGENTS.md: %v", err)
				}
			}
			if got := (agentsTarget{}).Present(base); got != testCase.want {
				t.Errorf("Present(%q) = %v, want %v", base, got, testCase.want)
			}
		})
	}
}

// TestAgentsTargetPresent_DefersToOpencode is the regression test for the bug where agentsTarget
// and opencodeTarget both claimed a root AGENTS.md: once an opencode dir is present,
// agentsTarget.Present must return false even though the shared AGENTS.md still exists, since
// opencodeTarget's own Uninstall already reverses it - claiming it here too would attempt the
// same removal twice.
func TestAgentsTargetPresent_DefersToOpencode(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	agentsPath := filepath.Join(base, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# AGENTS.md\n"), 0o600); err != nil {
		t.Fatalf("seed AGENTS.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".opencode"), 0o750); err != nil {
		t.Fatalf("mkdir .opencode: %v", err)
	}

	if got := (agentsTarget{}).Present(base); got {
		t.Error("Present() = true, want false (opencode already owns this AGENTS.md)")
	}
}

// TestAgentsTargetInstall asserts Install writes only the root AGENTS.md - no
// skills dir, no config file - reflecting the thin layout's one job.
func TestAgentsTargetInstall(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})

	report, err := agentsTarget{}.Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	agentsDoc := filepath.Join(base, "AGENTS.md")
	if _, statErr := os.Stat(agentsDoc); statErr != nil {
		t.Errorf("missing %q: %v", agentsDoc, statErr)
	}
	if _, statErr := os.Stat(filepath.Join(base, ".claude")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("agents target should not write .claude, stat err = %v", statErr)
	}
	if len(report.Notes) == 0 {
		t.Error("expected a note recording the write")
	}
}

// TestAgentsTargetUninstall covers the full round trip (Install then
// Uninstall) and idempotency (a second Uninstall is not an error).
func TestAgentsTargetUninstall(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (agentsTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	agentsDoc := filepath.Join(base, "AGENTS.md")
	if _, statErr := os.Stat(agentsDoc); statErr != nil {
		t.Fatalf("Install did not write %q: %v", agentsDoc, statErr)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err := (agentsTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(agentsDoc); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("expected %q to be removed, stat err = %v", agentsDoc, statErr)
	}

	if _, err := (agentsTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("second Uninstall: %v", err)
	}
}

// TestAgentsTargetUninstall_KeepSkillsKeepsAgents asserts --keep-skills
// leaves AGENTS.md in place.
func TestAgentsTargetUninstall_KeepSkillsKeepsAgents(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	ctx := newTestContext(t, base, []string{"go-style"})
	if _, err := (agentsTarget{}).Install(ctx); err != nil {
		t.Fatalf("Install: %v", err)
	}
	agentsDoc := filepath.Join(base, "AGENTS.md")

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry(), KeepSkills: true}
	if _, err := (agentsTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if _, statErr := os.Stat(agentsDoc); statErr != nil {
		t.Errorf("--keep-skills should have kept %q, stat err = %v", agentsDoc, statErr)
	}
}

// TestAgentsTargetUninstall_UserAuthoredAgentsSurvives is the highest-risk
// regression test for this change: a root AGENTS.md is a name a user very
// plausibly owns already, so Uninstall must never delete one this tool did
// not write, even when it runs right where Install would have written one.
func TestAgentsTargetUninstall_UserAuthoredAgentsSurvives(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	agentsDoc := filepath.Join(base, "AGENTS.md")
	foreign := "# My own AGENTS.md\nDo not touch.\n"
	if err := os.WriteFile(agentsDoc, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed user-authored AGENTS.md: %v", err)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err := (agentsTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}

	got, err := os.ReadFile(agentsDoc) //nolint:gosec // test fixture.
	if err != nil {
		t.Fatalf("expected user-authored AGENTS.md to survive, read err = %v", err)
	}
	if string(got) != foreign {
		t.Errorf("AGENTS.md content changed, got %q want %q", got, foreign)
	}
}

// TestAgentsTargetInstall_UserAuthoredAgentsSurvivesFullRoundTrip is the
// install-side half of the same regression: Install must not clobber a
// pre-existing user-authored root AGENTS.md either, and the file must still
// read back unchanged after Install then Uninstall both run against it.
func TestAgentsTargetInstall_UserAuthoredAgentsSurvivesFullRoundTrip(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	agentsDoc := filepath.Join(base, "AGENTS.md")
	foreign := "# My own AGENTS.md\nDo not touch, it predates pulsarules_cli.\n"
	if err := os.WriteFile(agentsDoc, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed user-authored AGENTS.md: %v", err)
	}

	ctx := newTestContext(t, base, []string{"go-style"})
	report, err := (agentsTarget{}).Install(ctx)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(report.Warnings) == 0 {
		t.Error("expected a warning that the foreign AGENTS.md was kept, not overwritten")
	}
	got, readErr := os.ReadFile(agentsDoc) //nolint:gosec // test fixture.
	if readErr != nil || string(got) != foreign {
		t.Fatalf(
			"Install changed a foreign AGENTS.md: err=%v got=%q want=%q",
			readErr,
			got,
			foreign,
		)
	}

	uctx := UninstallContext{Base: base, HookUninstallers: install.NewRegistry()}
	if _, err = (agentsTarget{}).Uninstall(uctx); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	got, readErr = os.ReadFile(agentsDoc) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("expected user-authored AGENTS.md to survive the full round trip: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("AGENTS.md content changed, got %q want %q", got, foreign)
	}
}

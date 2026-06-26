package output

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestInstallWorkflows asserts known workflows are written to
// <dest>/<id>/WORKFLOW.md with a .gitignore, and unknown ids are skipped.
func TestInstallWorkflows(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()

	installed, skipped, err := InstallWorkflows(idx, rnd, dest, []string{"refactoring", "ghost-wf"})
	if err != nil {
		t.Fatalf("InstallWorkflows: %v", err)
	}
	if len(installed) != 1 || installed[0] != "refactoring" {
		t.Errorf("installed = %v, want [refactoring]", installed)
	}
	if len(skipped) != 1 || skipped[0] != "ghost-wf" {
		t.Errorf("skipped = %v, want [ghost-wf]", skipped)
	}
	if _, err := os.Stat(filepath.Join(dest, "refactoring", "WORKFLOW.md")); err != nil {
		t.Errorf("WORKFLOW.md not written: %v", err)
	}
	data, readErr := os.ReadFile(filepath.Join(dest, "refactoring", ".gitignore"))
	if readErr != nil {
		t.Fatalf("read workflow .gitignore: %v", readErr)
	}
	if want := "WORKFLOW.md\n.gitignore\n"; string(data) != want {
		t.Errorf("workflow .gitignore content = %q, want %q", string(data), want)
	}
}

// TestInstall_RouterAndUnknown asserts known ids are written to <dest>/<id>/SKILL.md
// (with a sibling .gitignore) and unknown ids are skipped (not written, reported).
func TestInstall_RouterAndUnknown(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()

	installed, skipped, err := Install(idx, rnd, dest, []string{"project-router", "ghost"}, nil)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if len(installed) != 1 || installed[0] != "project-router" {
		t.Errorf("installed = %v, want [project-router]", installed)
	}
	if len(skipped) != 1 || skipped[0] != "ghost" {
		t.Errorf("skipped = %v, want [ghost]", skipped)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "project-router", "SKILL.md")); statErr != nil {
		t.Errorf("router SKILL.md not written: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "project-router", ".gitignore")); statErr != nil {
		t.Errorf("router .gitignore not written: %v", statErr)
	}
	data, err := os.ReadFile(filepath.Join(dest, "project-router", ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if want := "SKILL.md\n.gitignore\n"; string(data) != want {
		t.Errorf(".gitignore content = %q, want %q", string(data), want)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "ghost")); !os.IsNotExist(statErr) {
		t.Errorf("ghost should not be written, stat err = %v", statErr)
	}
}

// TestInstallDocs_RenderFailureLeavesPriorFiles proves the render-failure
// branch: a mid-loop render error stops the loop and propagates the error,
// and (unlike Package, which defers a cleanup) installDocs has no cleanup
// path - the doc already written for an earlier id stays on disk, and the
// failing/unreached ids never get a directory at all.
func TestInstallDocs_RenderFailureLeavesPriorFiles(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	wantErr := errors.New("boom")
	render := func(id string) (string, bool, error) {
		if id == "broken" {
			return "", true, wantErr
		}
		return "body for " + id, true, nil
	}

	installed, skipped, err := installDocs(
		dest,
		"SKILL.md",
		[]string{"good", "broken", "unreached"},
		render,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
	if len(installed) != 1 || installed[0] != "good" {
		t.Errorf("installed = %v, want [good]", installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "good", "SKILL.md")); statErr != nil {
		t.Errorf("expected good/SKILL.md to remain after a later failure, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "broken")); !os.IsNotExist(statErr) {
		t.Errorf("broken must not have written a partial dir, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "unreached")); !os.IsNotExist(statErr) {
		t.Errorf("unreached must never be processed, stat err = %v", statErr)
	}
}

// TestInstallDocs_WriteDocFailureLeavesPriorFiles proves the WriteDoc-failure
// branch: a destination whose directory a prior file blocks fails WriteDoc,
// installDocs wraps and returns that error, and - the same no-cleanup
// contract as the render-failure case - the earlier id's file stays.
func TestInstallDocs_WriteDocFailureLeavesPriorFiles(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	blocked := filepath.Join(dest, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	render := func(id string) (string, bool, error) {
		return "body for " + id, true, nil
	}

	installed, skipped, err := installDocs(dest, "SKILL.md", []string{"good", "blocked"}, render)
	if err == nil {
		t.Fatal("installDocs: expected an error from the blocked destination")
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none (a write error is not an unknown id)", skipped)
	}
	if len(installed) != 1 || installed[0] != "good" {
		t.Errorf("installed = %v, want [good]", installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "good", "SKILL.md")); statErr != nil {
		t.Errorf("expected good/SKILL.md to remain after a later failure, stat err = %v", statErr)
	}
}

// TestInstall_WriteDocFailure exercises the same WriteDoc-failure contract
// through the public Install entrypoint (not just the shared installDocs
// helper): a blocked destination for the second id fails the call, and the
// first id's already-written SKILL.md is left on disk rather than cleaned up.
func TestInstall_WriteDocFailure(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()
	blocked := filepath.Join(dest, "code-minimalism")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	installed, skipped, err := Install(
		idx,
		rnd,
		dest,
		[]string{"project-router", "code-minimalism"},
		nil,
	)
	if err == nil {
		t.Fatal("Install: expected an error from the blocked destination")
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
	if len(installed) != 1 || installed[0] != "project-router" {
		t.Errorf("installed = %v, want [project-router]", installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "project-router", "SKILL.md")); statErr != nil {
		t.Errorf(
			"expected project-router/SKILL.md to remain after a later failure, stat err = %v",
			statErr,
		)
	}
}

// TestInstallWorkflows_WriteDocFailure is TestInstall_WriteDocFailure's
// counterpart for InstallWorkflows, proving the same behavior through that
// entrypoint.
func TestInstallWorkflows_WriteDocFailure(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()
	blocked := filepath.Join(dest, "refactoring")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	installed, skipped, err := InstallWorkflows(
		idx,
		rnd,
		dest,
		[]string{"code-review", "refactoring"},
	)
	if err == nil {
		t.Fatal("InstallWorkflows: expected an error from the blocked destination")
	}
	if len(skipped) != 0 {
		t.Errorf("skipped = %v, want none", skipped)
	}
	if len(installed) != 1 || installed[0] != "code-review" {
		t.Errorf("installed = %v, want [code-review]", installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "code-review", "WORKFLOW.md")); statErr != nil {
		t.Errorf(
			"expected code-review/WORKFLOW.md to remain after a later failure, stat err = %v",
			statErr,
		)
	}
}

package docs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func TestInstallWorkflows(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()

	result, err := InstallWorkflows(
		idx,
		rnd,
		dest,
		[]string{"refactoring", "ghost-wf"},
	)
	if err != nil {
		t.Fatalf("InstallWorkflows: %v", err)
	}
	if len(result.BackedUp) != 0 {
		t.Errorf("backedUp = %v, want none (fresh dest)", result.BackedUp)
	}
	if len(result.Installed) != 1 || result.Installed[0] != "refactoring" {
		t.Errorf("installed = %v, want [refactoring]", result.Installed)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "ghost-wf" {
		t.Errorf("skipped = %v, want [ghost-wf]", result.Skipped)
	}
	if _, err = os.Stat(filepath.Join(dest, "refactoring", "WORKFLOW.md")); err != nil {
		t.Errorf("WORKFLOW.md not written: %v", err)
	}
	giContent, readErr := os.ReadFile(filepath.Join(dest, "refactoring", ".gitignore"))
	if readErr != nil {
		t.Fatalf("read workflow .gitignore: %v", readErr)
	}
	if want := "# " + ownedfile.Marker + "\nWORKFLOW.md\n.gitignore\n"; string(
		giContent,
	) != want {
		t.Errorf("workflow .gitignore content = %q, want %q", string(giContent), want)
	}
}

func TestInstall_RouterAndUnknown(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()

	result, err := Install(
		idx,
		rnd,
		dest,
		InstallRequest{IDs: []string{"project-router", "ghost"}},
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	assertIDs(t, "backedUp", result.BackedUp)
	assertIDs(t, "installed", result.Installed, "project-router")
	assertIDs(t, "skipped", result.Skipped, "ghost")
	assertPresent(t, filepath.Join(dest, "project-router", "SKILL.md"))
	assertContent(
		t,
		filepath.Join(dest, "project-router", ".gitignore"),
		"# "+ownedfile.Marker+"\nSKILL.md\n.gitignore\n",
	)
	assertMissing(t, filepath.Join(dest, "ghost"))
}

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

	result, err := installDocs(
		dest,
		"SKILL.md",
		[]string{"good", "broken", "unreached"},
		render,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
	if len(result.Skipped) != 0 {
		t.Errorf("skipped = %v, want none", result.Skipped)
	}
	if len(result.Installed) != 1 || result.Installed[0] != "good" {
		t.Errorf("installed = %v, want [good]", result.Installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "good", "SKILL.md")); statErr != nil {
		t.Errorf("expected good/SKILL.md to remain after a later failure, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "broken")); !errors.Is(statErr, fs.ErrNotExist) {
		t.Errorf("broken must not have written a partial dir, stat err = %v", statErr)
	}
	if _, statErr := os.Stat(
		filepath.Join(dest, "unreached"),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("unreached must never be processed, stat err = %v", statErr)
	}
}

func TestInstallDocs_WriteDocFailure(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	blocked := filepath.Join(dest, "blocked")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	render := func(id string) (string, bool, error) {
		return "body for " + id, true, nil
	}

	result, err := installDocs(dest, "SKILL.md", []string{"good", "blocked"}, render)
	if err == nil {
		t.Fatal("installDocs: expected an error from the blocked destination")
	}
	if len(result.Skipped) != 0 {
		t.Errorf("skipped = %v, want none (a write error is not an unknown id)", result.Skipped)
	}
	if len(result.Installed) != 1 || result.Installed[0] != "good" {
		t.Errorf("installed = %v, want [good]", result.Installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "good", "SKILL.md")); statErr != nil {
		t.Errorf("expected good/SKILL.md to remain after a later failure, stat err = %v", statErr)
	}
}

func TestInstall_WriteDocFailureLeavesPriorFiles(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()
	blocked := filepath.Join(dest, "code-minimalism")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	result, err := Install(
		idx,
		rnd,
		dest,
		InstallRequest{IDs: []string{"project-router", "code-minimalism"}},
	)
	if err == nil {
		t.Fatal("Install: expected an error from the blocked destination")
	}
	if len(result.Skipped) != 0 {
		t.Errorf("skipped = %v, want none", result.Skipped)
	}
	if len(result.Installed) != 1 || result.Installed[0] != "project-router" {
		t.Errorf("installed = %v, want [project-router]", result.Installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "project-router", "SKILL.md")); statErr != nil {
		t.Errorf(
			"expected project-router/SKILL.md to remain after a later failure, stat err = %v",
			statErr,
		)
	}
}

func TestInstallWorkflows_WriteDocFailure(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()
	blocked := filepath.Join(dest, "refactoring")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	result, err := InstallWorkflows(
		idx,
		rnd,
		dest,
		[]string{"code-review", "refactoring"},
	)
	if err == nil {
		t.Fatal("InstallWorkflows: expected an error from the blocked destination")
	}
	if len(result.Skipped) != 0 {
		t.Errorf("skipped = %v, want none", result.Skipped)
	}
	if len(result.Installed) != 1 || result.Installed[0] != "code-review" {
		t.Errorf("installed = %v, want [code-review]", result.Installed)
	}
	if _, statErr := os.Stat(filepath.Join(dest, "code-review", "WORKFLOW.md")); statErr != nil {
		t.Errorf(
			"expected code-review/WORKFLOW.md to remain after a later failure, stat err = %v",
			statErr,
		)
	}
}

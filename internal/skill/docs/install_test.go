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

	installed, skipped, backedUp, err := InstallWorkflows(
		idx,
		rnd,
		dest,
		[]string{"refactoring", "ghost-wf"},
	)
	if err != nil {
		t.Fatalf("InstallWorkflows: %v", err)
	}
	if len(backedUp) != 0 {
		t.Errorf("backedUp = %v, want none (fresh dest)", backedUp)
	}
	if len(installed) != 1 || installed[0] != "refactoring" {
		t.Errorf("installed = %v, want [refactoring]", installed)
	}
	if len(skipped) != 1 || skipped[0] != "ghost-wf" {
		t.Errorf("skipped = %v, want [ghost-wf]", skipped)
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

	installed, skipped, backedUp, err := Install(
		idx,
		rnd,
		dest,
		[]string{"project-router", "ghost"},
		nil,
	)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	assertIDs(t, "backedUp", backedUp)
	assertIDs(t, "installed", installed, "project-router")
	assertIDs(t, "skipped", skipped, "ghost")
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

	installed, skipped, _, err := installDocs(
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

	installed, skipped, _, err := installDocs(
		dest, "SKILL.md", []string{"good", "blocked"}, render,
	)
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

func TestInstall_WriteDocFailureLeavesPriorFiles(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()
	blocked := filepath.Join(dest, "code-minimalism")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	installed, skipped, _, err := Install(
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

func TestInstallWorkflows_WriteDocFailure(t *testing.T) {
	t.Parallel()

	idx, rnd := loadFixture(t)
	dest := t.TempDir()
	blocked := filepath.Join(dest, "refactoring")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	installed, skipped, _, err := InstallWorkflows(
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

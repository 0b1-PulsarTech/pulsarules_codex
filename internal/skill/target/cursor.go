package target

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/contract"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsx"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/cursorwire"
)

// why: the only thing this layout ever writes under .cursor is RulesDir
// itself, so reaping the root once that leaf is empty is always safe.
const cursorRootDir = ".cursor"

// pointerDescription is the always-on pointer rule's Cursor description.
const pointerDescription = "Mandatory pulsarules_codex engineering contract - read before any " +
	"Go/SQL/config change"

// cursorTarget renders each selected skill into <base>/.cursor/rules as one
// .mdc file, plus one always-on pointer rule carrying the routing contract.
// why: Cursor injects every alwaysApply:true rule into EVERY request, capped
// around ~2000 tokens; the five always_load skills alone render ~52KB, so
// only the pointer rule (under 900 bytes) is alwaysApply; the rest load on demand.
type cursorTarget struct{}

var _ Target = cursorTarget{}

func (cursorTarget) Name() string { return "cursor" }

// Present reports whether base holds a .cursor/rules dir this layout's
// Install could have written.
func (cursorTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, cursorwire.RulesDir))
	return err == nil
}

// Install renders the selected skills as Cursor .mdc rules under
// .cursor/rules, one file per skill, plus the always-on pointer rule.
func (cursorTarget) Install(ctx Context) (Report, error) {
	var report Report
	if err := writeCursorPointer(ctx, &report); err != nil {
		return report, err
	}
	if err := writeCursorRules(ctx, &report); err != nil {
		return report, err
	}
	return report, nil
}

// writeCursorPointer writes the small, always-applied pointer rule that
// carries the routing contract (see the cursorTarget doc comment for the
// size rationale behind keeping it, and only it, alwaysApply: true).
func writeCursorPointer(ctx Context, report *Report) error {
	contractText, err := contract.Session(ctx.Templates)
	if err != nil {
		return fmt.Errorf("routing contract: %w", err)
	}
	body := buildPointerBody(contractText)
	return writeCursorFile(ctx.Base, cursorwire.PointerID, body, report)
}

// buildPointerBody assembles the pointer rule's frontmatter and body. It is
// plain string assembly rather than a text/template file: the content is
// fixed (only the contract text varies, and that already comes pre-rendered
// from the contract package), so a template adds no value here (see
// code-minimalism).
func buildPointerBody(contractText string) string {
	return fmt.Sprintf(
		"---\ndescription: %s\nglobs:\nalwaysApply: true\n---\n"+
			"<!-- %s -->\n\n"+
			"# pulsarules_codex engineering contract\n\n%s\n\n"+
			"Skill rules under %s are pulled in on demand by Cursor, matched against "+
			"each rule's description; this is the only always-applied rule, kept small "+
			"on purpose (Cursor recommends always-applied content stay near 2000 tokens).\n",
		pointerDescription, marker.Installed, contractText, cursorwire.RulesDir,
	)
}

// writeCursorRules renders every selected skill as a Cursor .mdc rule and
// writes it under .cursor/rules, one file per skill id.
func writeCursorRules(ctx Context, report *Report) error {
	for _, id := range ctx.IDs {
		skill, ok := ctx.Index.Skill(id)
		if !ok {
			report.warn("skipped (unknown skill): %s", id)
			continue
		}
		body, err := ctx.Renderer.RenderSkillCursor(ctx.Index, skill, ctx.RouterFilter)
		if err != nil {
			return fmt.Errorf("render %q: %w", id, err)
		}
		if err = writeCursorFile(ctx.Base, id, body, report); err != nil {
			return err
		}
	}
	return nil
}

// writeCursorFile writes body to <base>/.cursor/rules/<id>.mdc via
// cursorwire.WriteRule, noting the write or warning that a foreign file at
// that path was kept.
func writeCursorFile(base, id, body string, report *Report) error {
	wrote, err := cursorwire.WriteRule(base, id, body)
	if err != nil {
		return fmt.Errorf("write cursor rule %q: %w", id, err)
	}
	path := filepath.Join(base, cursorwire.RulesDir, id+".mdc")
	if wrote {
		report.note("installed: %s", path)
	} else {
		report.warn("kept existing user-authored %s (not overwritten)", path)
	}
	return nil
}

// Uninstall removes every .mdc rule Install wrote under .cursor/rules
// (unless ctx.KeepSkills), proven by the ownership marker each carries,
// removes the rules directory once it is left empty, then reaps the .cursor
// root too - a no-op via fsx.RemoveEmptyDir when anything else still lives
// there.
func (cursorTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	if ctx.KeepSkills {
		return report, nil
	}
	removed, err := cursorwire.RemoveRules(ctx.Base)
	if err != nil {
		return report, fmt.Errorf("remove cursor rules: %w", err)
	}
	dir := filepath.Join(ctx.Base, cursorwire.RulesDir)
	for _, id := range removed {
		report.note("removed: %s", filepath.Join(dir, id+".mdc"))
	}
	if err = fsx.RemoveEmptyDir(filepath.Join(ctx.Base, cursorRootDir)); err != nil {
		return report, fmt.Errorf("remove empty cursor dir: %w", err)
	}
	return report, nil
}

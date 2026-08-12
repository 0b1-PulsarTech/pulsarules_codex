package host

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/cursor"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/contract"
)

type cursorTarget struct{}

var _ Target = cursorTarget{}

func (cursorTarget) Name() string { return "cursor" }

func (cursorTarget) Present(base string) bool {
	_, err := os.Stat(filepath.Join(base, cursor.RulesDir))
	return err == nil
}

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

func writeCursorPointer(ctx Context, report *Report) error {
	contractText, err := contract.Session(ctx.Templates)
	if err != nil {
		return fmt.Errorf("routing contract: %w", err)
	}
	body := buildPointerBody(contractText)
	return writeCursorFile(ctx.Base, cursor.PointerID, body, report)
}

func buildPointerBody(contractText string) string {
	const pointerDescription = "Mandatory pulsarules_codex engineering contract - read before " +
		"any Go/SQL/config change"
	// Only this rule is alwaysApply: Cursor injects those into every request (~2000 token cap).
	return fmt.Sprintf(
		"---\ndescription: %s\nglobs:\nalwaysApply: true\n---\n"+
			"<!-- %s -->\n\n"+
			"# pulsarules_codex engineering contract\n\n%s\n\n"+
			"Skill rules under %s are pulled in on demand by Cursor, matched against "+
			"each rule's description; this is the only always-applied rule, kept small "+
			"on purpose (Cursor recommends always-applied content stay near 2000 tokens).\n",
		pointerDescription, ownedfile.Marker, contractText, cursor.RulesDir,
	)
}

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

func writeCursorFile(base, id, body string, report *Report) error {
	wrote, err := cursor.WriteRule(base, id, body)
	if err != nil {
		return fmt.Errorf("write cursor rule %q: %w", id, err)
	}
	path := filepath.Join(base, cursor.RulesDir, id+".mdc")
	if wrote {
		report.note("installed: %s", path)
	} else {
		report.warn("kept existing user-authored %s (not overwritten)", path)
	}
	return nil
}

func (cursorTarget) Uninstall(ctx UninstallContext) (Report, error) {
	var report Report
	if ctx.KeepSkills {
		return report, nil
	}
	removed, err := cursor.RemoveRules(ctx.Base)
	if err != nil {
		return report, fmt.Errorf("remove cursor rules: %w", err)
	}
	dir := filepath.Join(ctx.Base, cursor.RulesDir)
	for _, id := range removed {
		report.note("removed: %s", filepath.Join(dir, id+".mdc"))
	}
	const cursorRootDir = ".cursor"
	// RulesDir is the only thing written under .cursor, so reaping the empty root is safe.
	if err = disk.RemoveEmptyDir(filepath.Join(ctx.Base, cursorRootDir)); err != nil {
		return report, fmt.Errorf("remove empty cursor dir: %w", err)
	}
	return report, nil
}

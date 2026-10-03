package movepurity

import (
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/movepurity/gosrc"
)

func (moves *moveContext) renameRules(
	oldFile, newFile gosrc.File, scope *gosrc.ImportScope, dirs [2]string,
) []gosrc.QualRule {
	oldDir, newDir := dirs[0], dirs[1]
	oldFull, newFull := scope.OldMod.ImportPath(oldDir), scope.NewMod.ImportPath(newDir)
	scope.DropOK, scope.AddOK = nonEmpty(newFull), nonEmpty(oldFull)
	var rules []gosrc.QualRule
	if name := gosrc.LocalName(newFile.Imports, oldFull, oldFile.Pkg); name != "" {
		rules = append(rules, gosrc.QualRule{
			NewQual: name,
			Declared: func(ident string) bool {
				return moves.declared(oldDir, oldFile.Pkg)[ident]
			},
		})
	}
	if name := gosrc.LocalName(oldFile.Imports, newFull, newFile.Pkg); name != "" {
		rules = append(rules, gosrc.QualRule{
			OldQual: name,
			Declared: func(ident string) bool {
				return moves.declared(newDir, newFile.Pkg)[ident]
			},
		})
	}
	return rules
}

func (moves *moveContext) dependantRules(
	oldFile, newFile gosrc.File, scope *gosrc.ImportScope, dir string,
) []gosrc.QualRule {
	var rules []gosrc.QualRule
	for _, moved := range moves.moveInfos() {
		scope.DropOK = append(scope.DropOK, nonEmpty(moved.OldFull)...)
		scope.AddOK = append(scope.AddOK, nonEmpty(moved.NewFull)...)
		oldName := gosrc.LocalName(oldFile.Imports, moved.OldFull, moved.OldPkg)
		newName := gosrc.LocalName(newFile.Imports, moved.NewFull, moved.NewPkg)
		declared := func(ident string) bool { return moved.Decls[ident] }
		if dir == moved.OldDir && oldFile.Pkg == moved.OldPkg && newName != "" {
			rules = append(rules, gosrc.QualRule{NewQual: newName, Declared: declared})
		}
		if dir == moved.NewDir && oldFile.Pkg == moved.NewPkg && oldName != "" {
			rules = append(rules, gosrc.QualRule{OldQual: oldName, Declared: declared})
		}
		if oldName != "" && newName != "" {
			rules = append(
				rules,
				gosrc.QualRule{OldQual: oldName, NewQual: newName, Declared: declared},
			)
		}
	}
	return slices.Clip(rules)
}

func nonEmpty(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

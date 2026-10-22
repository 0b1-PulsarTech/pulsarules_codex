package movepurity

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func (moves *moveContext) relocationVerdict(oldFile, newFile gosrc.File, edit fileEdit) verdict {
	dirs := edit.dirs()
	if gosrc.RelocationUnsafe(oldFile, newFile) || moves.takesOverName(newFile, edit) ||
		!moves.loadFollowsDir(newFile, edit) || !moves.embedsFollow(newFile.Embeds, dirs) {
		return verdictClauseOnly
	}
	file, listed := moves.strandedCompiled(dirs)
	switch {
	case !listed:
		return verdictClauseOnly
	case file != "":
		moves.stranded[edit.newPath] = file
		return verdictStrands
	}
	return verdictPure
}

func (moves *moveContext) strandedCompiled(dirs dirPair) (file string, listed bool) {
	for _, dir := range []string{dirs.oldDir, dirs.newDir} {
		files, err := moves.staged.dir(dir)
		if err != nil {
			return "", false
		}
		// An assembly or C file outside the move changes which sources the package builds.
		idx := slices.IndexFunc(files, func(file string) bool {
			return gosrc.IsCompiled(file) && !isGoFile(file) && !moves.moved[file]
		})
		if idx >= 0 {
			return files[idx], true
		}
	}
	return "", true
}

func (moves *moveContext) loadFollowsDir(newFile gosrc.File, edit fileEdit) bool {
	if !newFile.Loads && !newFile.TopLevel["TestMain"] {
		return true
	}
	// A production package moved whole still reorders cross-package init,
	// so only tests are exempt.
	return isTestFile(edit.newPath) && moves.isWholeDirMove(edit)
}

func (moves *moveContext) isWholeDirMove(edit fileEdit) bool {
	dirs := edit.dirs()
	left, leftErr := moves.staged.dir(dirs.oldDir)
	present, presentErr := moves.head.dir(dirs.newDir)
	if leftErr != nil || presentErr != nil ||
		slices.ContainsFunc(left, isGoFile) || slices.ContainsFunc(present, isGoFile) {
		return false
	}
	// A rename that leaves oldDir for elsewhere, or reaches newDir from elsewhere,
	// splits or merges the package.
	return !slices.ContainsFunc(moves.renames, func(rename pass.Rename) bool {
		moved := renameDirs(rename)
		return (moved.oldDir == dirs.oldDir) != (moved.newDir == dirs.newDir)
	})
}

func (moves *moveContext) unlinkedImport(oldFile, newFile gosrc.File) string {
	for _, moved := range moves.moveInfos() {
		if dropsImport(oldFile, newFile, moved.OldFull) && moves.dirKeepsLoad(moved.OldDir) {
			return moved.OldFull
		}
	}
	return ""
}

func dropsImport(oldFile, newFile gosrc.File, target string) bool {
	imports := func(file gosrc.File) bool {
		return slices.ContainsFunc(file.Imports, func(spec gosrc.ImportSpec) bool {
			return spec.Path == target
		})
	}
	return target != "" && imports(oldFile) && !imports(newFile)
}

func (moves *moveContext) dirKeepsLoad(dir string) bool {
	files, err := moves.staged.dir(dir)
	if err != nil {
		return true
	}
	return slices.ContainsFunc(files, func(file string) bool {
		if !isProductionGo(file) || gosrc.IsOutsidePackage(file) {
			return false
		}
		parsed, err := moves.staged.parse(file)
		return err != nil || (parsed.Loads && !parsed.IsIgnored)
	})
}

func (moves *moveContext) embedsFollow(patterns []string, dirs dirPair) bool {
	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, "*?[\\") || strings.HasPrefix(pattern, "all:") ||
			!moves.pathFollows(path.Join(dirs.oldDir, pattern), path.Join(dirs.newDir, pattern)) {
			return false
		}
	}
	return true
}

func (moves *moveContext) takesOverName(newFile gosrc.File, edit fileEdit) bool {
	owned := moves.headNamesIn(newDeclScope(edit))
	for name := range newFile.TopLevel {
		// A surviving owner would clash at build time, so an overlap is an owner that left.
		if name != "_" && name != "init" && owned[name] {
			return true
		}
	}
	return false
}

func (moves *moveContext) headNamesIn(scope declScope) map[string]bool {
	return cached(moves.owned, scope, func() map[string]bool {
		set := map[string]bool{}
		files, err := moves.head.dir(scope.newDir)
		if err != nil {
			return set
		}
		for _, file := range files {
			// Every .go file counts, whatever its package clause or test-ness.
			if isGoFile(file) {
				moves.head.addAnyDecls(file, set)
			}
		}
		return set
	})
}

func relocationUnsafeFinding(
	reporter finding.Reporter, rename pass.Rename, lead string,
) finding.Finding {
	return reporter.At(
		rename.NewPath,
		0,
		fmt.Sprintf(
			"move of %s -> %s %s not behaviour-neutral: in %s "+
				"a name resolves differently or load-time code changes package",
			rename.OldPath, rename.NewPath, lead, path.Dir(rename.NewPath),
		),
		"first commit an edit in the old directory that removes the conflict "+
			"(rename or qualify the clashing name, move init/registration and other "+
			"load-time work out of the file), then commit the move with only its "+
			"package clause and import fixups; "+
			"test files can move with their whole directory in one commit",
	)
}

func inclusionChangeFinding(reporter finding.Reporter, rename pass.Rename) finding.Finding {
	return reporter.At(
		rename.NewPath,
		0,
		fmt.Sprintf(
			"move of %s -> %s is byte-identical but changes how the build includes the file "+
				"(build-ignored path, root vendor/, _test suffix, GOOS/GOARCH suffix "+
				"or extension); it is not a pure move",
			rename.OldPath, rename.NewPath,
		),
		"commit this rename on its own and state the inclusion change in its message",
	)
}

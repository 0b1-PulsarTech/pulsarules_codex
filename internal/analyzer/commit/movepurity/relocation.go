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
	dirs := [2]string{path.Dir(edit.oldPath), path.Dir(edit.newPath)}
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

func (moves *moveContext) strandedCompiled(dirs [2]string) (file string, listed bool) {
	for _, dir := range dirs {
		files, err := moves.reader.StagedDir(dir)
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
	return strings.HasSuffix(edit.newPath, "_test.go") && moves.isWholeDirMove(edit)
}

func (moves *moveContext) isWholeDirMove(edit fileEdit) bool {
	oldDir, newDir := path.Dir(edit.oldPath), path.Dir(edit.newPath)
	left, leftErr := moves.reader.StagedDir(oldDir)
	present, presentErr := moves.reader.HeadDir(newDir)
	if leftErr != nil || presentErr != nil ||
		slices.ContainsFunc(left, isGoFile) || slices.ContainsFunc(present, isGoFile) {
		return false
	}
	// A rename that leaves oldDir for elsewhere, or reaches newDir from elsewhere,
	// splits or merges the package.
	return !slices.ContainsFunc(moves.renames, func(rename pass.Rename) bool {
		return (path.Dir(rename.OldPath) == oldDir) != (path.Dir(rename.NewPath) == newDir)
	})
}

func isGoFile(file string) bool { return strings.HasSuffix(file, ".go") }

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
	files, err := moves.reader.StagedDir(dir)
	if err != nil {
		return true
	}
	return slices.ContainsFunc(files, func(file string) bool {
		if !isProductionGo(file) || gosrc.IsOutsidePackage(file) {
			return false
		}
		src, readErr := moves.reader.StagedBlob(file)
		parsed, parseErr := gosrc.Parse(src)
		return readErr != nil || parseErr != nil || (parsed.Loads && !parsed.IsIgnored)
	})
}

func (moves *moveContext) embedsFollow(patterns []string, dirs [2]string) bool {
	for _, pattern := range patterns {
		if strings.ContainsAny(pattern, "*?[\\") || strings.HasPrefix(pattern, "all:") ||
			moves.movedTo[path.Join(dirs[0], pattern)] != path.Join(dirs[1], pattern) {
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
	return moves.cachedDecls("owned", scope, "", func(set map[string]bool) {
		files, err := moves.reader.HeadDir(scope.newDir)
		if err != nil {
			return
		}
		for _, file := range files {
			// Every .go file counts, whatever its package clause or test-ness.
			if !strings.HasSuffix(file, ".go") {
				continue
			}
			if src, readErr := moves.reader.HeadBlob(file); readErr == nil {
				gosrc.CollectAnyDecls(src, set)
			}
		}
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

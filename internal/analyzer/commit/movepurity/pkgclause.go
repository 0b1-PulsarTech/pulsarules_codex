package movepurity

import (
	"path"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
)

const testSuffix = "_test"

type (
	cleanName struct {
		name    string
		isMixed bool
	}
	dirPackages struct {
		staying cleanName
		moving  cleanName
	}
)

func (current cleanName) merge(next string) cleanName {
	switch {
	case next == "" || current.name == next:
		return current
	case current.name == "" && !current.isMixed:
		return cleanName{name: next}
	}
	return cleanName{isMixed: true}
}

func (names dirPackages) want() (string, bool) {
	if names.staying.isMixed || (names.staying.name == "" && names.moving.isMixed) {
		return "", false
	}
	if names.staying.name != "" {
		return names.staying.name, true
	}
	return names.moving.name, true
}

func (moves *moveContext) packageAllowed(oldFile, newFile gosrc.File, edit fileEdit) bool {
	if oldFile.IsIgnored || newFile.IsIgnored {
		return oldFile.Pkg == newFile.Pkg
	}
	dirs := edit.dirs()
	if dirs.oldDir == dirs.newDir && oldFile.Pkg != newFile.Pkg {
		return false
	}
	if strings.HasSuffix(oldFile.Pkg, testSuffix) != strings.HasSuffix(newFile.Pkg, testSuffix) {
		return false
	}
	names := moves.packagesOf(dirs.newDir)
	want, isClean := names.want()
	if !isClean || strings.TrimSuffix(newFile.Pkg, testSuffix) != want {
		return false
	}
	return newFile.Pkg != "main" || oldFile.Pkg == "main" ||
		names.staying.name == "main"
}

func (moves *moveContext) packagesOf(dir string) dirPackages {
	return cached(moves.dirNames, dir, func() dirPackages {
		var names dirPackages
		if staged, err := moves.staged.dir(dir); err == nil {
			for _, file := range staged {
				if !moves.moved[file] && !isTestFile(file) {
					names.staying = names.staying.merge(moves.cleanPackage(file))
				}
			}
		}
		for _, rename := range moves.renames {
			if path.Dir(rename.NewPath) == dir {
				names.moving = names.moving.merge(moves.cleanPackage(rename.NewPath))
			}
		}
		return names
	})
}

func (moves *moveContext) cleanPackage(file string) string {
	if !isGoFile(file) {
		return ""
	}
	name, isIgnored := moves.staged.clause(file)
	if isIgnored {
		return ""
	}
	return strings.TrimSuffix(name, testSuffix)
}

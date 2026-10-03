package movepurity

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/commit/movepurity/gosrc"
)

const testSuffix = "_test"

type dirPackages struct {
	staying string
	moving  string
}

func (moves *moveContext) packageAllowed(oldFile, newFile gosrc.File, edit fileEdit) bool {
	if oldFile.IsIgnored || newFile.IsIgnored {
		return oldFile.Pkg == newFile.Pkg
	}
	oldDir, newDir := path.Dir(edit.oldPath), path.Dir(edit.newPath)
	if oldDir == newDir && oldFile.Pkg != newFile.Pkg {
		return false
	}
	if strings.HasSuffix(oldFile.Pkg, testSuffix) != strings.HasSuffix(newFile.Pkg, testSuffix) {
		return false
	}
	names := moves.packagesOf(newDir)
	want := cmp.Or(names.staying, names.moving)
	if strings.TrimSuffix(newFile.Pkg, testSuffix) != want {
		return false
	}
	return newFile.Pkg != "main" || oldFile.Pkg == "main" ||
		names.staying == "main"
}

func (moves *moveContext) packagesOf(dir string) dirPackages {
	if names, ok := moves.dirNames[dir]; ok {
		return names
	}
	var names dirPackages
	if staged, err := moves.reader.StagedDir(dir); err == nil {
		for _, file := range staged {
			if !moves.moved[file] && !strings.HasSuffix(file, "_test.go") {
				names.staying = mergeName(names.staying, moves.cleanPackage(file))
			}
		}
	}
	for _, rename := range moves.renames {
		if path.Dir(rename.NewPath) == dir {
			names.moving = mergeName(names.moving, moves.cleanPackage(rename.NewPath))
		}
	}
	moves.dirNames[dir] = names
	return names
}

func (moves *moveContext) cleanPackage(file string) string {
	if path.Ext(file) != ".go" {
		return ""
	}
	src, err := moves.reader.StagedBlob(file)
	if err != nil {
		return ""
	}
	name, isIgnored := gosrc.PackageClause(src)
	if isIgnored {
		return ""
	}
	return strings.TrimSuffix(name, testSuffix)
}

func (moves *moveContext) packageName(key moduleKey) string {
	if name, ok := moves.pkgNames[key]; ok {
		return name
	}
	list, read := moves.reader.HeadDir, moves.reader.HeadBlob
	if key.isStaged {
		list, read = moves.reader.StagedDir, moves.reader.StagedBlob
	}
	name := ""
	if files, err := list(key.dir); err == nil {
		slices.Sort(files)
		name = firstPackage(files, read)
	}
	moves.pkgNames[key] = name
	return name
}

func firstPackage(files []string, read func(string) (string, error)) string {
	for _, file := range files {
		if !isProductionGo(file) {
			continue
		}
		src, err := read(file)
		clause, isIgnored := gosrc.PackageClause(src)
		if err == nil && !isIgnored && clause != "" {
			return clause
		}
	}
	return ""
}

func mergeName(current, next string) string {
	if next == "" || current == next {
		return current
	}
	if current == "" {
		return next
	}
	return "\x00mixed"
}

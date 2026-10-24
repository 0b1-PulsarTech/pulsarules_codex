package dartsrc

import (
	"path"
	"slices"
	"strings"
)

type (
	Source struct {
		File  File
		Path  string
		Roots func() Roots
	}
	Pair struct {
		Old Source
		New Source
	}
	linker struct {
		pair    Pair
		follows func(oldTarget, newTarget string) bool
	}
)

func Equivalent(
	pair Pair,
	follows func(oldTarget, newTarget string) bool,
) (isEquivalent bool, rewrites int) {
	oldDirs, newDirs := pair.Old.File.Directives, pair.New.File.Directives
	if pair.Old.File.Body != pair.New.File.Body || len(oldDirs) != len(newDirs) {
		return false, 0
	}
	link := linker{pair: pair, follows: follows}
	unmatched := slices.Clone(newDirs)
	var rest []Directive
	for _, directive := range oldDirs {
		var isFound bool
		unmatched, isFound = take(unmatched, directive, link.sameText)
		if !isFound {
			rest = append(rest, directive)
		}
	}
	for _, directive := range rest {
		var isFound bool
		unmatched, isFound = take(unmatched, directive, link.changedText)
		if !isFound {
			return false, 0
		}
	}
	return true, len(rest)
}

func take(
	pool []Directive,
	want Directive,
	matches func(oldURI, newURI string) bool,
) (remaining []Directive, isFound bool) {
	index := slices.IndexFunc(pool, func(candidate Directive) bool {
		if candidate.Shape != want.Shape || len(candidate.URIs) != len(want.URIs) {
			return false
		}
		for position, uri := range want.URIs {
			if !matches(uri, candidate.URIs[position]) {
				return false
			}
		}
		return true
	})
	if index < 0 {
		return pool, false
	}
	return slices.Delete(pool, index, index+1), true
}

func (link linker) resolve(source Source, uri string) (target string, ok bool) {
	var roots Roots
	if strings.HasPrefix(uri, "package:") {
		roots = source.Roots()
	}
	return Resolve(source.Path, uri, roots)
}

func (link linker) sameText(oldURI, newURI string) bool {
	if oldURI != newURI {
		return false
	}
	name, isPackage := packageOf(oldURI)
	oldTarget, oldOK := link.resolve(link.pair.Old, oldURI)
	newTarget, newOK := link.resolve(link.pair.New, newURI)
	switch {
	case oldOK && newOK:
		return link.follows(oldTarget, newTarget)
	case isPackage:
		_, isOldKnown := link.pair.Old.Roots()[name]
		_, isNewKnown := link.pair.New.Roots()[name]
		return !isOldKnown && !isNewKnown
	case isRelative(oldURI):
		return !oldOK && !newOK && path.Dir(link.pair.Old.Path) == path.Dir(link.pair.New.Path)
	}
	return true
}

func (link linker) changedText(oldURI, newURI string) bool {
	if oldURI == newURI {
		return link.sameText(oldURI, newURI)
	}
	oldTarget, oldOK := link.resolve(link.pair.Old, oldURI)
	newTarget, newOK := link.resolve(link.pair.New, newURI)
	if !oldOK || !newOK || !link.follows(oldTarget, newTarget) {
		return false
	}
	isImporterMoved := link.pair.Old.Path != link.pair.New.Path
	if oldTarget == newTarget && !isImporterMoved {
		return false
	}
	return link.sameForm(oldURI, newURI, oldTarget, newTarget)
}

func (link linker) sameForm(oldURI, newURI, oldTarget, newTarget string) bool {
	oldName, isOldPackage := packageOf(oldURI)
	newName, isNewPackage := packageOf(newURI)
	switch {
	case isOldPackage && isNewPackage:
		return oldName == newName
	case isOldPackage:
		return link.inLib(oldName, oldTarget, newTarget)
	case isNewPackage:
		return link.inLib(newName, oldTarget, newTarget)
	}
	return true
}

func (link linker) inLib(name, oldTarget, newTarget string) bool {
	within := func(source Source, target string) bool {
		dir := source.Roots()[name]
		lib := path.Join(dir, "lib") + "/"
		return strings.HasPrefix(source.Path, lib) && strings.HasPrefix(target, lib)
	}
	return within(link.pair.Old, oldTarget) && within(link.pair.New, newTarget)
}

func packageOf(uri string) (name string, isPackage bool) {
	rest, hasPrefix := strings.CutPrefix(uri, "package:")
	name, _, _ = strings.Cut(rest, "/")
	return name, hasPrefix
}

func isRelative(uri string) bool {
	scheme, _, _ := strings.Cut(uri, "/")
	return !strings.Contains(scheme, ":") && !strings.HasPrefix(uri, "/")
}

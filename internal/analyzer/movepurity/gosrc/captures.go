package gosrc

import (
	"go/token"
	"go/types"
	"slices"
)

type CaptureSets struct {
	Left  map[string]bool
	Taken map[string]bool
}

func (sets CaptureSets) captures(name string) bool {
	return sets.Taken[name] || (sets.Left[name] && types.Universe.Lookup(name) != nil)
}

func capturesBare(newFile File, newPos int, bind Binding) bool {
	tok := newFile.body[newPos]
	if tok.kind != token.IDENT || (!newFile.unresolved[tok.off] && !newFile.keys[tok.off]) {
		return false
	}
	return bind.Captures.captures(tok.lit)
}

func RelocationUnsafe(oldFile, newFile File) bool {
	for _, spec := range slices.Concat(oldFile.Imports, newFile.Imports) {
		if spec.Name == "." || spec.Path == "C" {
			return true
		}
	}
	for name := range newFile.TopLevel {
		if types.Universe.Lookup(name) != nil {
			return true
		}
	}
	return false
}

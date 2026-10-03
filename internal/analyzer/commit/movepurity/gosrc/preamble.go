package gosrc

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func PreambleMatches(oldFile, newFile File) bool {
	if len(oldFile.preamble) != len(newFile.preamble) {
		return false
	}
	for idx, oldTok := range oldFile.preamble {
		newTok := newFile.preamble[idx]
		if oldTok.kind != newTok.kind {
			return false
		}
		lit := oldTok.lit
		if oldTok.off >= oldFile.docStart && oldTok.off < oldFile.docEnd {
			lit = replacePackageWord(lit, oldFile.Pkg, newFile.Pkg)
		}
		if lit != newTok.lit {
			return false
		}
	}
	return true
}

func replacePackageWord(comment, oldPkg, newPkg string) string {
	needle := "Package " + oldPkg
	before, rest, found := strings.Cut(comment, needle)
	if !found {
		return comment
	}
	if next, _ := utf8.DecodeRuneInString(
		rest,
	); next == '_' || unicode.IsLetter(next) ||
		unicode.IsDigit(next) {
		return comment
	}
	return before + "Package " + newPkg + rest
}

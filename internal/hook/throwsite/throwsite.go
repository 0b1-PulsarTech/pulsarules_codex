package throwsite

import (
	"regexp"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func patternFor(ext string) string {
	switch ext {
	case ".dart":
		return `rethrow|throw|Future\.error`
	case ".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs", ".svelte":
		return `throw|Promise\.reject`
	}
	return ""
}

func Handles(ext string) bool { return patternFor(strings.ToLower(ext)) != "" }

type Hit struct {
	Line int
	Word string
}

func Scan(ext, src string) []Hit {
	ext = strings.ToLower(ext)
	pattern := patternFor(ext)
	if pattern == "" {
		return nil
	}
	if ext == ".svelte" {
		src = maskMarkupComments(src)
	}
	literals := language.ScanLiterals(
		ext, src, language.StringsEndAtNewline(), language.KeepSpansAtEOF(),
	)
	var hits []Hit
	for _, found := range regexp.MustCompile(pattern).FindAllStringIndex(src, -1) {
		start, end := found[0], found[1]
		if !standsAlone(src, start, end) || !inCode(literals, start) {
			continue
		}
		line := 1 + strings.Count(src[:start], "\n")
		hit := Hit{Line: line, Word: src[start:end]}
		if !repeated(hits, hit) {
			hits = append(hits, hit)
		}
	}
	return hits
}

func repeated(hits []Hit, hit Hit) bool {
	for at := len(hits) - 1; at >= 0 && hits[at].Line == hit.Line; at-- {
		if hits[at] == hit {
			return true
		}
	}
	return false
}

func inCode(literals language.Literals, offset int) bool {
	// A comment or literal open at EOF runs to EOF, so nothing after its opener is searched.
	return literals.Covers(offset) && !literals.Contains(offset) && !literals.InComment(offset)
}

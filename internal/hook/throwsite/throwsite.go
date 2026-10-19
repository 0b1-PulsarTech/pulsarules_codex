package throwsite

import (
	"regexp"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type rule struct {
	languages []language.ID
	pattern   *regexp.Regexp
	skill     string
}

type Scanner struct {
	rules []rule
}

func New() *Scanner {
	return &Scanner{rules: []rule{
		{
			languages: []language.ID{language.Dart},
			pattern:   regexp.MustCompile(`rethrow|throw|Future\.error`),
			skill:     "dart-errors",
		},
		{
			languages: []language.ID{language.TypeScript, language.JavaScript, language.Svelte},
			pattern:   regexp.MustCompile(`throw|Promise\.reject`),
			skill:     "ts-errors",
		},
	}}
}

func (scanner *Scanner) ruleFor(ext string) (rule, bool) {
	id := language.ForExtension(ext).ID
	for _, candidate := range scanner.rules {
		if slices.Contains(candidate.languages, id) {
			return candidate, true
		}
	}
	return rule{}, false
}

func (scanner *Scanner) Handles(ext string) bool {
	_, found := scanner.ruleFor(ext)
	return found
}

func (scanner *Scanner) SkillFor(ext string) string {
	found, _ := scanner.ruleFor(ext)
	return found.skill
}

type Hit struct {
	Line int
	Word string
}

func (scanner *Scanner) Scan(ext, src string) []Hit {
	matched, handled := scanner.ruleFor(ext)
	if !handled {
		return nil
	}
	literals := language.ScanLiterals(
		ext, src, language.StringsEndAtNewline(), language.KeepSpansAtEOF(),
	)
	var hits []Hit
	for _, found := range matched.pattern.FindAllStringIndex(src, -1) {
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

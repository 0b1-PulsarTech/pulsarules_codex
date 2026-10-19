package throwsite

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func SkillFor(ext string) string {
	switch {
	case strings.ToLower(ext) == ".dart":
		return "dart-errors"
	case Handles(ext):
		return "ts-errors"
	}
	return ""
}

func Notice(file string, hits []Hit, skill string) string {
	const maxListed = 5
	var lines printable.LineList
	for _, hit := range hits {
		lines.Add(hit.Line)
	}
	where := lines.Noun() + " " + lines.Head(maxListed)
	if extra := lines.Extra(maxListed); extra > 0 {
		where += fmt.Sprintf(" and %d more", extra)
	}
	text := fmt.Sprintf(
		"%s at %s of %s: return a result (the error last, or named) instead of %s; "+
			"a throw stays only for a broken invariant or a framework-mandated boundary.",
		matched(hits), where, file, instead(hits),
	)
	if skill != "" {
		text += " See the " + skill + " skill."
	}
	return text
}

func matched(hits []Hit) string {
	var words []string
	for _, hit := range hits {
		if !slices.Contains(words, hit.Word) {
			words = append(words, hit.Word)
		}
	}
	return "`" + strings.Join(words, "` / `") + "`"
}

func instead(hits []Hit) string {
	var phrases []string
	for _, hit := range hits {
		phrase := "throwing"
		switch hit.Word {
		case "Promise.reject":
			phrase = "returning a rejected promise"
		case "Future.error":
			phrase = "returning a failed future"
		}
		if !slices.Contains(phrases, phrase) {
			phrases = append(phrases, phrase)
		}
	}
	return strings.Join(phrases, " or ")
}

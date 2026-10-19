package throwsite

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ordered"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func Notice(file string, hits []Hit, skill string) string {
	const maxListed = 5
	var lines printable.LineList
	for _, hit := range hits {
		lines.Add(hit.Line)
	}
	where := lines.Label(maxListed)
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
	words := make([]string, 0, len(hits))
	for _, hit := range hits {
		words = append(words, hit.Word)
	}
	return "`" + strings.Join(ordered.Dedupe(words), "` / `") + "`"
}

func instead(hits []Hit) string {
	phrases := make([]string, 0, len(hits))
	for _, hit := range hits {
		phrase := "throwing"
		switch hit.Word {
		case "Promise.reject":
			phrase = "returning a rejected promise"
		case "Future.error":
			phrase = "returning a failed future"
		}
		phrases = append(phrases, phrase)
	}
	return strings.Join(ordered.Dedupe(phrases), " or ")
}

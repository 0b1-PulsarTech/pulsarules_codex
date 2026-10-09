package throwscan

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	var lines []string
	for _, hit := range hits {
		line := strconv.Itoa(hit.Line)
		if len(lines) == 0 || lines[len(lines)-1] != line {
			lines = append(lines, line)
		}
	}
	where := "line " + strings.Join(lines[:min(len(lines), maxListed)], ", ")
	if len(lines) > 1 {
		where = "lines " + strings.Join(lines[:min(len(lines), maxListed)], ", ")
	}
	if extra := len(lines) - maxListed; extra > 0 {
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

package character

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible/cleaner"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
)

const codeNote = "it may be literal data in code here; judge it, then remove it by hand"

type hint struct {
	text    string
	caution int
}

const (
	plain = iota
	unscanned
	literal
)

func unreadNote(literals language.Literals, found invisible.Mark) string {
	const escapeNote = "; inside a string literal it is data, " +
		"so write the language's escape instead"

	line := literals.UnreadFrom()
	switch {
	case line > 0 && found.Line >= line:
		return fmt.Sprintf("; could not read this file's strings from line %d on", line) +
			escapeNote
	case line > 0:
		return fmt.Sprintf(
			"; could not read this file's strings reliably: a literal on line %d never closes",
			line,
		) + escapeNote
	}
	if literals.Incomplete() {
		return "; could not read this file's strings" + escapeNote
	}
	return escapeNote
}

func markdownCode(ext, src string) markdown.Code {
	if !strings.EqualFold(ext, ".md") {
		return markdown.Code{}
	}
	// Only markdown skips code regions; a literal in a language file gets the escape hint instead.
	return markdown.CodeRegions(src)
}

func carrierAdvice(
	found invisible.Mark, ext string, literals language.Literals, code markdown.Code,
) hint {
	switch {
	case literals.Contains(found.Offset):
		return hint{escapeAdvice(literals, found), literal}
	case found.Class == invisible.ClassContextual:
		return hint{"it may be load-bearing here; judge it, then remove it by hand", plain}
	case code.Possible(found.Offset):
		return hint{codeNote, unscanned}
	case cleaner.Handles(ext) && !literals.Incomplete():
		return hint{"run `pulsarules_cli clean --write`, which removes this class", plain}
	case !literals.Covers(found.Offset) || literals.Incomplete():
		return hint{"remove it by hand" + unreadNote(literals, found), unscanned}
	}
	return hint{"remove it by hand", plain}
}

func typographicAdvice(
	found invisible.Mark, ext string, literals language.Literals, code markdown.Code,
) hint {
	const byHand = "replace it with the ASCII form by hand"
	switch {
	case literals.Contains(found.Offset):
		return hint{escapeAdvice(literals, found), literal}
	case code.Possible(found.Offset):
		return hint{"it may be shown code here; judge it, then " + byHand, unscanned}
	case unread(found, ext, literals):
		return hint{byHand + unreadNote(literals, found), unscanned}
	}
	return hint{byHand, plain}
}

func escapeAdvice(literals language.Literals, found invisible.Mark) string {
	const lastBMP = 0xFFFF
	escape := literals.Escape(found.Rune)
	switch {
	case literals.Raw(found.Offset):
		return "a raw string cannot hold an escape: move it into an interpreted string as " +
			escape
	case literals.Regex(found.Offset) && found.Rune > lastBMP:
		return "write it as the escape " + escape + " and add the u flag"
	}
	return "write it as the escape " + escape
}

func unread(found invisible.Mark, ext string, literals language.Literals) bool {
	if strings.EqualFold(ext, ".md") {
		return false
	}
	return !literals.Covers(found.Offset) || literals.Incomplete()
}

package cleaner

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

func EscapeAdvice(literals language.Literals, found invisible.Mark) string {
	const lastBMP = 0xFFFF
	if !literals.Lexed() {
		return fmt.Sprintf("write the language's escape for U+%04X instead", found.Rune)
	}
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

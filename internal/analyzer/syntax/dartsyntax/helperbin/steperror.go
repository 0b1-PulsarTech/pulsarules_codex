package helperbin

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/process"
)

func StepError(step string, result process.Result, err error) error {
	// The captured output is appended below, so the wrapped Stderr would repeat it.
	if execErr, ok := errors.AsType[*process.Error](err); ok {
		err = execErr.Err
	}
	output := tail(result.Stderr + "\n" + result.Stdout)
	if output == "" {
		return fmt.Errorf("dart %s: %w", step, err)
	}
	return fmt.Errorf("dart %s: %w: %s", step, err, output)
}

func clip(output string) (text string, isClipped bool) {
	const maxChars = 400
	text = strings.TrimSpace(output)
	if len(text) <= maxChars {
		return text, false
	}
	start := len(text) - maxChars
	for start < len(text) && !utf8.RuneStart(text[start]) {
		start++
	}
	return text[start:], true
}

func tail(output string) string {
	text, isClipped := clip(output)
	// Raw output stays byte-exact; only a clipped tail may start inside a control sequence.
	if isClipped {
		return printable.Line(text)
	}
	return text
}

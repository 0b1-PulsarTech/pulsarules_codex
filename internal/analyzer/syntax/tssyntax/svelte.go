package tssyntax

import (
	"errors"
	"fmt"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/tssyntax/facts"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/sfc"
)

func summarizeSvelte(summary *syntax.File, options facts.Options, text string) {
	scripts, err := sfc.Extract(text)
	partial := errors.Is(err, sfc.ErrDroppedScript) || errors.Is(err, sfc.ErrUnclosed)
	if err != nil && !partial {
		summary.ParseError = fmt.Sprintf("skipped: %v", err)
		return
	}
	for block, script := range scripts {
		kind := core.ScriptKindJS
		if script.TypeScript {
			kind = core.ScriptKindTS
		}
		before := summary.ParseError
		summarizeScript(summary, options, scriptBlock{text: script.Text, kind: kind, index: block})
		if before != "" {
			summary.ParseError = before
		}
	}
	if partial && summary.ParseError == "" {
		summary.ParseError = err.Error()
	}
}

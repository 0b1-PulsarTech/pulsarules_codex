package tssyntax

import (
	"errors"
	"fmt"

	"github.com/Zzzen/typescript-go/use-at-your-own-risk/core"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/svelte"
)

func summarizeSvelte(summary *syntax.File, source syntax.Source, text string) {
	scripts, err := svelte.Extract(text)
	partial := errors.Is(err, svelte.ErrDroppedScript) || errors.Is(err, svelte.ErrUnclosed)
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
		summarizeScript(summary, source, scriptBlock{text: script.Text, kind: kind, index: block})
		if before != "" {
			summary.ParseError = before
		}
	}
	if partial && summary.ParseError == "" {
		summary.ParseError = err.Error()
	}
}

package throwscan

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/langspec"
)

func maskMarkupComments(src string) string {
	// Without KeepSpansAtEOF: a fake script region that runs to its end must stay trimmed.
	scripts := langspec.ScanLiterals(".svelte", src, langspec.StringsEndAtNewline())
	masked := []byte(src)
	for from := 0; ; {
		opener := strings.Index(src[from:], "<!--")
		if opener < 0 {
			return string(masked)
		}
		start := from + opener
		end := len(src)
		if closer := strings.Index(src[start:], "-->"); closer >= 0 {
			end = start + closer + len("-->")
		}
		// The lexer opens a script at any "<script", also inside <!-- -->; a comment that
		// starts inside a real script block is script text and stays, and its closer may
		// belong to a later markup comment.
		if scripts.Covers(start) {
			from = start + len("<!--")
			continue
		}
		from = end
		for at := start; at < end; at++ {
			if masked[at] != '\n' {
				masked[at] = ' '
			}
		}
	}
}

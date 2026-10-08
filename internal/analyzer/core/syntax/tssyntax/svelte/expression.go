package svelte

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax/tssyntax/tsguard"
)

func skipBraces(lower string, from int) tsguard.Scan {
	return tsguard.BraceEnd(lower, from+1)
}

func skipMustache(lower string, from int) tsguard.Scan {
	return tsguard.BraceEnd(lower, afterSigil(lower, from+1))
}

func afterSigil(lower string, body int) int {
	for body < len(lower) && strings.IndexByte(blanks, lower[body]) >= 0 {
		body++
	}
	if body >= len(lower) || strings.IndexByte("#/:@", lower[body]) < 0 {
		return body
	}
	if lower[body] == '/' && !isBlockClose(lower[body:]) {
		return body
	}
	body++
	for body < len(lower) && isNameStart(lower[body]) {
		body++
	}
	return body
}

func isBlockClose(body string) bool {
	return len(body) > 1 && body[0] == '/' && isNameStart(body[1])
}

func (scan *markup) view(from int) string {
	if !scan.bounded {
		return scan.lower
	}
	end := strings.IndexByte(scan.lower[from:], '\n')
	if end < 0 {
		return scan.lower
	}
	return scan.lower[:from+end]
}

func scanWork(braces tsguard.Scan, view string, from int) int {
	if braces.Exhausted {
		return len(view) - from
	}
	return braces.End - from
}

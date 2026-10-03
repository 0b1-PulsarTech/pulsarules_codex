package narration

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

const (
	openNone = iota
	openAlways
	openRust
	openScalar
	openCLike
)

type rules struct {
	open      int
	backslash bool
	rawSingle bool
	backtick  bool
	sfc       bool
	verbatim  bool
}

func rulesFor(syntax lexer.CommentSyntax, ext string) rules {
	hasBlock := func(open string) bool {
		return slices.ContainsFunc(syntax.Blocks, func(block lexer.BlockDelim) bool {
			return block.Open == open
		})
	}
	switch {
	case ext == ".rs":
		return rules{open: openRust, backslash: true}
	case slices.Contains([]string{".yaml", ".yml", ".toml"}, ext):
		return rules{open: openScalar, backslash: true, rawSingle: true}
	case slices.Contains([]string{".sh", ".bash", ".zsh"}, ext):
		return rules{open: openAlways, backslash: true, rawSingle: true}
	case slices.Contains(syntax.Line, "--") && hasBlock("/*"):
		return rules{open: openAlways}
	case slices.Contains(syntax.Line, "--"), slices.Contains(syntax.Line, "#"):
		return rules{open: openAlways, backslash: true}
	case slices.Contains(syntax.Line, "//"):
		return rules{
			open: openCLike, backslash: true, backtick: true, sfc: hasBlock("<!--"),
			verbatim: ext == ".cs",
		}
	}
	return rules{open: openNone}
}

func isIdentByte(char byte) bool {
	return char == '_' || char >= '0' && char <= '9' ||
		char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

func identBefore(line string, at int) string {
	start := at
	for start > 0 && isIdentByte(line[start-1]) {
		start--
	}
	return line[start:at]
}

func rawHashes(line string, at int) (count int, prefix string) {
	start := at
	for start > 0 && line[start-1] == '#' {
		start--
	}
	return at - start, identBefore(line, start)
}

func (quoting rules) stringEnd(line string, at int) (end int, closed bool) {
	if hashes, raw := quoting.rawString(line, at); raw {
		return rawEnd(line, at, hashes)
	}
	if quoting.verbatim && line[at] == '"' && verbatimBefore(line, at) {
		return verbatimEnd(line, at)
	}
	return quoting.quotedEnd(line, at)
}

func rawEnd(line string, at, hashes int) (end int, closed bool) {
	closer := string(line[at]) + strings.Repeat("#", hashes)
	found := strings.Index(line[at+1:], closer)
	if found < 0 {
		return len(line), false
	}
	return at + 1 + found + len(closer), true
}

func (quoting rules) quotedEnd(line string, at int) (end int, closed bool) {
	quote := line[at]
	escapes := quoting.backslash && (quote != '\'' || !quoting.rawSingle)
	for pos := at + 1; pos < len(line); pos++ {
		switch {
		case line[pos] == quote:
			return pos + 1, true
		case line[pos] == '\\' && escapes:
			pos++
		}
	}
	return len(line), false
}

func (quoting rules) rawString(line string, at int) (hashes int, raw bool) {
	var prefix string
	hashes, prefix = rawHashes(line, at)
	isRaw := prefix == "r" && (quoting.open == openRust || quoting.open == openCLike) ||
		(prefix == "br" || prefix == "cr") && quoting.open == openRust
	return hashes, isRaw && (hashes == 0 || line[at] == '"')
}

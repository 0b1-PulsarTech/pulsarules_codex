package historynarration

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type scanner struct {
	syntax   core.CommentSyntax
	rules    rules
	closer   string
	rawOpen  bool
	segments []segment
}

func scanSegments(src []byte, syntax core.CommentSyntax, ext string) []segment {
	sc := &scanner{syntax: syntax, rules: rulesFor(syntax, ext)}
	for index, line := range strings.Split(string(src), "\n") {
		sc.scanLine(index+1, line)
	}
	return sc.segments
}

func (sc *scanner) add(line int, text string) {
	sc.segments = append(sc.segments, segment{line: line, text: text})
}

func (sc *scanner) scanLine(num int, line string) {
	for pos := 0; pos < len(line); {
		next, done := sc.step(num, line, pos)
		if done {
			return
		}
		pos = next
	}
}

func (sc *scanner) step(num int, line string, pos int) (next int, done bool) {
	if sc.closer != "" {
		end := strings.Index(line[pos:], sc.closer)
		if end < 0 {
			sc.add(num, line[pos:])
			return len(line), true
		}
		sc.add(num, line[pos:pos+end])
		next = pos + end + len(sc.closer)
		sc.closer = ""
		return next, false
	}
	if sc.rawOpen {
		return sc.rawStep(line, pos)
	}
	if skip := sc.rules.quoteLen(line, pos); skip > 0 {
		return pos + skip, false
	}
	if line[pos] == '`' && sc.rules.backtick {
		sc.rawOpen = true
		return pos + 1, false
	}
	return sc.openComment(num, line, pos)
}

func (sc *scanner) rawStep(line string, pos int) (next int, done bool) {
	for ; pos < len(line); pos++ {
		switch {
		case line[pos] == '\\' && sc.rules.backslash:
			pos++
		case line[pos] == '`':
			sc.rawOpen = false
			return pos + 1, false
		}
	}
	return len(line), true
}

func (sc *scanner) openComment(num int, line string, pos int) (next int, done bool) {
	rest := line[pos:]
	for _, block := range sc.syntax.Blocks {
		if !strings.HasPrefix(rest, block.Open) {
			continue
		}
		sc.closer = block.Close
		return pos + len(block.Open), false
	}
	for _, prefix := range sc.syntax.Line {
		if strings.HasPrefix(rest, prefix) && lineStart(line, pos, prefix) {
			sc.add(num, rest[len(prefix):])
			return len(line), true
		}
	}
	return pos + 1, false
}

func lineStart(line string, pos int, prefix string) bool {
	if pos == 0 {
		return true
	}
	before := line[pos-1]
	switch prefix {
	case "#":
		return before == ' ' || before == '\t'
	case "//":
		return before != ':'
	}
	return true
}

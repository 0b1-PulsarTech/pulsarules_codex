package narration

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type scanner struct {
	syntax   pass.CommentSyntax
	closer   string
	segments []segment
}

func scanSegments(src []byte, syntax pass.CommentSyntax) []segment {
	sc := &scanner{syntax: syntax}
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
	return sc.openComment(num, line, pos)
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

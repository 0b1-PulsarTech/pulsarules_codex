package narration

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

const (
	regionMarkup = iota
	regionScript
	regionStyle
)

type scanner struct {
	syntax   lexer.CommentSyntax
	rules    rules
	closer   string
	rawOpen  bool
	region   int
	segments []segment
}

func scanSegments(src []byte, lang language.Language) []segment {
	sc := &scanner{syntax: lang.Comments, rules: rulesFor(lang)}
	sc.region = regionScript
	if sc.rules.sfc {
		sc.region = regionMarkup
	}
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
	if sc.region == regionMarkup {
		return sc.markupStep(line, pos), false
	}
	if tail := sc.regionEnd(line[pos:]); sc.rules.sfc && tail > 0 {
		sc.region = regionMarkup
		return pos + tail, false
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
		if block.Open == "<!--" && sc.rules.sfc || !strings.HasPrefix(rest, block.Open) {
			continue
		}
		sc.closer = block.Close
		return pos + len(block.Open), false
	}
	for _, prefix := range sc.syntax.Line {
		if sc.region != regionStyle && strings.HasPrefix(rest, prefix) &&
			lineStart(line, pos, prefix) {
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

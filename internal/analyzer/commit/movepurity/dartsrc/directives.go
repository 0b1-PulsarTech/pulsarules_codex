package dartsrc

import (
	"errors"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

var ErrMalformed = errors.New("malformed dart directive")

type (
	Directive struct {
		Kind  string
		URIs  []string
		Shape string
	}
	File struct {
		Directives []Directive
		Body       string
	}
	scanner struct {
		src  string
		lits language.Literals
		pos  int
	}
)

func Parse(src string) (File, error) {
	lits := language.ScanLiterals(".dart", src)
	if lits.Incomplete() {
		return File{}, ErrMalformed
	}
	scan := scanner{src: src, lits: lits}
	var file File
	var body strings.Builder
	if strings.HasPrefix(src, "#!") {
		line, _, _ := strings.Cut(src, "\n")
		scan.pos = len(line)
	}
	body.WriteString(src[:scan.pos])
	for {
		triviaStart := scan.pos
		scan.pos = scan.trivia(scan.pos)
		body.WriteString(src[triviaStart:scan.pos])
		kind := scan.kind()
		if kind == "" {
			break
		}
		end, err := scan.directiveEnd()
		if err != nil {
			return File{}, err
		}
		parsed, err := scan.directive(kind, end)
		if err != nil {
			return File{}, err
		}
		file.Directives = append(file.Directives, parsed)
		scan.pos = end
	}
	body.WriteString(src[scan.pos:])
	file.Body = body.String()
	return file, nil
}

func (scan *scanner) trivia(from int) int {
	for from < len(scan.src) {
		isSpace := unicode.IsSpace(rune(scan.src[from]))
		if !isSpace && !scan.lits.InComment(from) {
			return from
		}
		from++
	}
	return from
}

func (scan *scanner) wordAt(from int) (word string, end int) {
	end = from
	for end < len(scan.src) && isWordByte(scan.src[end]) {
		end++
	}
	return scan.src[from:end], end
}

func isWordByte(char byte) bool { return isIdentStart(char) || char >= '0' && char <= '9' }

func (scan *scanner) kind() string {
	word, end := scan.wordAt(scan.pos)
	next := scan.trivia(end)
	switch word {
	case "import", "export":
		if scan.isLiteralStart(next) {
			return word
		}
	case "library":
		if scan.isIdentAt(next) || next < len(scan.src) && scan.src[next] == ';' {
			return word
		}
	case "part":
		return scan.partKind(next)
	}
	return ""
}

func (scan *scanner) partKind(next int) string {
	if scan.isLiteralStart(next) {
		return "part"
	}
	ofWord, after := scan.wordAt(next)
	target := scan.trivia(after)
	if ofWord == "of" && (scan.isLiteralStart(target) || scan.isIdentAt(target)) {
		return "part of"
	}
	return ""
}

func (scan *scanner) isIdentAt(at int) bool {
	return at < len(scan.src) && isIdentStart(scan.src[at])
}

func (scan *scanner) isLiteralStart(at int) bool {
	if at < len(scan.src) && scan.src[at] == 'r' {
		at++
	}
	return at < len(scan.src) && (scan.src[at] == '\'' || scan.src[at] == '"')
}

func isIdentStart(char byte) bool {
	return char == '_' || char == '$' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
}

func (scan *scanner) directiveEnd() (int, error) {
	for offset := scan.pos; offset < len(scan.src); offset++ {
		isCode := !scan.lits.Contains(offset) && !scan.lits.InComment(offset)
		if isCode && scan.src[offset] == ';' {
			return offset + 1, nil
		}
	}
	return 0, ErrMalformed
}

func (scan *scanner) directive(kind string, end int) (Directive, error) {
	result := Directive{Kind: kind}
	var shape strings.Builder
	uriEnd := scan.pos
	for offset := scan.pos; offset < end; {
		if !scan.lits.Contains(offset) || scan.lits.InComment(offset) {
			shape.WriteByte(scan.src[offset])
			offset++
			continue
		}
		runEnd := offset
		for runEnd < end && scan.lits.Contains(runEnd) {
			runEnd++
		}
		run := scan.src[offset:runEnd]
		if isConditionValue(scan.src[scan.pos:offset]) {
			shape.WriteString(run)
		} else {
			uri, err := literalValue(run)
			if err != nil || !scan.isPlaced(result, uriEnd, offset) {
				return Directive{}, ErrMalformed
			}
			result.URIs = append(result.URIs, uri)
			shape.WriteString("\x00")
			uriEnd = runEnd
		}
		offset = runEnd
	}
	result.Shape = shape.String()
	return result, nil
}

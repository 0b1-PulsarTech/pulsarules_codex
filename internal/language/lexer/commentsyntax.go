package lexer

import (
	"bytes"
	"strings"
)

type BlockDelim struct {
	Open   string
	Close  string
	Nested bool
}

type CommentSyntax struct {
	Line   []string
	Blocks []BlockDelim
}

func AnyCommentSyntax() CommentSyntax {
	return CommentSyntax{
		Line: []string{"//", "#", "--"},
		Blocks: []BlockDelim{
			{Open: "/*", Close: "*/"},
			{Open: "<!--", Close: "-->"},
			{Open: "--[[", Close: "]]"},
		},
	}
}

func TrimBOM(src []byte) []byte {
	return bytes.TrimPrefix(src, []byte("\xef\xbb\xbf"))
}

func BlockEnd(lines []string, delim BlockDelim) (int, bool) {
	for idx, line := range lines {
		rest := line
		if idx == 0 {
			rest = strings.TrimSpace(line)[len(delim.Open):]
		}
		if strings.Contains(rest, delim.Close) {
			return idx, true
		}
	}
	return len(lines) - 1, false
}

func CommentBody(text string, syntax CommentSyntax) string {
	if body, ok := openedBody(text, syntax); ok {
		return body
	}
	body := strings.TrimSpace(text)
	for _, block := range syntax.Blocks {
		body = strings.TrimSuffix(body, block.Close)
	}
	return trimStars(body)
}

func openedBody(text string, syntax CommentSyntax) (string, bool) {
	body := strings.TrimSpace(text)
	for _, block := range syntax.Blocks {
		if rest, ok := strings.CutPrefix(body, block.Open); ok {
			return trimStars(strings.TrimSuffix(strings.TrimSpace(rest), block.Close)), true
		}
	}
	for _, prefix := range syntax.Line {
		if rest, ok := strings.CutPrefix(body, prefix); ok {
			return trimStars(rest), true
		}
	}
	return "", false
}

func trimStars(text string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(text), "*"))
}

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
	// LineOpens keeps the "#" of a word or the "//" of a URL from opening a comment.
	LineOpens func(src string, at int) bool
	// keepStars keeps a leading "*" as text, as go/ast does; unset, it is JSDoc gutter.
	keepStars bool
}

func SlashSyntax() CommentSyntax {
	return CommentSyntax{
		Line:   []string{"//"},
		Blocks: []BlockDelim{{Open: "/*", Close: "*/"}},
	}
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
	return tidy(trimClosers(strings.TrimSpace(text), syntax), syntax)
}

func openedBody(text string, syntax CommentSyntax) (string, bool) {
	body := strings.TrimSpace(text)
	for _, block := range syntax.Blocks {
		if rest, ok := strings.CutPrefix(body, block.Open); ok {
			return tidy(openedRest(rest, block, syntax), syntax), true
		}
	}
	for _, prefix := range syntax.Line {
		if rest, ok := strings.CutPrefix(body, prefix); ok {
			return tidy(trimClosers(strings.TrimSpace(rest), syntax), syntax), true
		}
	}
	return "", false
}

func openedRest(rest string, block BlockDelim, syntax CommentSyntax) string {
	body := strings.TrimSuffix(strings.TrimSpace(rest), block.Close)
	if syntax.keepStars {
		return body
	}
	return strings.TrimPrefix(body, "*")
}

func trimClosers(body string, syntax CommentSyntax) string {
	for _, block := range syntax.Blocks {
		body = strings.TrimSuffix(body, block.Close)
	}
	return body
}

func tidy(text string, syntax CommentSyntax) string {
	if syntax.keepStars {
		return strings.TrimSpace(text)
	}
	return trimStars(text)
}

func trimStars(text string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(text), "*"))
}

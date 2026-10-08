package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

const ellipsis = "..."

type edge int

const headEdge, tailEdge edge = 0, 1

type collector struct {
	path   string
	stored []string
	bytes  int
	total  int
	walked int
}

func (col *collector) add(line int, at *fieldPath, format string, args ...any) {
	col.total++
	if len(col.stored) >= MaxReported || col.bytes >= MaxReportedBytes {
		return
	}
	text := col.path + ": "
	if line > 0 {
		text += "line " + strconv.Itoa(line) + ": "
	}
	if field := at.String(); field != "" {
		text += clipPath(field) + ": "
	}
	text += fmt.Sprintf(format, args...)
	col.bytes += len(text)
	col.stored = append(col.stored, text)
}

func (col *collector) err() error {
	texts := col.stored
	switch extra := col.total - len(col.stored); {
	case extra == 1:
		texts = append(texts, "... and 1 more problem")
	case extra > 1:
		texts = append(texts, fmt.Sprintf("... and %d more problems", extra))
	}
	if len(texts) == 0 {
		return nil
	}
	return errors.New(strings.Join(texts, "; "))
}

func ClipHead(text string, limit int) string {
	return clip(text, limit, len(ellipsis), headEdge)
}

func Fit(text string, limit int) (string, bool) {
	return fit(text, limit, headEdge, escapedWidth)
}

func clip(text string, limit, reserve int, side edge) string {
	kept, whole := fit(text, limit, side, escapedWidth)
	if !whole {
		kept, _ = fit(text, max(limit-reserve, 0), side, escapedWidth)
	}
	switch {
	case whole:
		return escape.Line(kept)
	case side == tailEdge:
		return ellipsis + escape.Line(kept)
	}
	return escape.Line(trimBrokenEscape(kept)) + ellipsis
}

func trimBrokenEscape(kept string) string {
	// A cut backslash or "\x" before the ellipsis would read as a broken escape.
	for {
		cut := strings.LastIndexByte(kept, '\\')
		if cut < 0 || !partialEscape(kept[cut+1:]) {
			return kept
		}
		kept = kept[:cut]
	}
}

func partialEscape(rest string) bool {
	if rest == "" {
		return true
	}
	for _, shape := range []string{"xhh", "uhhhh", "Uhhhhhhhh"} {
		if rest[0] == shape[0] {
			return len(rest) < len(shape) && strings.Trim(rest[1:], "0123456789abcdef") == ""
		}
	}
	return false
}

func fit(text string, limit int, side edge, width func(string) int) (string, bool) {
	lo, hi, used := 0, len(text), 0
	for lo < hi {
		unit := edgeUnit(text[lo:hi], side)
		if used += width(unit); used > limit {
			break
		}
		if side == tailEdge {
			hi -= len(unit)
		} else {
			lo += len(unit)
		}
	}
	switch {
	case lo >= hi:
		return text, true
	case side == tailEdge:
		return text[hi:], false
	}
	return text[:lo], false
}

func edgeUnit(text string, side edge) string {
	if side == tailEdge {
		_, size := utf8.DecodeLastRuneInString(text)
		return text[len(text)-size:]
	}
	_, size := utf8.DecodeRuneInString(text)
	return text[:size]
}

func escapedWidth(unit string) int { return len(escape.Line(unit)) }

func clipParseText(text string) string {
	prefix := linePrefix(text)
	return prefix + clipValue(strings.TrimPrefix(text, prefix))
}

func leftover(path string, root *yaml.Node, err error) error {
	text := plainText(err)
	if strings.Contains(text, "excessive aliasing") {
		line := firstAliasLine(root)
		return fmt.Errorf("%s: line %d: aliases expand into too many nodes", path, line)
	}
	for _, leaked := range []string{"cannot", "!!", "yaml:", "config.", "[]", "map["} {
		if strings.Contains(text, leaked) {
			return fmt.Errorf("%s: %sinvalid value", path, linePrefix(text))
		}
	}
	return fmt.Errorf("%s: %s", path, text)
}

func linePrefix(text string) string {
	rest, found := strings.CutPrefix(text, "line ")
	if !found {
		return ""
	}
	digits, _, _ := strings.Cut(rest, ":")
	if _, err := strconv.Atoi(digits); err != nil {
		return ""
	}
	return "line " + digits + ": "
}

func plainText(err error) string {
	if typeErr, ok := errors.AsType[*yaml.TypeError](err); ok {
		return strings.Join(typeErr.Errors, "; ")
	}
	text := strings.TrimPrefix(err.Error(), "yaml: ")
	lines := strings.Split(text, "\n")
	for idx, line := range lines {
		lines[idx] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "; ")
}

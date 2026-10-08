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

func clip(text string, limit int) string {
	kept, whole := fit(text, limit)
	if whole {
		return escape.Line(kept)
	}
	return ellipsis + escape.Line(kept)
}

func fit(text string, limit int) (string, bool) {
	hi, used := len(text), 0
	for hi > 0 {
		_, size := utf8.DecodeLastRuneInString(text[:hi])
		if used += escapedWidth(text[hi-size : hi]); used > limit {
			break
		}
		hi -= size
	}
	if hi == 0 {
		return text, true
	}
	return text[hi:], false
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

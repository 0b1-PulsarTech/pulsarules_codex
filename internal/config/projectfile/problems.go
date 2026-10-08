package projectfile

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Problems struct {
	path   string
	stored []string
	bytes  int
	total  int
	walked int
}

func (col *Problems) add(line int, at *Field, format string, args ...any) {
	col.total++
	if len(col.stored) >= maxReported || col.bytes >= maxReportedBytes {
		return
	}
	text := col.path + ": "
	if line > 0 {
		text += "line " + strconv.Itoa(line) + ": "
	}
	if field := at.String(); field != "" {
		text += field + ": "
	}
	text += fmt.Sprintf(format, args...)
	col.bytes += len(text)
	col.stored = append(col.stored, text)
}

func (col *Problems) err() error {
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

func clipParseText(text string) string {
	prefix := linePrefix(text)
	return prefix + clipValue(strings.TrimPrefix(text, prefix))
}

func leftover(path string, err error) error {
	text := plainText(err)
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

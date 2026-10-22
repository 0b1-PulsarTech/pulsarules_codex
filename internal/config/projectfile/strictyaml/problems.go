package strictyaml

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const ellipsis = "..."

type Problems struct {
	path   string
	stored []string
	bytes  int
	total  int
	walked int
}

func NewProblems(path string) *Problems { return &Problems{path: path} }

func (problems *Problems) Report(at *Field, format string, args ...any) {
	if !at.covered(&problems.walked) {
		problems.add(0, at, format, args...)
	}
}

func (problems *Problems) add(line int, at *Field, format string, args ...any) {
	problems.total++
	if len(problems.stored) >= maxReported || problems.bytes >= maxReportedBytes {
		return
	}
	text := problems.path + ": "
	if line > 0 {
		text += "line " + strconv.Itoa(line) + ": "
	}
	if field := at.String(); field != "" {
		text += clipPath(field) + ": "
	}
	text += fmt.Sprintf(format, args...)
	problems.bytes += len(text)
	problems.stored = append(problems.stored, text)
}

func (problems *Problems) Err() error {
	texts := problems.stored
	switch extra := problems.total - len(problems.stored); {
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
	return prefix + ClipValue(strings.TrimPrefix(text, prefix))
}

func leftover[Target any](path string, root *yaml.Node, err error) error {
	text := plainText(err)
	if strings.Contains(text, "excessive aliasing") {
		line := firstAliasLine(root)
		return fmt.Errorf("%s: line %d: aliases expand into too many nodes", path, line)
	}
	needles := []string{"cannot", "!!", "yaml:", decodeTypePrefix[Target](), "[]", "map["}
	for _, leaked := range needles {
		if strings.Contains(text, leaked) {
			return fmt.Errorf("%s: %sinvalid value", path, linePrefix(text))
		}
	}
	return fmt.Errorf("%s: %s", path, text)
}

func decodeTypePrefix[Target any]() string {
	// The package name the decoder would print before the target type, such as "projectfile.".
	pkg, _, _ := strings.Cut(reflect.TypeFor[Target]().String(), ".")
	return pkg + "."
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

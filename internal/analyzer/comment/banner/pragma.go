package banner

import (
	"slices"
	"strings"
)

func isToolPragma(line string) bool {
	body, ok := strings.CutPrefix(strings.TrimSpace(line), "//")
	if !ok {
		return false
	}
	fields := strings.Fields(body)
	return len(fields) > 0 && pragmaArgsMatch(fields[0], fields[1:])
}

func pragmaArgsMatch(tag string, args []string) bool {
	switch tag {
	case "@flow":
		return len(args) == 0 || oneOf(args, "strict", "strict-local")
	case "@noflow":
		return len(args) == 0
	case "@jsx", "@jsxFrag", "@jsxImportSource":
		return len(args) == 1
	case "@jsxRuntime":
		return oneOf(args, "classic", "automatic")
	case "@refresh":
		return oneOf(args, "reset")
	}
	return false
}

func oneOf(args []string, values ...string) bool {
	return len(args) == 1 && slices.Contains(values, args[0])
}

package budget

import (
	"fmt"
	"strings"
)

func checkGauges(text string) string {
	// Unguarded, the worst 32000-bracket shape peaks at 133 MiB.
	if openers := countOpeners(text); openers > MaxOpeners {
		return fmt.Sprintf("skipped: %d brackets is over the %d parse limit", openers, MaxOpeners)
	}
	// Operator bytes drive recursion that brackets do not show: 64Ki of them peak at 106 MiB.
	// Real bundles stay well under both bounds.
	if operators := countOperators(text); operators > MaxOperators {
		return fmt.Sprintf(
			"skipped: %d operators is over the %d parse limit",
			operators,
			MaxOperators,
		)
	}
	return ""
}

func countOpeners(text string) int {
	return strings.Count(text, "(") + strings.Count(text, "[") +
		strings.Count(text, "{") + strings.Count(text, "<")
}

func countOperators(text string) int {
	count := 0
	for idx := range len(text) {
		if isOperator(text[idx]) {
			count++
		}
	}
	return count
}

func isOperator(char byte) bool {
	return strings.IndexByte("!~=?:*+-&|^%>,", char) >= 0
}

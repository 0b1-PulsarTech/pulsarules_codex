package tsguard

import "fmt"

const (
	MaxFileBytes   = 1 << 20
	MaxNesting     = 500
	MaxSpeculative = 10
	MaxOpeners     = 1 << 15
	MaxOperators   = 1 << 16
)

func CheckSize(text string) string {
	if len(text) > MaxFileBytes {
		return fmt.Sprintf("skipped: %d bytes is over the %d byte limit", len(text), MaxFileBytes)
	}
	return ""
}

func Check(text string) string {
	if reason := CheckSize(text); reason != "" {
		return reason
	}
	// A cheap early-out only.
	if reason := checkGauges(text); reason != "" {
		return reason
	}
	scan := newScanner(text)
	deepest := scan.run()
	if deepest > MaxNesting {
		return fmt.Sprintf(
			"skipped: nesting depth %d is over the %d parse limit",
			deepest,
			MaxNesting,
		)
	}
	if scan.widest > MaxSpeculative {
		return fmt.Sprintf(
			"skipped: %d nested arrow-like parentheses is over the %d parse limit",
			scan.widest,
			MaxSpeculative,
		)
	}
	return ""
}

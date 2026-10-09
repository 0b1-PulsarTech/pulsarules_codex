package filesize

import (
	"math"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

const (
	ParamMaxLines   = "max_lines"
	ParamErrorLines = "error_lines"
)

type lineLimits struct {
	production     int
	test           int
	hardProduction int
	hardTest       int
}

func newLineLimits(maxLines, errorLines int) lineLimits {
	return lineLimits{
		production:     maxLines,
		test:           scaleForTests(maxLines),
		hardProduction: errorLines,
		hardTest:       scaleForTests(errorLines),
	}
}

func scaleForTests(limit int) int {
	const testLineMultiplier = 2.6
	// A table-driven test carries a case table with no counterpart in the code under
	// test; one shared number would push the table out of the file it documents.
	return int(math.Round(float64(limit) * testLineMultiplier))
}

func (limits lineLimits) forFile(fc core.FileChange) int {
	if fc.IsTest {
		return limits.test
	}
	return limits.production
}

func (limits lineLimits) hardFor(fc core.FileChange) int {
	if fc.IsTest {
		return limits.hardTest
	}
	return limits.hardProduction
}

func errorLines(params core.ParamSet, extension string) int {
	return params.NonNegativeInt(ParamErrorLines, defaultErrorLines(extension))
}

func defaultErrorLines(extension string) int {
	const svelteErrorLines = 290
	if strings.EqualFold(extension, ".svelte") {
		return svelteErrorLines
	}
	return 0
}

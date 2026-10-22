package projectfile

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
)

func (val *validator) checkGlobs(at *strictyaml.Field, globs []string) []string {
	passed := make([]string, 0, len(globs))
	for idx, pattern := range globs {
		if fault := globFault(pattern); fault != "" {
			val.fail(at.Item(idx), "%s", fault)
			continue
		}
		if _, err := glob.Compile(pattern); err != nil {
			reason := strings.TrimPrefix(err.Error(), fmt.Sprintf("compile glob %q: ", pattern))
			val.fail(
				at.Item(idx), "compile glob %q: %s",
				strictyaml.ClipValue(pattern), strictyaml.ClipValue(reason),
			)
			continue
		}
		passed = append(passed, pattern)
	}
	return passed
}

func globFault(pattern string) string {
	const (
		maxGlobBytes = 256
		maxGlobStars = 4
	)
	switch {
	case len(pattern) > maxGlobBytes:
		return fmt.Sprintf("glob longer than %d bytes", maxGlobBytes)
	case glob.HasBrace(pattern):
		return "brace alternatives are not supported; list each glob"
	case glob.StarCount(pattern) > maxGlobStars:
		return fmt.Sprintf(
			"glob has more than %d * (** and lone * segments are free)", maxGlobStars,
		)
	}
	return ""
}

func (val *validator) checkGlobBudget(globs []string) bool {
	const (
		maxTotalGlobs     = 256
		maxTotalGlobBytes = 4096
		maxWildcardWork   = 256
		maxStarTextBytes  = 2048
	)
	size, wildcards, tails := 0, 0, 0
	for _, pattern := range globs {
		size += len(pattern)
		wildcards += glob.WildcardWork(pattern)
		tails += glob.StarTail(pattern)
	}
	at := val.root.Key("overrides and tests")
	switch {
	case len(globs) > maxTotalGlobs:
		val.fail(at, "more than %d globs in total", maxTotalGlobs)
	case size > maxTotalGlobBytes:
		val.fail(at, "globs exceed %d bytes in total", maxTotalGlobBytes)
	case wildcards > maxWildcardWork:
		val.fail(at, "wildcard classes and ? exceed %d in total", maxWildcardWork)
	case tails > maxStarTextBytes:
		val.fail(at, "text after the first * of each segment exceeds %d bytes in total",
			maxStarTextBytes)
	default:
		return true
	}
	return false
}

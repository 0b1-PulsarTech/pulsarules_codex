package command

import (
	"maps"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type diagnostic struct {
	code       string
	severity   finding.Severity
	file       string
	line       int
	message    string
	suggestion string
}

type format struct {
	decode         func(output []byte) ([]diagnostic, error)
	reportingExits []int
}

func formats() map[string]format {
	const (
		dartExitInfos = iota + 1
		dartExitWarnings
		dartExitErrors
	)
	return map[string]format{
		"dart-analyze-json": {
			decode:         decodeDartAnalyze,
			reportingExits: []int{dartExitInfos, dartExitWarnings, dartExitErrors},
		},
	}
}

func Formats() []string {
	return slices.Sorted(maps.Keys(formats()))
}

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

type format interface {
	reportingExits() []int
	decode(output []byte) ([]diagnostic, error)
	precheck(root, dir string, listed listing) string
	codegen(found []diagnostic, where scope) (kept []diagnostic, notice string)
}

type dartAnalyzeJSON struct{}

var _ format = dartAnalyzeJSON{}

func (dartAnalyzeJSON) reportingExits() []int {
	const (
		dartExitInfos = iota + 1
		dartExitWarnings
		dartExitErrors
	)
	return []int{dartExitInfos, dartExitWarnings, dartExitErrors}
}

func (dartAnalyzeJSON) decode(output []byte) ([]diagnostic, error) {
	return decodeDartAnalyze(output)
}

func (dartAnalyzeJSON) precheck(root, dir string, listed listing) string {
	return dartPrecheck(root, dir, listed)
}

func (dartAnalyzeJSON) codegen(found []diagnostic, where scope) ([]diagnostic, string) {
	return dartCodegen(found, where)
}

func formats() map[string]format {
	return map[string]format{"dart-analyze-json": dartAnalyzeJSON{}}
}

func Formats() []string {
	return slices.Sorted(maps.Keys(formats()))
}

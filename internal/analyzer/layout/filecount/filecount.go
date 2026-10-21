package filecount

import (
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

const ID = "package-file-count"

const (
	paramMaxFiles = "max_files"
	paramExempt   = "exempt"
)

type Analyzer struct {
	reporter finding.Reporter
}

func NewAnalyzer() *Analyzer {
	return &Analyzer{
		reporter: finding.NewReporter(ID, finding.SeverityWarning, finding.CatAST),
	}
}

func (analyzer *Analyzer) ID() string             { return ID }
func (analyzer *Analyzer) Stage() finding.StageID { return finding.StageAST }

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if ctx.Sources == nil {
		return nil
	}
	params := ctx.Params(ID)
	maxFiles := params.Int(paramMaxFiles, 10)
	exempt := exemptions{patterns: params.Strings(paramExempt, nil)}
	eligible := pass.ExtensionFilter(ctx, ID, []string{".go"})
	reporter := pass.ResolveReporter(analyzer.reporter, ctx)
	anchors := changedAnchors(ctx, eligible)

	var findings []finding.Finding
	if len(ctx.ChangedFiles) > 0 {
		pinned := malformedAnchor(ctx, anchors)
		line := 1
		if pinned == "" {
			line = 0
		}
		for _, problem := range exempt.invalid() {
			findings = append(findings, reporter.At(pinned, line, problem, "fix the exempt param"))
		}
	}
	wanted := make(map[string]bool, len(anchors))
	for _, changed := range anchors {
		wanted[changed.dir] = true
	}
	counts := countByDir(ctx, eligible, wanted)
	for _, changed := range anchors {
		entry := counts[changed.dir]
		if entry.files <= maxFiles || entry.isEntSchema || exempt.matches(changed.dir) {
			continue
		}
		findings = append(findings, reporter.At(
			changed.file,
			1,
			fmt.Sprintf(
				"package %s has %d non-test source files, max %d",
				changed.dir,
				entry.files,
				maxFiles,
			),
			"split the package by responsibility into smaller packages",
		))
	}
	return findings
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "code-placement-go",
		Params: []string{paramMaxFiles, paramExempt, pass.ParamExtensions},
	}
}

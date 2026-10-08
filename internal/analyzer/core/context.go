package core

import (
	"go/ast"
	"iter"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/sources"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/testpolicy"
)

type FileChange struct {
	Path      string
	Extension string
	IsTest    bool
	Generated bool
	Staged    bool
}

type Rename struct {
	OldPath string
	NewPath string
	Score   int
}

type GitCommitEntry struct {
	Subject string
}

type AnalysisContext struct {
	ProjectDir    string
	ChangedFiles  []FileChange
	StagedRenames []Rename
	CommitMsg     string
	GitHistory    []GitCommitEntry
	Config        *AnalysisConfig
	TestPolicy    testpolicy.Policy
	ASTCache      *astcache.Cache
	Syntax        *syntax.Cache
	Sources       sources.Provider
	Skips         sources.SkipSet
	Findings      []Finding
}

func (ctx *AnalysisContext) ChangedGoASTs() iter.Seq2[FileChange, *ast.File] {
	return func(yield func(FileChange, *ast.File) bool) {
		if ctx.ASTCache == nil {
			return
		}
		// _test.go files stay in so the AST analyzers see tests as well.
		for _, fc := range ctx.ChangedFiles {
			if fc.Extension != ".go" {
				continue
			}
			file := ctx.ASTCache.Get(fc.Path)
			if file == nil {
				continue
			}
			if !yield(fc, file) {
				return
			}
		}
	}
}

func RunPerGoFile(
	ctx *AnalysisContext,
	check func(fc FileChange, file *ast.File) []Finding,
) []Finding {
	if ctx.ASTCache == nil {
		return nil
	}
	var findings []Finding
	for fc, file := range ctx.ChangedGoASTs() {
		findings = append(findings, check(fc, file)...)
	}
	return findings
}

type AnalysisConfig struct {
	Analyzers        map[string]AnalyzerConfig
	IncludeGenerated bool
	TestGlobs        []string
	Overrides        []Override
	Delegated        []DelegatedCommand
}

type Override struct {
	Files     []string
	Analyzers map[string]OverrideConfig
}

type OverrideConfig struct {
	Enabled  *bool
	Severity string
}

type DelegatedCommand struct {
	ID, Dir, Format, Rule string
	Run                   []string
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

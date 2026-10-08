package pass

import (
	"go/ast"
	"iter"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/testfile"
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

type Context struct {
	ProjectDir    string
	ChangedFiles  []FileChange
	StagedRenames []Rename
	CommitMsg     string
	GitHistory    []GitCommitEntry
	Config        *AnalysisConfig
	TestPolicy    testfile.Classifier
	ASTCache      *astcache.Cache
	Syntax        *syntax.Cache
	Sources       source.Provider
	Skips         source.SkipSet
	Findings      []finding.Finding
}

func (ctx *Context) ChangedGoASTs() iter.Seq2[FileChange, *ast.File] {
	return func(yield func(FileChange, *ast.File) bool) {
		if ctx.ASTCache == nil {
			return
		}
		// _test.go files stay in so the AST analyzers see tests as well.
		for _, fc := range ctx.ChangedFiles {
			if fc.Extension != ".go" || ctx.Skips.SkipsPath(fc.Path) {
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
	ctx *Context,
	check func(fc FileChange, file *ast.File) []finding.Finding,
) []finding.Finding {
	if ctx.ASTCache == nil {
		return nil
	}
	var findings []finding.Finding
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
}

type Override struct {
	Files     []string
	Analyzers map[string]OverrideConfig
}

type OverrideConfig struct {
	Enabled  *bool
	Severity string
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

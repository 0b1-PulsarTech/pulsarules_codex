package pass

import (
	"go/ast"
	"iter"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
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
	ASTCache      *astcache.Cache
	Sources       SourceProvider
	Findings      []finding.Finding
}

func (ctx *Context) ChangedGoASTs() iter.Seq2[FileChange, *ast.File] {
	return func(yield func(FileChange, *ast.File) bool) {
		if ctx.ASTCache == nil {
			return
		}
		// _test.go files stay in so the AST analyzers see tests as well.
		for _, fc := range ctx.ChangedFiles {
			if fc.Extension != ".go" {
				continue
			}
			f := ctx.ASTCache.Get(fc.Path)
			if f == nil {
				continue
			}
			if !yield(fc, f) {
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
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

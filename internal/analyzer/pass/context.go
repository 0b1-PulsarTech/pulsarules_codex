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
	Staged    bool
}

type GitCommitEntry struct {
	Subject string
}

type Context struct {
	ProjectDir   string
	ChangedFiles []FileChange
	CommitMsg    string
	GitHistory   []GitCommitEntry
	Config       *AnalysisConfig
	ASTCache     *astcache.Cache
	Sources      SourceProvider
	Findings     []finding.Finding
}

func (ctx *Context) ChangedGoASTs() iter.Seq2[FileChange, *ast.File] {
	return func(yield func(FileChange, *ast.File) bool) {
		if ctx.ASTCache == nil {
			return
		}
		for _, fc := range ctx.ChangedFiles {
			if fc.Extension != ".go" || fc.IsTest {
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

type AnalysisConfig struct {
	Analyzers map[string]AnalyzerConfig
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

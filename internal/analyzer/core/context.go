package core

import (
	"go/ast"
	"iter"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/astcache"
)

// FileChange describes a single file modification in the working tree.
type FileChange struct {
	// Path is the repo-relative file path.
	Path string
	// Extension is the lowercase file extension (e.g. ".go").
	Extension string
	// IsTest is true for Go test files (ending in _test.go).
	IsTest bool
	// Generated is true when the file carries the Go generated-code marker.
	// Findings in such a file are suppressed by default: the next codegen run
	// would undo any edit made to satisfy them.
	Generated bool
	// Staged is true when the change is present in the index, used by the
	// move-purity analyzer to tell a staged edit from a mere worktree one.
	Staged bool
}

// Rename is one staged rename git's own similarity detection paired: a
// staged deletion and a staged addition it judged to be the same file moved.
type Rename struct {
	// OldPath is the staged-deleted path.
	OldPath string
	// NewPath is the staged-added path judged to be its replacement.
	NewPath string
	// Score is git's rename-similarity score, from 0 to 100.
	Score int
}

// GitCommitEntry is a single recent commit's subject line, used by the
// emoji-variance check to compare against the ±3 window.
type GitCommitEntry struct {
	// Subject is the raw commit subject line.
	Subject string
}

// AnalysisContext is the shared state every analyzer receives. It is built in
// the StageContext stage and enriched by each subsequent stage. Analyzers read
// from it but must not mutate it (the pipeline owns it).
type AnalysisContext struct {
	// ProjectDir is the resolved project root (may be a worktree root).
	ProjectDir string
	// ChangedFiles is the set of files modified in the working tree.
	ChangedFiles []FileChange
	// StagedRenames is the set of staged renames git's own similarity
	// detection paired, used by the move-purity analyzer. Populated whenever
	// a repository is available, regardless of scope.
	StagedRenames []Rename
	// CommitMsg is the parsed commit message, or empty if not a commit event.
	CommitMsg string
	// GitHistory is the list of recent commit subjects, oldest first.
	GitHistory []GitCommitEntry
	// Config holds the governance configuration (analyzer enable/disable,
	// thresholds, emoji mappings).
	Config *AnalysisConfig
	// ASTCache is a shared per-invocation cache of parsed Go ASTs. It is
	// populated before StageAST runs and may be nil in earlier stages.
	ASTCache *astcache.Cache
	// Sources provides file contents without analyzers touching the filesystem
	// directly. Built by the pipeline in buildContext.
	Sources SourceProvider
	// Findings accumulates findings across stages. Analyzers append to this.
	Findings []Finding
}

// ChangedGoASTs yields each changed Go file already parsed in the AST
// cache; nothing yields if no cache was built.
//
// why: it used to skip fc.IsTest, blinding every AST analyzer to _test.go
// - including time-discipline, whose point is catching sleeps/clocks in tests.
func (ctx *AnalysisContext) ChangedGoASTs() iter.Seq2[FileChange, *ast.File] {
	return func(yield func(FileChange, *ast.File) bool) {
		if ctx.ASTCache == nil {
			return
		}
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

// RunPerGoFile runs check against every changed Go file the AST cache
// holds (see ChangedGoASTs), collecting findings in file order.
//
// why: all seven Go-AST analyzers duplicated this shape, so the AST-cache
// test-file bug had to be fixed in seven files instead of one.
func RunPerGoFile(
	ctx *AnalysisContext,
	check func(fc FileChange, f *ast.File) []Finding,
) []Finding {
	if ctx.ASTCache == nil {
		return nil
	}
	var findings []Finding
	for fc, f := range ctx.ChangedGoASTs() {
		findings = append(findings, check(fc, f)...)
	}
	return findings
}

// AnalysisConfig is the runtime config holder consumed by analyzers.
type AnalysisConfig struct {
	// Analyzers maps analyzer ID to its runtime config.
	Analyzers map[string]AnalyzerConfig
	// IncludeGenerated keeps findings that fall in generated files instead of
	// suppressing them, for a caller that wants the raw picture.
	IncludeGenerated bool
}

// AnalyzerConfig holds the per-analyzer enable/disable state and parameters.
type AnalyzerConfig struct {
	// Enabled controls whether the analyzer runs.
	Enabled bool
	// Params is an arbitrary key-value map for analyzer-specific settings
	// (e.g. max_file_lines=180).
	Params map[string]any
}

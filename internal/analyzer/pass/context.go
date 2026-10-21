package pass

import (
	"go/ast"
	"iter"
	"sync"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax/gosyntax/astcache"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/glob"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/gomod"
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

type Module struct {
	Path string
	Err  error
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

	goPackages     *gomod.Index
	goPackagesOnce sync.Once
	module         Module
	moduleOnce     sync.Once
}

func (ctx *Context) GoPackages() *gomod.Index {
	ctx.goPackagesOnce.Do(func() { ctx.goPackages = gomod.NewIndex(ctx.Sources) })
	return ctx.goPackages
}

func (ctx *Context) GoModule() Module {
	// Lazy: go.mod is read only when arch or the Go syntax provider asks for it.
	ctx.moduleOnce.Do(func() {
		if ctx.ProjectDir == "" {
			return
		}
		path, err := ModulePath(ctx.ProjectDir)
		ctx.module = Module{Path: path, Err: err}
	})
	return ctx.module
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

type AnalysisConfig struct {
	Analyzers        map[string]AnalyzerConfig
	IncludeGenerated bool
	TestGlobs        []string
	Overrides        []Override
	Delegated        []DelegatedCommand
}

type Override struct {
	Files     glob.Set
	Analyzers map[string]OverrideConfig
}

func NewOverride(files []string, analyzers map[string]OverrideConfig) Override {
	return Override{Files: glob.CompileSet(files), Analyzers: analyzers}
}

func (override Override) Covers(path string) bool {
	// A scoped override never claims a finding that has no file.
	if !override.Files.IsOpen() && path == "" {
		return false
	}
	return override.Files.Match(path)
}

type OverrideConfig struct {
	Enabled  *bool
	Severity string
	Params   map[string]any
}

type DelegatedCommand struct {
	ID, Dir, Format, Rule string
	Run                   []string
}

type AnalyzerConfig struct {
	Enabled bool
	Params  map[string]any
}

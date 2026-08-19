package branchname

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

const analyzerID = "branch-name"

type branchReader interface {
	CurrentBranch() (string, error)
}

type Analyzer struct {
	repo     branchReader
	reporter core.Reporter
}

var _ core.Analyzer = (*Analyzer)(nil)

func NewAnalyzer(repo branchReader) *Analyzer {
	return &Analyzer{
		repo:     repo,
		reporter: core.NewReporter(analyzerID, core.SeverityError, core.CatCommit),
	}
}

func (a *Analyzer) ID() string { return analyzerID }

func (a *Analyzer) Stage() core.StageID      { return core.StageStatic }
func (a *Analyzer) Needs() core.Requirements { return core.Requirements{} }

func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.repo == nil {
		return nil
	}
	branch, err := a.repo.CurrentBranch()
	// Silent on an unreadable repo: failing the run because git was unavailable
	// would block work over a naming rule with nothing to say.
	if err != nil || branch == "" || isExemptBranch(branch) {
		return nil
	}
	allowed := allowedTypes(ctx.Params(analyzerID))
	if hasAllowedType(branch, allowed) {
		return nil
	}
	return []core.Finding{a.reporter.New(fmt.Sprintf(
		"branch %q does not start with a type prefix (want <type>/<description>, type one of %s)",
		branch, strings.Join(allowed, "|"),
	))}
}

func isExemptBranch(branch string) bool {
	switch branch {
	// A long-lived trunk names a line of history, not the work landing on it.
	case "main", "master", "develop":
		return true
	}
	return false
}

func allowedTypes(params core.ParamSet) []string {
	gitflowTypes := []string{"feature", "release", "hotfix", "bugfix", "support"}
	// Gitflow prefixes are allowed by default: the check exists to stop a tool-generated
	// name reaching a remote, not to block a project's own gitflow convention.
	allowed := slices.Concat(commitmsg.AllowedTypes(), gitflowTypes)
	for extra := range strings.SplitSeq(params.String("extra_types", ""), ",") {
		if trimmed := strings.TrimSpace(extra); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	slices.Sort(allowed)
	return slices.Compact(allowed)
}

func hasAllowedType(branch string, allowed []string) bool {
	prefix, description, found := strings.Cut(branch, "/")
	if !found || description == "" {
		return false
	}
	if scopeStart := strings.IndexByte(prefix, '('); scopeStart >= 0 {
		if !strings.HasSuffix(prefix, ")") {
			return false
		}
		prefix = prefix[:scopeStart]
	}
	return slices.Contains(allowed, prefix)
}

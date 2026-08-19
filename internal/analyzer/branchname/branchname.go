package branchname

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

const analyzerID = "branch-name"

var reporter = core.NewReporter(analyzerID, core.SeverityError, core.CatCommit)

// exemptBranches carry no type prefix because they are not a change: a
// long-lived trunk names a line of history, not the work landing on it.
var exemptBranches = []string{"main", "master", "develop"}

// gitflowTypes are branch prefixes gitflow uses that are not commit types.
// why: default, not opt-in - the check exists to stop a tool-generated name
// reaching a remote, and a project on gitflow would otherwise have its own
// convention blocked by the rule meant to protect it.
var gitflowTypes = []string{"feature", "release", "hotfix", "bugfix", "support"}

// branchReader is the consumer-declared slice of the repository this analyzer
// needs: only the checked-out branch's name.
type branchReader interface {
	CurrentBranch() (string, error)
}

// Analyzer checks that the checked-out branch names its change the way a commit
// does, so `<type>/<description>` survives from branch to subject line.
type Analyzer struct {
	repo branchReader
}

var _ core.Analyzer = (*Analyzer)(nil)

func NewAnalyzer(repo branchReader) *Analyzer { return &Analyzer{repo: repo} }

func (a *Analyzer) ID() string   { return analyzerID }
func (a *Analyzer) Name() string { return "Branch name" }

func (a *Analyzer) Description() string {
	return "Checks the branch name uses a Conventional Commit type prefix"
}
func (a *Analyzer) Stage() core.StageID      { return core.StageStatic }
func (a *Analyzer) Category() core.Category  { return core.CatCommit }
func (a *Analyzer) Needs() core.Requirements { return core.Requirements{} }

// Analyze reports the checked-out branch when its name carries no recognized
// type prefix. A detached HEAD and an exempt trunk report nothing.
// why: silent on a repo it cannot read - failing the run because git was
// unavailable would block work over a naming rule with nothing to say.
func (a *Analyzer) Analyze(ctx *core.AnalysisContext) []core.Finding {
	if a.repo == nil {
		return nil
	}
	branch, err := a.repo.CurrentBranch()
	if err != nil || branch == "" || slices.Contains(exemptBranches, branch) {
		return nil
	}
	allowed := allowedTypes(ctx.Params(analyzerID))
	if hasAllowedType(branch, allowed) {
		return nil
	}
	return []core.Finding{reporter.New(fmt.Sprintf(
		"branch %q does not start with a type prefix (want <type>/<description>, type one of %s)",
		branch, strings.Join(allowed, "|"),
	))}
}

// allowedTypes returns the prefixes a branch may lead with: the Conventional
// Commit types, the gitflow lines, plus any comma-separated "extra_types" a
// project adds of its own.
func allowedTypes(params core.ParamSet) []string {
	allowed := slices.Concat(commitmsg.AllowedTypes, gitflowTypes)
	for extra := range strings.SplitSeq(params.String("extra_types", ""), ",") {
		if trimmed := strings.TrimSpace(extra); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	slices.Sort(allowed)
	return slices.Compact(allowed)
}

// hasAllowedType reports whether branch reads as <type>/<description>, with an
// optional Conventional Commit (scope) between them.
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

// ValidExtraTypes reports whether spec is a well-formed comma-separated list of
// branch types: each entry lowercase letters, digits or dashes, none empty.
//
// why: the value is baked into a generated shell script, so anything outside
// that alphabet is rejected at the boundary rather than quoted and hoped for.
func ValidExtraTypes(spec string) bool {
	if spec == "" {
		return true
	}
	for entry := range strings.SplitSeq(spec, ",") {
		if entry == "" {
			return false
		}
		for _, char := range entry {
			isAllowed := char == '-' ||
				(char >= 'a' && char <= 'z') ||
				(char >= '0' && char <= '9')
			if !isAllowed {
				return false
			}
		}
	}
	return true
}

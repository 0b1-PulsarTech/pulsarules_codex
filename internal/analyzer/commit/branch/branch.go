package branch

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/commitmsg"
)

const (
	ID              = "branch-name"
	ParamExtraTypes = "extra_types"
)

type branchReader interface {
	CurrentBranch() (string, error)
}

type Analyzer struct {
	repo     branchReader
	reporter finding.Reporter
}

var _ pass.Analyzer = (*Analyzer)(nil)

func NewAnalyzer(repo branchReader) *Analyzer {
	return &Analyzer{
		repo:     repo,
		reporter: finding.NewReporter(ID, finding.SeverityError, finding.CatCommit),
	}
}

func (analyzer *Analyzer) Analyze(ctx *pass.Context) []finding.Finding {
	if analyzer.repo == nil {
		return nil
	}
	current, err := analyzer.repo.CurrentBranch()
	// Silent on an unreadable repo: failing the run because git was unavailable
	// would block work over a naming rule with nothing to say.
	if err != nil || current == "" || isExemptBranch(current) {
		return nil
	}
	allowed := allowedTypes(ctx.Params(ID))
	if hasAllowedType(current, allowed) {
		return nil
	}
	return []finding.Finding{analyzer.reporter.New(fmt.Sprintf(
		"branch %q does not start with a type prefix (want <type>/<description>, type one of %s)",
		current, strings.Join(allowed, "|"),
	))}
}

func isExemptBranch(name string) bool {
	switch name {
	// A long-lived trunk names a line of history, not the work landing on it.
	case "main", "master", "develop":
		return true
	}
	return false
}

func allowedTypes(params pass.ParamSet) []string {
	gitflowTypes := []string{"feature", "release", "hotfix", "bugfix", "support"}
	// Gitflow prefixes are allowed by default: the check exists to stop a tool-generated
	// name reaching a remote, not to block a project's own gitflow convention.
	allowed := slices.Concat(commitmsg.AllowedTypes(), gitflowTypes)
	allowed = append(allowed, params.Strings(ParamExtraTypes, nil)...)
	slices.Sort(allowed)
	return slices.Compact(allowed)
}

func hasAllowedType(name string, allowed []string) bool {
	prefix, description, found := strings.Cut(name, "/")
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

func ValidateExtraTypes(spec string) error {
	if validExtraTypes(spec) {
		return nil
	}
	return fmt.Errorf(
		"%q is not a comma-separated list of lowercase names, e.g. spike,experiment", spec,
	)
}

func validExtraTypes(spec string) bool {
	if spec == "" {
		return true
	}
	// The value is baked into a generated shell script, so anything outside
	// lowercase letters, digits and dashes is rejected at the boundary.
	for entry := range strings.SplitSeq(spec, ",") {
		if entry == "" {
			return false
		}
		if strings.ContainsFunc(entry, func(char rune) bool { return !isExtraTypeChar(char) }) {
			return false
		}
	}
	return true
}

func isExtraTypeChar(char rune) bool {
	return char == '-' || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
}

func Describe() pass.Descriptor {
	return pass.Descriptor{
		ID:     ID,
		Rule:   "commits",
		Params: []string{ParamExtraTypes},
	}
}

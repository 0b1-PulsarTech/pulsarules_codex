package registry

import (
	"maps"
	"strings"
	"testing"
)

func TestRulesMatchTheBaseMap(t *testing.T) {
	t.Parallel()

	const baseRules = `
		file-size source-files
		text-markers source-files
		typographic-markers source-files
		ai-frontmatter source-files
		import-groups imports
		naming naming
		top-of-file comments
		big-comment comments
		history-narration comments
		doc-comment effective-go
		body-header-comment effective-go
		too-many-results effective-go
		ungrouped-interfaces effective-go
		single-letter-name naming
		variadic-index flag-arguments
		package-level-var code-smells
		package-file-count code-placement-go
		test-comment comments
		simplification-path minimalism
		single-use-declaration minimalism
		control-flow code-smells
		shadowing effective-go
		complexity code-smells
		short-decl-reuse named-returns
		named-returns named-returns
		time-discipline concurrency
		context-discipline concurrency
		pprof-labels security
		arch-boundary dependency-rule
		import-cycle module-boundaries
		golangci-lint code-smells
		branch-name commits
		commit-lint commits
		commit-initial commits
		commit-merge commits
		commit-emoji-required commits
		commit-emoji-count commits
		commit-emoji-catalog commits
		commit-emoji-prohibited commits
		commit-emoji-repeat commits
		commit-emoji-soft-repeat commits
		commit-type-required commits
		commit-type-enum commits
		commit-scope-charset commits
		commit-desc-required commits
		commit-desc-length commits
		commit-desc-capitalize commits
		commit-desc-no-period commits
		commit-body-length commits
		commit-body-total-length commits
		commit-no-coauthor commits
		commit-move-purity commits
	`

	fields := strings.Fields(baseRules)
	want := make(map[string]string, len(fields)/2)
	for pos := 0; pos+1 < len(fields); pos += 2 {
		want[fields[pos]] = fields[pos+1]
	}
	got := Rules()
	if !maps.Equal(got, want) {
		for id, rule := range want {
			if got[id] != rule {
				t.Errorf("rule for %q = %q, want %q", id, got[id], rule)
			}
		}
		for id := range got {
			if _, ok := want[id]; !ok {
				t.Errorf("unexpected rule entry %q", id)
			}
		}
	}
}

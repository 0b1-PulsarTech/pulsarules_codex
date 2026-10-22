package hook

import (
	"os"
	"slices"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/loader"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/findings"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

type governor struct {
	index  *knowledge.Index
	loader func() *loader.Layered
}

func (gov governor) check(
	catalog func() (*emoji.Catalog, error),
) func(root string, status vcs.Status) (string, int) {
	return func(root string, status vcs.Status) (string, int) {
		resolved, err := catalog()
		if err != nil {
			printHookError(os.Stderr, "get emoji catalog", err)
			return "", 0
		}
		return gov.changes(root, status, resolved)
	}
}

const governanceHeader = "Governance checks:"

func (gov governor) changes(
	root string,
	status vcs.Status,
	catalog *emoji.Catalog,
) (string, int) {
	repo, err := vcs.Open(root)
	if err != nil {
		return "\n" + governanceHeader + "\n  open repository: " + printable.Line(
			err.Error(),
		) + "\n", 1
	}
	// The empty commit message is deliberate: only the git commit-msg hook validates one.
	cfg, configFindings := gov.loader().ForRoot(repo.Root())
	sess := governance.NewSession(repo, gov.index, catalog, cfg)
	result := sess.Analyze(governance.Request{
		Scope:  governance.ScopeChanged,
		Status: &status,
		Files:  governance.FileSetChanged,
	})
	result.Findings = slices.Concat(configFindings, result.Findings)
	return governanceBlock(result)
}

func governanceBlock(report findings.Report) (block string, count int) {
	text := report.Text(findings.Hook(), governanceHeader)
	if text == "" {
		return "", 0
	}
	if note := report.Suppression(); note != "" {
		// The hook has no footer of its own, so the suppression is stated here
		// rather than dropped: a filtered block must say it was filtered.
		text += "  " + note + "\n"
	}
	return "\n" + text, len(report.Findings)
}

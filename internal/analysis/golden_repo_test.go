package analysis

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

// goldenRepo is a fake vcs.Repository over a plain directory tree, no real
// .git anywhere. Enough for the golden-corpus harness: Discover, on
// ScopeChanged/FileSetAll, only calls Root and StagedRenames (error
// swallowed) - WorktreeStatus, HeadSubject, HeadAuthorEpoch, StagedDiff, and
// StagedRenameDiff exist only to satisfy the interface, unexercised here.
type goldenRepo struct {
	root string
}

var _ vcs.Repository = (*goldenRepo)(nil)

func (g *goldenRepo) Root() string { return g.root }

func (g *goldenRepo) HeadSubject() (string, error) { return "", nil }

func (g *goldenRepo) CurrentBranch() (string, error) { return "", nil }

func (g *goldenRepo) HeadAuthorEpoch() (int64, bool, error) { return 0, false, nil }

func (g *goldenRepo) RecentSubjects(int) ([]string, error) { return nil, nil }

func (g *goldenRepo) WorktreeStatus() (vcs.Status, error) { return vcs.Status{}, nil }

func (g *goldenRepo) StagedRenames(int) ([]vcs.Rename, error) { return nil, nil }

func (g *goldenRepo) StagedDiff(string) (string, error) { return "", nil }

func (g *goldenRepo) StagedRenameDiff(string, string, int) (string, error) { return "", nil }

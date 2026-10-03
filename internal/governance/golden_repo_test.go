package governance

import (
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type goldenRepo struct {
	root string
}

var _ vcs.Repository = (*goldenRepo)(nil)

func (g *goldenRepo) Root() string { return g.root }

func (g *goldenRepo) CurrentBranch() (string, error) { return "", nil }

func (g *goldenRepo) HeadAuthorEpoch() (int64, bool, error) { return 0, false, nil }

func (g *goldenRepo) RecentSubjects(int) ([]string, error) { return nil, nil }

func (g *goldenRepo) WorktreeStatus() (vcs.Status, error) { return vcs.Status{}, nil }

func (g *goldenRepo) StagedRenames(int) ([]vcs.Rename, error) { return nil, nil }

func (g *goldenRepo) StagedDiff(string) (string, error) { return "", nil }

func (g *goldenRepo) StagedRenameDiff(string, string, int) (string, error) { return "", nil }

func (g *goldenRepo) HeadBlob(string) (string, error) { return "", nil }

func (g *goldenRepo) StagedBlob(string) (string, error) { return "", nil }

func (g *goldenRepo) HeadEntry(string) (mode, id string, err error) { return "", "", nil }

func (g *goldenRepo) StagedEntry(string) (mode, id string, err error) { return "", "", nil }

func (g *goldenRepo) HeadDir(string) ([]string, error) { return nil, nil }

func (g *goldenRepo) StagedDir(string) ([]string, error) { return nil, nil }

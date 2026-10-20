package governance

import (
	"io/fs"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type goldenRepo struct {
	root string
}

var _ Repository = (*goldenRepo)(nil)

func (repo *goldenRepo) Root() string { return repo.root }

func (repo *goldenRepo) CurrentBranch() (string, error) { return "", nil }

func (repo *goldenRepo) HeadAuthorEpoch() (int64, bool, error) { return 0, false, nil }

func (repo *goldenRepo) RecentSubjects(int) ([]string, error) { return nil, nil }

func (repo *goldenRepo) WorktreeStatus() (vcs.Status, error) { return vcs.Status{}, nil }

func (repo *goldenRepo) StagedRenames(int) ([]vcs.Rename, error) { return nil, nil }

func (repo *goldenRepo) Blob(vcs.Side, string) (string, error) { return "", nil }

func (repo *goldenRepo) Entry(vcs.Side, string) (mode, id string, err error) { return "", "", nil }

func (repo *goldenRepo) Dir(vcs.Side, string) ([]string, error) { return nil, nil }

func (repo *goldenRepo) ListFiles() ([]string, error) {
	var files []string
	err := filepath.WalkDir(repo.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		rel, relErr := filepath.Rel(repo.root, path)
		if relErr != nil {
			return relErr
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	return files, err
}

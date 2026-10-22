package command

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func TestOptionalRepository(t *testing.T) {
	t.Parallel()

	wiringErr := errors.New("boom: injector wiring failed")
	dir := t.TempDir()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	present, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	testCases := []struct {
		name     string
		repo     *vcs.Repository
		repoErr  error
		wantNil  bool
		wantErr  error
		wantRoot string
	}{
		{name: "absent repository is a true nil", repoErr: vcs.ErrNoRepository, wantNil: true},
		{name: "wiring failure is returned", repoErr: wiringErr, wantNil: true, wantErr: wiringErr},
		{name: "present repository is returned", repo: present, wantRoot: present.Root()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
			remy.RegisterConstructorErr(
				inj, remy.Factory[*vcs.Repository],
				func() (*vcs.Repository, error) { return testCase.repo, testCase.repoErr },
			)

			repo, err := OptionalRepository(inj)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("OptionalRepository() error = %v, want %v", err, testCase.wantErr)
			}
			if (repo == nil) != testCase.wantNil {
				t.Fatalf("repo == nil is %v, want %v (typed nil?)", repo == nil, testCase.wantNil)
			}
			if repo != nil && repo.Root() != testCase.wantRoot {
				t.Fatalf("Root() = %q, want %q", repo.Root(), testCase.wantRoot)
			}
		})
	}
}

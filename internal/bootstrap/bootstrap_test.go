package bootstrap

import (
	"testing"
)

func TestNew_ResolvesEveryService(t *testing.T) {
	t.Parallel()

	svc, err := New(Options{ProjectDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	eager := []struct {
		name string
		set  bool
	}{
		{"knowledge index", svc.Index != nil},
		{"templates", svc.Templates.FS != nil},
		{"renderer", svc.Renderer != nil},
		{"target registry", svc.Targets != nil},
		{"open repository", svc.OpenRepository != nil},
		{"open logger", svc.OpenLogger != nil},
		{"load catalog", svc.LoadCatalog != nil},
	}
	for _, testCase := range eager {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if !testCase.set {
				t.Fatalf("%s is not wired", testCase.name)
			}
		})
	}
	if _, err = svc.LoadCatalog(); err != nil {
		t.Errorf("LoadCatalog: %v", err)
	}
	logging, err := svc.OpenLogger()
	if err != nil || logging.Logger == nil || logging.Closer == nil {
		t.Errorf("OpenLogger() = %+v, %v, want a logger and a closer", logging, err)
	}
	if repo, repoErr := svc.OpenRepository(); repoErr != nil || repo == nil {
		t.Errorf(
			"OpenRepository() = %v, %v, want the repository of the working tree",
			repo,
			repoErr,
		)
	}
}

func TestNew_KnowledgeLoadFailure(t *testing.T) {
	t.Parallel()

	_, err := New(Options{Root: t.TempDir()})
	if err == nil {
		t.Fatal("New: want error for a dev root with no knowledge directory, got nil")
	}
}

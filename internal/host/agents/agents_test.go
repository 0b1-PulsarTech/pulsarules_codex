package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestWrite_RendersOnlySelectedSkills(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	projectDir := t.TempDir()
	wrote, err := Write(fakeTemplates(), projectDir, idx, []string{"go-style"})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if wrote == ownedfile.Kept {
		t.Fatalf("outcome = %v, want Wrote for a fresh project dir", wrote)
	}

	//nolint:gosec // temp dir.
	raw, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	got := string(raw)
	for _, want := range []string{
		filepath.Base(projectDir), "`go-style`", ownedfile.Marker, "routing contract text.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("AGENTS.md missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "`git-history`") {
		t.Errorf("AGENTS.md lists git-history, which was never selected:\n%s", got)
	}
}

func TestWrite_OwnershipDiscipline(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	testCases := []struct {
		name        string
		seedContent string
		wantWrote   bool
	}{
		{
			name:        "foreign file is not overwritten",
			seedContent: "# My own AGENTS.md\nDo not touch.\n",
			wantWrote:   false,
		},
		{
			name:        "tool-written file is refreshed",
			seedContent: "# AGENTS.md - old\n\n<!-- Installed by pulsarules_cli -->\nstale\n",
			wantWrote:   true,
		},
		{
			name:        "no existing file",
			seedContent: "",
			wantWrote:   true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			path := filepath.Join(projectDir, "AGENTS.md")
			if testCase.seedContent != "" {
				if seedErr := os.WriteFile(
					path,
					[]byte(testCase.seedContent),
					0o600,
				); seedErr != nil {
					t.Fatalf("seed AGENTS.md: %v", seedErr)
				}
			}

			wrote, writeErr := Write(fakeTemplates(), projectDir, idx, []string{"go-style"})
			if writeErr != nil {
				t.Fatalf("Write: %v", writeErr)
			}
			if (wrote != ownedfile.Kept) != testCase.wantWrote {
				t.Errorf("wrote = %v, want %v", wrote, testCase.wantWrote)
			}

			//nolint:gosec // temp dir.
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read AGENTS.md: %v", readErr)
			}
			if !testCase.wantWrote && string(got) != testCase.seedContent {
				t.Errorf("foreign content changed, got %q want %q", got, testCase.seedContent)
			}
		})
	}
}

func TestWrite_Errors(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	testCases := []struct {
		name         string
		templates    fstest.MapFS
		dirBlocked   bool
		agentsIsADir bool
	}{
		{
			name:      "missing template",
			templates: fstest.MapFS{},
		},
		{
			name: "template references unknown field",
			templates: fstest.MapFS{
				"docs/AGENTS.md.tmpl":     {Data: []byte(`{{.NotAField}}`)},
				"hooks/contract.txt":      {Data: []byte("routing contract text.\n")},
				"hooks/contract-tail.txt": {Data: []byte("commit tail text.\n")},
			},
		},
		{
			name: "missing contract asset",
			templates: fstest.MapFS{
				"docs/AGENTS.md.tmpl": {Data: []byte(`{{.Contract}}`)},
			},
		},
		{
			name:       "project dir blocked by a file",
			templates:  fakeTemplates(),
			dirBlocked: true,
		},
		{
			name:         "AGENTS.md path occupied by a directory",
			templates:    fakeTemplates(),
			agentsIsADir: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projectDir := t.TempDir()
			if testCase.dirBlocked {
				blocked := filepath.Join(projectDir, "blocked")
				if mkErr := os.WriteFile(blocked, []byte("x"), 0o600); mkErr != nil {
					t.Fatalf("seed blocker: %v", mkErr)
				}
				projectDir = filepath.Join(blocked, "child")
			}
			if testCase.agentsIsADir {
				if mkErr := os.Mkdir(filepath.Join(projectDir, "AGENTS.md"), 0o750); mkErr != nil {
					t.Fatalf("seed directory in place of AGENTS.md: %v", mkErr)
				}
			}
			if _, writeErr := Write(
				testCase.templates,
				projectDir,
				idx,
				[]string{"go-style"},
			); writeErr == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestWrite_MkdirErrorWording(t *testing.T) {
	t.Parallel()

	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	parent := t.TempDir()
	if err = os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(parent, 0o700) })

	projectDir := filepath.Join(parent, "project")
	_, err = Write(fakeTemplates(), projectDir, idx, []string{"go-style"})
	want := "mkdir " + `"` + projectDir + `": mkdir ` + projectDir + ": permission denied"
	if err == nil || err.Error() != want {
		t.Fatalf("Write err = %v, want %q", err, want)
	}
}

package agentswire

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestWriteAgents asserts the AGENTS.md is written at the project root,
// scoped to the selected ids, carrying the project name and the ownership
// marker RemoveAgents relies on - and that a skill outside the selection is
// left out, since AGENTS.md now lives at the root where a dead reference to
// an unrendered SKILL.md would mislead every host reading only this file.
func TestWriteAgents(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}
	projectDir := t.TempDir()
	wrote, err := WriteAgents(fakeTemplates(), projectDir, idx, []string{"go-style"})
	if err != nil {
		t.Fatalf("WriteAgents: %v", err)
	}
	if !wrote {
		t.Fatal("wrote = false, want true for a fresh project dir")
	}

	//nolint:gosec // temp dir.
	raw, err := os.ReadFile(filepath.Join(projectDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	got := string(raw)
	for _, want := range []string{
		filepath.Base(projectDir), "`go-style`", marker.Installed, "routing contract text.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("AGENTS.md missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "`git-history`") {
		t.Errorf("AGENTS.md lists git-history, which was never selected:\n%s", got)
	}
}

// TestWriteAgents_OwnershipDiscipline asserts WriteAgents leaves a
// pre-existing user-authored AGENTS.md untouched (the same ownership check
// RemoveAgents applies to deletion, applied here to writing, since a root
// AGENTS.md is a name a user very plausibly owns already), while a file this
// tool wrote before is refreshed on a second run.
func TestWriteAgents_OwnershipDiscipline(t *testing.T) {
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

			wrote, writeErr := WriteAgents(fakeTemplates(), projectDir, idx, []string{"go-style"})
			if writeErr != nil {
				t.Fatalf("WriteAgents: %v", writeErr)
			}
			if wrote != testCase.wantWrote {
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

// TestWriteAgents_Errors covers a template that fails to parse, one that
// fails to execute, and a project dir a file blocks from being created.
func TestWriteAgents_Errors(t *testing.T) {
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
			if _, writeErr := WriteAgents(
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

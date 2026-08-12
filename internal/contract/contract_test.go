package contract

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestSessionSubagentDeriveFromSameSource is the drift test: it renders both
// consumer variants against the repo's real embedded templates (not a
// fixture, so a hand-edit that breaks the invariant fails here) and asserts
// the subagent variant equals the session variant minus the commit tail.
func TestSessionSubagentDeriveFromSameSource(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load embedded knowledge: %v", err)
	}

	session, err := Session(templates)
	if err != nil {
		t.Fatalf("Session: %v", err)
	}
	subagent, err := Subagent(templates)
	if err != nil {
		t.Fatalf("Subagent: %v", err)
	}

	if subagent == "" {
		t.Fatal("Subagent returned empty text")
	}
	if !strings.HasPrefix(session, subagent) {
		t.Fatalf(
			"Session does not start with Subagent's text:\nsession=%q\nsubagent=%q",
			session,
			subagent,
		)
	}
	tail := strings.TrimSpace(strings.TrimPrefix(session, subagent))
	if tail == "" {
		t.Fatal("Session carries no commit tail beyond the subagent contract")
	}
	if strings.Contains(subagent, tail) {
		t.Errorf("subagent variant leaked the commit tail:\n%s", subagent)
	}
}

// TestReadAssetErrors covers both Session and Subagent surfacing a read
// failure instead of swallowing it, exercised through a templates FS missing
// each of the two assets in turn.
func TestReadAssetErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		templates fstest.MapFS
		render    func(fstest.MapFS) (string, error)
	}{
		{
			name:      "Session missing contract.txt",
			templates: fstest.MapFS{},
			render:    func(fs fstest.MapFS) (string, error) { return Session(fs) },
		},
		{
			name: "Session missing contract-tail.txt",
			templates: fstest.MapFS{
				"hooks/contract.txt": {Data: []byte("contract\n")},
			},
			render: func(fs fstest.MapFS) (string, error) { return Session(fs) },
		},
		{
			name:      "Subagent missing contract.txt",
			templates: fstest.MapFS{},
			render:    func(fs fstest.MapFS) (string, error) { return Subagent(fs) },
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := testCase.render(testCase.templates); err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

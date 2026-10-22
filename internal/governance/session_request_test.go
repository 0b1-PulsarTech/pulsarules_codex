package governance

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestSession_Analyze_CommitMsgBelongsToTheRequest(t *testing.T) {
	t.Parallel()

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	sess := NewSession(nil, idx, testCatalog(t), nil)

	testCases := []struct {
		name      string
		commitMsg string
		want      bool
	}{
		{"an invalid message is linted", "bad commit message", true},
		{"the next request carries none, so nothing is linted", "", false},
		{"the same session lints a later message again", "still bad", true},
	}
	for _, testCase := range testCases {
		got := sess.Analyze(Request{Scope: ScopeCommit, CommitMsg: testCase.commitMsg}).Findings
		if (len(got) > 0) != testCase.want {
			t.Errorf("%s: findings = %+v, want findings = %v", testCase.name, got, testCase.want)
		}
	}
}

func TestNewSession_InvalidTestGlobsSeedEveryRun(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.TestGlobs = []string{"[unclosed"}
	sess := NewSession(nil, nil, testCatalog(t), cfg)

	for run := range 2 {
		got := sess.Analyze(Request{Scope: ScopeChanged}).Findings
		if len(got) != 1 || got[0].AnalyzerID != "config" || got[0].File != ".pulsarules.yaml" {
			t.Fatalf("run %d: findings = %+v, want the one config finding", run, got)
		}
	}
}

func TestNewSession_PolicyIsFixedAtConstruction(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.TestGlobs = []string{"check/**"}
	sess := NewSession(nil, nil, testCatalog(t), cfg)
	sess.cfgAnalysis.TestGlobs = nil

	if !sess.classifier.IsTest("check/a.go") {
		t.Fatal("the test glob was read after construction, not at it")
	}
}

package commit

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

func newTestAnalyzer(tb testing.TB) *Analyzer {
	tb.Helper()
	catalog, err := emoji.NewCatalog()
	if err != nil {
		tb.Fatalf("NewCatalog: %v", err)
	}
	return NewAnalyzer(catalog)
}

func historyOf(shortcodes ...string) []core.GitCommitEntry {
	entries := make([]core.GitCommitEntry, 0, len(shortcodes))
	for _, shortcode := range shortcodes {
		entries = append(
			entries,
			core.GitCommitEntry{Subject: ":" + shortcode + ": chore: Something"},
		)
	}
	return entries
}

func TestCommitAnalyzerID(t *testing.T) {
	t.Parallel()

	analyzer := newTestAnalyzer(t)
	if analyzer.ID() != "commit-lint" {
		t.Errorf("ID = %q, want commit-lint", analyzer.ID())
	}
	if analyzer.Stage() != core.StageStatic {
		t.Errorf("Stage = %v, want StageStatic", analyzer.Stage())
	}
	if analyzer.Category() != core.CatCommit {
		t.Errorf("Category = %v, want CatCommit", analyzer.Category())
	}
}

// Requiring git history would make the pipeline skip the analyzer whenever
// history is missing, disabling every commit rule on a first commit.
func TestCommitAnalyzerDoesNotRequireGitHistory(t *testing.T) {
	t.Parallel()

	if newTestAnalyzer(t).Needs().NeedsGitHistory {
		t.Fatal("commit lint must run without git history")
	}
}

type analyzeCase struct {
	name        string
	commitMsg   string
	gitHistory  []core.GitCommitEntry
	wantFinders []string
	wantNone    []string
}

func analyzeCases() []analyzeCase {
	return []analyzeCase{
		{
			name:       "valid commit with no history",
			commitMsg:  ":wrench: feat(goscan): Detect shadowing",
			gitHistory: nil,
			wantNone: []string{
				"commit-emoji-required",
				"commit-type-enum",
				"commit-emoji-catalog",
				"commit-emoji-repeat",
			},
		},
		{
			name:       "valid commit with non-repeating history",
			commitMsg:  ":wrench: feat: Add thing",
			gitHistory: historyOf("memo", "tea", "bug", "gear", "package"),
			wantNone:   []string{"commit-emoji-repeat", "commit-emoji-soft-repeat"},
		},
		{
			name:        "repeat inside the hard window blocks",
			commitMsg:   ":wrench: feat: Add thing",
			gitHistory:  historyOf("memo", "wrench", "bug", "gear", "package"),
			wantFinders: []string{"commit-emoji-repeat"},
		},
		{
			name:        "off-catalog emoji blocks",
			commitMsg:   ":nonexistent_emoji: feat: Add thing",
			gitHistory:  nil,
			wantFinders: []string{"commit-emoji-catalog"},
		},
		{
			name:        "prohibited emoji blocks",
			commitMsg:   ":sparkles: feat: Add thing",
			gitHistory:  nil,
			wantFinders: []string{"commit-emoji-prohibited"},
		},
		{
			name:        "lowercase subject blocks",
			commitMsg:   ":wrench: feat: add thing",
			gitHistory:  nil,
			wantFinders: []string{"commit-desc-capitalize"},
		},
		{
			name:        "trailing period blocks",
			commitMsg:   ":wrench: feat: Add thing.",
			gitHistory:  nil,
			wantFinders: []string{"commit-desc-no-period"},
		},
		{
			name:        "empty commit msg produces no findings",
			commitMsg:   "",
			gitHistory:  nil,
			wantFinders: nil,
		},
		{
			name:        "co-author rejected",
			commitMsg:   ":wrench: feat: Add thing\n\nCo-Authored-By: Bot <x@y.com>",
			gitHistory:  nil,
			wantFinders: []string{"commit-no-coauthor"},
		},
		{
			name:        "claude session trailer rejected",
			commitMsg:   ":wrench: feat: Add thing\n\nClaude-Session: https://example.test/s",
			gitHistory:  nil,
			wantFinders: []string{"commit-no-coauthor"},
		},
	}
}

func TestCommitAnalyzerAnalyze(t *testing.T) {
	t.Parallel()

	for _, testCase := range analyzeCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := &core.AnalysisContext{
				CommitMsg:  testCase.commitMsg,
				GitHistory: testCase.gitHistory,
			}
			findings := newTestAnalyzer(t).Analyze(ctx)
			for _, want := range testCase.wantFinders {
				if !hasAnalyzer(findings, want) {
					t.Errorf("expected finding %s, got: %+v", want, findings)
				}
			}
			for _, none := range testCase.wantNone {
				if hasAnalyzer(findings, none) {
					t.Errorf("did not expect finding %s, got: %+v", none, findings)
				}
			}
		})
	}
}

// The windows are configurable so a project can widen them without a rebuild.
func TestCommitAnalyzerHonoursConfiguredWindow(t *testing.T) {
	t.Parallel()

	ctx := &core.AnalysisContext{
		CommitMsg:  ":wrench: feat: Add thing",
		GitHistory: historyOf("wrench", "memo", "tea", "bug", "gear", "package"),
		Config: &core.AnalysisConfig{
			Analyzers: map[string]core.AnalyzerConfig{
				"commit-lint": {Enabled: true, Params: map[string]any{"emoji_hard_window": 10}},
			},
		},
	}

	findings := newTestAnalyzer(t).Analyze(ctx)
	if !hasAnalyzer(findings, "commit-emoji-repeat") {
		t.Fatalf("a widened hard window must block the repeat, got %+v", findings)
	}
}

func TestEmojiWindowConfigMerge(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		params   core.ParamSet
		wantHard int
		wantSoft int
	}{
		{"nil params keeps defaults", nil, 5, 20},
		{"empty params keeps defaults", core.ParamSet{}, 5, 20},
		{
			name:     "zero values are ignored",
			params:   core.ParamSet{"emoji_hard_window": 0},
			wantHard: 5,
			wantSoft: 20,
		},
		{
			name:     "wrong types are ignored",
			params:   core.ParamSet{"emoji_hard_window": "eight"},
			wantHard: 5,
			wantSoft: 20,
		},
		{
			name: "set values win",
			params: core.ParamSet{
				"emoji_hard_window": 8,
				"emoji_soft_window": 30,
			},
			wantHard: 8,
			wantSoft: 30,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			merged := DefaultEmojiWindowConfig().merge(testCase.params)
			if merged.HardWindow != testCase.wantHard {
				t.Errorf("HardWindow = %d, want %d", merged.HardWindow, testCase.wantHard)
			}
			if merged.SoftWindow != testCase.wantSoft {
				t.Errorf("SoftWindow = %d, want %d", merged.SoftWindow, testCase.wantSoft)
			}
		})
	}
}

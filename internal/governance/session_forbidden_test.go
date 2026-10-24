package governance

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func forbiddenRows(t *testing.T, overrides []pass.Override) []string {
	t.Helper()

	dir := t.TempDir()
	writeFileAtDir(t, dir, "web/src/routes/drafts/list.ts", "export const draft = 1;\n")
	writeFileAtDir(t, dir, "web/src/lib/drafts/store.ts", "export const kept = 1;\n")
	cfg := config.Defaults()
	cfg.Analyzers["forbidden-token"] = pass.AnalyzerConfig{
		Enabled: true,
		Params:  map[string]any{"tokens": []any{"draft", "drafts"}},
	}
	cfg.Overrides = overrides

	sess := NewSession(&goldenRepo{root: dir}, nil, testCatalog(t), cfg)
	var rows []string
	for _, hit := range sess.Analyze(Request{Scope: ScopeChanged, Files: FileSetAll}).Findings {
		if hit.AnalyzerID == "forbidden-token" || hit.AnalyzerID == "forbidden-token/dir" {
			rows = append(rows, hit.AnalyzerID+" "+hit.File)
		}
	}
	slices.Sort(rows)
	return rows
}

func TestForbiddenTokenDirOverride(t *testing.T) {
	t.Parallel()

	off := false
	routesDirOff := pass.NewOverride([]string{"web/src/routes/**"}, map[string]pass.OverrideConfig{
		"forbidden-token/dir": {Enabled: &off},
	})
	routesAllOff := pass.NewOverride([]string{"web/src/routes/**"}, map[string]pass.OverrideConfig{
		"forbidden-token": {Enabled: &off},
	})
	testCases := []struct {
		name      string
		overrides []pass.Override
		want      []string
	}{
		{
			"no override reports both ids",
			nil,
			[]string{
				"forbidden-token web/src/routes/drafts/list.ts",
				"forbidden-token/dir web/src/lib/drafts",
				"forbidden-token/dir web/src/routes/drafts",
			},
		},
		{
			"dir override silences only the dir id",
			[]pass.Override{routesDirOff},
			[]string{
				"forbidden-token web/src/routes/drafts/list.ts",
				"forbidden-token/dir web/src/lib/drafts",
			},
		},
		{
			"base override reaches the dir id through the fallback",
			[]pass.Override{routesAllOff},
			[]string{"forbidden-token/dir web/src/lib/drafts"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := forbiddenRows(t, testCase.overrides); !reflect.DeepEqual(got, testCase.want) {
				t.Fatalf("rows = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestForbiddenTokenSkipsGonePaths(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFileAtDir(t, dir, "src/items/view.ts", "export const kept = 1;\n")
	writeFileAtDir(t, dir, "src/moved/new_view.ts", "export const kept = 1;\n")
	cfg := config.Defaults()
	cfg.Analyzers["forbidden-token"] = pass.AnalyzerConfig{
		Enabled: true,
		Params:  map[string]any{"tokens": []any{"drafts", "legacy"}},
	}
	status := &vcs.Status{Changes: []vcs.Change{
		{Path: "src/drafts/legacy_view.ts", Extension: ".ts", Staging: 'D', Staged: true},
		{Path: "src/legacy/old_view.ts", Extension: ".ts", Staging: 'D', Staged: false},
		{Path: "src/items/view.ts", Extension: ".ts", Staging: 'A', Staged: true},
		{Path: "src/moved/new_view.ts", Extension: ".ts", Staging: 'R', Staged: true},
	}}

	sess := NewSession(&goldenRepo{root: dir}, nil, testCatalog(t), cfg)
	for _, hit := range sess.Analyze(Request{Scope: ScopeCommit, Status: status}).Findings {
		if hit.AnalyzerID == "forbidden-token" || hit.AnalyzerID == "forbidden-token/dir" {
			t.Errorf("finding on a path that is gone: %s %s", hit.AnalyzerID, hit.File)
		}
	}
}

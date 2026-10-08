package projectcfg

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analysis/overrides"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

type winsCase struct {
	name, base, private string
	wantOff             string
	wantOverrides       int
	wantKept            []string
}

func enable(files string, ids ...string) string {
	rules := ""
	for _, id := range ids {
		rules += id + ": {enabled: true}, "
	}
	return "{files: ['" + files + "'], analyzers: {" + rules + "}}"
}

func enableDoc(files string, ids ...string) string {
	return overridesOf(enable(files, ids...))
}

func overridesOf(entries ...string) string {
	return "overrides: [" + strings.Join(entries, ", ") + "]\n"
}

func winsCases() []winsCase {
	const sizeOff = "analyzers: {file-size: {enabled: false}}\n"
	return []winsCase{
		{
			name: "explicit disable beats a base enabling override",
			base: doc(enableDoc("lib/**", "file-size")), private: doc(sizeOff),
			wantOff: "file-size",
		},
		{
			name: "explicit disable beats a base sub id override",
			base: doc(enableDoc("lib/**", "file-size/sub")), private: doc(sizeOff),
			wantOff: "file-size",
		},
		{
			name: "private preset beats a base enabling override",
			base: doc(enableDoc("lib/**", "doc-comment")), private: doc("preset: minimal"),
			wantOff: "doc-comment",
		},
		{
			name: "an entry holding only the dropped analyzer disappears",
			base: doc(overridesOf(
				enable("lib/**", "file-size"), enable("web/**", "big-comment"),
			)),
			private: doc(sizeOff), wantOff: "file-size", wantOverrides: 1,
			wantKept: []string{"big-comment"},
		},
		{
			name:    "the dropped analyzer leaves its entry's other analyzers",
			base:    doc(enableDoc("lib/**", "file-size", "big-comment")),
			private: doc(sizeOff), wantOff: "file-size", wantOverrides: 1,
			wantKept: []string{"big-comment"},
		},
	}
}

func TestPrivateWinsOverBaseOverrides(t *testing.T) {
	t.Parallel()

	for _, testCase := range winsCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: testCase.base, private: testCase.private, git: true}
			root := spec.build(t)
			cfg, findings := ForRoot(root)
			if len(findings) != 0 {
				t.Fatalf("ForRoot() findings = %+v, want none", findings)
			}
			if got := cfg.Analyzers[testCase.wantOff]; got.Enabled {
				t.Fatalf("%s enabled = true, want off", testCase.wantOff)
			}
			if len(cfg.Overrides) != testCase.wantOverrides {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverrides)
			}
			for _, override := range cfg.Overrides {
				for id := range override.Analyzers {
					if !slices.Contains(testCase.wantKept, id) {
						t.Fatalf("override kept %q, want only %v", id, testCase.wantKept)
					}
				}
			}
		})
	}
}

func keptCases() []winsCase {
	return []winsCase{
		{
			name:    "private re-enable keeps the base override",
			base:    doc(enableDoc("lib/**", "file-size")),
			private: doc("analyzers: {file-size: {enabled: true}}"), wantOverrides: 1,
			wantKept: []string{"file-size"},
		},
		{
			name: "private re-enable survives its own disabling preset",
			base: doc(overridesOf("{files: ['**/*.mjs'], analyzers: " +
				"{history-narration: {enabled: true, severity: error}}}")),
			private: doc(
				"preset: minimal", "analyzers: {history-narration: {enabled: true}}",
			),
			wantOverrides: 1, wantKept: []string{"history-narration"},
		},
		{
			name: "base explicit enable survives a private preset",
			base: doc("analyzers: {doc-comment: {enabled: true}}",
				enableDoc("lib/**", "doc-comment")),
			private: doc("preset: minimal"), wantOverrides: 1, wantKept: []string{"doc-comment"},
		},
		{
			name: "private override narrowing is kept",
			base: doc("preset: minimal"), private: doc(enableDoc("lib/**", "file-size")),
			wantOverrides: 1, wantKept: []string{"file-size"},
		},
	}
}

func TestPrivateWinsKeepsWhatItShould(t *testing.T) {
	t.Parallel()

	for _, testCase := range keptCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: testCase.base, private: testCase.private, git: true}
			cfg, findings := ForRoot(spec.build(t))
			if len(findings) != 0 {
				t.Fatalf("ForRoot() findings = %+v, want none", findings)
			}
			if len(cfg.Overrides) != testCase.wantOverrides {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverrides)
			}
		})
	}
}

func TestPrivateWins_NarrowsEachFile(t *testing.T) {
	t.Parallel()

	const (
		sizeOff     = "analyzers: {file-size: {enabled: false}}\n"
		privateName = ".git/pulsarules.yaml: overrides[0].analyzers.file-size.enabled"
	)
	testCases := []struct {
		name, base, private string
		wantFile, wantErr   string
	}{
		{
			name: "private override enabling after a private disable",
			base: doc("preset: strict"), private: doc(sizeOff, enableDoc("a", "file-size")),
			wantFile: ".git/pulsarules.yaml", wantErr: privateName,
		},
		{
			name: "base widening alone fails even when the private file disables too",
			base: doc(sizeOff, enableDoc("a", "file-size")), private: doc(sizeOff),
			wantFile: config.ProjectFileName,
			wantErr:  ".pulsarules.yaml: overrides[0].analyzers.file-size.enabled",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: testCase.base, private: testCase.private, git: true}
			_, findings := ForRoot(spec.build(t))
			assertConfigFinding(t, findings, testCase.wantFile, testCase.wantErr)
		})
	}
}

func loadAndBuild(root, flag string) (*config.GovernanceConfig, error) {
	file, err := Load(root, "", flag)
	if err != nil {
		return nil, err
	}
	return Build(file, flag)
}

func requireBlocked(t *testing.T, blocking string, errs ...error) {
	t.Helper()
	for _, got := range errs {
		if got == nil || !strings.Contains(got.Error(), blocking) {
			t.Fatalf("error = %v, want containing %q", got, blocking)
		}
	}
}

func TestPrivateWinsUsesTheEffectivePreset(t *testing.T) {
	t.Parallel()

	const blocking = "overrides[0].analyzers.doc-comment.enabled: doc-comment is disabled"
	base := doc(enableDoc("lib/**", "doc-comment"))
	testCases := []struct {
		name, private, flag string
		wantOverrides       int
		wantEnabled         bool
		wantBlocked         bool
	}{
		{"private preset disables", doc("preset: minimal"), "", 0, false, false},
		{"flag overrides it", doc("preset: minimal"), "recommended", 1, true, false},
		{"flag repeats it", doc("preset: minimal"), "minimal", 0, false, false},
		{"no preset anywhere", doc("skip_dirs: [x]"), "", 1, true, false},
		{"flag alone, no choice", doc("skip_dirs: [x]"), "minimal", 0, false, true},
		{"flag alone, version only", doc(), "minimal", 0, false, true},
		{"flag disables, private enables", doc("preset: recommended"), "minimal", 0, false, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: base, private: testCase.private, git: true}
			cfg, err := loadAndBuild(spec.build(t), testCase.flag)
			_, aloneErr := loadAndBuild(layout{base: base, git: true}.build(t), testCase.flag)
			if testCase.wantBlocked {
				requireBlocked(t, blocking, err, aloneErr)
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if len(cfg.Overrides) != testCase.wantOverrides {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverrides)
			}
			entry, listed := cfg.Analyzers["doc-comment"]
			if got := !listed || entry.Enabled; got != testCase.wantEnabled {
				t.Fatalf("doc-comment enabled = %v, want %v", got, testCase.wantEnabled)
			}
		})
	}
}

func overrideOf(glob, rules string) string {
	return "overrides: [{files: ['" + glob + "'], analyzers: {file-size: {" + rules + "}}}]\n"
}

type precedenceCase struct {
	name, base, private, file string
	wantKept                  bool
	wantSeverity              core.Severity
}

func precedenceCases() []precedenceCase {
	wideInfo := overrideOf("web/**", "severity: info")
	narrowErr := overrideOf("web/test/**", "severity: error")
	wideOff := overrideOf("web/**", "enabled: false")
	narrowOn := overrideOf("web/test/**", "enabled: true")
	const (
		inTest, inLib, elsewhere = "web/test/a.dart", "web/lib/b.dart", "other/c.dart"
	)
	return []precedenceCase{
		{"private severity beats base", wideInfo, narrowErr, inTest, true, core.SeverityError},
		{"base severity outside glob", wideInfo, narrowErr, inLib, true, core.SeverityInfo},
		{"untouched file", wideInfo, narrowErr, elsewhere, true, core.SeverityWarning},
		{"private enable beats disable", wideOff, narrowOn, inTest, true, core.SeverityWarning},
		{"base disable outside private glob", wideOff, narrowOn, inLib, false, 0},
		{"base alone", wideInfo, "", inTest, true, core.SeverityInfo},
	}
}

func TestOverridePrecedence(t *testing.T) {
	t.Parallel()

	for _, testCase := range precedenceCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: "version: 1\n" + testCase.base, git: true}
			if testCase.private != "" {
				spec.private = "version: 1\n" + testCase.private
			}
			file := loadWithin(t, spec.build(t), "")
			if file.err != nil {
				t.Fatalf("Load() error = %v", file.err)
			}
			cfg, err := Build(file.file, "")
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			found := []core.Finding{
				{AnalyzerID: "file-size", File: testCase.file, Severity: core.SeverityWarning},
			}
			got := overrides.Apply(found, "file-size", cfg.Overrides)
			if kept := len(got) == 1; kept != testCase.wantKept {
				t.Fatalf("Apply() = %+v, wantKept %v", got, testCase.wantKept)
			}
			if testCase.wantKept && got[0].Severity != testCase.wantSeverity {
				t.Fatalf("severity = %v, want %v", got[0].Severity, testCase.wantSeverity)
			}
		})
	}
}

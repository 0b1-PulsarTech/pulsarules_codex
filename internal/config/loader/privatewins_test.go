package loader

import (
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
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
			cfg, findings := testLoader().ForRoot(root)
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
			cfg, findings := testLoader().ForRoot(spec.build(t))
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
			wantFile: projectfile.Name,
			wantErr:  ".pulsarules.yaml: overrides[0].analyzers.file-size.enabled",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: testCase.base, private: testCase.private, git: true}
			_, findings := testLoader().ForRoot(spec.build(t))
			assertConfigFinding(t, findings, testCase.wantFile, testCase.wantErr)
		})
	}
}

func loadAndBuild(root, flag string) (*config.GovernanceConfig, error) {
	file, err := testLoader().Load(root, "", flag)
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
	wantSeverity              finding.Severity
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
		{"private severity beats base", wideInfo, narrowErr, inTest, true, finding.SeverityError},
		{"base severity outside glob", wideInfo, narrowErr, inLib, true, finding.SeverityInfo},
		{"untouched file", wideInfo, narrowErr, elsewhere, true, finding.SeverityWarning},
		{"private enable beats disable", wideOff, narrowOn, inTest, true, finding.SeverityWarning},
		{"base disable outside private glob", wideOff, narrowOn, inLib, false, 0},
		{"base alone", wideInfo, "", inTest, true, finding.SeverityInfo},
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
			found := []finding.Finding{
				{AnalyzerID: "file-size", File: testCase.file, Severity: finding.SeverityWarning},
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

type liftCase struct {
	name, user, flag string
	spec             layout
	wantErr          string
	wantLifted       []string
	// wantScopedAt names the analyzer an override must enable at big/ only, path-less included.
	wantScopedAt string
	wantOffEvery string
}

func liftCases() []liftCase {
	const sizeOff = "analyzers: {file-size: {enabled: false}}\n"
	const (
		branchOff  = "analyzers: {branch-name: {enabled: false}}\n"
		overNarrow = "an override can only narrow it"
	)
	return []liftCase{
		{
			name: "a user disable never invalidates a team override", user: baseStrict + sizeOff,
			spec:       layout{base: doc(enableDoc("big/**", "file-size")), git: true},
			wantLifted: []string{"file-size"}, wantScopedAt: "file-size",
		},
		{
			name: "a user preset never invalidates a team override", user: privateMinimal,
			spec:       layout{base: doc(enableDoc("big/**", "complexity")), git: true},
			wantLifted: []string{"complexity"}, wantScopedAt: "complexity",
		},
		{
			name: "a user disable never invalidates a private override", user: doc(sizeOff),
			spec:       layout{private: doc(enableDoc("big/**", "file-size")), git: true},
			wantLifted: []string{"file-size"}, wantScopedAt: "file-size",
		},
		{
			name: "a user disable holds for a finding with no path", user: doc(branchOff),
			spec:       layout{base: doc(enableDoc("big/**", "branch-name")), git: true},
			wantLifted: []string{"branch-name"}, wantScopedAt: "branch-name",
		},
		{
			name: "a team override that disables it does not lift", user: doc(sizeOff),
			spec: layout{
				base: doc(overridesOf(
					"{files: ['big/**'], analyzers: {file-size: {enabled: false}}}",
				)),
				git: true,
			},
			wantOffEvery: "file-size",
		},
		{
			name: "an empty user file hides no preset flag violation", user: doc(), flag: "minimal",
			spec:    layout{base: doc(enableDoc("big/**", "complexity")), git: true},
			wantErr: overNarrow,
		},
		{
			name: "no user file reports the preset flag violation", flag: "minimal",
			spec:    layout{base: doc(enableDoc("big/**", "complexity")), git: true},
			wantErr: overNarrow,
		},
		{
			name: "a preset flag over a user preset lifts nothing", user: privateMinimal,
			flag: "strict",
			spec: layout{base: doc(enableDoc("big/**", "complexity")), git: true},
		},
		{
			name: "nothing to lift emits no personal override", user: doc(sizeOff),
			spec: layout{base: doc(enableDoc("big/**", "complexity")), git: true},
		},
	}
}

func TestUnderlayLiftsOnlyWhatAnOverrideEnables(t *testing.T) {
	t.Parallel()

	for _, testCase := range liftCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			layered := New(testLoader().rules, placeUser(t, testCase.user))
			file, err := layered.Load(testCase.spec.build(t), "", testCase.flag)
			if err == nil {
				var cfg *config.GovernanceConfig
				if cfg, err = Build(file, testCase.flag); err == nil {
					assertLifted(t, file, cfg, testCase)
					return
				}
			}
			if testCase.wantErr == "" || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func assertLifted(
	t *testing.T, file *projectfile.File, cfg *config.GovernanceConfig, testCase liftCase,
) {
	t.Helper()
	if testCase.wantErr != "" {
		t.Fatalf("no error, want one containing %q", testCase.wantErr)
	}
	if !slices.Equal(file.PersonalOff, testCase.wantLifted) {
		t.Fatalf("PersonalOff = %v, want %v", file.PersonalOff, testCase.wantLifted)
	}
	if len(testCase.wantLifted) == 0 {
		assertNoOpenOverride(t, file, cfg)
	}
	if id := testCase.wantOffEvery; id != "" && cfg.Enabled(id) {
		t.Fatalf("%s runs, want it off for the whole project", id)
	}
	if id := testCase.wantScopedAt; id != "" {
		assertScopedAt(t, cfg, id)
	}
}

func assertNoOpenOverride(t *testing.T, file *projectfile.File, cfg *config.GovernanceConfig) {
	t.Helper()
	if len(file.Overrides) != len(cfg.Overrides) {
		t.Fatalf("overrides = %+v, want only the file's %d", cfg.Overrides, len(file.Overrides))
	}
	for _, override := range cfg.Overrides {
		if override.Files.IsOpen() {
			t.Fatalf("open override %+v emitted with nothing lifted", override)
		}
	}
}

func assertScopedAt(t *testing.T, cfg *config.GovernanceConfig, id string) {
	t.Helper()
	if !cfg.Enabled(id) {
		t.Fatalf("%s is off for the whole project, an override cannot reach it", id)
	}
	for path, want := range map[string]bool{"big/a.go": true, "other/a.go": false, "": false} {
		hit := finding.Finding{AnalyzerID: id, File: path}
		kept := len(overrides.Apply([]finding.Finding{hit}, id, cfg.Overrides)) == 1
		if kept != want {
			t.Errorf("%s kept at %q = %v, want %v", id, path, kept, want)
		}
	}
}

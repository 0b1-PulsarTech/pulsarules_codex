package loader

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/governance/overrides"
)

type delegatedCase struct {
	name         string
	private      string
	wantSeverity string
	wantOverride int
	wantErr      string
}

func delegatedCases() []delegatedCase {
	const named = ".git/pulsarules.yaml: "
	tune := func(rules string) string {
		return doc("overrides: [{files: [a], analyzers: {" + rules + "}}]")
	}
	return []delegatedCase{
		{
			name:         "severity in analyzers",
			private:      doc("analyzers: {dart-an: {severity: info}}"),
			wantSeverity: "info",
		},
		{
			name:         "enabled in analyzers",
			private:      doc("analyzers: {dart-an: {enabled: false}}"),
			wantSeverity: "error",
		},
		{
			name:         "override keyed by the id",
			private:      tune("dart-an: {severity: error}"),
			wantSeverity: "error",
			wantOverride: 1,
		},
		{
			name:         "override keyed by a sub id",
			private:      tune("dart-an/rule: {enabled: false}"),
			wantSeverity: "error",
			wantOverride: 1,
		},
		{
			name:    "an id in neither file",
			private: doc("analyzers: {nope: {severity: info}}"),
			wantErr: named + "analyzers.nope: unknown analyzer",
		},
		{
			name:    "an override id in neither file",
			private: tune("nope: {enabled: false}"),
			wantErr: named + "overrides[0].analyzers.nope: unknown analyzer",
		},
	}
}

func TestPrivateTunesBaseDelegated(t *testing.T) {
	t.Parallel()

	base := doc("delegated: [{id: dart-an, run: [dart, analyze], format: dart-analyze-json}]")
	for _, testCase := range delegatedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: base, private: testCase.private, git: true}
			result := loadWithin(t, spec.build(t), "")
			if testCase.wantErr != "" {
				assertErrorHas(t, result.err, []string{testCase.wantErr})
				return
			}
			if result.err != nil {
				t.Fatalf("Load() error = %v", result.err)
			}
			cfg, err := Build(result.file, "")
			if err != nil {
				t.Fatalf("Build() error = %v", err)
			}
			if got := dartSeverity(cfg); got != testCase.wantSeverity {
				t.Fatalf("severity = %q, want %q", got, testCase.wantSeverity)
			}
			if len(cfg.Overrides) != testCase.wantOverride {
				t.Fatalf("overrides = %+v, want %d", cfg.Overrides, testCase.wantOverride)
			}
		})
	}
}

func TestPrivateRulesWithoutBase(t *testing.T) {
	t.Parallel()

	declare := func(ids ...string) *projectfile.File {
		file := &projectfile.File{}
		for _, id := range ids {
			file.Delegated = append(file.Delegated, projectfile.DelegatedEntry{ID: id})
		}
		return file
	}
	testCases := []struct {
		name      string
		base      *projectfile.File
		wantExtra []string
	}{
		{"no base", nil, nil},
		{"one delegated id", declare("x"), []string{"x"}},
		{"two delegated ids", declare("x", "y"), []string{"x", "y"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := privateRules(testLoader().rules, testCase.base).AnalyzerIDs
			want := slices.Concat(testLoader().rules.AnalyzerIDs, testCase.wantExtra)
			if !slices.Equal(got, want) {
				t.Fatalf("AnalyzerIDs = %d ids, want %d", len(got), len(want))
			}
		})
	}
}

func TestPrivateRulesLeaveTheSharedIDsAlone(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		delegated []string
	}{
		{"one delegated id", []string{"x"}},
		{"two delegated ids", []string{"x", "y"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			shared := make([]string, 1, 8)
			shared[0] = "file-size"
			base := &projectfile.File{}
			for _, id := range testCase.delegated {
				base.Delegated = append(base.Delegated, projectfile.DelegatedEntry{ID: id})
			}

			got := privateRules(projectfile.Rules{AnalyzerIDs: shared}, base).AnalyzerIDs
			want := slices.Concat([]string{"file-size"}, testCase.delegated)
			if !slices.Equal(got, want) {
				t.Fatalf("ids = %v, want %v", got, want)
			}
			if spare := shared[:len(want)]; slices.Contains(spare[1:], "x") {
				t.Fatalf("shared spare slots = %v, want the loader rules untouched", spare)
			}
		})
	}
}

func TestFailedBaseDoesNotBlameDelegatedIDs(t *testing.T) {
	t.Parallel()

	const declared = "delegated: [{id: dart-an, run: [dart, analyze], format: dart-analyze-json}]"
	testCases := []struct {
		name, private string
		wantAlso      bool
	}{
		{"private tunes the id", doc("analyzers: {dart-an: {severity: info}}"), false},
		{
			"private override keyed by it",
			doc("overrides: [{files: [a], analyzers: {dart-an/x: {enabled: false}}}]"), false,
		},
		{"another private problem is still reported", doc("bogus: 1"), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{base: doc("bogus: 1", declared), private: testCase.private, git: true}
			_, findings := testLoader().ForRoot(spec.build(t))
			assertConfigFinding(
				t, findings, projectfile.Name, ".pulsarules.yaml: line 2: bogus")
			named := strings.Contains(findings[0].Message, ".git/pulsarules.yaml")
			if named != testCase.wantAlso {
				t.Fatalf("message = %q, names the private file = %v, want %v",
					findings[0].Message, named, testCase.wantAlso)
			}
		})
	}
}

func dartSeverity(cfg *config.GovernanceConfig) string {
	probe := []finding.Finding{{AnalyzerID: "dart-an", Severity: finding.SeverityError}}
	rules := slices.Concat(cfg.SeverityRules, cfg.Overrides)
	return overrides.Apply(probe, "dart-an", rules)[0].Severity.String()
}

type userCase struct {
	name         string
	user         string
	noUserDir    bool
	spec         layout
	wantPreset   string
	wantFileSize string
	wantNil      bool
	wantErr      []string
	wantFile     string
}

func userLayerCases() []userCase {
	return slices.Concat(userOverrideCases(), userOverflowCases(), userMergeCases())
}

const (
	sizeOff  = "analyzers: {file-size: {enabled: false}}\n"
	userPath = "pulsarules/pulsarules.yaml"
)

func userOverrideCases() []userCase {
	const dartAn = "delegated: [{id: dart-an, run: [dart, analyze], format: dart-analyze-json}]\n"
	return []userCase{
		{
			name: "the team tunes a command only the user file defines",
			user: "version: 1\n" + dartAn,
			spec: layout{base: "version: 1\nanalyzers: {dart-an: {severity: info}}\n", git: true},
		},
		{
			name: "the private file tunes a command only the user file defines",
			user: "version: 1\n" + dartAn,
			spec: layout{
				base: "version: 1\n", git: true,
				private: "version: 1\nanalyzers: {dart-an: {severity: info}}\n",
			},
		},
	}
}

func userOverflowCases() []userCase {
	return []userCase{
		{
			name: "three layers overflowing a list blame the user file",
			user: doc("skip_dirs: " + names("u", 400)),
			spec: layout{
				base: doc("skip_dirs: " + names("t", 400)), git: true,
				private: doc("skip_dirs: " + names("p", 400)),
			},
			wantErr:  []string{"skip_dirs: more than 1024 entries"},
			wantFile: userPath,
		},
		{
			name: "project layers overflowing a list blame the team file",
			user: "version: 1\n" + sizeOff,
			spec: layout{
				base: doc("skip_dirs: " + names("t", 600)), git: true,
				private: doc("skip_dirs: " + names("p", 600)),
			},
			wantErr:  []string{"skip_dirs: more than 1024 entries"},
			wantFile: ".pulsarules.yaml",
		},
	}
}

func userMergeCases() []userCase {
	const sizeOn = "analyzers: {file-size: {enabled: true}}\n"
	return []userCase{
		{
			name: "user only", user: baseStrict, spec: layout{git: true},
			wantPreset: "strict",
		},
		{
			name: "user plus team, team wins", user: baseStrict + sizeOn,
			spec:       layout{base: privateMinimal + sizeOff, git: true},
			wantPreset: "minimal", wantFileSize: "off",
		},
		{
			name: "all three layers", user: baseStrict + sizeOn,
			spec: layout{
				base: "version: 1\n" + sizeOff, private: "version: 1\n" + sizeOn, git: true,
			},
			wantPreset: "strict", wantFileSize: "on",
		},
		{
			name: "private still wins over both", user: baseStrict + sizeOn,
			spec: layout{
				base:    "version: 1\npreset: recommended\n" + sizeOn,
				private: privateMinimal + sizeOff,
				git:     true,
			},
			wantPreset: "minimal", wantFileSize: "off",
		},
		{
			name: "a user preset under a team preset", user: baseStrict,
			spec:       layout{base: privateMinimal, git: true},
			wantPreset: "minimal",
		},
		{
			name: "no user dir", noUserDir: true, user: bogusKey,
			spec: layout{base: baseStrict, git: true}, wantPreset: "strict",
		},
		{
			name: "no user file", spec: layout{base: baseStrict, git: true},
			wantPreset: "strict",
		},
		{
			name: "nothing anywhere", spec: layout{git: true}, wantNil: true,
		},
		{
			name: "a fault is blamed on the user file", user: bogusKey,
			spec:     layout{base: baseStrict, git: true},
			wantErr:  []string{userPath + ": line 2: bogus: unknown key"},
			wantFile: userPath,
		},
		{
			name: "faults of user and team are both named", user: bogusKey,
			spec:    layout{base: bogusKey, git: true},
			wantErr: []string{userPath + ": line 2: bogus", "; .pulsarules.yaml: line 2: bogus"},
		},
	}
}

func TestLoadUserLayer(t *testing.T) {
	t.Parallel()

	for _, testCase := range userLayerCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			userDir := placeUser(t, testCase.user)
			if testCase.noUserDir {
				userDir = ""
			}
			layered := New(testLoader().rules, userDir)
			file, err := layered.Load(testCase.spec.build(t), "", "")
			if testCase.wantErr != nil {
				assertErrorHas(t, err, testCase.wantErr)
				if got := faultFile(err, nil); testCase.wantFile != "" &&
					!strings.HasSuffix(got, testCase.wantFile) {
					t.Fatalf("faultFile() = %q, want the user file", got)
				}
				return
			}
			if err != nil || (file == nil) != testCase.wantNil {
				t.Fatalf("Load() = %+v, %v", file, err)
			}
			if file != nil {
				assertMerged(t, file, testCase)
			}
		})
	}
}

func placeUser(t *testing.T, content string) string {
	t.Helper()
	userDir := t.TempDir()
	if content == "" {
		return userDir
	}
	if err := os.MkdirAll(filepath.Join(userDir, userDirName), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	write(t, filepath.Join(userDir, userDirName, privateFileName), content)
	return userDir
}

func assertMerged(t *testing.T, file *projectfile.File, testCase userCase) {
	t.Helper()
	if file.Preset != testCase.wantPreset {
		t.Errorf("Preset = %q, want %q", file.Preset, testCase.wantPreset)
	}
	if got := enabledOf(file, "file-size"); got != testCase.wantFileSize {
		t.Errorf("file-size = %q, want %q", got, testCase.wantFileSize)
	}
}

func enabledOf(file *projectfile.File, id string) string {
	switch enabled := file.Analyzers[id].Enabled; {
	case enabled == nil:
		return ""
	case *enabled:
		return "on"
	default:
		return "off"
	}
}

func TestLoadUserEmptyDir(t *testing.T) {
	cwd := placeUser(t, baseStrict)
	t.Chdir(cwd)
	rules := testLoader().rules
	file, err := New(rules, "").loadUser(t.TempDir(), rules)
	if file != nil || err != nil {
		t.Fatalf("loadUser(empty dir) = %+v, %v, want nil, nil", file, err)
	}
}

//go:build unix

package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

func withEntry(
	change func(*core.DelegatedCommand),
) func(*testing.T, string) core.DelegatedCommand {
	return func(t *testing.T, root string) core.DelegatedCommand {
		t.Helper()
		entry := recordedWithExit(0)(t, root)
		change(&entry)
		if err := os.WriteFile(filepath.Join(root, "plain"), nil, 0o600); err != nil {
			t.Fatalf("write plain: %v", err)
		}
		return entry
	}
}

func configFaultCases() []runCase {
	configIDs := []string{entryID + "/config"}
	configFiles := []string{""}
	configSev := []core.Severity{core.SeverityError}
	return []runCase{
		{
			"dir escapes the root",
			withEntry(func(entry *core.DelegatedCommand) { entry.Dir = "../out" }),
			configIDs, configFiles, configSev, `dir "../out" must stay inside the project`,
		},
		{
			"dir is absolute",
			withEntry(func(entry *core.DelegatedCommand) { entry.Dir = "/tmp" }),
			configIDs, configFiles, configSev, `dir "/tmp" must stay inside the project`,
		},
		{
			"dir does not exist",
			withEntry(func(entry *core.DelegatedCommand) { entry.Dir = "missing" }),
			configIDs, configFiles, configSev, `dir "missing" is not a directory`,
		},
		{
			"dir is a file",
			withEntry(func(entry *core.DelegatedCommand) { entry.Dir = "plain" }),
			configIDs, configFiles, configSev, `dir "plain" is not a directory`,
		},
		{
			"unknown rule",
			withEntry(func(entry *core.DelegatedCommand) { entry.Rule = "no-such-rule" }),
			configIDs, configFiles, configSev, `rule "no-such-rule" is not in the knowledge index`,
		},
		{
			"unknown format",
			withEntry(func(entry *core.DelegatedCommand) { entry.Format = "nope" }),
			configIDs, configFiles, configSev, `unknown format "nope"`,
		},
		{
			"empty run",
			withEntry(func(entry *core.DelegatedCommand) { entry.Run = nil }),
			configIDs, configFiles, configSev, "run is empty",
		},
	}
}

func reportAt(file string) string {
	return `printf '{"version":1,"diagnostics":[{"code":"c","severity":"ERROR",` +
		`"location":{"file":"%s"}}]}' ` + file + "\nexit 3\n"
}

func linkedEntry(
	body, name, target string,
) func(*testing.T, string) core.DelegatedCommand {
	return func(t *testing.T, root string) core.DelegatedCommand {
		t.Helper()
		dest := t.TempDir()
		if target != "" {
			dest = filepath.Join(root, target)
			if err := os.Mkdir(dest, 0o700); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}
		if err := os.Symlink(dest, filepath.Join(root, name)); err != nil {
			t.Fatalf("symlink: %v", err)
		}
		return core.DelegatedCommand{
			ID: entryID, Dir: name, Format: "dart-analyze-json", Run: writeScript(t, body),
		}
	}
}

func symlinkCases() []runCase {
	outside := `dir "link" resolves outside the project`
	configIDs := []string{entryID + "/config"}
	inlinkIDs := []string{entryID + "/c"}
	files, sev := []string{""}, []core.Severity{core.SeverityError}
	return []runCase{
		{
			"relative tool through a link outside", linkedEntry(reportAt(`"f.dart"`), "link", ""),
			configIDs, files, sev, outside,
		},
		{
			"PWD tool through a link outside", linkedEntry(reportAt(`"$PWD/f.dart"`), "link", ""),
			configIDs, files, sev, outside,
		},
		{
			"physical tool through a link outside",
			linkedEntry(reportAt(`"$(pwd -P)/f.dart"`), "link", ""),
			configIDs, files, sev, outside,
		},
		{
			"relative tool through a link inside",
			linkedEntry(reportAt(`"f.dart"`), "inlink", "sub"),
			inlinkIDs,
			[]string{"sub/f.dart"},
			sev, "",
		},
		{
			"PWD tool through a link inside",
			linkedEntry(reportAt(`"$PWD/f.dart"`), "inlink", "sub"),
			inlinkIDs,
			[]string{"sub/f.dart"},
			sev, "",
		},
	}
}

func TestRunEntrySymlinkedDir(t *testing.T) {
	t.Parallel()

	runRunCases(t, symlinkCases())
}

func TestRunEntryConfigFaults(t *testing.T) {
	t.Parallel()

	runRunCases(t, configFaultCases())
}

func TestResolveInDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		run  string
		want string
	}{
		{"dart", "dart"},
		{"./fake.sh", "/proj/sub/fake.sh"},
		{"/usr/bin/dart", "/usr/bin/dart"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.run, func(t *testing.T) {
			t.Parallel()

			if got := resolveInDir("/proj/sub", testCase.run); got != testCase.want {
				t.Errorf("resolveInDir = %q, want %q", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest // a parallel fork would hold fake.sh open for writing: ETXTBSY
func TestRunEntryRelativeExecutable(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	script := "#!/bin/sh\nprintf '%s' '{\"version\":1,\"diagnostics\":[]}'\n"
	if err := os.WriteFile(filepath.Join(sub, "fake.sh"), []byte(script), 0o700); err != nil {
		t.Fatalf("write script: %v", err)
	}
	entry := core.DelegatedCommand{
		ID: entryID, Dir: "sub", Format: "dart-analyze-json", Run: []string{"./fake.sh"},
	}

	if findings := NewAnalyzer(nil).runEntry(root, entry); len(findings) != 0 {
		t.Fatalf("findings = %+v, want none (./fake.sh runs inside sub)", findings)
	}
}

func linkedRoot(t *testing.T) (logical, physical string) {
	t.Helper()
	physical = t.TempDir()
	if err := os.Mkdir(filepath.Join(physical, "sub"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	logical = filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(physical, logical); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	return logical, physical
}

func TestRunEntryLinkedRoot(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
		want string
	}{
		{"dir unset", "", "f.dart"},
		{"dir dot", ".", "f.dart"},
		{"dir inside", "sub", "sub/f.dart"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			logical, _ := linkedRoot(t)
			entry := core.DelegatedCommand{
				ID: entryID, Dir: testCase.dir, Format: "dart-analyze-json",
				Run: writeScript(t, reportAt(`"$(pwd -P)/f.dart"`)),
			}

			findings := NewAnalyzer(nil).runEntry(logical, entry)
			if len(findings) != 1 || findings[0].File != testCase.want ||
				findings[0].AnalyzerID != entryID+"/c" {
				t.Fatalf("findings = %+v, want one %s/c at %s", findings, entryID, testCase.want)
			}
		})
	}
}

func TestRunEntryRelativeFileWithoutDir(t *testing.T) {
	t.Parallel()

	ids, sev := []string{entryID + "/c"}, []core.Severity{core.SeverityError}
	runRunCases(t, []runCase{
		{
			"dir unset",
			entryRunning(func(*testing.T, string) string { return reportAt(`"f.dart"`) }),
			ids,
			[]string{"f.dart"},
			sev, "",
		},
	})
}

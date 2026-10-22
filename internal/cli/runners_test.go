package cli

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/command"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/cli/flags"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/emoji"
)

func seedGovernanceRepo(t *testing.T, dir, source string) {
	t.Helper()

	writeFile(t, filepath.Join(dir, "alpha.go"), source)
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "add", "alpha.go")
}

func outcome(err error) string {
	if err == nil {
		return "ok"
	}
	if errors.Is(err, command.ErrFindings) {
		return "exit 1"
	}
	return err.Error()
}

func TestRun_GovernanceReportsFindings(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		scope  string
		source string
		want   string
	}{
		{name: "clean tree", source: "package alpha\n", want: "ok"},
		{name: "warnings do not fail the run", source: "package alpha\n\nfunc {\n", want: "ok"},
		{name: "errors exit 1", source: "package alpha\n\nvar _ = \"\u2014\"\n", want: "exit 1"},
		{name: "invalid scope", scope: "fulll", want: "invalid --scope"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			seedGovernanceRepo(t, dir, testCase.source)
			svc := servicesIn(t, dir)
			args := []string{"governance", "--project", dir, "--all-files"}
			if testCase.scope != "" {
				args = append(args, "--scope", testCase.scope)
			}
			call, err := Parse(args, flags.Environment{}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := outcome(call.Run(svc)); !strings.HasPrefix(got, testCase.want) {
				t.Errorf("Run outcome = %q, want prefix %q", got, testCase.want)
			}
		})
	}
}

//nolint:paralleltest // t.Setenv and the os.Stdin/os.Stdout swaps bar t.Parallel.
func TestParse_HookReceivesTheEnvironment(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	project, skills := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(project, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(skills, "gopls-navigation", "SKILL.md"), "x")
	payload := `{"session_id":"s","tool_name":"Grep","tool_input":{"pattern":"a.go"}}`

	testCases := []struct {
		name string
		env  flags.Environment
		want bool
	}{
		{"both given", flags.Environment{ProjectDir: project, SkillsDir: skills}, true},
		{"no skills dir", flags.Environment{ProjectDir: project}, false},
		{"no project dir", flags.Environment{SkillsDir: skills}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			svc := servicesIn(t, project)
			call, err := Parse([]string{"hook", "pre-search"}, testCase.env, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			out, _ := capture(t, payload, func() {
				if err = call.Run(svc); err != nil {
					t.Errorf("Run() = %v, want nil", err)
				}
			})
			if got := out != ""; got != testCase.want {
				t.Errorf("pre-search emitted = %v, want %v:\n%s", got, testCase.want, out)
			}
		})
	}
}

func repoWithHistory(t *testing.T, dir string) {
	t.Helper()

	gitIn(t, dir, "init", "-q")
	writeFile(t, filepath.Join(dir, "alpha.go"), "package alpha\n")
	gitIn(t, dir, "add", "alpha.go")
	gitIn(t, dir, "commit", "-q", "-m", ":scissors: refactor(alpha): Start the package")
}

func writeUnlistedAnalyzer(t *testing.T, dir string) {
	t.Helper()

	repoWithHistory(t, dir)
	writeFile(
		t, filepath.Join(dir, ".pulsarules.yaml"),
		"version: 1\nanalyzers: {not-an-analyzer: {enabled: false}}\n",
	)
}

func hideGopls(t *testing.T, _ string) {
	t.Helper()

	t.Setenv("PATH", t.TempDir())
}

func writeOwnedSkill(t *testing.T, dir string) {
	t.Helper()

	owned := "# Go style\n\n<!-- Installed by pulsarules_cli; do not hand-edit. -->\n"
	writeFile(t, filepath.Join(dir, ".claude", "skills", "go-style", "SKILL.md"), owned)
}

func writeIgnoredMarker(t *testing.T, dir string) {
	t.Helper()

	repoWithHistory(t, dir)
	writeFile(t, filepath.Join(dir, "tracked.md"), "a \u2014 b\n")
	writeFile(t, filepath.Join(dir, "ignored.md"), "a \u2014 b\n")
	writeFile(t, filepath.Join(dir, ".gitignore"), "ignored.md\n")
	gitIn(t, dir, "add", "tracked.md", ".gitignore")
}

func writeArtifact(t *testing.T, dir string) {
	t.Helper()

	writeFile(t, filepath.Join(dir, "artifact.txt"), "package alpha\n")
}

func writeDashedSource(t *testing.T, dir string) {
	t.Helper()

	repoWithHistory(t, dir)
	writeFile(t, filepath.Join(dir, "alpha.go"), "package alpha\n\nvar _ = \"\u2014\"\n")
}

func failCatalog(svc *bootstrap.Services) {
	svc.LoadCatalog = func() (*emoji.Catalog, error) { return nil, errors.New("catalog boom") }
}

func failLogger(svc *bootstrap.Services) {
	svc.OpenLogger = func() (*slog.Logger, io.Closer, error) {
		return nil, nil, errors.New("logger boom")
	}
}

type runnerCase struct {
	name       string
	args       []string
	stdin      string
	setup      func(t *testing.T, dir string)
	mutate     func(svc *bootstrap.Services)
	wantResult string
	wantOut    []string
	wantErr    []string
	notOut     []string
	wantFiles  []string
	goneFiles  []string
}

const dirMark = "{dir}"

func renderCases() []runnerCase {
	return []runnerCase{
		{
			name:       "generate renders every skill from the index",
			args:       []string{"generate", "--out", dirMark + "/out"},
			wantResult: "ok",
			wantOut:    []string{"generated ", " skills into ", "/out/skills\n"},
			wantFiles:  []string{"out/skills/go-style/SKILL.md"},
		},
		{
			name:       "validate counts the index it was given",
			args:       []string{"validate"},
			wantResult: "ok",
			wantOut:    []string{"ok: ", " rules, ", " skills\n"},
		},
		{
			name:       "list defaults to skills",
			args:       []string{"list"},
			wantResult: "ok",
			wantOut:    []string{"SKILL", "LOAD", "go-style"},
		},
		{
			name:       "list forwards its kind argument",
			args:       []string{"list", "rules"},
			wantResult: "ok",
			wantOut:    []string{"ID", "DEPS"},
			notOut:     []string{"SKILL", "LOAD"},
		},
		{
			name:       "package writes the archive it reports",
			args:       []string{"package", "--out", dirMark + "/skills.zip"},
			wantResult: "ok",
			wantOut:    []string{"packaged: ", "/skills.zip\n"},
			wantFiles:  []string{"skills.zip"},
		},
	}
}

func installCases() []runnerCase {
	return []runnerCase{
		{
			name: "install renders skills and the gopls skill from the templates",
			args: []string{
				"install", "--project", dirMark, "--skills", "go-style", "--languages", "go",
				"--no-hooks", "--no-git-hooks",
			},
			wantResult: "ok",
			wantOut:    []string{"installed: "},
			wantFiles: []string{
				".claude/skills/go-style/SKILL.md", ".claude/skills/gopls-navigation/SKILL.md",
			},
		},
		{
			name: "install warns that gopls is not on PATH",
			args: []string{
				"install", "--project", dirMark, "--skills", "go-style", "--languages", "go",
				"--no-hooks", "--no-git-hooks",
			},
			setup:      hideGopls,
			wantResult: "ok",
			wantErr:    []string{"gopls not on PATH"},
			goneFiles:  []string{".claude/skills/gopls-navigation"},
		},
		{
			name: "install names the registered targets",
			args: []string{
				"install", "--project", dirMark, "--router-only", "--target", "bogus",
			},
			wantResult: "install: invalid --target",
		},
		{
			name:       "uninstall removes what the registry installed",
			args:       []string{"uninstall", "--project", dirMark},
			setup:      writeOwnedSkill,
			wantResult: "ok",
			goneFiles:  []string{".claude/skills/go-style"},
		},
		{
			name:       "uninstall lists the registered targets",
			args:       []string{"uninstall", "--project", dirMark, "--target", "bogus"},
			wantResult: `uninstall: invalid --target "bogus" (want agents|claude|cursor|opencode)`,
		},
	}
}

func lintCases() []runnerCase {
	const fresh = ":bug: fix(alpha): Handle the empty case"
	lint := []string{"commitlint", "--project", dirMark, "--msg"}
	return []runnerCase{
		{
			name:       "commitlint reads the history of the repository",
			args:       append(slices.Clone(lint), ":scissors: refactor(alpha): Repeat it"),
			setup:      repoWithHistory,
			wantResult: "exit 1",
			wantErr:    []string{"commit-emoji-repeat"},
		},
		{
			name:       "commitlint checks the emoji against the catalog",
			args:       append(slices.Clone(lint), ":notanemoji: refactor(alpha): Add it"),
			setup:      repoWithHistory,
			wantResult: "exit 1",
			wantErr:    []string{"commit-emoji-catalog"},
		},
		{
			name:       "commitlint accepts a fresh emoji",
			args:       append(slices.Clone(lint), fresh),
			setup:      repoWithHistory,
			wantResult: "ok",
			wantErr:    []string{"commit message is valid"},
		},
		{
			name:       "commitlint loads the catalog it was given",
			args:       append(slices.Clone(lint), fresh),
			mutate:     failCatalog,
			wantResult: "get emoji catalog: catalog boom",
		},
		{
			name:       "commitlint validates the config with the registry loader",
			args:       append(slices.Clone(lint), fresh),
			setup:      writeUnlistedAnalyzer,
			wantResult: "exit 1",
			wantErr:    []string{"unknown analyzer"},
		},
	}
}

func checkCases() []runnerCase {
	governance := []string{"governance", "--project", dirMark, "--all-files"}
	return []runnerCase{
		{
			name:       "governance validates the config with the registry loader",
			args:       governance,
			setup:      writeUnlistedAnalyzer,
			wantResult: ".pulsarules.yaml: analyzers.not-an-analyzer: unknown analyzer",
		},
		{
			name:       "governance loads the catalog it was given",
			args:       governance,
			setup:      repoWithHistory,
			mutate:     failCatalog,
			wantResult: "get emoji catalog: catalog boom",
		},
		{
			name:       "clean walks the files the repository lists",
			args:       []string{"clean", "--project", dirMark},
			setup:      writeIgnoredMarker,
			wantResult: "ok",
			wantOut:    []string{"tracked.md:1 found em dash", "1 marker(s) found"},
			notOut:     []string{"ignored.md"},
		},
		{
			name:       "evals grades the artifact it is given",
			args:       []string{"evals", "--artifact", dirMark + "/artifact.txt"},
			setup:      writeArtifact,
			wantResult: "exit 1",
			wantOut:    []string{"machine assertions passed across"},
		},
		{
			name:       "evals requires an artifact",
			args:       []string{"evals"},
			wantResult: "evals: --artifact is required",
		},
	}
}

func hookCases() []runnerCase {
	return []runnerCase{
		{
			name:       "hook stop reports governance findings with the checklist",
			args:       []string{"hook", "stop"},
			stdin:      `{"session_id":"stop-findings"}`,
			setup:      writeDashedSource,
			wantResult: "ok",
			wantOut:    []string{"typographic-markers", "gofmt -l clean"},
		},
		{
			name:       "hook stop loads the catalog it was given",
			args:       []string{"hook", "stop"},
			stdin:      `{"session_id":"stop-catalog"}`,
			setup:      writeDashedSource,
			mutate:     failCatalog,
			wantResult: "ok",
			wantErr:    []string{"hook: get emoji catalog: catalog boom"},
			notOut:     []string{"typographic-markers"},
		},
		{
			name:       "hook opens the logger it was given",
			args:       []string{"hook", "session-start"},
			stdin:      `{"session_id":"logger-failure"}`,
			mutate:     failLogger,
			wantResult: "ok",
			wantErr:    []string{"hook: build logger: logger boom"},
			notOut:     []string{"additionalContext"},
		},
	}
}

//nolint:paralleltest // t.Setenv and the os.Stdin/os.Stdout/os.Stderr swaps bar t.Parallel.
func TestRunners_PassEachCommandItsCollaborators(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	gopls := t.TempDir()
	answer := "#!/bin/sh\n[ \"$1 $2\" = \"mcp -instructions\" ] && echo fake gopls instructions\n"
	if err := os.WriteFile(filepath.Join(gopls, "gopls"), []byte(answer), 0o700); err != nil {
		t.Fatalf("write fake gopls: %v", err)
	}
	t.Setenv("PATH", gopls+string(os.PathListSeparator)+os.Getenv("PATH"))

	testCases := slices.Concat(
		renderCases(), installCases(), lintCases(), checkCases(), hookCases(),
	)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			dir := t.TempDir()
			if testCase.setup != nil {
				testCase.setup(t, dir)
			}
			svc := servicesIn(t, dir)
			if testCase.mutate != nil {
				testCase.mutate(&svc)
			}
			args := make([]string, len(testCase.args))
			for pos, arg := range testCase.args {
				args[pos] = strings.ReplaceAll(arg, dirMark, dir)
			}
			call, err := Parse(args, flags.Environment{ProjectDir: dir}, io.Discard)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			var runErr error
			stdout, stderr := capture(t, testCase.stdin, func() { runErr = call.Run(svc) })

			if got := outcome(runErr); !strings.HasPrefix(got, testCase.wantResult) {
				t.Errorf("Run outcome = %q, want prefix %q", got, testCase.wantResult)
			}
			assertHolds(t, "stdout", stdout, testCase.wantOut, testCase.notOut)
			assertHolds(t, "stderr", stderr, testCase.wantErr, nil)
			assertFiles(t, dir, testCase.wantFiles, testCase.goneFiles)
		})
	}
}

func assertHolds(t *testing.T, label, got string, want, unwanted []string) {
	t.Helper()

	for _, part := range want {
		if !strings.Contains(got, part) {
			t.Errorf("%s = %q, want it to contain %q", label, got, part)
		}
	}
	for _, part := range unwanted {
		if strings.Contains(got, part) {
			t.Errorf("%s = %q, want it free of %q", label, got, part)
		}
	}
}

func assertFiles(t *testing.T, dir string, written, removed []string) {
	t.Helper()

	for _, rel := range written {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("want %s written: %v", rel, err)
		}
	}
	for _, rel := range removed {
		if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
			t.Errorf("want %s removed", rel)
		}
	}
}

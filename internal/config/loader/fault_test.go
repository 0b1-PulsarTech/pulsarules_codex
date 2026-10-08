package loader

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func TestFaultFile(t *testing.T) {
	t.Parallel()

	bad := errors.New("bad")
	testCases := []struct {
		name string
		err  error
		file *projectfile.File
		want string
	}{
		{"typed error", &fileError{File: "x.yaml", Err: bad}, nil, "x.yaml"},
		{"no file defaults to the base", bad, nil, projectfile.Name},
		{"single file", bad, &projectfile.File{Path: "p.yaml"}, "p.yaml"},
		{
			"merged file by its base",
			bad,
			&projectfile.File{Path: "a.yaml + b + c.yaml (merged)"},
			"a.yaml",
		},
		{
			"private path holding the separator stays whole",
			bad,
			&projectfile.File{Path: "/x + y/pulsarules.yaml"},
			"/x + y/pulsarules.yaml",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := faultFile(testCase.err, testCase.file); got != testCase.want {
				t.Fatalf("faultFile() = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestFileErrorUnwraps(t *testing.T) {
	t.Parallel()

	cause := errors.New("cause")
	testCases := []struct {
		name    string
		inner   error
		wantMsg string
	}{
		{"plain cause", cause, "cause"},
		{"wrapped cause", fmt.Errorf("outer: %w", cause), "outer: cause"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fault := &fileError{File: "x.yaml", Err: testCase.inner}
			if !errors.Is(fault, cause) || fault.Error() != testCase.wantMsg {
				t.Fatalf("FileError = %v, want it to unwrap to its cause", fault)
			}
		})
	}
}

func TestCapReportShortensAMergedLead(t *testing.T) {
	t.Parallel()

	merged := projectfile.Name + mergedJoin +
		strings.Repeat("d", 1500) + "/.git/" + privateFileName + mergedSuffix
	problem := func(line int) string {
		filler := strings.Repeat("x", 2600)
		return fmt.Sprintf("%s: line %d: k: unknown key %s", merged, line, filler)
	}
	testCases := []struct {
		name       string
		text       string
		wantMarker bool
	}{
		{"one problem keeps its reason", problem(1), false},
		{"two problems are cut at the second lead", problem(1) + "; " + problem(2), true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fault := &fileError{
				File: projectfile.Name, Lead: merged, Err: errors.New(testCase.text),
			}
			got := capReport(fault, nil)
			requireFits(t, errorFinding(got), "unknown key", privateFileName)
			text := got.Error()
			if strings.HasSuffix(text, moreProblems) != testCase.wantMarker ||
				strings.Count(text, mergedSuffix+": ") != 1 || strings.Contains(text, merged) {
				t.Fatalf("capReport() = %.100q ... %.100q", text, text[len(text)-100:])
			}
		})
	}
}

func TestLoadCutsAMergedReportOnlyAtItsOwnLead(t *testing.T) {
	t.Parallel()

	const reason = "an override can only narrow it"
	rows := make([]string, 6)
	for idx := range rows {
		rows[idx] = fmt.Sprintf("  - files: ['lib%d/**']\n    analyzers: "+
			"{'package-level-var/a; .pulsarules.yaml: b': {enabled: true}}", idx)
	}
	base := "version: 1\noverrides:\n" + strings.Join(rows, "\n") + "\n"
	testCases := []struct {
		name       string
		privateLen int
		wantCut    bool
	}{
		{"four problems fit whole", 800, false},
		{"three problems then the marker", 825, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{
				worktree: true, privateLen: testCase.privateLen, base: base,
				private: "version: 1\n",
			}
			_, err := Load(spec.build(t), "", config.PresetMinimal)
			if err == nil {
				t.Fatal("Load() = nil, want the narrowing faults")
			}
			kept, cut := strings.CutSuffix(err.Error(), moreProblems)
			kept = strings.TrimSuffix(kept, "; ... and 2 more problems")
			requireFits(t, errorFinding(err), reason, privateFileName)
			if cut != testCase.wantCut || !strings.HasSuffix(kept, reason) ||
				strings.Count(kept, "(merged): overrides[") != strings.Count(kept, reason) {
				t.Fatalf("Load() = %.60q ... %.200q, want whole problems", err, kept)
			}
		})
	}
}

func TestLocateFaultDropsTheRoot(t *testing.T) {
	t.Parallel()

	resolve := func(dir string) (string, error) {
		return "", fmt.Errorf("git -C %s rev-parse: %w", dir, context.DeadlineExceeded)
	}
	testCases := []struct{ name, deep string }{
		{"a short root", ""},
		{"a 3950 byte root", deepOf(3950)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := layout{deep: testCase.deep}.build(t)
			_, err := loadLayers(root, resolve, "")
			got := capReport(err, nil)
			requireFits(t, errorFinding(got), "git -C . rev-parse: context deadline exceeded")
			if !errors.Is(got, context.DeadlineExceeded) || strings.Contains(got.Error(), root) {
				t.Fatalf("capReport() = %.120q, want the cause and no root", got)
			}
		})
	}
}

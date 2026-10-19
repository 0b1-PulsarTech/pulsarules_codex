package command

import (
	"slices"
	"testing"
)

func TestFormats(t *testing.T) {
	t.Parallel()

	if got := Formats(); !slices.Equal(got, []string{"dart-analyze-json"}) {
		t.Fatalf("Formats() = %v", got)
	}
}

func TestFormatsReportingExits(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		exit      int
		reporting bool
	}{
		{"clean", 0, false},
		{"infos", 1, true},
		{"warnings", 2, true},
		{"errors", 3, true},
		{"usage error", 64, false},
	}
	kind := formats()["dart-analyze-json"]
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := slices.Contains(
				kind.reportingExits(),
				testCase.exit,
			); got != testCase.reporting {
				t.Errorf("exit %d reporting = %v, want %v", testCase.exit, got, testCase.reporting)
			}
		})
	}
}

func TestDartAnalyzeJSONDecode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		output  string
		want    int
		wantErr bool
	}{
		{"one diagnostic", `{"version":1,"diagnostics":[{"code":"unused_import"}]}`, 1, false},
		{"empty report", `{"version":1,"diagnostics":[]}`, 0, false},
		{"wrong version", `{"version":2,"diagnostics":[]}`, 0, true},
		{"not json", `nope`, 0, true},
	}
	kind := formats()["dart-analyze-json"]
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			found, err := kind.decode([]byte(testCase.output))
			if (err != nil) != testCase.wantErr {
				t.Fatalf("decode err = %v, wantErr %v", err, testCase.wantErr)
			}
			if len(found) != testCase.want {
				t.Errorf("decode found %d diagnostics, want %d", len(found), testCase.want)
			}
		})
	}
}

func TestDartAnalyzeJSONPrecheckSkipsWithoutPackageConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	listed := listing{}
	if got := formats()["dart-analyze-json"].precheck(root, root, listed); got != "" {
		t.Errorf("precheck over a tree with no pubspec = %q, want no skip", got)
	}
}

func TestDartAnalyzeJSONCodegenKeepsPlainDiagnostics(t *testing.T) {
	t.Parallel()

	found := []diagnostic{{code: "unused_import", file: "lib/a.dart"}}
	kept, notice := formats()["dart-analyze-json"].codegen(found, scope{})
	if len(kept) != 1 || notice != "" {
		t.Errorf("codegen = %v, %q; want the diagnostic kept and no notice", kept, notice)
	}
}

package command

import (
	"os"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type decodeCase struct {
	name    string
	input   string
	want    []diagnostic
	wantErr string
}

func decodeCases(t *testing.T) []decodeCase {
	t.Helper()
	recorded, err := os.ReadFile("testdata/dart_analyze.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return []decodeCase{
		{
			name:  "recorded dart 3.13 output",
			input: string(recorded),
			want:  recordedDiagnostics(),
		},
		{
			name:  "unknown severity surfaces as a warning",
			input: `{"version":1,"diagnostics":[{"code":"c","severity":"FATAL"}]}`,
			want:  []diagnostic{{code: "c", severity: core.SeverityWarning}},
		},
		{name: "no diagnostics", input: `{"version":1,"diagnostics":[]}`, want: []diagnostic{}},
		{name: "not json", input: "Analyzing...", wantErr: "decode dart analyze json"},
		{name: "empty output", input: "", wantErr: "decode dart analyze json"},
		{
			name:    "other version",
			input:   `{"version":2,"diagnostics":[]}`,
			wantErr: "version 2, want 1",
		},
		{name: "missing version", input: `{"diagnostics":[]}`, wantErr: "version 0, want 1"},
	}
}

func TestDecodeDartAnalyze(t *testing.T) {
	t.Parallel()

	for _, testCase := range decodeCases(t) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := decodeDartAnalyze([]byte(testCase.input))
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeDartAnalyze: %v", err)
			}
			if len(got) != len(testCase.want) {
				t.Fatalf("got %d diagnostics, want %d", len(got), len(testCase.want))
			}
			for idx := range got {
				if got[idx] != testCase.want[idx] {
					t.Errorf("diagnostic %d = %+v, want %+v", idx, got[idx], testCase.want[idx])
				}
			}
		})
	}
}

func recordedDiagnostics() []diagnostic {
	return []diagnostic{
		{
			code:     "inference_failure_on_untyped_parameter",
			severity: core.SeverityWarning,
			file:     "{{ROOT}}/packages/nucleon/test/l10n/messages_for_test.dart",
			line:     21,
			message: "The type of 'message' can't be inferred; " +
				"a type must be explicitly provided.",
			suggestion: "Try specifying the type of the parameter.",
		},
		{
			code:       "prefer_interpolation_to_compose_strings",
			severity:   core.SeverityInfo,
			file:       "{{ROOT}}/packages/nucleon/test/settings/ai_config_scope_test.dart",
			line:       46,
			message:    "Use interpolation to compose strings and values.",
			suggestion: "Try using string interpolation to build the composite string.",
		},
		{
			code:     "uri_has_not_been_generated",
			severity: core.SeverityError,
			file:     "{{ROOT}}/packages/nucleon/lib/nucleon.dart",
			line:     117,
			message:  "Target of URI hasn't been generated: 'src/l10n/catalog.g.dart'.",
			suggestion: "Try running the generator that will generate " +
				"the file referenced by the URI.",
		},
	}
}

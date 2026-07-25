package hook

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestDispatchPostEdit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		filePath     string
		useTempDir   bool
		wantContains string
		wantEmpty    bool
	}{
		{
			name:         "go file emits skill list when no project dir",
			filePath:     "pkg/foo.go",
			wantContains: "go-style",
		},
		{
			name:         "go file falls back to generic when no skills installed",
			filePath:     "pkg/foo.go",
			useTempDir:   true,
			wantContains: "post-edit generic reminder",
		},
		{
			name:      "markdown extension produces no output",
			filePath:  "README.md",
			wantEmpty: true,
		},
		{
			name:         "sql file emits skill list when no project dir",
			filePath:     "query.sql",
			wantContains: "database-persistence",
		},
		{
			name:         "test file includes integration-tests skill",
			filePath:     "internal/service/service_test.go",
			wantContains: "integration-tests",
		},
		{
			name:         "proto file emits skill list when no project dir",
			filePath:     "api.proto",
			wantContains: "grpc-adapter",
		},
	}
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := ""
			if testCase.useTempDir {
				dir = t.TempDir()
			}
			disp, out := dispatchCapture(Deps{Index: idx, ProjectDir: dir})
			payload := fmt.Sprintf(
				`{"tool_input":{"file_path":%q},"session_id":%q}`,
				testCase.filePath, uniqueSessionID(t),
			)
			_ = disp.Dispatch("post-edit", []byte(payload))
			if testCase.wantEmpty {
				if strings.TrimSpace(out.String()) != "" {
					t.Errorf("expected no output for %q, got %q", testCase.filePath, out.String())
				}
				return
			}
			ctx := extractContext(t, out.String())
			if !strings.Contains(ctx, testCase.wantContains) {
				t.Errorf("output missing %q:\n%s", testCase.wantContains, ctx)
			}
		})
	}
}

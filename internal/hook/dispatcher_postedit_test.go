package hook

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

// TestDispatchPostEdit exercises every filePath/installed-skill combination
// through explicit Deps.SkillsDir values, never the ambient
// PULSARULES_SKILLS_DIR fallback, so it stays hermetic under t.Parallel().
func TestDispatchPostEdit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		filePath     string
		installSkill string
		wantContains string
	}{
		{
			name:         "go file emits the skill when it is installed",
			filePath:     "pkg/foo.go",
			installSkill: "go-style",
			wantContains: "go-style",
		},
		{
			name:         "go file falls back to generic when no skills installed",
			filePath:     "pkg/foo.go",
			wantContains: "post-edit generic reminder",
		},
		{
			name:         "markdown extension produces no output",
			filePath:     "README.md",
			wantContains: "",
		},
		{
			name:         "sql file emits the skill when it is installed",
			filePath:     "query.sql",
			installSkill: "database-persistence",
			wantContains: "database-persistence",
		},
		{
			name:         "test file emits the skill when it is installed",
			filePath:     "internal/service/service_test.go",
			installSkill: "integration-tests",
			wantContains: "integration-tests",
		},
		{
			name:         "proto file emits the skill when it is installed",
			filePath:     "api.proto",
			installSkill: "grpc-adapter",
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
			skillsDir := t.TempDir()
			if testCase.installSkill != "" {
				installSkillFixture(t, skillsDir, testCase.installSkill)
			}
			disp, out := dispatchCapture(Deps{Index: idx, SkillsDir: skillsDir})
			payload := fmt.Sprintf(
				`{"tool_input":{"file_path":%q},"session_id":%q}`,
				testCase.filePath, uniqueSessionID(t),
			)
			_ = disp.Dispatch("post-edit", []byte(payload))
			if testCase.wantContains == "" {
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

// TestDispatchPostEdit_SkillsDirUnset pins the B5 fallback contract: with no
// SkillsDir resolvable (Deps unset and PULSARULES_SKILLS_DIR unset), a
// matched Go edit degrades to the generic reminder instead of going silent
// or naming un-installed skills. It clears an environment variable, so it
// cannot run in parallel with a test that sets it.
func TestDispatchPostEdit_SkillsDirUnset(t *testing.T) {
	t.Setenv("PULSARULES_SKILLS_DIR", "")

	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	disp, out := dispatchCapture(Deps{Index: idx})
	payload := fmt.Sprintf(
		`{"tool_input":{"file_path":"pkg/foo.go"},"session_id":%q}`,
		uniqueSessionID(t),
	)
	_ = disp.Dispatch("post-edit", []byte(payload))
	ctx := extractContext(t, out.String())
	if !strings.Contains(ctx, "post-edit generic reminder") {
		t.Errorf("expected the generic reminder, got %q", ctx)
	}
}

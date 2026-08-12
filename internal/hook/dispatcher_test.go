package hook

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestDispatchSessionStart(t *testing.T) {
	t.Parallel()

	disp, out := dispatchCapture(Deps{})
	_ = disp.Dispatch("session-start", newSessionPayload(t))
	ctx := extractContext(t, out.String())
	for _, want := range []string{"contract text", "commit tail text"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("missing %q:\n%s", want, ctx)
		}
	}
}

func preEditIndex(t *testing.T) *knowledge.Index {
	t.Helper()
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return idx
}

func TestPreEditAsset_ByteBudget(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	body, err := fs.ReadFile(templates, "hooks/pre-edit.txt")
	if err != nil {
		t.Fatalf("read pre-edit.txt: %v", err)
	}
	text := string(body)

	const budgetBytes = 1200
	if len(body) >= budgetBytes {
		t.Errorf("pre-edit.txt is %d bytes, want < %d", len(body), budgetBytes)
	}
	if strings.ContainsRune(text, '\u2014') {
		t.Error("pre-edit.txt contains an em dash (U+2014)")
	}

	testCases := []struct {
		name string
		want string
	}{
		{"routing instruction kept", "project-router ran"},
		{"stdlib rung names slices/maps/cmp", "slices/maps/cmp"},
		{"stdlib rung names errors.Join", "errors.Join"},
		{"stdlib rung names iter.Seq", "iter.Seq"},
		{"two-call-site rule", ">=2 real call sites"},
		{"named-domain-type rule", "UserID string"},
		{"money as minor-unit int64", "minor-unit int64"},
		{"scope guard: never skip mandated pattern", "never skip a mandated"},
		{"scope guard: never trim guardrails", "never trim validation"},
		{"errors contract kept", "%w"},
		{"test shape kept", "testCases/testCase"},
		{"verify-before-commit line kept", "Validation checklist"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(text, testCase.want) {
				t.Errorf("pre-edit.txt missing %q:\n%s", testCase.want, text)
			}
		})
	}

	stdlibIdx := strings.Index(text, "stdlib")
	reuseIdx := strings.Index(text, "reuse")
	if stdlibIdx < 0 || reuseIdx < 0 || stdlibIdx > reuseIdx {
		t.Errorf(
			"decision ladder order wrong: stdlib must precede reuse (stdlibIdx=%d, reuseIdx=%d)",
			stdlibIdx,
			reuseIdx,
		)
	}
}

func TestUserPromptNudge_ShortAndKeepsPhrases(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	body, err := fs.ReadFile(templates, "hooks/user-prompt.txt")
	if err != nil {
		t.Fatalf("read user-prompt.txt: %v", err)
	}
	text := strings.TrimSpace(string(body))

	const nudgeByteCeiling = 300
	if got := len(text); got > nudgeByteCeiling {
		t.Errorf(
			"user-prompt.txt is %d bytes, want <= %d",
			got,
			nudgeByteCeiling,
		)
	}
	if strings.ContainsRune(text, '\u2014') {
		t.Error("user-prompt.txt contains an em dash (U+2014)")
	}

	testCases := []struct {
		name string
		want string
	}{
		{"routing question kept", "Have you invoked the project-router skill"},
		{"go/sql/config trigger kept", "Go/SQL/config"},
		{"skip consequence kept", "Skipping the router skips the rules"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(text, testCase.want) {
				t.Errorf("user-prompt.txt missing %q:\n%s", testCase.want, text)
			}
		})
	}

	notWantedCases := []struct {
		name   string
		phrase string
	}{
		{"contract.txt's baseline skill list", "code-minimalism"},
		{"contract.txt/pre-edit.txt already state the test shape", "testCase"},
		{"pre-edit.txt's minimalism ladder", "ladder"},
		{"pre-edit.txt's stdlib rung", "slices/maps/cmp"},
		{"pre-edit.txt's named-domain-type rule", "minor-unit int64"},
	}
	for _, notWantedCase := range notWantedCases {
		t.Run(notWantedCase.name, func(t *testing.T) {
			t.Parallel()
			if strings.Contains(text, notWantedCase.phrase) {
				t.Errorf(
					"user-prompt.txt duplicates contract.txt/pre-edit.txt phrase %q",
					notWantedCase.phrase,
				)
			}
		})
	}
}

func TestDispatchPreEdit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filePath string
		wantFire bool
	}{
		{"go file fires", "a.go", true},
		{"sql file fires", "q.sql", true},
		{"proto file fires", "a.proto", true},
		{"markdown skipped", "r.md", false},
	}
	idx := preEditIndex(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			disp, out := dispatchCapture(Deps{Index: idx})
			payload := fmt.Sprintf(
				`{"tool_input":{"file_path":%q},"session_id":%q}`,
				testCase.filePath, uniqueSessionID(t),
			)
			_ = disp.Dispatch("pre-edit", []byte(payload))
			if !testCase.wantFire {
				if strings.TrimSpace(out.String()) != "" {
					t.Errorf("expected no output, got %q", out.String())
				}
				return
			}
			ctx := extractContext(t, out.String())
			if !strings.Contains(ctx, "pre-edit reminder text") {
				t.Errorf("missing reminder text:\n%s", ctx)
			}
		})
	}
}

func TestDispatchPreEditPerFile_OncePerFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		filePaths []string
		wantFires []bool
	}{
		{"same file twice fires once", []string{"a.go", "a.go"}, []bool{true, false}},
		{"different files both fire", []string{"a.go", "b.go"}, []bool{true, true}},
	}
	idx := preEditIndex(t)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			id := uniqueSessionID(t)
			disp, out := dispatchCapture(Deps{Index: idx})
			for i, filePath := range testCase.filePaths {
				out.Reset()
				payload := fmt.Appendf(
					nil, `{"tool_input":{"file_path":%q},"session_id":%q}`, filePath, id,
				)
				_ = disp.Dispatch("pre-edit", payload)
				fired := strings.Contains(out.String(), "pre-edit reminder text")
				if fired != testCase.wantFires[i] {
					t.Errorf(
						"dispatch %d (%s): fired = %v, want %v",
						i,
						filePath,
						fired,
						testCase.wantFires[i],
					)
				}
			}
		})
	}
}

func TestDispatchUserPrompt_EveryDispatch(t *testing.T) {
	t.Parallel()

	payload := newSessionPayload(t)
	disp, out := dispatchCapture(Deps{})

	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Errorf("second call did not fire: %q", out.String())
	}
}

func TestDispatchSessionEnd_Markers(t *testing.T) {
	t.Parallel()

	id := uniqueSessionID(t)
	session := fmt.Appendf(nil, `{"session_id":%q}`, id)
	disp, _ := dispatchCapture(Deps{Index: preEditIndex(t)})

	_ = disp.Dispatch("session-start", session)
	_ = disp.Dispatch(
		"pre-edit",
		fmt.Appendf(nil, `{"tool_input":{"file_path":"a.go"},"session_id":%q}`, id),
	)

	markers := []string{
		"skill-route-session-start." + id,
		"skill-hook-" + preEditFileEvent("a.go") + "." + id,
	}
	for _, name := range markers {
		if _, err := os.Stat(filepath.Join(os.TempDir(), name)); err != nil {
			t.Errorf("marker %q not created before session-end: %v", name, err)
		}
	}

	_ = disp.Dispatch("session-end", session)

	for _, name := range markers {
		if _, err := os.Stat(filepath.Join(os.TempDir(), name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("marker %q not removed after session-end (stat err = %v)", name, err)
		}
	}
}

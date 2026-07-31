package hook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDispatchSessionStart(t *testing.T) {
	t.Parallel()

	disp, out := dispatchCapture(Deps{})
	_ = disp.Dispatch("session-start", newSessionPayload(t))
	ctx := extractContext(t, out.String())
	if !strings.Contains(ctx, "session contract text") {
		t.Errorf("missing contract text:\n%s", ctx)
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
		{"markdown skipped", "r.md", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			disp, out := dispatchCapture(Deps{})
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

func TestDispatchPreEditOncePerSession(t *testing.T) {
	t.Parallel()

	id := uniqueSessionID(t)
	payload := fmt.Appendf(nil, `{"tool_input":{"file_path":"a.go"},"session_id":%q}`, id)
	disp, out := dispatchCapture(Deps{})

	_ = disp.Dispatch("pre-edit", payload)
	if !strings.Contains(extractContext(t, out.String()), "pre-edit reminder text") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("pre-edit", payload)
	if strings.Contains(out.String(), "pre-edit reminder text") {
		t.Errorf("second call fired again: %q", out.String())
	}
}

func TestDispatchUserPrompt(t *testing.T) {
	t.Parallel()

	payload := newSessionPayload(t)
	disp, out := dispatchCapture(Deps{})

	_ = disp.Dispatch("user-prompt", payload)
	if !strings.Contains(extractContext(t, out.String()), "user-prompt routing reminder") {
		t.Fatalf("first call did not fire: %q", out.String())
	}
	out.Reset()
	_ = disp.Dispatch("user-prompt", payload)
	if strings.Contains(out.String(), "user-prompt routing reminder") {
		t.Errorf("second call fired again: %q", out.String())
	}
}

func TestDispatchSessionEnd(t *testing.T) {
	t.Parallel()

	id := uniqueSessionID(t)
	session := fmt.Appendf(nil, `{"session_id":%q}`, id)
	disp, _ := dispatchCapture(Deps{})

	_ = disp.Dispatch("session-start", session)
	_ = disp.Dispatch(
		"pre-edit",
		fmt.Appendf(nil, `{"tool_input":{"file_path":"a.go"},"session_id":%q}`, id),
	)
	_ = disp.Dispatch("user-prompt", session)

	markers := []string{
		"skill-route-session-start" + id,
		"skill-route-pre-edit" + id,
		"skill-route-prompt" + id,
	}
	for _, name := range markers {
		if _, err := os.Stat(filepath.Join(os.TempDir(), name)); err != nil {
			t.Errorf("marker %q not created before session-end: %v", name, err)
		}
	}

	_ = disp.Dispatch("session-end", session)

	for _, name := range markers {
		if _, err := os.Stat(filepath.Join(os.TempDir(), name)); !os.IsNotExist(err) {
			t.Errorf("marker %q not removed after session-end (stat err = %v)", name, err)
		}
	}
}

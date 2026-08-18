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

// preEditIndex loads the real knowledge index, since emitPreEdit now gates
// on Router.SkillsForFile, which reports no match at all over a nil index
// (see Router.SkillsForFile's doc). dispatchCapture's default Deps{} carries
// no Index, so any pre-edit test exercising the router match needs this.
func preEditIndex(t *testing.T) *knowledge.Index {
	t.Helper()
	idx, _, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	return idx
}

// TestPreEditAssetCarriesSubstance reads the REAL embedded pre-edit.txt
// (not hookTemplates' decoupled fixture) so a future edit that trims it
// back to bare skill-name pointers fails here, not ships silently. It also
// pins the byte budget and the decision-ladder order (stdlib before
// reuse-what-is-local), since minimalism.md's must item 2 puts them in that order.
func TestPreEditAssetCarriesSubstance(t *testing.T) {
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

// TestUserPromptNudgeStaysShortAndDistinct reads the REAL embedded
// user-prompt.txt (per-turn nudge, not hookTemplates' fixture). It fires
// every turn, so growth pays that cost every turn - the ceiling is scaled
// to a two-sentence question, not the larger session-start contract. It
// also proves the nudge doesn't restate contract.txt/pre-edit.txt.
func TestUserPromptNudgeStaysShortAndDistinct(t *testing.T) {
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

	const ceilingBytes = 300 // the text is a short routing question, not a digest
	if got := len(text); got > ceilingBytes {
		t.Errorf("user-prompt.txt is %d bytes, want <= %d", got, ceilingBytes)
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

	// why: these phrases belong to contract.txt/pre-edit.txt; the nudge
	// restating them would duplicate what those two assets already say every
	// session-start/first-write.
	notWanted := []string{
		"code-minimalism",  // contract.txt's baseline skill list
		"testCase",         // contract.txt/pre-edit.txt already state the test shape
		"ladder",           // pre-edit.txt's minimalism ladder
		"slices/maps/cmp",  // pre-edit.txt's stdlib rung
		"minor-unit int64", // pre-edit.txt's named-domain-type rule
	}
	for _, phrase := range notWanted {
		if strings.Contains(text, phrase) {
			t.Errorf("user-prompt.txt duplicates contract.txt/pre-edit.txt phrase %q", phrase)
		}
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

// TestDispatchPreEditPerFile is the failing-then-passing proof for the
// per-file gating change: the SAME file twice must emit once (already
// true), while two DIFFERENT files must emit both times (false under the
// old OncePerSession("pre-edit") gate, which muted every file after the
// session's first).
func TestDispatchPreEditPerFile(t *testing.T) {
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

// TestDispatchUserPromptFiresEveryTurn is the failing-then-passing proof for
// the per-turn digest: the old handler gated on OncePerSession("prompt"), so
// a second dispatch in the same session went silent - the exact bug that let
// mid-session drift off the router go uncorrected. Dispatching twice with the
// same session id must fire both times now.
func TestDispatchUserPromptFiresEveryTurn(t *testing.T) {
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

func TestDispatchSessionEnd(t *testing.T) {
	t.Parallel()

	id := uniqueSessionID(t)
	session := fmt.Appendf(nil, `{"session_id":%q}`, id)
	disp, _ := dispatchCapture(Deps{Index: preEditIndex(t)})

	_ = disp.Dispatch("session-start", session)
	_ = disp.Dispatch(
		"pre-edit",
		fmt.Appendf(nil, `{"tool_input":{"file_path":"a.go"},"session_id":%q}`, id),
	)
	// why: user-prompt no longer records a marker - it fires every turn now
	// (see emitUserPrompt), so there is nothing for session-end to clean up
	// there. session-start and pre-edit still gate per-session/per-file.

	markers := []string{
		"skill-route-session-start" + id,
		"skill-hook-" + preEditFileEvent("a.go") + id,
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

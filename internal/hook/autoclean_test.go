package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type autoCleanCase struct {
	name        string
	rel         string
	body        string
	wantCleaned bool
}

// Every row that expects no change is a precondition guarding the only place a
// hook dispatch writes a project file.
var autoCleanCases = []autoCleanCase{
	{"markdown carrier", "notes.md", "a\u200Bb\n", true},
	{"go carrier", "code.go", "package a // x\u200By\n", true},
	{"nbsp folds", "notes.md", "a\u00A0b\n", true},
	{"em dash is left for a human", "notes.md", "a\u2014b\n", false},
	{"emoji glue is left alone", "notes.md", "\U0001F468\u200D\U0001F469\n", false},
	{"clean file", "notes.md", "nothing here\n", false},
	{"unlisted extension", "notes.txt", "a\u200Bb\n", false},
	{"fixture under testdata", "testdata/f.md", "a\u200Bb\n", false},
}

func TestAutoCleanEdited(t *testing.T) {
	t.Parallel()

	for _, testCase := range autoCleanCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := filepath.Join(root, testCase.rel)
			if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(path, []byte(testCase.body), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}

			disp, _ := dispatchCapture(Deps{ProjectDir: root})
			notice := disp.autoCleanEdited(path)

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			changed := string(after) != testCase.body
			if changed != testCase.wantCleaned {
				t.Errorf(
					"file changed = %v, want %v (now %q)",
					changed,
					testCase.wantCleaned,
					after,
				)
			}
			if (notice != "") != testCase.wantCleaned {
				t.Errorf("notice = %q, want announced = %v", notice, testCase.wantCleaned)
			}
			if testCase.wantCleaned && !strings.Contains(notice, "Re-read it") {
				t.Errorf("notice = %q, want it to tell the agent to re-read", notice)
			}
		})
	}
}

func TestAutoCleanEdited_RefusesOutsideRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "victim.md")
	const body = "a\u200Bb\n"
	if err := os.WriteFile(outside, []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	disp, _ := dispatchCapture(Deps{ProjectDir: root})
	if notice := disp.autoCleanEdited(outside); notice != "" {
		t.Errorf("notice = %q, want none", notice)
	}
	after, err := os.ReadFile(outside)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(after) != body {
		t.Error("a file outside the project root was rewritten")
	}
}

func TestAutoCleanEdited_RefusesWithoutRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "a.md")
	const body = "a\u200Bb\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	disp, _ := dispatchCapture(Deps{})
	if notice := disp.autoCleanEdited(path); notice != "" {
		t.Errorf("notice = %q, want none", notice)
	}
	after, _ := os.ReadFile(path)
	if string(after) != body {
		t.Error("the file was rewritten with no project root resolved")
	}
}

// TestEmitPostEdit_AnnouncesTheCleanForMarkdown proves the placement: the skills
// gate returns early for .md, so a clean placed after it would never fire there.
func TestEmitPostEdit_AnnouncesTheCleanForMarkdown(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "notes.md")
	if err := os.WriteFile(path, []byte("a\u200Bb\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	disp, out := dispatchCapture(Deps{ProjectDir: root})
	if err := disp.Dispatch("post-edit", postEditPayload(path)); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}

	var got struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not one JSON line: %v (%q)", err, out.String())
	}
	if !strings.Contains(got.HookSpecificOutput.AdditionalContext, "zero width space") {
		t.Errorf(
			"context = %q, want it to name the codepoint",
			got.HookSpecificOutput.AdditionalContext,
		)
	}
}

// TestEmitPostEdit_SilentOnACleanFile pins that a clean edit behaves exactly as
// it did before this feature existed.
func TestEmitPostEdit_SilentOnACleanFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "notes.md")
	if err := os.WriteFile(path, []byte("nothing here\n"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	disp, out := dispatchCapture(Deps{ProjectDir: root})
	if err := disp.Dispatch("post-edit", postEditPayload(path)); err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("output = %q, want none", out.String())
	}
}

func postEditPayload(path string) []byte {
	payload, _ := json.Marshal(map[string]any{
		"session_id": "t",
		"tool_name":  "Write",
		"tool_input": map[string]string{"file_path": path},
	})
	return payload
}

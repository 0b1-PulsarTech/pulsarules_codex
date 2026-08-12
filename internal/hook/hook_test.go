package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func hookTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/contract.txt":  {Data: []byte("session contract text\n")},
		"hooks/pre-edit.txt":  {Data: []byte("pre-edit reminder text\n")},
		"hooks/post-edit.txt": {Data: []byte("post-edit generic reminder\n")},
		"hooks/post-edit-checklist.txt.tmpl": {Data: []byte(
			"checklist for {{.BaseName}}:\n" +
				"{{range .SkillIDs}}  - {{.}}\n{{end -}}" +
				"{{if .IsGo}}  - go doc-comment reminder\n{{end -}}" +
				"verify suite",
		)},
		"hooks/pre-search.txt":  {Data: []byte("pre-search gopls reminder\n")},
		"hooks/user-prompt.txt": {Data: []byte("user-prompt routing reminder\n")},
		"hooks/stop.txt":        {Data: []byte("Governance findings on the working tree.\n")},
	}
}

func dispatchCapture(deps Deps) (*Dispatcher, *bytes.Buffer) {
	var out bytes.Buffer
	deps.Out = &out
	if deps.Templates == nil {
		deps.Templates = hookTemplates()
	}
	return NewDispatcher(deps), &out
}

func extractContext(t *testing.T, raw string) string {
	t.Helper()
	var out Output
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		t.Fatalf("unmarshal hook output: %v (raw=%q)", err, raw)
	}
	return out.HookSpecificOutput.AdditionalContext
}

func uniqueSessionID(t *testing.T) string {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	id := fmt.Sprintf("test-%d-%s", os.Getpid(), name)
	NewSessionTrackerFromID(id).Cleanup()
	t.Cleanup(func() { NewSessionTrackerFromID(id).Cleanup() })
	return id
}

func newSessionPayload(t *testing.T) []byte {
	t.Helper()
	return fmt.Appendf(nil, `{"session_id":%q}`, uniqueSessionID(t))
}

func installSkillFixture(t *testing.T, skillsDir, id string) {
	t.Helper()
	dir := filepath.Join(skillsDir, id)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("stub\n"), 0o600); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
}

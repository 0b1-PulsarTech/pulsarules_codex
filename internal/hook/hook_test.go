package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"
)

// hookTemplates returns hook contract text fixtures decoupled from the repo's
// real templates, so a dispatcher test can't break just because the real
// copy under templates/hooks changed wording.
func hookTemplates() fstest.MapFS {
	return fstest.MapFS{
		"hooks/contract.txt":      {Data: []byte("contract text\n")},
		"hooks/contract-tail.txt": {Data: []byte("commit tail text\n")},
		"hooks/pre-edit.txt":      {Data: []byte("pre-edit reminder text\n")},
		"hooks/post-edit.txt":     {Data: []byte("post-edit generic reminder\n")},
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

// dispatchCapture builds a Dispatcher over deps with a private buffer as Out
// (defaulting Templates to hookTemplates() when unset) and returns both, so a
// test reads what Dispatch emitted without touching the process-global
// os.Stdout - the seam that used to force every dispatcher test to run
// serially.
func dispatchCapture(deps Deps) (*Dispatcher, *bytes.Buffer) {
	var out bytes.Buffer
	deps.Out = &out
	if deps.Templates == nil {
		deps.Templates = hookTemplates()
	}
	// why: without a sink, a Deps that resolves no project dir writes the
	// reinstall warning to the real stderr and spams the test run.
	if deps.ErrOut == nil {
		deps.ErrOut = io.Discard
	}
	return NewDispatcher(deps), &out
}

// extractContext unmarshals raw hook JSON output and returns its additional
// context, failing the test on malformed JSON.
func extractContext(t *testing.T, raw string) string {
	t.Helper()
	var out Output
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &out); err != nil {
		t.Fatalf("unmarshal hook output: %v (raw=%q)", err, raw)
	}
	return out.HookSpecificOutput.AdditionalContext
}

var sessionCounter atomic.Int64

// uniqueSessionID returns a session id unique to this test run and schedules
// cleanup of any per-session dedup markers it leaves behind in the OS temp
// dir, so dispatcher tests need no TMPDIR isolation and can run with
// t.Parallel().
func uniqueSessionID(t *testing.T) string {
	t.Helper()
	id := fmt.Sprintf("test-%d-%d", time.Now().UnixNano(), sessionCounter.Add(1))
	t.Cleanup(func() { NewSessionTrackerFromID(id).Cleanup() })
	return id
}

func newSessionPayload(t *testing.T) []byte {
	t.Helper()
	return fmt.Appendf(nil, `{"session_id":%q}`, uniqueSessionID(t))
}

// installSkillFixture writes a stub SKILL.md under skillsDir/id - the shape
// filterInstalled checks for - so a dispatcher test can mark a skill
// "installed" without touching a real skills directory.
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

// gitInit makes dir a git repository so git status --porcelain works in tests.
func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.CommandContext(t.Context(), "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
}

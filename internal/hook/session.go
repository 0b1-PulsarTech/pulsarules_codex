package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

// SessionTracker manages per-session marker files used to deduplicate hook
// emissions within a single Claude Code session. Markers live in the OS temp
// directory and are keyed by event name and session ID.
type SessionTracker struct {
	sessionID string
}

// NewSessionTrackerFromID builds a tracker from an already-known session id,
// defaulting to "nosession" when empty. Dispatch decodes the hook payload once
// and calls this instead of re-parsing the same bytes NewSessionTracker would.
func NewSessionTrackerFromID(sessionID string) *SessionTracker {
	if sessionID == "" {
		sessionID = "nosession"
	}
	return &SessionTracker{sessionID: sessionID}
}

// SessionID returns the session identifier this tracker was built from.
func (s *SessionTracker) SessionID() string { return s.sessionID }

// OncePerSession records that an event fired for this session and reports
// whether this is the first occurrence (true) or a repeat (false).
func (s *SessionTracker) OncePerSession(event string) bool {
	marker := s.markerPath("skill-route-" + event)
	if _, err := os.Stat(marker); err == nil {
		return false
	}
	_ = os.WriteFile(
		marker,
		nil,
		fsperm.FilePrivate,
	) //nolint:gosec // per-session marker, not sensitive.
	return true
}

// FirstEmission reports whether content differs from what was last emitted for
// this event and session, recording the new content hash. It returns false when
// identical content was already emitted, so a hook stays silent across turns
// until the underlying state actually changes.
func (s *SessionTracker) FirstEmission(event, content string) bool {
	marker := s.markerPath("skill-hook-" + event)
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])
	//nolint:gosec // marker is a deterministic temp path under our control
	if prev, err := os.ReadFile(marker); err == nil && string(prev) == hash {
		return false
	}
	_ = os.WriteFile(marker, []byte(hash), fsperm.FilePrivate) //nolint:gosec // per-session marker.
	return true
}

// Cleanup removes all per-session marker files for this session ID so a
// finished session leaves nothing behind in the temp directory. It scans the
// temp dir because each Dispatch call creates a fresh SessionTracker - the
// markers were recorded on prior instances that are no longer reachable.
//
// simplification: Cleanup only runs when Dispatch receives "session-end"
// (see hook.go). opencode's trigger() has no session-lifecycle hook name to
// send that from (see knowledge/templates/hooks/opencode-plugin.js's file
// header), so under opencode these markers accumulate in os.TempDir() for
// the life of the machine. Ceiling: no automatic reclamation for opencode
// sessions. Upgrade path: if opencode ever adds a session-end-equivalent
// hook, wire it to dispatch "session-end" the same way Claude Code's
// SessionEnd mode already does.
func (s *SessionTracker) Cleanup() {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, s.sessionID) {
			continue
		}
		if !strings.HasPrefix(name, "skill-route-") && !strings.HasPrefix(name, "skill-hook-") {
			continue
		}
		_ = os.Remove(filepath.Join(os.TempDir(), name))
	}
}

func (s *SessionTracker) markerPath(prefix string) string {
	return filepath.Join(os.TempDir(), prefix+s.sessionID)
}

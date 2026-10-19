package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

const maxSessionIDBytes = 128

const markerSeparator = "."

type sessionTracker struct {
	sessionID string
}

func newSessionTracker(sessionID string) *sessionTracker {
	if sessionID == "" {
		sessionID = "nosession"
	}
	if !isPlainToken(sessionID) {
		sessionID = "h-" + fingerprint([]byte(sessionID))
	}
	return &sessionTracker{sessionID: sessionID}
}

func fingerprint(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func isPlainToken(sessionID string) bool {
	if len(sessionID) > maxSessionIDBytes {
		return false
	}
	for _, char := range sessionID {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z':
		case char >= '0' && char <= '9', char == '_', char == '-':
		default:
			return false
		}
	}
	return true
}

func (tracker *sessionTracker) SessionID() string { return tracker.sessionID }

func (tracker *sessionTracker) OncePerSession(event string) bool {
	marker := tracker.markerPath("skill-route-" + event)
	if _, err := os.Stat(marker); err == nil {
		return false
	}
	_ = os.WriteFile(marker, nil, disk.FilePrivate)
	return true
}

func (tracker *sessionTracker) FirstEmission(event, content string) bool {
	marker := tracker.markerPath("skill-hook-" + event)
	hash := fingerprint([]byte(content))
	//nolint:gosec // marker is a deterministic temp path under our control
	if prev, err := os.ReadFile(marker); err == nil && string(prev) == hash {
		return false
	}
	//nolint:gosec // the id is a plain token or a hash, see newSessionTracker.
	_ = os.WriteFile(marker, []byte(hash), disk.FilePrivate)
	return true
}

func (tracker *sessionTracker) Cleanup() {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	// simplification: runs only on "session-end"; opencode's trigger() can't send it,
	// so markers accumulate for the machine's life. Upgrade path: an opencode equivalent.
	for _, entry := range entries {
		name := entry.Name()
		// A plain token holds no dot, so a short id never matches the tail of a longer one.
		if !strings.HasSuffix(name, markerSeparator+tracker.sessionID) {
			continue
		}
		if !strings.HasPrefix(name, "skill-route-") && !strings.HasPrefix(name, "skill-hook-") {
			continue
		}
		_ = os.Remove(filepath.Join(os.TempDir(), name))
	}
}

func (tracker *sessionTracker) markerPath(prefix string) string {
	return filepath.Join(os.TempDir(), prefix+markerSeparator+tracker.sessionID)
}

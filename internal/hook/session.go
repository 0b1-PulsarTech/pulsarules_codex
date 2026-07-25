package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
)

const maxSessionIDBytes = 128

const markerSeparator = "."

type SessionTracker struct {
	sessionID string
}

func NewSessionTrackerFromID(sessionID string) *SessionTracker {
	if sessionID == "" {
		sessionID = "nosession"
	}
	if !isPlainToken(sessionID) {
		sum := sha256.Sum256([]byte(sessionID))
		sessionID = "h-" + hex.EncodeToString(sum[:])
	}
	return &SessionTracker{sessionID: sessionID}
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

func (tracker *SessionTracker) SessionID() string { return tracker.sessionID }

func (tracker *SessionTracker) OncePerSession(event string) bool {
	marker := tracker.markerPath("skill-route-" + event)
	if _, err := os.Stat(marker); err == nil {
		return false
	}
	_ = os.WriteFile(marker, nil, fsperm.FilePrivate)
	return true
}

func (tracker *SessionTracker) FirstEmission(event, content string) bool {
	marker := tracker.markerPath("skill-hook-" + event)
	sum := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(sum[:])
	//nolint:gosec // marker is a deterministic temp path under our control
	if prev, err := os.ReadFile(marker); err == nil && string(prev) == hash {
		return false
	}
	//nolint:gosec // the id is a plain token or a hash, see NewSessionTrackerFromID.
	_ = os.WriteFile(marker, []byte(hash), fsperm.FilePrivate)
	return true
}

func (tracker *SessionTracker) Cleanup() {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
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

func (tracker *SessionTracker) markerPath(prefix string) string {
	return filepath.Join(os.TempDir(), prefix+markerSeparator+tracker.sessionID)
}

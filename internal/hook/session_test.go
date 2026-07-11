package hook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		id   string
		want string
	}{
		{"with session id", "abc-123", "abc-123"},
		{"empty id falls back", "", "nosession"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tr := NewSessionTrackerFromID(testCase.id)
			if tr.SessionID() != testCase.want {
				t.Fatalf("SessionID = %q, want %q", tr.SessionID(), testCase.want)
			}
		})
	}
}

func TestOncePerSession(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	tr := NewSessionTrackerFromID("once-test")
	defer tr.Cleanup()

	if !tr.OncePerSession("PreToolUse") {
		t.Fatal("first OncePerSession should return true")
	}
	if tr.OncePerSession("PreToolUse") {
		t.Fatal("second OncePerSession for same event should return false")
	}
	if !tr.OncePerSession("Stop") {
		t.Fatal("OncePerSession for different event should return true")
	}
}

func TestFirstEmission(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	tr := NewSessionTrackerFromID("emit-test")
	defer tr.Cleanup()

	if !tr.FirstEmission("Stop", "content-A") {
		t.Fatal("first emission should return true")
	}
	if tr.FirstEmission("Stop", "content-A") {
		t.Fatal("identical content should return false")
	}
	if !tr.FirstEmission("Stop", "content-B") {
		t.Fatal("different content should return true")
	}
}

func TestCleanupRemovesMarkers(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	tr := NewSessionTrackerFromID("cleanup-test")

	_ = tr.OncePerSession("PreToolUse")
	_ = tr.FirstEmission("Stop", "x")

	preEditMarker := filepath.Join(os.TempDir(), "skill-route-PreToolUsecleanup-test")
	if _, err := os.Stat(preEditMarker); err != nil {
		t.Fatalf("PreToolUse marker not created: %v", err)
	}

	tr.Cleanup()

	for _, marker := range []string{
		"skill-route-PreToolUsecleanup-test",
		"skill-hook-Stopcleanup-test",
	} {
		path := filepath.Join(os.TempDir(), marker)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("marker %q not removed (stat err = %v)", marker, err)
		}
	}
}

package hook

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
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
			tr := newSessionTracker(testCase.id)
			if tr.SessionID() != testCase.want {
				t.Fatalf("SessionID = %q, want %q", tr.SessionID(), testCase.want)
			}
		})
	}
}

func TestOncePerSession(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	tr := newSessionTracker("once-test")
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

	tr := newSessionTracker("emit-test")
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

	tr := newSessionTracker("cleanup-test")

	_ = tr.OncePerSession("PreToolUse")
	_ = tr.FirstEmission("Stop", "x")

	preEditMarker := filepath.Join(os.TempDir(), "skill-route-PreToolUse.cleanup-test")
	if _, err := os.Stat(preEditMarker); err != nil {
		t.Fatalf("PreToolUse marker not created: %v", err)
	}

	tr.Cleanup()

	for _, marker := range []string{
		"skill-route-PreToolUse.cleanup-test",
		"skill-hook-Stop.cleanup-test",
	} {
		path := filepath.Join(os.TempDir(), marker)
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("marker %q not removed (stat err = %v)", marker, err)
		}
	}
}

func TestSessionID_HashesAnUnsafeID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		id   string
	}{
		{"parent traversal", "/../../victim"},
		{"path separator", "a/b"},
		{"dot", "a.b"},
		{"space", "a b"},
		{"non ascii", "sess\u00e9"},
		{"too long", strings.Repeat("a", maxSessionIDBytes+1)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := newSessionTracker(testCase.id).SessionID()
			if !strings.HasPrefix(got, "h-") || !isPlainToken(got) {
				t.Fatalf("SessionID = %q, want a hashed plain token", got)
			}
			sum := sha256.Sum256([]byte(testCase.id))
			if want := "h-" + hex.EncodeToString(sum[:]); got != want {
				t.Fatalf("SessionID = %q, want %q", got, want)
			}
			if again := newSessionTracker(testCase.id).SessionID(); again != got {
				t.Fatalf("hash not deterministic: %q then %q", got, again)
			}
		})
	}
}

func TestSessionMarkers_StayInTempDir(t *testing.T) {
	root := t.TempDir()
	markerDir := filepath.Join(root, "markers")
	if err := os.Mkdir(markerDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", markerDir)

	tr := newSessionTracker("/../../victim")
	_ = tr.OncePerSession("PreToolUse")
	_ = tr.FirstEmission("PostToolUse", "x")

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "markers" {
		t.Fatalf("a marker escaped the temp dir: %v", entries)
	}
	markers, err := os.ReadDir(markerDir)
	if err != nil || len(markers) != 2 {
		t.Fatalf("markers = %v (err %v), want 2 inside the temp dir", markers, err)
	}

	tr.Cleanup()
	left, err := os.ReadDir(markerDir)
	if err != nil || len(left) != 0 {
		t.Fatalf("Cleanup left %v (err %v)", left, err)
	}
}

func TestCleanup_SparesASessionWhoseIDEndsTheSame(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	short, long := newSessionTracker("ab"), newSessionTracker("sess-ab")
	_ = long.OncePerSession("PreToolUse")
	_ = long.FirstEmission("Stop", "x")
	_ = short.OncePerSession("PreToolUse")

	short.Cleanup()

	left, err := os.ReadDir(os.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 {
		t.Fatalf("Cleanup for the short id left %v, want only the long id's two markers", left)
	}
	for _, entry := range left {
		if !strings.HasSuffix(entry.Name(), ".sess-ab") {
			t.Errorf("marker %q is not the long id's", entry.Name())
		}
	}
}

func TestFingerprint(t *testing.T) {
	t.Parallel()

	const emptySum = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := fingerprint(nil); got != emptySum {
		t.Errorf("fingerprint(nil) = %q, want %q", got, emptySum)
	}
	if got := fileEvent("p-", "a.go"); len(got) != len("p-")+32 {
		t.Errorf("fileEvent key = %q, want a 32-char suffix", got)
	}
}

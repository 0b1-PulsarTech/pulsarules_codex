package dartsyntax

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func writeMarker(t *testing.T, dir string, marker failureMarker) {
	t.Helper()
	payload, err := json.Marshal(marker)
	if err != nil {
		t.Fatal(err)
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(dir), 0o700); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}
	if writeErr := os.WriteFile(markerPath(dir), payload, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
}

func TestCacheDir(t *testing.T) {
	t.Parallel()
	got := cacheDir("/base", "3.13.2", "abc123")
	key := "3.13.2-" + runtime.GOOS + "-" + runtime.GOARCH + "-abc123"
	want := filepath.Join("/base", "pulsarules", "dart-syntax", key)
	if got != want {
		t.Errorf("cacheDir = %q, want %q", got, want)
	}
}

func TestPublish(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("directory rename semantics differ on windows")
	}
	testCases := []struct {
		name    string
		existed string
		noWork  bool
		wantErr string
		wantExe bool
	}{
		{name: "fresh target", wantExe: true},
		{name: "nothing to publish", noWork: true, wantErr: "publish helper"},
		{name: "complete target stays", existed: exeName(), wantExe: true},
		{name: "broken target repaired", existed: "junk", wantExe: true},
		{name: "broken target stuck", existed: "locked/junk", wantErr: "remove broken"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			work, dir := filepath.Join(root, "work"), filepath.Join(root, "dir")
			if !testCase.noWork {
				writeBuilt(t, work)
			}
			prepareExisting(t, dir, testCase.existed)
			if testCase.wantErr != "" {
				lockDir(t, filepath.Dir(filepath.Join(dir, testCase.existed)))
			}
			err := publish(work, dir)
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("publish error = %v, want containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil || exeExists(dir) != testCase.wantExe {
				t.Errorf("publish = %v, exe = %v, want %v", err, exeExists(dir), testCase.wantExe)
			}
		})
	}
}

func writeBuilt(t *testing.T, work string) {
	t.Helper()
	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, exeName()), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func prepareExisting(t *testing.T, dir, file string) {
	t.Helper()
	if file == "" {
		return
	}
	target := filepath.Join(dir, file)
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func lockDir(t *testing.T, dir string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
}

func TestRecentFailure(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		marker  *markerSeed
		wantErr string
	}{
		{name: "no marker"},
		{name: "fresh marker", marker: &markerSeed{age: time.Minute}, wantErr: "boom"},
		{name: "expired marker", marker: &markerSeed{age: failureWindow + time.Second}},
		{name: "marker dated in the future", marker: &markerSeed{age: -time.Minute}},
		{name: "marker of another key", marker: &markerSeed{key: "other", age: time.Second}},
		{name: "corrupt marker", marker: &markerSeed{raw: "{"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "current-0123456789abcdef")
			if testCase.marker != nil {
				testCase.marker.write(t, dir)
			}
			err := recentFailure(dir)
			assertErrContains(t, err, testCase.wantErr)
		})
	}
}

type markerSeed struct {
	key string
	age time.Duration
	raw string
}

func (seed markerSeed) write(t *testing.T, dir string) {
	t.Helper()
	if seed.raw == "" {
		key := seed.key
		if key == "" {
			key = filepath.Base(dir)
		}
		writeMarker(t, dir, failureMarker{
			Key: key, At: time.Now().Add(-seed.age), Message: "boom",
		})
		return
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markerPath(dir), []byte(seed.raw), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFailureMarkerFiles(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		blocked  bool
		clears   int
		wantErr  string
		wantLoad bool
	}{
		{name: "records and loads", wantLoad: true},
		{name: "clear removes and is idempotent", clears: 2},
		{name: "record under a file fails", blocked: true, wantErr: "create helper cache"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := filepath.Join(t.TempDir(), "sub", "current-0123456789abcdef")
			if testCase.blocked {
				dir = filepath.Join(blockerFile(t), "x", "key")
			}
			err := recordFailure(dir, errors.New("pub exploded"))
			assertErrContains(t, err, testCase.wantErr)
			for range testCase.clears {
				if clearErr := clearFailure(dir); clearErr != nil {
					t.Errorf("clearFailure: %v", clearErr)
				}
			}
			marker, ok := loadMarker(dir)
			if ok != testCase.wantLoad || (ok && marker.Message != "pub exploded") {
				t.Errorf("marker = %+v, %v; want loaded %v", marker, ok, testCase.wantLoad)
			}
		})
	}
}

func blockerFile(t *testing.T) string {
	t.Helper()
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return blocker
}

func assertErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("error = %v, want nil", err)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want containing %q", err, want)
	}
}

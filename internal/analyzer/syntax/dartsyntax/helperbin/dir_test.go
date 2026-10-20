package helperbin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCacheDir(t *testing.T) {
	t.Parallel()
	got := string(newHelperDir("/base", "3.13.2", "abc123"))
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
			err := helperDir(dir).publish(work)
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("publish error = %v, want containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil || helperDir(dir).exists() != testCase.wantExe {
				t.Errorf(
					"publish = %v, exe = %v, want %v",
					err,
					helperDir(dir).exists(),
					testCase.wantExe,
				)
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

package helperbin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
	if writeErr := os.WriteFile(helperDir(dir).markerPath(), payload, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
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
			err := helperDir(dir).recentFailure()
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
	if err := os.WriteFile(helperDir(dir).markerPath(), []byte(seed.raw), 0o600); err != nil {
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
			err := helperDir(dir).recordFailure(errors.New("pub exploded"))
			assertErrContains(t, err, testCase.wantErr)
			for range testCase.clears {
				if clearErr := helperDir(dir).clearFailure(); clearErr != nil {
					t.Errorf("clearFailure: %v", clearErr)
				}
			}
			marker, ok := helperDir(dir).loadMarker()
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

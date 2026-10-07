package dartsyntax

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type seed struct {
	name string
	file string
	age  time.Duration
}

func seedEntry(t *testing.T, root string, entry seed) {
	t.Helper()
	path := filepath.Join(root, entry.name)
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	stamped := []string{path}
	if entry.file != "" {
		file := filepath.Join(path, entry.file)
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		stamped = append([]string{file}, stamped...)
	}
	modified := time.Now().Add(-entry.age)
	for _, target := range stamped {
		if err := os.Chtimes(target, modified, modified); err != nil {
			t.Fatal(err)
		}
	}
}

type staleCase struct {
	name  string
	entry seed
	want  bool
}

func staleCases() []staleCase {
	const (
		day   = 24 * time.Hour
		month = 31 * day
		other = "bbbbbbbbbbbbbbbb"
		same  = "9.9.9-linux-amd64-" + other
		sdk   = "8.8.8-linux-amd64-" + other
	)
	return []staleCase{
		{name: "old build dir", entry: seed{name: "build-abc", age: 2 * BuildTimeout}, want: true},
		{name: "fresh build dir", entry: seed{name: "build-abc"}},
		{name: "current key", entry: seed{name: staleCurrent, file: exeName(), age: month}},
		{name: "same key in use", entry: seed{name: same, file: exeName(), age: day}},
		{name: "same key, 6 days", entry: seed{name: same, file: exeName(), age: 6 * day}},
		{
			name:  "same key, 8 days",
			entry: seed{name: same, file: exeName(), age: 8 * day}, want: true,
		},
		{
			name:  "same key, just under a week",
			entry: seed{name: same, file: exeName(), age: 7*day - time.Hour},
		},
		{
			name:  "same key, just over a week",
			entry: seed{name: same, file: exeName(), age: 7*day + time.Hour}, want: true,
		},
		{
			name:  "other arch in use",
			entry: seed{name: "9.9.9-plan9-mips-" + other, file: exeName(), age: time.Hour},
		},
		{name: "other sdk in use", entry: seed{name: sdk, file: exeName(), age: time.Hour}},
		{name: "other sdk, 6 days", entry: seed{name: sdk, file: exeName(), age: 6 * day}},
		{
			name:  "other sdk, 8 days",
			entry: seed{name: sdk, file: exeName(), age: 8 * day}, want: true,
		},
		{name: "no exe, old dir", entry: seed{name: sdk, age: 8 * day}, want: true},
		{name: "no exe, fresh dir", entry: seed{name: sdk, age: day}},
		{name: "unrelated dir", entry: seed{name: "notes", age: month}},
	}
}

const staleCurrent = "9.9.9-linux-amd64-aaaaaaaaaaaaaaaa"

func TestStale(t *testing.T) {
	t.Parallel()
	for _, testCase := range staleCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			seedEntry(t, root, testCase.entry)
			entries, err := os.ReadDir(root)
			if err != nil {
				t.Fatal(err)
			}
			if got := stale(root, entries[0], staleCurrent); got != testCase.want {
				t.Errorf("stale(%s) = %v, want %v", testCase.entry.name, got, testCase.want)
			}
		})
	}
}

func TestPruneCacheErrors(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions differ on windows")
	}
	testCases := []struct {
		name    string
		missing bool
		wantErr string
	}{
		{name: "missing root", missing: true, wantErr: "list"},
		{name: "undeletable entry", wantErr: "prune"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if testCase.missing {
				root = filepath.Join(root, "absent")
			} else {
				lockEntry(t, root, "1.0.0-linux-amd64-0123456789abcdef")
			}
			err := pruneCache(filepath.Join(root, "9.9.9-linux-amd64-aaaaaaaaaaaaaaaa"))
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("pruneCache = %v, want containing %q", err, testCase.wantErr)
			}
		})
	}
}

func lockEntry(t *testing.T, root, name string) {
	t.Helper()
	seedEntry(t, root, seed{name: name, file: exeName(), age: 40 * 24 * time.Hour})
	inner := filepath.Join(root, name, "inner")
	if err := os.MkdirAll(inner, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inner, "file"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	lockDir(t, inner)
}

type layoutCase struct {
	name     string
	seeds    []seed
	wantKept []string
	wantGone []string
	wantUsed []string
}

func layoutCases() []layoutCase {
	return append(keepLayoutCases(), removeLayoutCases()...)
}

func keepLayoutCases() []layoutCase {
	const (
		family = "{family}"
		day    = 24 * time.Hour
		arch   = "9.9.9-plan9-mips-0123456789abcdef"
		sdk    = "8.8.8-linux-amd64-0123456789abcdef"
	)
	exe := exeName()
	return []layoutCase{
		{
			name:  "other arch survives",
			seeds: []seed{{name: arch, file: exe}}, wantKept: []string{arch + "/" + exe},
		},
		{
			name:  "other sdk survives",
			seeds: []seed{{name: sdk, file: exe}}, wantKept: []string{sdk + "/" + exe},
		},
		{
			name:     "in-use hash of the same key stays",
			seeds:    []seed{{name: family + "0123456789abcdef", file: exe, age: day}},
			wantKept: []string{family + "0123456789abcdef/" + exe},
		},
		{
			name:     "reuse refreshes the exe time",
			seeds:    []seed{{name: "{current}", file: exe, age: 31 * day}},
			wantUsed: []string{"{current}/" + exe},
		},
	}
}

func removeLayoutCases() []layoutCase {
	const (
		family = "{family}"
		day    = 24 * time.Hour
		sdk    = "8.8.8-linux-amd64-0123456789abcdef"
	)
	exe := exeName()
	return []layoutCase{
		{
			name:     "old hash of the same key goes",
			seeds:    []seed{{name: family + "0123456789abcdef", file: exe, age: 8 * day}},
			wantGone: []string{family + "0123456789abcdef"},
		},
		{
			name:     "other key unused for a month goes",
			seeds:    []seed{{name: sdk, file: exe, age: 31 * day}},
			wantGone: []string{sdk},
		},
		{
			name: "build dirs by age, other dirs stay",
			seeds: []seed{
				{name: "build-stale", age: 2 * BuildTimeout},
				{name: "build-fresh"},
				{name: "notes", age: 31 * day},
			},
			wantKept: []string{"build-fresh", "notes"},
			wantGone: []string{"build-stale"},
		},
		{
			name:     "dir without exe is repaired",
			seeds:    []seed{{name: "{current}", file: "junk"}},
			wantKept: []string{"{current}/" + exe},
			wantGone: []string{"{current}/junk"},
		},
	}
}

func TestEnsureHelperCacheLayout(t *testing.T) {
	dart := writeFakeDart(t)
	t.Parallel()
	for _, testCase := range layoutCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			root := cacheRoot(base)
			current := filepath.Base(currentDir(t, base))
			expand := strings.NewReplacer(
				"{family}", keyPrefix("9.9.9"), "{current}", current,
			).Replace
			for _, entry := range testCase.seeds {
				entry.name = expand(entry.name)
				seedEntry(t, root, entry)
			}
			if _, err := ensureHelper(context.Background(), base, dart); err != nil {
				t.Fatalf("ensureHelper: %v", err)
			}
			for _, name := range testCase.wantKept {
				assertPresent(t, root, expand(name))
			}
			for _, name := range testCase.wantGone {
				assertAbsent(t, root, expand(name))
			}
			for _, name := range testCase.wantUsed {
				assertTouched(t, filepath.Join(root, expand(name)))
			}
		})
	}
}

func assertPresent(t *testing.T, root, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, name)); err != nil {
		t.Errorf("%s is missing: %v", name, err)
	}
}

func assertAbsent(t *testing.T, root, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s still exists: %v", name, err)
	}
}

func assertTouched(t *testing.T, path string) {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil || time.Since(stat.ModTime()) > time.Minute {
		t.Errorf("%s was not touched on reuse: %v", path, err)
	}
}

type markerCase struct {
	name     string
	file     string
	age      time.Duration
	wantGone bool
}

func markerCases() []markerCase {
	const marker = "failed-9.9.9-x-0123456789abcdef.json"
	return []markerCase{
		{
			name: "expired marker goes", file: marker,
			age: failureWindow + time.Minute, wantGone: true,
		},
		{name: "fresh marker stays", file: marker, age: time.Minute},
		{name: "five minute marker stays", file: marker, age: 5 * time.Minute},
		{name: "marker inside the window stays", file: marker, age: failureWindow - time.Minute},
		{name: "old file of another name stays", file: "failed-notes.txt", age: 2 * failureWindow},
		{name: "old file without the prefix stays", file: "other.json", age: 2 * failureWindow},
	}
}

func TestPruneCacheMarkers(t *testing.T) {
	t.Parallel()
	for _, testCase := range markerCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			path := filepath.Join(root, testCase.file)
			if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
				t.Fatal(err)
			}
			modified := time.Now().Add(-testCase.age)
			if err := os.Chtimes(path, modified, modified); err != nil {
				t.Fatal(err)
			}
			current := filepath.Join(root, "9.9.9-linux-amd64-aaaaaaaaaaaaaaaa")
			if err := pruneCache(current); err != nil {
				t.Fatalf("pruneCache: %v", err)
			}
			_, err := os.Stat(path)
			if gone := errors.Is(err, os.ErrNotExist); gone != testCase.wantGone {
				t.Errorf("%s gone = %v, want %v", testCase.file, gone, testCase.wantGone)
			}
		})
	}
}

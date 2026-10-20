package helperbin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func fakeToolchain(t *testing.T) (compiles, argv string) {
	t.Helper()

	dart := writeFakeDart(t)
	t.Setenv("PATH", filepath.Dir(dart)+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("FAKE_COMPILES", filepath.Join(t.TempDir(), "compiles.log"))
	return os.Getenv("FAKE_COMPILES"), fakeArgv(t)
}

func countVersionCalls(argv string) int {
	count := 0
	for _, line := range readLines(argv) {
		if strings.Contains(line, "--version") {
			count++
		}
	}
	return count
}

func TestLocatorBuildsAndLocatesOnce(t *testing.T) {
	t.Setenv("FAKE_FAIL", "")
	compiles, argv := fakeToolchain(t)
	locator := NewLocator(testFiles(), ".")
	const callers = 8
	found := make([]string, callers)
	var group sync.WaitGroup
	for index := range callers {
		group.Go(func() {
			exe, err := locator.Exe(context.Background())
			if err != nil {
				t.Errorf("caller %d: %v", index, err)
			}
			found[index] = exe
		})
	}
	group.Wait()
	for _, exe := range found {
		if exe == "" || exe != found[0] {
			t.Fatalf("exes %q, want one shared path", found)
		}
	}
	if got := readLines(compiles); len(got) != 1 {
		t.Errorf("compiles = %q, want one build for %d callers", got, callers)
	}
	if got := countVersionCalls(argv); got != 1 {
		t.Errorf("dart --version ran %d times, want 1", got)
	}
}

func TestLocatorRebuildsAPrunedExe(t *testing.T) {
	t.Setenv("FAKE_FAIL", "")
	compiles, _ := fakeToolchain(t)
	locator := NewLocator(testFiles(), ".")
	exe, err := locator.Exe(context.Background())
	if err != nil {
		t.Fatalf("first Exe: %v", err)
	}
	if err = os.RemoveAll(filepath.Dir(exe)); err != nil {
		t.Fatal(err)
	}
	again, err := locator.Exe(context.Background())
	if err != nil || again != exe {
		t.Fatalf("second Exe = %q, %v; want %q", again, err, exe)
	}
	if got := readLines(compiles); !slices.Equal(got, []string{"9.9.9", "9.9.9"}) {
		t.Errorf("compiles = %q, want a rebuild after the prune", got)
	}
}

func TestLocatorErrors(t *testing.T) {
	testCases := []struct {
		name      string
		path      func(t *testing.T) string
		cacheHome string
		home      string
		want      string
	}{
		{
			name: "no dart on PATH",
			path: func(t *testing.T) string { t.Helper(); return t.TempDir() },
			want: "locate dart",
		},
		{
			name: "no user cache dir",
			path: func(t *testing.T) string { t.Helper(); return filepath.Dir(writeFakeDart(t)) },
			want: "locate user cache dir",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("PATH", testCase.path(t))
			t.Setenv("XDG_CACHE_HOME", testCase.cacheHome)
			t.Setenv("HOME", testCase.home)
			_, err := NewLocator(testFiles(), ".").Exe(context.Background())
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("Exe error = %v, want containing %q", err, testCase.want)
			}
		})
	}
}

func TestLocatorHonorsContextDeadline(t *testing.T) {
	t.Setenv("FAKE_FAIL", "hang")
	fakeToolchain(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	_, err := NewLocator(testFiles(), ".").Exe(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Exe error = %v, want context deadline exceeded", err)
	}
	base := os.Getenv("XDG_CACHE_HOME")
	if _, marked := helperDir(currentDir(t, base)).loadMarker(); marked {
		t.Error("a parent deadline wrote a failure marker")
	}
}

func TestBoundBuild(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		elapsed  time.Duration
		wantDone bool
	}{
		{"just before the bound", BuildTimeout - time.Nanosecond, false},
		{"at the bound", BuildTimeout, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := boundBuild(context.Background())
				defer cancel()
				synctest.Sleep(testCase.elapsed)
				if isDone := ctx.Err() != nil; isDone != testCase.wantDone {
					t.Fatalf(
						"done = %v after %v, want %v",
						isDone,
						testCase.elapsed,
						testCase.wantDone,
					)
				}
				if testCase.wantDone && !errors.Is(context.Cause(ctx), errBuildBound) {
					t.Errorf("cause = %v, want errBuildBound", context.Cause(ctx))
				}
			})
		})
	}
}

func TestLocatorWaitHonorsContext(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		locator := NewLocator(testFiles(), ".")
		locator.slot <- struct{}{}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := locator.Exe(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Exe error = %v, want a deadline while waiting for the build slot", err)
		}
	})
}

package astcache

import (
	"go/token"
	"strings"
	"testing"
)

func TestParseAndCache(t *testing.T) {
	t.Parallel()

	cache := New()
	src := []byte("package main\n\nfunc main() {}\n")

	file, err := cache.Parse("test.go", src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if file == nil {
		t.Fatal("parsed file is nil")
	}
	if file.Name.Name != "main" {
		t.Errorf("package name = %q, want main", file.Name.Name)
	}

	cached := cache.Get("test.go")
	if cached == nil {
		t.Fatal("Get returned nil after Parse")
	}
	if cached != file {
		t.Error("Get returned a different pointer than Parse")
	}
}

func TestParseErrorCachedAcrossRepeatedParses(t *testing.T) {
	t.Parallel()

	cache := New()
	src := []byte("package main\n\nfunc broken(\n")

	var first error
	for call := range 3 {
		file, err := cache.Parse("broken.go", src)
		if err == nil {
			t.Fatalf("call %d: expected parse error for broken code", call)
		}
		if file != nil {
			t.Fatalf("call %d: parsed file = %v, want nil", call, file)
		}
		if call == 0 {
			first = err
		} else if err.Error() != first.Error() {
			t.Errorf("call %d: error = %v, want %v", call, err, first)
		}
	}

	files := 0
	cache.FileSet().Iterate(func(*token.File) bool {
		files++
		return true
	})
	if files != 1 {
		t.Errorf("FileSet holds %d files after 3 parses, want 1", files)
	}
}

func TestFileSetShared(t *testing.T) {
	t.Parallel()

	cache := New()
	_, _ = cache.Parse("a.go", []byte("package a\n"))
	_, _ = cache.Parse("b.go", []byte("package b\n"))

	fs := cache.FileSet()
	if fs == nil {
		t.Fatal("FileSet is nil")
	}
	count := 0
	fs.Iterate(func(*token.File) bool {
		count++
		return true
	})
	if count == 0 {
		t.Fatal("FileSet has no files")
	}
}

func TestErr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		path    string
		src     string
		parse   bool
		wantErr string
	}{
		{
			name: "parse failure is kept", path: "bad.go", src: "package\n",
			parse: true, wantErr: "parse bad.go: ",
		},
		{name: "parsed file has none", path: "ok.go", src: "package ok\n", parse: true},
		{name: "unknown path has none", path: "never.go"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cache := New()
			if testCase.parse {
				_, _ = cache.Parse(testCase.path, []byte(testCase.src))
			}
			err := cache.Err(testCase.path)
			if (err == nil) != (testCase.wantErr == "") {
				t.Fatalf("Err = %v, want prefix %q", err, testCase.wantErr)
			}
			if err != nil && !strings.HasPrefix(err.Error(), testCase.wantErr) {
				t.Fatalf("Err = %q, want prefix %q", err, testCase.wantErr)
			}
		})
	}
}

package vcs

import (
	"strings"
	"testing"
)

func TestBlobs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})
	writeFile(t, dir, "a.go", "package b\n")
	runGitOrFatal(t, dir, "add", "a.go")

	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	type blobCase struct {
		name    string
		side    Side
		path    string
		want    string
		wantErr bool
	}
	testCases := []blobCase{
		{name: "head blob", side: Head, path: "a.go", want: "package a"},
		{name: "staged blob", side: Staged, path: "a.go", want: "package b"},
		{name: "head blob missing", side: Head, path: "none.go", wantErr: true},
		{name: "staged blob missing", side: Staged, path: "none.go", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := repo.Blob(testCase.side, testCase.path)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestBlob_ErrorNamesTheSide(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	initRepo(t, dir)
	writeAndCommit(t, dir, "initial", map[string]string{"a.go": "package a\n"})
	repo, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	testCases := []struct {
		name string
		side Side
		want string
	}{
		{"head", Head, "read HEAD blob none.go:"},
		{"staged", Staged, "read staged blob none.go:"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := repo.Blob(testCase.side, "none.go")
			if err == nil || !strings.HasPrefix(err.Error(), testCase.want) {
				t.Fatalf("err = %v, want prefix %q", err, testCase.want)
			}
		})
	}
}

package vcs

import "testing"

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
		read    func(string) (string, error)
		path    string
		want    string
		wantErr bool
	}
	testCases := []blobCase{
		{name: "head blob", read: repo.HeadBlob, path: "a.go", want: "package a"},
		{name: "staged blob", read: repo.StagedBlob, path: "a.go", want: "package b"},
		{name: "head blob missing", read: repo.HeadBlob, path: "none.go", wantErr: true},
		{name: "staged blob missing", read: repo.StagedBlob, path: "none.go", wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := testCase.read(testCase.path)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if got != testCase.want {
				t.Fatalf("got %q, want %q", got, testCase.want)
			}
		})
	}
}

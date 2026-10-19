package movepurity

import (
	"maps"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type recordingRepo struct {
	fakeRepo
	calls map[string]int
}

func newRecordingRepo(head, staged map[string]string) recordingRepo {
	return recordingRepo{fakeRepo: newFakeRepo(head, staged), calls: map[string]int{}}
}

func (repo recordingRepo) HeadBlob(path string) (string, error) {
	repo.calls["HeadBlob "+path]++
	return repo.fakeRepo.HeadBlob(path)
}

func (repo recordingRepo) StagedBlob(path string) (string, error) {
	repo.calls["StagedBlob "+path]++
	return repo.fakeRepo.StagedBlob(path)
}

func (repo recordingRepo) HeadEntry(path string) (mode, id string, err error) {
	repo.calls["HeadEntry "+path]++
	return repo.fakeRepo.HeadEntry(path)
}

func (repo recordingRepo) StagedEntry(path string) (mode, id string, err error) {
	repo.calls["StagedEntry "+path]++
	return repo.fakeRepo.StagedEntry(path)
}

func (repo recordingRepo) HeadDir(dir string) ([]string, error) {
	repo.calls["HeadDir "+dir]++
	return repo.fakeRepo.HeadDir(dir)
}

func (repo recordingRepo) StagedDir(dir string) ([]string, error) {
	repo.calls["StagedDir "+dir]++
	return repo.fakeRepo.StagedDir(dir)
}

func TestSnapshot_ReadsEachObjectOnce(t *testing.T) {
	t.Parallel()

	tree := map[string]string{"a/x.go": "package a\n\nfunc F() {}\n", "a/y.go": "package a\n"}
	testCases := []struct {
		name string
		read func(snap *snapshot)
		want map[string]int
	}{
		{
			name: "blob and missing blob",
			read: func(snap *snapshot) {
				_, _ = snap.blob("a/x.go")
				_, _ = snap.blob("gone.go")
			},
			want: map[string]int{"HeadBlob a/x.go": 1, "HeadBlob gone.go": 1},
		},
		{
			name: "entry",
			read: func(snap *snapshot) { _, _ = snap.entryOf("a/x.go") },
			want: map[string]int{"HeadEntry a/x.go": 1},
		},
		{
			name: "dir",
			read: func(snap *snapshot) { _, _ = snap.dir("a") },
			want: map[string]int{"HeadDir a": 1},
		},
		{
			name: "parse reads the blob once",
			read: func(snap *snapshot) {
				_, _ = snap.parse("a/x.go")
				_, _ = snap.blob("a/x.go")
			},
			want: map[string]int{"HeadBlob a/x.go": 1},
		},
		{
			name: "clause and decls share one blob read",
			read: func(snap *snapshot) {
				snap.clause("a/x.go")
				snap.addDecls("a/x.go", "a", map[string]bool{})
				snap.addAnyDecls("a/x.go", map[string]bool{})
			},
			want: map[string]int{"HeadBlob a/x.go": 1},
		},
		{
			name: "package name lists the dir and reads each clause once",
			read: func(snap *snapshot) { snap.packageName("a") },
			want: map[string]int{"HeadDir a": 1, "HeadBlob a/x.go": 1},
		},
		{
			name: "module walks up once",
			read: func(snap *snapshot) { snap.module("a/b") },
			want: map[string]int{
				"HeadBlob a/b/go.mod": 1,
				"HeadBlob a/go.mod":   1,
				"HeadBlob go.mod":     1,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := newRecordingRepo(tree, tree)
			snap := newHeadSnapshot(repo)
			for range 3 {
				testCase.read(snap)
			}
			if !maps.Equal(repo.calls, testCase.want) {
				t.Fatalf("git calls = %v, want %v", repo.calls, testCase.want)
			}
		})
	}
}

func TestSnapshot_Clause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		src         string
		wantName    string
		wantIgnored bool
	}{
		{"clause", "package p\n\nfunc F() {}\n", "p", false},
		{"clause with broken body", "package q\n\nfunc (\n", "q", false},
		{"build ignored", "//go:build ignore\n\npackage r\n", "r", true},
		{"no clause", "func F() {}\n", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := newFakeRepo(map[string]string{"f.go": testCase.src}, map[string]string{})
			name, isIgnored := newHeadSnapshot(repo).clause("f.go")
			if name != testCase.wantName || isIgnored != testCase.wantIgnored {
				t.Fatalf("clause() = %q, %v, want %q, %v",
					name, isIgnored, testCase.wantName, testCase.wantIgnored)
			}
		})
	}
	if name, isIgnored := newHeadSnapshot(
		newFakeRepo(map[string]string{}, map[string]string{}),
	).clause("gone.go"); name != "" ||
		isIgnored {
		t.Fatalf("clause() of a missing blob = %q, %v, want empty", name, isIgnored)
	}
}

func TestSnapshot_PackageName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			"first production file in sorted order",
			map[string]string{"d/b.go": "package b\n", "d/a.go": "package a\n"},
			"a",
		},
		{
			"skips tests, ignored files and non-go files",
			map[string]string{
				"d/a_test.go": "package a_test\n",
				"d/b.go":      "//go:build ignore\n\npackage ignored\n",
				"d/c.txt":     "package c\n",
				"d/d.go":      "package d\n",
			},
			"d",
		},
		{
			"a first file without a clause is skipped",
			map[string]string{"d/a.go": "func F() {}\n", "d/b.go": "package b\n"},
			"b",
		},
		{"an empty dir", map[string]string{}, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			snap := newHeadSnapshot(newFakeRepo(testCase.files, map[string]string{}))
			if got := snap.packageName("d"); got != testCase.want {
				t.Fatalf("packageName() = %q, want %q", got, testCase.want)
			}
		})
	}
}

type failingDirRepo struct{ fakeRepo }

func (failingDirRepo) HeadDir(string) ([]string, error) { return nil, errMissing }

func TestSnapshot_PackageNameDirError(t *testing.T) {
	t.Parallel()

	repo := failingDirRepo{
		newFakeRepo(map[string]string{"d/a.go": "package a\n"}, map[string]string{}),
	}
	if got := newHeadSnapshot(repo).packageName("d"); got != "" {
		t.Fatalf("packageName() = %q, want empty when the directory cannot be read", got)
	}
}

type unsortedDirRepo struct{ fakeRepo }

func (unsortedDirRepo) HeadDir(string) ([]string, error) {
	return []string{"d/b.go", "d/a.go"}, nil
}

func TestSnapshot_PackageNameKeepsDirOrder(t *testing.T) {
	t.Parallel()

	repo := unsortedDirRepo{newFakeRepo(
		map[string]string{"d/a.go": "package a\n", "d/b.go": "package b\n"}, map[string]string{},
	)}
	snap := newHeadSnapshot(repo)
	if got := snap.packageName("d"); got != "a" {
		t.Fatalf("packageName() = %q, want a", got)
	}
	files, err := snap.dir("d")
	if err != nil || !slices.Equal(files, []string{"d/b.go", "d/a.go"}) {
		t.Fatalf("dir() = %q, %v, want the listing order untouched", files, err)
	}
}

func TestSnapshot_Decls(t *testing.T) {
	t.Parallel()

	const src = "package p\n\ntype T struct{}\n\nfunc (T) Method() {}\n\nfunc F() {}\n\n" +
		"const C, D = 1, 2\n\nvar (\n\tV = 1\n\t_ = 2\n)\n"
	testCases := []struct {
		name  string
		src   string
		pkg   string
		isAny bool
		want  []string
	}{
		{"every top-level name", src, "p", false, []string{"C", "D", "F", "T", "V", "_"}},
		{"another package clause", src, "q", false, nil},
		{"any package clause", src, "q", true, []string{"C", "D", "F", "T", "V", "_"}},
		{"a syntax error", "package p\n\nfunc F( {\n", "p", false, nil},
		{"a syntax error, any package", "package p\n\nfunc F( {\n", "", true, nil},
		{
			"a build-ignored file still counts",
			"//go:build ignore\n\npackage p\n\nfunc G() {}\n",
			"p", false,
			[]string{"G"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			snap := newHeadSnapshot(
				newFakeRepo(map[string]string{"f.go": testCase.src}, map[string]string{}),
			)
			set := map[string]bool{}
			if testCase.isAny {
				snap.addAnyDecls("f.go", set)
			} else {
				snap.addDecls("f.go", testCase.pkg, set)
			}
			if got := slices.Sorted(maps.Keys(set)); !slices.Equal(got, testCase.want) {
				t.Fatalf("decls = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestAnalyzer_ReadsEachGitObjectOnce(t *testing.T) {
	t.Parallel()

	head := map[string]string{
		"a/x.go": "package a\n\nfunc X() {}\n",
		"a/y.go": "package a\n\nfunc Y() { X() }\n",
		"a/z.go": "package a\n\nfunc Z() {}\n",
		"b/k.go": "package b\n\nfunc K() {}\n",
	}
	staged := maps.Clone(head)
	delete(staged, "a/x.go")
	delete(staged, "a/y.go")
	staged["b/x.go"] = "package b\n\nfunc X() {}\n"
	staged["b/y.go"] = "package b\n\nfunc Y() { X() }\n"
	staged["a/z.go"] = "package a\n\nfunc Z() { Z() }\n"
	repo := newRecordingRepo(head, staged)
	NewAnalyzer(repo).Analyze(&pass.Context{
		StagedRenames: []pass.Rename{pair("a/x.go", "b/x.go"), pair("a/y.go", "b/y.go")},
		ChangedFiles:  []pass.FileChange{{Path: "a/z.go", Staged: true}},
	})
	if len(repo.calls) == 0 {
		t.Fatal("the analyzer made no git calls")
	}
	for call, count := range repo.calls {
		if count != 1 {
			t.Errorf("%q ran %d times, want once per run", call, count)
		}
	}
}

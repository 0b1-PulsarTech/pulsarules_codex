package movepurity

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

type recordingRepo struct {
	fakeRepo
	calls map[string]int
}

func newRecordingRepo(head, staged map[string]string) recordingRepo {
	return recordingRepo{fakeRepo: newFakeRepo(head, staged), calls: map[string]int{}}
}

func (repo recordingRepo) Blob(side vcs.Side, path string) (string, error) {
	repo.calls[side.String()+" blob "+path]++
	return repo.fakeRepo.Blob(side, path)
}

func (repo recordingRepo) Entry(side vcs.Side, path string) (mode, id string, err error) {
	repo.calls[side.String()+" entry "+path]++
	return repo.fakeRepo.Entry(side, path)
}

func (repo recordingRepo) Dir(side vcs.Side, dir string) ([]string, error) {
	repo.calls[side.String()+" dir "+dir]++
	return repo.fakeRepo.Dir(side, dir)
}

func (repo recordingRepo) Files(side vcs.Side, dir string) ([]string, error) {
	repo.calls[side.String()+" tree "+dir]++
	return repo.fakeRepo.Files(side, dir)
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
			want: map[string]int{"HEAD blob a/x.go": 1, "HEAD blob gone.go": 1},
		},
		{
			name: "entry",
			read: func(snap *snapshot) { _, _ = snap.entryOf("a/x.go") },
			want: map[string]int{"HEAD entry a/x.go": 1},
		},
		{
			name: "dir",
			read: func(snap *snapshot) { _, _ = snap.dir("a") },
			want: map[string]int{"HEAD dir a": 1},
		},
		{
			name: "tree",
			read: func(snap *snapshot) {
				_, _ = snap.tree("a")
				_, _ = snap.tree("a")
			},
			want: map[string]int{"HEAD tree a": 1},
		},
		{
			name: "parse reads the blob once",
			read: func(snap *snapshot) {
				_, _ = snap.parse("a/x.go")
				_, _ = snap.blob("a/x.go")
			},
			want: map[string]int{"HEAD blob a/x.go": 1},
		},
		{
			name: "clause and decls share one blob read",
			read: func(snap *snapshot) {
				snap.clause("a/x.go")
				snap.addDecls("a/x.go", "a", map[string]bool{})
				snap.addAnyDecls("a/x.go", map[string]bool{})
			},
			want: map[string]int{"HEAD blob a/x.go": 1},
		},
		{
			name: "package name lists the dir and reads each clause once",
			read: func(snap *snapshot) { snap.packageName("a") },
			want: map[string]int{"HEAD dir a": 1, "HEAD blob a/x.go": 1},
		},
		{
			name: "module walks up once",
			read: func(snap *snapshot) { snap.module("a/b") },
			want: map[string]int{
				"HEAD blob a/b/go.mod": 1,
				"HEAD blob a/go.mod":   1,
				"HEAD blob go.mod":     1,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := newRecordingRepo(tree, tree)
			snap := newSnapshot(repo, vcs.Head)
			for range 3 {
				testCase.read(snap)
			}
			if !maps.Equal(repo.calls, testCase.want) {
				t.Fatalf("git calls = %v, want %v", repo.calls, testCase.want)
			}
		})
	}
}

func TestSnapshot_ReadsFromItsSide(t *testing.T) {
	t.Parallel()

	tree := map[string]string{"a/x.go": "package a\n"}
	testCases := []struct {
		name string
		side vcs.Side
		want map[string]int
	}{
		{
			name: "head",
			side: vcs.Head,
			want: map[string]int{"HEAD blob a/x.go": 1, "HEAD entry a/x.go": 1, "HEAD dir a": 1},
		},
		{
			name: "staged",
			side: vcs.Staged,
			want: map[string]int{
				"staged blob a/x.go": 1, "staged entry a/x.go": 1, "staged dir a": 1,
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := newRecordingRepo(tree, tree)
			snap := newSnapshot(repo, testCase.side)
			_, _ = snap.blob("a/x.go")
			_, _ = snap.entryOf("a/x.go")
			_, _ = snap.dir("a")
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
			name, isIgnored := newSnapshot(repo, vcs.Head).clause("f.go")
			if name != testCase.wantName || isIgnored != testCase.wantIgnored {
				t.Fatalf("clause() = %q, %v, want %q, %v",
					name, isIgnored, testCase.wantName, testCase.wantIgnored)
			}
		})
	}
	if name, isIgnored := newSnapshot(
		newFakeRepo(map[string]string{}, map[string]string{}), vcs.Head,
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

			snap := newSnapshot(newFakeRepo(testCase.files, map[string]string{}), vcs.Head)
			if got := snap.packageName("d"); got != testCase.want {
				t.Fatalf("packageName() = %q, want %q", got, testCase.want)
			}
		})
	}
}

type failingDirRepo struct{ fakeRepo }

func (failingDirRepo) Dir(vcs.Side, string) ([]string, error) { return nil, errMissing }

func TestSnapshot_PackageNameDirError(t *testing.T) {
	t.Parallel()

	repo := failingDirRepo{
		newFakeRepo(map[string]string{"d/a.go": "package a\n"}, map[string]string{}),
	}
	if got := newSnapshot(repo, vcs.Head).packageName("d"); got != "" {
		t.Fatalf("packageName() = %q, want empty when the directory cannot be read", got)
	}
}

type unsortedDirRepo struct{ fakeRepo }

func (unsortedDirRepo) Dir(vcs.Side, string) ([]string, error) {
	return []string{"d/b.go", "d/a.go"}, nil
}

func TestSnapshot_PackageNameKeepsDirOrder(t *testing.T) {
	t.Parallel()

	repo := unsortedDirRepo{newFakeRepo(
		map[string]string{"d/a.go": "package a\n", "d/b.go": "package b\n"}, map[string]string{},
	)}
	snap := newSnapshot(repo, vcs.Head)
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

			snap := newSnapshot(
				newFakeRepo(map[string]string{"f.go": testCase.src}, map[string]string{}), vcs.Head,
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

func embedderMovingWith(filesPerDir int) (head, staged map[string]string, renames []pass.Rename) {
	const oldGo, newGo = "internal/util/a.go", "internal/routing/a.go"
	src := embedSrc("util", "//go:embed static")
	head = map[string]string{oldGo: src}
	staged = map[string]string{newGo: embedSrc("routing", "//go:embed static")}
	renames = make([]pass.Rename, 1, 1+3*filesPerDir)
	renames[0] = pair(oldGo, newGo)
	for dir := range 3 {
		for file := range filesPerDir {
			name := fmt.Sprintf("static/d%d/f%d.txt", dir, file)
			head["internal/util/"+name], staged["internal/routing/"+name] = "x\n", "x\n"
			renames = append(renames, pair("internal/util/"+name, "internal/routing/"+name))
		}
	}
	return head, staged, renames
}

func TestSnapshot_EmbedderMoveListsIgnoreFileCount(t *testing.T) {
	t.Parallel()

	listings := func(filesPerDir int) (total, trees int) {
		head, staged, renames := embedderMovingWith(filesPerDir)
		repo := newRecordingRepo(head, staged)
		moves := newMoveContext(repo, renames)
		if got := moves.renameVerdict(renames[0]); got != verdictPure {
			t.Fatalf("renameVerdict() = %v, want pure", got)
		}
		for call, count := range repo.calls {
			if strings.Contains(call, " tree ") {
				trees += count
			}
			if strings.Contains(call, " tree ") || strings.Contains(call, " dir ") {
				total += count
			}
		}
		return total, trees
	}
	smallTotal, _ := listings(2)
	bigTotal, bigTrees := listings(40)
	if bigTotal != smallTotal || bigTrees != 2 {
		t.Fatalf(
			"listing calls: %d with 40 files per dir (%d trees), %d with 2; want equal, 2 trees",
			bigTotal,
			bigTrees,
			smallTotal,
		)
	}
}

package source

import "testing"

func TestSkipSet_SkipsDir(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		set  SkipSet
		dir  string
		want bool
	}{
		{"zero value falls back to defaults", SkipSet{}, "node_modules", true},
		{"zero value keeps go tree", SkipSet{}, "internal", false},
		{"configured empty set skips nothing", NewSkipSet(nil), "node_modules", false},
		{"configured set matches its own entries", NewSkipSet([]string{"dist"}), "dist", true},
		{
			"configured set does not fall back to defaults",
			NewSkipSet([]string{"dist"}),
			"node_modules",
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.set.SkipsDir(testCase.dir); got != testCase.want {
				t.Errorf("SkipsDir(%q) = %v, want %v", testCase.dir, got, testCase.want)
			}
		})
	}
}

func TestSkipSet_SkipsPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		set  SkipSet
		path string
		want bool
	}{
		{"file inside node_modules is skipped", SkipSet{}, "node_modules/pkg/index.js", true},
		{
			"file inside a nested vendored dir is skipped",
			SkipSet{},
			"apps/web/node_modules/pkg/index.js",
			true,
		},
		{"file directly under root is not skipped", SkipSet{}, "main.go", false},
		{
			"a file merely named vendor.go is not skipped: the rule prunes " +
				"directories, not filenames",
			SkipSet{},
			"internal/vendor.go",
			false,
		},
		{"a real vendor directory is skipped", SkipSet{}, "vendor/pkg/pkg.go", true},
		{
			"configured set never consults the default dirs",
			NewSkipSet([]string{"dist"}),
			"vendor/pkg/pkg.go",
			false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.set.SkipsPath(testCase.path); got != testCase.want {
				t.Errorf("SkipsPath(%q) = %v, want %v", testCase.path, got, testCase.want)
			}
		})
	}
}

func TestDefaultSkipDirs_ClonedEachCall(t *testing.T) {
	t.Parallel()

	first := DefaultSkipDirs()
	first[0] = "mutated"
	second := DefaultSkipDirs()
	if second[0] == "mutated" {
		t.Fatal("DefaultSkipDirs must clone, not share, its backing slice")
	}
}

func TestGitSkipDirs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		dir  string
		want bool
	}{
		{"target is real source under git", "target", false},
		{"build is real source under git", "build", false},
		{"generated is real source under git", "generated", false},
		{"dist is real source under git", "dist", false},
		{"out is real source under git", "out", false},
		{"coverage is real source under git", "coverage", false},
		{"vendor stays skipped", "vendor", true},
		{"node_modules stays skipped", "node_modules", true},
		{"dot git stays skipped", ".git", true},
	}
	set := NewSkipSet(GitSkipDirs())
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := set.SkipsDir(testCase.dir); got != testCase.want {
				t.Errorf("SkipsDir(%q) = %v, want %v", testCase.dir, got, testCase.want)
			}
		})
	}
}

func TestGitSkipDirs_ClonedEachCall(t *testing.T) {
	t.Parallel()

	first := GitSkipDirs()
	first[0] = "mutated"
	if GitSkipDirs()[0] == "mutated" {
		t.Fatal("GitSkipDirs must not share a backing slice")
	}
}

package detect

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDetect(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		files []string
		want  []string
	}{
		{name: "empty tree detects nothing", files: nil, want: nil},
		{name: "go module", files: []string{"go.mod"}, want: []string{"go"}},
		{
			name:  "go workspace marker also counts as go",
			files: []string{"go.work"},
			want:  []string{"go"},
		},
		{
			name: "dart + svelte + typescript monorepo",
			files: []string{
				"pubspec.yaml",
				"web/svelte.config.js",
				"web/tsconfig.json",
			},
			want: []string{"dart", "svelte", "typescript"},
		},
		{
			name: "a vendored marker under node_modules does not count",
			files: []string{
				"go.mod",
				"node_modules/some-pkg/package.json",
			},
			want: []string{"go"},
		},
		{
			name: "a vendored marker under vendor does not count",
			files: []string{
				"go.mod",
				"vendor/some/pubspec.yaml",
			},
			want: []string{"go"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			for _, rel := range testCase.files {
				writeMarker(t, root, rel)
			}

			got, err := scan(root, DefaultMaxDepth)
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if diff := got.Strings(); !slices.Equal(diff, testCase.want) {
				t.Errorf("Detect(%v) = %v, want %v", testCase.files, diff, testCase.want)
			}
		})
	}
}

func TestDetect_MaxDepth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		marker   string
		maxDepth int
		want     []string
	}{
		{
			name:     "root marker at depth 0 within cap",
			marker:   "go.mod",
			maxDepth: 0,
			want:     []string{"go"},
		},
		{
			name:     "marker at exactly the cap is found",
			marker:   "a/b/c/go.mod",
			maxDepth: 3,
			want:     []string{"go"},
		},
		{
			name:     "marker one level past the cap is not found",
			marker:   "a/b/c/d/go.mod",
			maxDepth: 3,
			want:     nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			writeMarker(t, root, testCase.marker)

			got, err := scan(root, testCase.maxDepth)
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if diff := got.Strings(); !slices.Equal(diff, testCase.want) {
				t.Errorf("Detect(%q, maxDepth=%d) = %v, want %v",
					testCase.marker, testCase.maxDepth, diff, testCase.want)
			}
		})
	}
}

func writeMarker(tb testing.TB, root, rel string) {
	tb.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		tb.Fatalf("mkdir %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o600); err != nil {
		tb.Fatalf("write %q: %v", path, err)
	}
}

func TestDetect_UnreadableRoot(t *testing.T) {
	t.Parallel()

	got, err := scan(filepath.Join(t.TempDir(), "no-such-dir"), DefaultMaxDepth)
	if err == nil {
		t.Fatalf("Detect on a missing root returned no error, set = %v", got.Strings())
	}
	if !got.Empty() {
		t.Fatalf("set = %v, want empty on error", got.Strings())
	}
}

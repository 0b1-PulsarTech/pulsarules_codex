package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type containCase struct {
	name    string
	rel     string
	wantErr bool
}

var containCases = []containCase{
	{"plain child", "a.txt", false},
	{"nested child", "sub/a.txt", false},
	{"child that does not exist yet", "sub/new.txt", false},
	{"dot segment resolves back inside", "sub/../a.txt", false},
	{"parent escape", "../outside.txt", true},
	{"deep parent escape", "sub/../../outside.txt", true},
}

func TestResolveInside(t *testing.T) {
	t.Parallel()

	for _, testCase := range containCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "sub"), 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			got, err := ResolveInside(root, filepath.Join(root, testCase.rel))
			if testCase.wantErr {
				if !errors.Is(err, ErrOutsideRoot) {
					t.Fatalf("err = %v, want ErrOutsideRoot", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveInside: %v", err)
			}
			if !filepath.IsAbs(got) {
				t.Errorf("got %q, want an absolute path", got)
			}
		})
	}
}

// TestResolveInside_SymlinkEscaping is the case a naive prefix check misses: the
// path string sits under root, but the link it names does not.
func TestResolveInside_SymlinkEscaping(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	link := filepath.Join(root, "looks-inside.txt")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := ResolveInside(root, link); !errors.Is(err, ErrOutsideRoot) {
		t.Fatalf("err = %v, want ErrOutsideRoot", err)
	}
}

// TestResolveInside_SymlinkStayingInside proves the check rejects by destination,
// not by the mere presence of a link.
func TestResolveInside_SymlinkStayingInside(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target := filepath.Join(root, "real.txt")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	link := filepath.Join(root, "alias.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := ResolveInside(root, link); err != nil {
		t.Errorf("ResolveInside rejected a link that stays inside: %v", err)
	}
}

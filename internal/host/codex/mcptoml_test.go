package codex

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/mcpserver"
)

func readText(tb testing.TB, path string) string {
	tb.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return string(body)
}

func TestWriteMCP_ThenRemove(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
		wantAfterRm string
	}{
		{name: "new file", seed: ""},
		{
			name: "user content kept", seed: "[mcp_servers.goland]\ncommand = \"g\"\n",
			wantAfterRm: "[mcp_servers.goland]\ncommand = \"g\"\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(t.TempDir(), ".codex")
			path := filepath.Join(dir, ConfigFile)
			if testCase.seed != "" {
				seedConfig(t, dir, testCase.seed)
			}
			wired, clashed, err := WriteMCP(dir, []mcpserver.Server{goplsServer()})
			if err != nil || !slices.Equal(wired, []string{"gopls"}) || clashed != nil {
				t.Fatalf("WriteMCP = %v, %v, %v", wired, clashed, err)
			}
			first := readText(t, path)
			if _, _, err = WriteMCP(dir, []mcpserver.Server{goplsServer()}); err != nil {
				t.Fatalf("second WriteMCP: %v", err)
			}
			if again := readText(t, path); again != first {
				t.Errorf("a rewrite changed the file:\n%s\nvs\n%s", first, again)
			}
			assertMode(t, path, testCase.seed)
			assertRemoved(t, dir, testCase.wantAfterRm)
		})
	}
}

func seedConfig(tb testing.TB, dir, body string) {
	tb.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(body), 0o644); err != nil {
		tb.Fatal(err)
	}
}

func assertRemoved(tb testing.TB, dir, wantText string) {
	tb.Helper()
	path := filepath.Join(dir, ConfigFile)
	changed, err := RemoveMCP(dir)
	if err != nil || !changed {
		tb.Fatalf("RemoveMCP = %v, %v", changed, err)
	}
	if wantText == "" {
		if _, statErr := os.Stat(path); !errors.Is(statErr, fs.ErrNotExist) {
			tb.Errorf("a file we created must go, stat err = %v", statErr)
		}
	} else if got := readText(tb, path); got != wantText {
		tb.Errorf("after remove = %q, want %q", got, wantText)
	}
	if changed, err = RemoveMCP(dir); err != nil || changed {
		tb.Errorf("second RemoveMCP = %v, %v, want a quiet no-op", changed, err)
	}
}

func assertMode(tb testing.TB, path, seed string) {
	tb.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		tb.Fatal(err)
	}
	want := fs.FileMode(0o600)
	if seed != "" {
		want = 0o644
	}
	if stat.Mode().Perm() != want {
		tb.Errorf("mode = %v, want %v", stat.Mode().Perm(), want)
	}
}

func TestWriteMCP_ClashLeavesTheFileAlone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFile)
	const seed = "[mcp_servers.gopls]\ncommand = \"mine\"\n"
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	wired, clashed, err := WriteMCP(dir, []mcpserver.Server{goplsServer()})
	if err != nil || wired != nil || !slices.Equal(clashed, []string{"gopls"}) {
		t.Fatalf("WriteMCP = %v, %v, %v", wired, clashed, err)
	}
	if got := readText(t, path); got != seed {
		t.Errorf("file = %q, want it untouched", got)
	}
}

func TestWriteMCP_NothingToWriteCreatesNothing(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), ".codex")
	if _, _, err := WriteMCP(dir, nil); err != nil {
		t.Fatalf("WriteMCP: %v", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("no servers must create no directory, stat err = %v", err)
	}
}

func TestWriteMCP_Failures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		prepare func(t *testing.T, dir string)
		wantErr error
	}{
		{
			name: "unterminated block",
			prepare: func(t *testing.T, dir string) {
				t.Helper()
				body := blockBegin + "\n[mcp_servers.gopls]\n"
				seedConfig(t, dir, body)
			},
			wantErr: ErrUnterminatedBlock,
		},
		{
			name: "config is a directory",
			prepare: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, ConfigFile), 0o750); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			testCase.prepare(t, dir)
			_, _, err := WriteMCP(dir, []mcpserver.Server{goplsServer()})
			if err == nil {
				t.Fatal("WriteMCP succeeded, want an error")
			}
			if testCase.wantErr != nil && !errors.Is(err, testCase.wantErr) {
				t.Errorf("err = %v, want %v", err, testCase.wantErr)
			}
			if _, removeErr := RemoveMCP(dir); removeErr == nil {
				t.Error("RemoveMCP succeeded, want an error")
			}
		})
	}
}

func TestWriteMCP_DirectoryIsAFile(t *testing.T) {
	t.Parallel()

	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(blocker, ".codex")
	if _, _, err := WriteMCP(dir, []mcpserver.Server{goplsServer()}); err == nil {
		t.Error("WriteMCP succeeded under a file, want an error")
	}
}

func TestMCP_SymlinkedConfigIsLeftAlone(t *testing.T) {
	t.Parallel()

	for _, withTarget := range []bool{true, false} {
		t.Run(fmt.Sprintf("target exists %v", withTarget), func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			target := filepath.Join(dir, "target.toml")
			if withTarget {
				if err := os.WriteFile(target, []byte(goplsBlock), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(dir, ConfigFile)
			if err := os.Symlink(target, path); err != nil {
				t.Fatal(err)
			}
			_, _, writeErr := WriteMCP(dir, []mcpserver.Server{goplsServer()})
			_, removeErr := RemoveMCP(dir)
			if !errors.Is(writeErr, ErrSymlink) || !errors.Is(removeErr, ErrSymlink) {
				t.Errorf("errors = %v, %v, want ErrSymlink", writeErr, removeErr)
			}
			wantTarget := ""
			if withTarget {
				wantTarget = goplsBlock
			}
			assertLinkAndTarget(t, path, target, wantTarget)
		})
	}
}

func assertLinkAndTarget(tb testing.TB, path, target, wantTarget string) {
	tb.Helper()
	if stat, err := os.Lstat(path); err != nil || stat.Mode()&os.ModeSymlink == 0 {
		tb.Errorf("the symlink must survive, got %v, %v", stat, err)
	}
	if wantTarget != "" && readText(tb, target) != wantTarget {
		tb.Errorf("the target was written through")
	}
}

func TestRenderMCP(t *testing.T) {
	t.Parallel()

	if got := RenderMCP([]mcpserver.Server{goplsServer()}); got != goplsBlock {
		t.Errorf("RenderMCP = %q, want %q", got, goplsBlock)
	}
}

func TestClashFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if got, err := ClashFound(dir, "gopls"); err != nil || got != "" {
		t.Errorf("no config: ClashFound = %q, %v, want empty", got, err)
	}
	seed := "[mcp_servers]\ngopls = { command = \"mine\" }\n"
	if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := ClashFound(dir, "gopls"); err != nil || got != "gopls under [mcp_servers]" {
		t.Errorf("ClashFound = %q, %v", got, err)
	}
}

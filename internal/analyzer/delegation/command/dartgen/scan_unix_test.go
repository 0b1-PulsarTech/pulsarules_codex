//go:build unix

package dartgen

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"
)

func TestMissingSkipsWhatIsNotAPlainFile(t *testing.T) {
	t.Parallel()

	const deadline = 5 * time.Second
	root, _ := writeTree(t, map[string]string{"lib/real.dart": "export 'c.g.dart';\n"})
	outside := filepath.Join(t.TempDir(), "outside.dart")
	if err := os.WriteFile(outside, []byte("export 'x.g.dart';\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "lib", "escape.dart")); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := syscall.Mkfifo(filepath.Join(root, "lib", "pipe.dart"), 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	listed := []string{"lib/escape.dart", "lib/pipe.dart", "lib/real.dart"}
	done := make(chan []string, 1)
	go func() { done <- Missing(root, "", listed) }()
	ctx, cancel := context.WithTimeout(t.Context(), deadline)
	defer cancel()
	select {
	case got := <-done:
		if want := []string{"lib/c.g.dart"}; !slices.Equal(got, want) {
			t.Fatalf("Missing = %v, want %v", got, want)
		}
	case <-ctx.Done():
		t.Fatalf("Missing still blocked after %v on a fifo", deadline)
	}
}

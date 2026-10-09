//go:build unix

package hook

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestEmitPostEdit_ThrowNoticeSkipsAFIFO(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "pipe.ts")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("fifos unavailable: %v", err)
	}
	disp, out := dispatchCapture(Deps{Index: preEditIndex(t), ProjectDir: root})
	payload := postEditPayload(t, path, uniqueSessionID(t))

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = disp.Dispatch("post-edit", payload)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("post-edit blocked reading a FIFO")
	}
	if out.Len() != 0 {
		t.Errorf("output = %q, want silence", out.String())
	}
}

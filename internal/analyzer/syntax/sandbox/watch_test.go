package sandbox

import (
	"bytes"
	"io"
	"testing"
	"testing/synctest"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/syntax"
)

func TestInUse(t *testing.T) {
	t.Parallel()

	if inUse() == 0 {
		t.Fatal("no memory in use")
	}
}

func TestWatchFiresOnTheCeiling(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		exited := make(chan int, 1)
		release := make(chan struct{})
		hold := make(chan struct{})
		handle := func(Request) *syntax.File {
			<-release
			return &syntax.File{}
		}
		input := io.MultiReader(bytes.NewReader(seedFrames(t, "a.ts")), gate{hold})
		done := make(chan int)
		go func() {
			done <- serve(input, io.Discard, handle, serveLimits{Ceiling: 1, Exit: func(code int) {
				select {
				case exited <- code:
				default:
				}
			}})
		}()
		select {
		case code := <-exited:
			if code != exitMemory {
				t.Errorf("exit %d, want %d", code, exitMemory)
			}
		case <-time.After(time.Second):
			t.Error("the watchdog never fired")
		}
		close(release)
		synctest.Wait()
		close(hold)
		if code := <-done; code != 0 {
			t.Errorf("serve %d", code)
		}
	})
}

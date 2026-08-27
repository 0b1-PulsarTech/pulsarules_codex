//go:build unix

package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func KillGroupOnCancel(execCmd *exec.Cmd) {
	execCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// The default cancel kills only the direct child; a surviving grandchild holds the
	// pipes open and Wait would block, so Command.Timeout would bound nothing.
	execCmd.Cancel = func() error {
		killErr := syscall.Kill(-execCmd.Process.Pid, syscall.SIGKILL)
		switch {
		case killErr == nil:
			return nil
		case errors.Is(killErr, syscall.ESRCH):
			// Wait ignores os.ErrProcessDone, so the real exit surfaces.
			return os.ErrProcessDone
		default:
			return fmt.Errorf("kill process group: %w", killErr)
		}
	}
}

//go:build !unix

package process

import "os/exec"

func KillGroupOnCancel(_ *exec.Cmd) {}

//go:build !unix

package execx

import "os/exec"

func wireProcessGroupKill(_ *exec.Cmd) {}

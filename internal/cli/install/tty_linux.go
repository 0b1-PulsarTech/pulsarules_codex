//go:build linux

package install

import (
	"os"
	"syscall"
	"unsafe"
)

// why: os.ModeCharDevice (via os.Stdin.Stat) is not enough to detect a real
// terminal - /dev/null is a character device too, so a CI run redirecting
// stdin from /dev/null would misreport as interactive. TCGETS only succeeds
// on an actual tty fd, which is what "is stdin a terminal" needs to mean.
const ioctlGetTermios = 0x5401 // TCGETS

func stdinIsTerminal() bool {
	var termios syscall.Termios
	//nolint:gosec // ioctl output buffer, not a conversion of arbitrary data.
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		os.Stdin.Fd(),
		ioctlGetTermios,
		uintptr(unsafe.Pointer(&termios)),
	)
	return errno == 0
}

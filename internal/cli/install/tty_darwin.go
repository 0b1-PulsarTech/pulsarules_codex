//go:build darwin

package install

import (
	"os"
	"syscall"
	"unsafe"
)

// why: see tty_linux.go - TIOCGETA is darwin's termios-get ioctl request.
const ioctlGetTermios = 0x40487413 // TIOCGETA

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

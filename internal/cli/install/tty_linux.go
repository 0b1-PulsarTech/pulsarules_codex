//go:build linux

package install

import (
	"os"
	"syscall"
	"unsafe"
)

func stdinIsTerminal() bool {
	var termios syscall.Termios
	// ModeCharDevice would also match /dev/null; TCGETS succeeds only on a real tty fd.
	//nolint:gosec // ioctl output buffer, not a conversion of arbitrary data.
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		os.Stdin.Fd(),
		syscall.TCGETS,
		uintptr(unsafe.Pointer(&termios)),
	)
	return errno == 0
}

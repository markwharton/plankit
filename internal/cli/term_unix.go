//go:build unix

package cli

import (
	"syscall"
	"unsafe"
)

// termSize reports the terminal width for fd. A false return means fd is
// not a terminal; the ioctl doubles as the TTY probe.
func termSize(fd uintptr) (int, bool) {
	var ws struct{ Row, Col, X, Y uint16 }
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&ws)))
	if errno != 0 || ws.Col == 0 {
		return 0, false
	}
	return int(ws.Col), true
}

// enableVT is a no-op on unix; ANSI is native.
func enableVT(uintptr) bool { return true }

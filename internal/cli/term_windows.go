//go:build windows

package cli

import (
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

const enableVTProcessing = 0x0004

type coord struct{ X, Y int16 }
type smallRect struct{ Left, Top, Right, Bottom int16 }
type consoleInfo struct {
	Size      coord
	Cursor    coord
	Attrs     uint16
	Window    smallRect
	MaxWindow coord
}

// termSize reports the console width for fd. A false return means fd is
// not a console; the query doubles as the TTY probe.
func termSize(fd uintptr) (int, bool) {
	var info consoleInfo
	r, _, _ := procGetConsoleInfo.Call(fd, uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return 0, false
	}
	w := int(info.Window.Right-info.Window.Left) + 1
	if w <= 0 {
		return 0, false
	}
	return w, true
}

// enableVT turns on virtual terminal processing so ANSI escapes render.
// A false return means the console refused; the caller falls back to
// undecorated output rather than printing escape litter.
func enableVT(fd uintptr) bool {
	var mode uint32
	if r, _, _ := procGetConsoleMode.Call(fd, uintptr(unsafe.Pointer(&mode))); r == 0 {
		return false
	}
	if mode&enableVTProcessing != 0 {
		return true
	}
	r, _, _ := procSetConsoleMode.Call(fd, uintptr(mode|enableVTProcessing))
	return r != 0
}

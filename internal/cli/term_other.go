//go:build !unix && !windows

package cli

func termSize(uintptr) (int, bool) { return 0, false }
func enableVT(uintptr) bool        { return false }

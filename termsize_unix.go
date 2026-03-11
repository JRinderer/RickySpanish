//go:build darwin || linux || freebsd

package main

import (
	"syscall"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func tuiWidth() int {
	ws := &winsize{}
	syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		syscall.TIOCGWINSZ,
		uintptr(unsafe.Pointer(ws)),
	)
	w := int(ws.Col)
	if w < 40 {
		return 72
	}
	return w
}

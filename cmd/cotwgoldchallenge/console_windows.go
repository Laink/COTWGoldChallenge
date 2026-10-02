package main

import (
	"syscall"
	"unsafe"
)

// setupConsole switches the console to UTF-8 and turns on its colours when it has them.
func setupConsole() {
	k := syscall.NewLazyDLL("kernel32.dll")
	k.NewProc("SetConsoleOutputCP").Call(65001)
	k.NewProc("SetConsoleCP").Call(65001)
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	if r, _, _ := k.NewProc("GetConsoleMode").Call(uintptr(h), uintptr(unsafe.Pointer(&mode))); r == 0 {
		return
	}
	const virtualTerminal = 0x0004 // ENABLE_VIRTUAL_TERMINAL_PROCESSING
	r, _, _ := k.NewProc("SetConsoleMode").Call(uintptr(h), uintptr(mode|virtualTerminal))
	colors = r != 0
}

// setTitle sets the title of the console window.
func setTitle(s string) {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return
	}
	syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleTitleW").Call(uintptr(unsafe.Pointer(p)))
}

package hotkey

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32    = syscall.NewLazyDLL("user32.dll")
	kernel32  = syscall.NewLazyDLL("kernel32.dll")
	keyState  = user32.NewProc("GetAsyncKeyState")
	fgWindow  = user32.NewProc("GetForegroundWindow")
	windowPID = user32.NewProc("GetWindowThreadProcessId")
	openProc  = kernel32.NewProc("OpenProcess")
	imageName = kernel32.NewProc("QueryFullProcessImageNameW")
	closeH    = kernel32.NewProc("CloseHandle")
)

// Supported tells whether keys can be watched on this system.
const Supported = true

// Down tells whether the key is held.
func Down(vk int) bool {
	r, _, _ := keyState.Call(uintptr(vk))
	return r&0x8000 != 0
}

// GameInFront tells whether the window in front belongs to the game.
func GameInFront() bool {
	w, _, _ := fgWindow.Call()
	if w == 0 {
		return false
	}
	var pid uint32
	windowPID.Call(w, uintptr(unsafe.Pointer(&pid)))
	h, _, _ := openProc.Call(0x1000, 0, uintptr(pid)) // PROCESS_QUERY_LIMITED_INFORMATION
	if h == 0 {
		return false
	}
	defer closeH.Call(h)
	buf := make([]uint16, 1024)
	n := uint32(len(buf))
	if r, _, _ := imageName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); r == 0 {
		return false
	}
	name := strings.ToLower(filepath.Base(syscall.UTF16ToString(buf[:n])))
	return strings.HasPrefix(name, "thehuntercotw")
}

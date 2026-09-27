package game

import "syscall"

// isJunction tells whether path is a reparse point (junction or symbolic link).
func isJunction(path string) bool {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	a, err := syscall.GetFileAttributes(p)
	return err == nil && a&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}

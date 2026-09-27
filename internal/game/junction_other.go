//go:build !windows

package game

func isJunction(path string) bool { return false }

// Package hotkey watches a key while the game window is in front, without taking it from the game.
package hotkey

import (
	"strconv"
	"strings"
)

// Parse reads a key setting: "<virtual-key code>:<name>" as written by the settings page, or a key
// name of Keys. It returns the code and the name, or 0.
func Parse(s string) (int, string) {
	if c, name, ok := strings.Cut(s, ":"); ok {
		if vk, err := strconv.Atoi(c); err == nil && vk > 0 && vk < 256 {
			return vk, name
		}
		return 0, ""
	}
	return VK(s), s
}

// Keys lists key names accepted in the settings file, with their Windows virtual-key code.
var Keys = []struct {
	Name string
	VK   int
}{
	{"F1", 0x70}, {"F2", 0x71}, {"F3", 0x72}, {"F4", 0x73}, {"F5", 0x74}, {"F6", 0x75},
	{"F7", 0x76}, {"F8", 0x77}, {"F9", 0x78}, {"F10", 0x79}, {"F11", 0x7A}, {"F12", 0x7B},
	{"Insert", 0x2D}, {"Delete", 0x2E}, {"Home", 0x24}, {"End", 0x23}, {"PageUp", 0x21}, {"PageDown", 0x22},
	{"Pause", 0x13}, {"ScrollLock", 0x91},
}

// VK returns the virtual-key code of a key name, or 0.
func VK(name string) int {
	for _, k := range Keys {
		if k.Name == name {
			return k.VK
		}
	}
	return 0
}

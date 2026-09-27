//go:build !windows

package hotkey

// Supported tells whether keys can be watched on this system.
const Supported = false

// Down tells whether the key is held.
func Down(vk int) bool { return false }

// GameInFront tells whether the window in front belongs to the game.
func GameInFront() bool { return false }

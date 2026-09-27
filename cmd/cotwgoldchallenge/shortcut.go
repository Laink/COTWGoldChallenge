package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/hotkey"
	"github.com/Laink/COTWGoldChallenge/internal/settings"
)

// ToggleFile tells the mod whether the overlay is shown: "<beat> <0|1>", rewritten every second.
// When the beat stops changing, the mod shows the overlay again.
const ToggleFile = "ui/cotwgc_toggle.txt"

// shortcut watches the overlay key in the background while the program runs.
var shortcut struct {
	sync.Mutex
	stop chan struct{}
	done chan struct{}
	key  string // name of the key, "" when none is set
}

// shortcutOn tells whether the shortcut runs, and its key.
func shortcutOn() (bool, string) {
	shortcut.Lock()
	defer shortcut.Unlock()
	return shortcut.stop != nil, shortcut.key
}

// toggleShortcut starts the shortcut, or stops it when it runs.
func toggleShortcut(ui *UI, dropzone, lang string) error {
	if on, _ := shortcutOn(); on {
		stopShortcut()
		return nil
	}
	if !hotkey.Supported {
		return errors.New(ui.t("shortcut_windows"))
	}
	stop, done := make(chan struct{}), make(chan struct{})
	vk, name := hotkey.Parse(settings.Read(dropzone, lang)["toggleKey"])
	if vk == 0 {
		name = ""
	}
	shortcut.Lock()
	shortcut.stop, shortcut.done, shortcut.key = stop, done, name
	shortcut.Unlock()
	go watchKey(dropzone, lang, stop, done)
	return nil
}

func stopShortcut() {
	shortcut.Lock()
	stop, done := shortcut.stop, shortcut.done
	shortcut.stop, shortcut.done = nil, nil
	shortcut.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}
}

// watchKey toggles the overlay when the key is pressed while the game is in front. The key is
// read again from the settings every 3 s.
func watchKey(dropzone, lang string, stop, done chan struct{}) {
	defer close(done)
	path := filepath.Join(dropzone, filepath.FromSlash(ToggleFile))
	defer os.Remove(path)
	shown, beat := true, 0
	write := func() {
		beat++
		state := "1"
		if !shown {
			state = "0"
		}
		os.WriteFile(path, []byte(strconv.Itoa(beat)+" "+state+"\n"), 0o644)
	}
	vk := 0
	readKey := func() {
		var name string
		vk, name = hotkey.Parse(settings.Read(dropzone, lang)["toggleKey"])
		if vk == 0 {
			name = ""
		}
		shortcut.Lock()
		shortcut.key = name
		shortcut.Unlock()
	}
	readKey()
	write()
	tick := time.NewTicker(30 * time.Millisecond)
	defer tick.Stop()
	held := false
	last := time.Now()
	for n := 1; ; n++ {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		down := vk != 0 && hotkey.Down(vk)
		if down && !held && hotkey.GameInFront() {
			shown = !shown
			write()
			last = time.Now()
		}
		held = down
		if time.Since(last) >= time.Second {
			write()
			last = time.Now()
		}
		if n%100 == 0 {
			readKey()
		}
	}
}

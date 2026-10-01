package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/hotkey"
	"github.com/Laink/COTWGoldChallenge/internal/settings"
)

// ToggleFile tells the mod whether the overlay is shown and whether the names key is held:
// "<beat> <0|1> <0|1>", rewritten every second. When the beat stops changing, the mod shows the
// overlay again.
const ToggleFile = "ui/cotwgc_toggle.txt"

// shortcut watches the overlay keys in the background while the program runs.
var shortcut struct {
	sync.Mutex
	stop  chan struct{}
	done  chan struct{}
	key   string // name of the show/hide key, "" when none is set
	names string // name of the key held to show the names, "" when none is set
}

// shortcutOn tells whether the shortcut runs, and its keys.
func shortcutOn() (bool, string, string) {
	shortcut.Lock()
	defer shortcut.Unlock()
	return shortcut.stop != nil, shortcut.key, shortcut.names
}

// keysLabel describes the keys set, for the menu.
func keysLabel(ui *UI, key, names string) string {
	var parts []string
	if key != "" {
		parts = append(parts, fmt.Sprintf(ui.t("keys_toggle"), key))
	}
	if names != "" {
		parts = append(parts, fmt.Sprintf(ui.t("keys_names"), names))
	}
	return strings.Join(parts, ", ")
}

// readKeys returns the keys of the settings: show/hide, then names.
func readKeys(dropzone, lang string) (int, string, int, string) {
	s := settings.Read(dropzone, lang)
	vk, name := hotkey.Parse(s["toggleKey"])
	if vk == 0 {
		name = ""
	}
	nvk, nname := hotkey.Parse(s["namesKey"])
	if nvk == 0 {
		nname = ""
	}
	return vk, name, nvk, nname
}

// toggleShortcut starts the shortcut, or stops it when it runs.
func toggleShortcut(ui *UI, dropzone, lang string) error {
	if on, _, _ := shortcutOn(); on {
		stopShortcut()
		return nil
	}
	if !hotkey.Supported {
		return errors.New(ui.t("shortcut_windows"))
	}
	stop, done := make(chan struct{}), make(chan struct{})
	_, name, _, names := readKeys(dropzone, lang)
	shortcut.Lock()
	shortcut.stop, shortcut.done, shortcut.key, shortcut.names = stop, done, name, names
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

// watchKey toggles the overlay when the key is pressed while the game is in front, and shows the
// names while the names key is held. The keys are read again from the settings every 3 s.
func watchKey(dropzone, lang string, stop, done chan struct{}) {
	defer close(done)
	path := filepath.Join(dropzone, filepath.FromSlash(ToggleFile))
	defer os.Remove(path)
	shown, names, beat := true, false, 0
	bit := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	write := func() {
		beat++
		os.WriteFile(path, []byte(strconv.Itoa(beat)+" "+bit(shown)+" "+bit(names)+"\n"), 0o644)
	}
	vk, nvk := 0, 0
	readKey := func() {
		var name, nname string
		vk, name, nvk, nname = readKeys(dropzone, lang)
		shortcut.Lock()
		shortcut.key, shortcut.names = name, nname
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
		changed := false
		down := vk != 0 && hotkey.Down(vk)
		if down && !held && hotkey.GameInFront() {
			shown = !shown
			changed = true
		}
		held = down
		// names: from the press in the game until the release, wherever it happens
		ndown := nvk != 0 && hotkey.Down(nvk)
		if ndown && !names && hotkey.GameInFront() {
			names, changed = true, true
		} else if !ndown && names {
			names, changed = false, true
		}
		if changed || time.Since(last) >= time.Second {
			write()
			last = time.Now()
		}
		if n%100 == 0 {
			readKey()
		}
	}
}

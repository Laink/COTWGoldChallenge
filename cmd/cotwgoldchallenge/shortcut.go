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

// ToggleFile tells the mod whether the overlay is shown and which keys are held:
// "<beat> <shown 0|1> <names 0|1> <opaque 0|1>", rewritten every second. When the beat stops
// changing, the mod shows the overlay again.
const ToggleFile = "ui/cotwgc_toggle.txt"

// Overlay keys: the first one shows or hides the overlay, the others act while held.
var overlayKeys = []struct{ setting, text string }{
	{"toggleKey", "keys_toggle"},
	{"namesKey", "keys_names"},
	{"opaqueKey", "keys_opaque"},
}

// shortcut watches the overlay keys in the background while the program runs.
var shortcut struct {
	sync.Mutex
	stop  chan struct{}
	done  chan struct{}
	names []string // name of each overlay key, "" when none is set
}

// shortcutOn tells whether the shortcut runs, and the names of its keys.
func shortcutOn() (bool, []string) {
	shortcut.Lock()
	defer shortcut.Unlock()
	return shortcut.stop != nil, shortcut.names
}

// keysLabel describes the keys set, for the menu; "" when none is.
func keysLabel(ui *UI, names []string) string {
	var parts []string
	for i, n := range names {
		if n != "" {
			parts = append(parts, fmt.Sprintf(ui.t(overlayKeys[i].text), n))
		}
	}
	return strings.Join(parts, ", ")
}

// readKeys returns the virtual-key code and name of each overlay key of the settings.
func readKeys(dropzone, lang string) ([]int, []string) {
	s := settings.Read(dropzone, lang)
	vks := make([]int, len(overlayKeys))
	names := make([]string, len(overlayKeys))
	for i, k := range overlayKeys {
		vks[i], names[i] = hotkey.Parse(s[k.setting])
		if vks[i] == 0 {
			names[i] = ""
		}
	}
	return vks, names
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
	_, names := readKeys(dropzone, lang)
	shortcut.Lock()
	shortcut.stop, shortcut.done, shortcut.names = stop, done, names
	shortcut.Unlock()
	go watchKey(dropzone, lang, stop, done)
	return nil
}

// startKeys starts the shortcut when the mod is installed and the shortcut is off.
func startKeys(dropzone, lang string) {
	if _, err := os.Stat(filepath.Join(dropzone, marker)); err != nil || !hotkey.Supported {
		return
	}
	if on, _ := shortcutOn(); !on {
		toggleShortcut(nil, dropzone, lang)
	}
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

// watchKey toggles the overlay when its key is pressed while the game is in front, and tells
// which held keys are down, from the press in the game until the release. The keys are read
// again from the settings every 3 s.
func watchKey(dropzone, lang string, stop, done chan struct{}) {
	defer close(done)
	path := filepath.Join(dropzone, filepath.FromSlash(ToggleFile))
	defer os.Remove(path)
	shown, beat := true, 0
	held := make([]bool, len(overlayKeys)) // held[0]: toggle key down at the last tick
	bit := func(b bool) string {
		if b {
			return "1"
		}
		return "0"
	}
	write := func() {
		beat++
		line := strconv.Itoa(beat) + " " + bit(shown)
		for _, h := range held[1:] {
			line += " " + bit(h)
		}
		os.WriteFile(path, []byte(line+"\n"), 0o644)
	}
	var vks []int
	readKey := func() {
		var names []string
		vks, names = readKeys(dropzone, lang)
		shortcut.Lock()
		shortcut.names = names
		shortcut.Unlock()
	}
	readKey()
	write()
	tick := time.NewTicker(30 * time.Millisecond)
	defer tick.Stop()
	last := time.Now()
	for n := 1; ; n++ {
		select {
		case <-stop:
			return
		case <-tick.C:
		}
		changed := false
		for i, vk := range vks {
			down := vk != 0 && hotkey.Down(vk)
			switch {
			case i == 0:
				if down && !held[0] && hotkey.GameInFront() {
					shown = !shown
					changed = true
				}
				held[0] = down
			case down && !held[i] && hotkey.GameInFront():
				held[i], changed = true, true
			case !down && held[i]:
				held[i], changed = false, true
			}
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

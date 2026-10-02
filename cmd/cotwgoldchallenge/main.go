// COTWGoldChallenge shows the medal potential of the spotted animal in the binoculars panel
// of theHunter: Call of the Wild. Designed by Laink.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
	"github.com/Laink/COTWGoldChallenge/internal/game"
	"github.com/Laink/COTWGoldChallenge/internal/icons"
	"github.com/Laink/COTWGoldChallenge/internal/modclass"
	"github.com/Laink/COTWGoldChallenge/internal/moddata"
	"github.com/Laink/COTWGoldChallenge/internal/patch"
	"github.com/Laink/COTWGoldChallenge/internal/settings"
	"github.com/Laink/COTWGoldChallenge/internal/swf"
)

var version = "dev"

const marker = "COTWGoldChallenge.json"

// iconsFile holds the species icons as SVG, by icon number, relative to dropzone/ui.
const iconsFile = "cotwgc_icons.json"

// legacyMarker is the marker of version 1, published as COTWSpottingPlus.
const legacyMarker = "COTWSpottingPlus.json"

type install struct {
	Version  string    `json:"version"`
	Language string    `json:"language"`
	Species  int       `json:"species"`
	Date     time.Time `json:"date"`
	Game     string    `json:"game,omitempty"` // gameSignature at installation
}

// gameSignature changes when a game update replaces one of the movies patched by the mod.
func gameSignature(arc *apex.Archives) string {
	var b strings.Builder
	fmt.Fprint(&b, len(arc.Entries))
	for _, p := range moviePaths() {
		for _, e := range arc.Find(p) {
			fmt.Fprintf(&b, "|%s:%d:%d", filepath.Base(e.Arc), e.Offset, e.Size)
		}
	}
	return b.String()
}

// printState tells whether the mod is installed, and whether it should be installed again:
// it returns true when an update is required.
func printState(ui *UI, inst *game.Install, dropzone string) bool {
	fmt.Println()
	if _, err := os.Stat(filepath.Join(dropzone, legacyMarker)); err == nil {
		fmt.Println(ui.t("state_legacy"))
		return true
	}
	b, err := os.ReadFile(filepath.Join(dropzone, marker))
	var info install
	if err != nil || json.Unmarshal(b, &info) != nil {
		fmt.Println(ui.t("state_none"))
		return false
	}
	date := info.Date.Local().Format(ui.t("date_format"))
	fmt.Printf(ui.t("state_installed")+"\n", date, info.Version)
	switch {
	case info.Version != version:
		fmt.Println(ui.t("state_program"))
		return true
	case info.Game != "":
		if arc, err := apex.Open(filepath.Join(inst.Dir, "archives_win64")); err == nil {
			defer arc.Close()
			if gameSignature(arc) != info.Game {
				fmt.Println(ui.t("state_game"))
				return true
			}
		}
	}
	return false
}

// lines are the lines typed in the console, read by one goroutine so that a page closed by its
// button does not leave a reader waiting for the next menu choice.
var lines = make(chan string)

func init() {
	go func() {
		r := bufio.NewReader(os.Stdin)
		for {
			l, err := r.ReadString('\n')
			if l != "" || err == nil {
				lines <- l
			}
			if err != nil {
				close(lines)
				return
			}
		}
	}()
}

// readLine waits for a line typed in the console ("" when the input is closed).
func readLine() string {
	return <-lines
}

func main() {
	gameDir := flag.String("game", "", "game folder (theHunterCotW)")
	lang := flag.String("lang", "", "language code or Steam language name")
	uninstall := flag.Bool("uninstall", false, "remove the mod")
	edit := flag.Bool("settings", false, "open the settings page")
	shortcut := flag.Bool("shortcut", false, "watch the key that shows or hides the wall")
	yes := flag.Bool("yes", false, "no questions")
	out := flag.String("out", "", "write the patched file here instead of installing it")
	previewData := flag.String("preview", "", "debug: id,min,max test data shown outside the game")
	flag.Parse()
	setupConsole()
	checkRelease()

	ui := uiFor("")
	err := run(&ui, *gameDir, *lang, *uninstall, *edit, *shortcut, *yes, *out, *previewData)
	if err == errQuit {
		return
	}
	if err != nil {
		fmt.Println()
		fmt.Println(ui.t("error"), err)
	}
	if !*yes {
		fmt.Println()
		fmt.Print(ui.t("press_enter"))
		readLine()
	}
	if err != nil {
		os.Exit(1)
	}
}

func run(ui *UI, gameDir, lang string, uninstall, edit, shortcut, yes bool, out, previewData string) error {
	all := game.LocateAll(gameDir)
	inst, ok := game.Locate(gameDir)
	if ok {
		inst = all[0]
	}
	*ui = uiFor(inst.Language)
	fmt.Printf("COTWGoldChallenge %s — %s\n", version, ui.t("credit"))
	fmt.Println(strings.Repeat("-", 78))
	fmt.Println(wrap(ui.t("preamble"), 78))
	fmt.Println()
	// Steam and Epic Games versions on the same computer: the player chooses.
	if len(all) > 1 && !yes {
		fmt.Println(ui.t("several"))
		for i, in := range all {
			store := "Steam"
			if in.Epic {
				store = "Epic Games"
			}
			fmt.Printf("  %d  %s: %s\n", i+1, store, in.Dir)
		}
		fmt.Print(ui.t("choice_game"))
		if n, err := strconv.Atoi(strings.TrimSpace(readLine())); err == nil && n >= 1 && n <= len(all) {
			inst = all[n-1]
		}
		*ui = uiFor(inst.Language)
		fmt.Println()
	}
	if !ok {
		if yes {
			return errors.New(ui.t("not_found"))
		}
		fmt.Println(ui.t("not_found"))
		for !ok {
			fmt.Print(ui.t("ask_folder"))
			line := readLine()
			line = strings.Trim(strings.TrimSpace(line), `"`)
			if line == "" {
				return errors.New(ui.t("not_found"))
			}
			if inst, ok = game.Locate(line); !ok {
				fmt.Println(ui.t("not_game"))
			}
		}
		*ui = uiFor(inst.Language)
	}
	if inst.Epic {
		fmt.Println(ui.t("game"), inst.Dir, "(Epic Games)")
	} else {
		fmt.Println(ui.t("game"), inst.Dir)
	}
	if lang == "" {
		lang = inst.Language
	}
	code := patch.LanguageFor(lang)
	fmt.Println(ui.t("language"), code)

	dropzone := filepath.Join(inst.Dir, "dropzone")
	linkSaves(ui, inst, dropzone)
	startRecorder(dropzone)
	if uninstall {
		return remove(ui, dropzone)
	}
	if edit {
		if err := openSettings(ui, dropzone, settingsLang(code)); err != nil {
			return err
		}
		fmt.Println(ui.t("wait_enter"))
		readLine()
		return errQuit
	}
	if shortcut {
		if err := toggleShortcut(ui, dropzone, settingsLang(code)); err != nil {
			return err
		}
		fmt.Println(ui.t("wait_enter"))
		readLine()
		stopShortcut()
		return errQuit
	}
	if yes || out != "" {
		return doInstall(ui, inst, dropzone, code, yes, out, previewData)
	}
	// Menu, shown again after each choice, until Enter. The settings page, the shortcut and the
	// recorder run in the background meanwhile. The overlay keys start with the program.
	defer stopShortcut()
	update := printState(ui, inst, dropzone)
	startKeys(dropzone, settingsLang(code))
	for {
		printBanner(ui)
		fmt.Println()
		switch {
		case !update:
			fmt.Println(ui.t("menu_install"))
		case colors:
			fmt.Println(ui.t("menu_install") + "  \x1b[30;43m " + ui.t("menu_update") + " \x1b[0m")
		default:
			fmt.Println(ui.t("menu_install") + "  <-- " + ui.t("menu_update"))
		}
		fmt.Println(ui.t("menu"))
		on, keys := shortcutOn()
		switch label := keysLabel(ui, keys); {
		case !on:
			fmt.Println(ui.t("menu_key_off"))
		case label == "":
			fmt.Println(ui.t("menu_key_nokey"))
		default:
			fmt.Printf(ui.t("menu_key_on")+"\n", label)
		}
		if on {
			fmt.Println(ui.t("menu_quit_on"))
		} else {
			fmt.Println(ui.t("menu_quit"))
		}
		fmt.Print(ui.t("choice"))
		line := readLine()
		var err error
		switch strings.TrimSpace(line) {
		case "1":
			if err = doInstall(ui, inst, dropzone, code, yes, out, previewData); err == nil {
				update = printState(ui, inst, dropzone)
				startRecorder(dropzone)
				startKeys(dropzone, settingsLang(code))
			}
		case "2":
			stopShortcut()
			if err = remove(ui, dropzone); err == nil {
				update = printState(ui, inst, dropzone)
			}
		case "3":
			err = openSettings(ui, dropzone, settingsLang(code))
		case "4":
			if err = toggleShortcut(ui, dropzone, settingsLang(code)); err == nil {
				fmt.Println()
				on, keys := shortcutOn()
				switch label := keysLabel(ui, keys); {
				case !on:
					fmt.Println(ui.t("key_off"))
				case label == "":
					fmt.Println(ui.t("key_nokey"))
				default:
					fmt.Printf(ui.t("key_on")+"\n", label)
				}
			}
		default:
			return errQuit
		}
		if err != nil {
			fmt.Println()
			fmt.Println(ui.t("error"), err)
		}
	}
}

// errQuit ends the program from the menu, without asking for Enter again.
var errQuit = errors.New("quit")

// doInstall installs or updates the mod.
func doInstall(ui *UI, inst *game.Install, dropzone, code string, yes bool, out, previewData string) error {
	arc, err := apex.Open(filepath.Join(inst.Dir, "archives_win64"))
	if err != nil {
		return err
	}
	defer arc.Close()
	entries := arc.Find(patch.GamePath)
	if len(entries) == 0 {
		return errors.New(ui.t("no_hud"))
	}
	original, err := arc.Read(entries[len(entries)-1])
	if err != nil {
		return err
	}
	fmt.Println(ui.t("reading"))
	species, err := game.LoadSpecies(arc, func(step string, done, total int) {
		fmt.Printf("\r  %s %3d %%", ui.t("step_"+step), done*100/max(total, 1))
	})
	fmt.Print("\r" + strings.Repeat(" ", 40) + "\r")
	if err != nil {
		return err
	}
	fmt.Printf(ui.t("species")+"\n", len(species))

	abc, err := modclass.Tags()
	if err != nil {
		return err
	}
	opts := patch.Options{Language: code, Extra: abc, Hook: patch.ModClass, Icons: true}
	if previewData != "" {
		var p [3]int
		if _, err := fmt.Sscanf(previewData, "%d,%d,%d", &p[0], &p[1], &p[2]); err != nil {
			return err
		}
		opts.Preview = &p
	}
	patched, err := patch.Apply(original, species, opts)
	if err != nil {
		return fmt.Errorf("%s: %w", ui.t("patch_failed"), err)
	}
	if out != "" {
		return os.WriteFile(out, patched, 0o644)
	}
	// The other movies: HUD (overlay) and reserve selection.
	movies := map[string][]byte{patch.GamePath: patched}
	read := func(p string) ([]byte, error) {
		e := arc.Find(p)
		if len(e) == 0 {
			return nil, fmt.Errorf("%s: %s", ui.t("patch_failed"), p)
		}
		return arc.Read(e[len(e)-1])
	}
	hud, err := read(patch.HUDPath)
	if err != nil {
		return err
	}
	if movies[patch.HUDPath], err = patch.ApplyHUD(hud, original, abc); err != nil {
		return fmt.Errorf("%s: %w", ui.t("patch_failed"), err)
	}
	for _, p := range patch.MenuPaths {
		m, err := read(p)
		if err != nil {
			return err
		}
		if movies[p], err = patch.ApplyMenu(m, original, abc); err != nil {
			return fmt.Errorf("%s: %w", ui.t("patch_failed"), err)
		}
	}
	cx, err := game.LoadCodex(arc, code)
	if err != nil {
		return err
	}

	// First installation: no marker yet (the settings may already have been made with choice 3).
	first := true
	for _, mk := range []string{marker, legacyMarker} {
		if _, err := os.Stat(filepath.Join(dropzone, mk)); err == nil {
			first = false
		}
	}

	// Files of other mods are kept as .bak.
	var others []string
	for _, p := range moviePaths() {
		t := filepath.Join(dropzone, filepath.FromSlash(p))
		if _, err := os.Stat(t); err == nil && !ours(t) {
			others = append(others, p)
		}
	}
	if len(others) > 0 {
		fmt.Printf(ui.t("other_mod")+"\n", strings.Join(others, ", "))
		if !yes {
			fmt.Print(ui.t("confirm"))
			a := readLine()
			if !isYes(a) {
				return nil
			}
		}
		for _, p := range others {
			t := filepath.Join(dropzone, filepath.FromSlash(p))
			os.Rename(t, t+".bak")
		}
	}
	for _, p := range moviePaths() {
		t := filepath.Join(dropzone, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(t), 0o755); err != nil {
			return writeError(ui, err)
		}
		if err := os.WriteFile(t, movies[p], 0o644); err != nil {
			return writeError(ui, err)
		}
	}
	if err := moddata.Write(filepath.Join(dropzone, "ui"), species, cx); err != nil {
		return writeError(ui, err)
	}
	// species icons for the settings page; the page does without them
	if svgs, err := icons.Extract(original); err == nil {
		if b, err := json.Marshal(svgs); err == nil {
			os.WriteFile(filepath.Join(dropzone, "ui", iconsFile), b, 0o644)
		}
	}
	// The hunting log is read through a link to the save folder.
	if saves, err := game.SavesDir(inst.Epic); err != nil {
		fmt.Println(ui.t("no_saves"))
	} else if err := game.LinkSaves(dropzone, saves); err != nil {
		fmt.Println(ui.t("no_link"), err)
	}
	info, _ := json.MarshalIndent(install{version, code, len(species), time.Now(), gameSignature(arc)}, "", "  ")
	os.WriteFile(filepath.Join(dropzone, marker), info, 0o644)
	os.Remove(filepath.Join(dropzone, legacyMarker))
	if _, err := settings.Ensure(dropzone, settingsLang(code)); err != nil {
		return writeError(ui, err)
	}
	fmt.Println()
	fmt.Println(ui.t("installed"))
	launchOptions(ui, inst)
	if first {
		fmt.Println()
		fmt.Println(wrap(ui.t("first_run"), 78))
	}
	return nil
}

// linkSaves makes the link to the save folder when the mod is installed without it: the save
// folder did not exist yet at installation (game never played), or was moved.
func linkSaves(ui *UI, inst *game.Install, dropzone string) {
	if _, err := os.Stat(filepath.Join(dropzone, marker)); err != nil {
		return
	}
	if _, err := os.Stat(filepath.Join(dropzone, game.SavesLink, "hunting_log_adf")); err == nil {
		return
	}
	saves, err := game.SavesDir(inst.Epic)
	if err != nil {
		return
	}
	if err := game.LinkSaves(dropzone, saves); err == nil {
		fmt.Println(ui.t("saves_linked"))
	}
}

// moviePaths lists the game movies replaced by the mod, relative to dropzone.
func moviePaths() []string {
	return append([]string{patch.GamePath, patch.HUDPath}, patch.MenuPaths...)
}

func launchOptions(ui *UI, inst *game.Install) {
	set := false
	if inst.Epic {
		set = game.EpicLaunchOptionsSet()
	} else {
		for _, o := range game.CurrentLaunchOptions(inst.Steam) {
			set = set || game.HasDropzone(o)
		}
	}
	if set {
		if inst.Epic {
			fmt.Println(ui.t("launch_ok_epic"))
		} else {
			fmt.Println(ui.t("launch_ok"))
		}
		fmt.Println(ui.t("restart"))
		return
	}
	fmt.Println()
	if inst.Epic {
		fmt.Println(wrap(ui.t("launch_todo_epic"), 78))
	} else {
		fmt.Println(ui.t("launch_todo"))
	}
	fmt.Println()
	fmt.Println("    " + game.LaunchOptions)
	fmt.Println()
	fmt.Println(ui.t("whole_line"))
	if copyClipboard(game.LaunchOptions) {
		fmt.Println(ui.t("copied"))
	}
}

// ours reports whether a movie was made by COTWGoldChallenge, or by version 1 published as
// COTWSpottingPlus.
func ours(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	m, err := swf.Load(b)
	return err == nil && (bytes.Contains(m.Body(), []byte("GoldChallenge")) || bytes.Contains(m.Body(), []byte("SpottingPlus")))
}

// remove uninstalls the mod: its movies (the files of other mods come back), data files and
// link to the saves. The settings are kept for a later installation.
func remove(ui *UI, dropzone string) error {
	found := false
	for _, p := range moviePaths() {
		t := filepath.Join(dropzone, filepath.FromSlash(p))
		if !ours(t) {
			continue
		}
		found = true
		os.Remove(t)
		if _, err := os.Stat(t + ".bak"); err == nil {
			os.Rename(t+".bak", t)
		}
	}
	for _, mk := range []string{marker, legacyMarker} {
		if _, err := os.Stat(filepath.Join(dropzone, mk)); err == nil {
			found = true
			os.Remove(filepath.Join(dropzone, mk))
		}
	}
	if !found {
		fmt.Println(ui.t("not_installed"))
		return nil
	}
	for _, f := range append(moddata.Files, "cotwgc_toggle.txt", iconsFile) {
		os.Remove(filepath.Join(dropzone, "ui", f))
	}
	game.UnlinkSaves(dropzone)
	fmt.Println(ui.t("removed"))
	return nil
}

func writeError(ui *UI, err error) error {
	if errors.Is(err, os.ErrPermission) {
		return errors.New(ui.t("denied"))
	}
	return err
}

func isYes(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "o" || s == "oui" || s == "y" || s == "yes"
}

func copyClipboard(s string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	c := exec.Command("cmd", "/c", "clip")
	c.Stdin = strings.NewReader(s)
	return c.Run() == nil
}

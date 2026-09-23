// COTWSpottingPlus shows the medal potential of the spotted animal in the binoculars panel
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
	"strings"
	"time"

	"github.com/Laink/COTWSpottingPlus/internal/apex"
	"github.com/Laink/COTWSpottingPlus/internal/game"
	"github.com/Laink/COTWSpottingPlus/internal/patch"
	"github.com/Laink/COTWSpottingPlus/internal/swf"
)

var version = "dev"

const marker = "COTWSpottingPlus.json"

type install struct {
	Version  string    `json:"version"`
	Language string    `json:"language"`
	Species  int       `json:"species"`
	Date     time.Time `json:"date"`
}

var in = bufio.NewReader(os.Stdin)

func main() {
	gameDir := flag.String("game", "", "game folder (theHunterCotW)")
	lang := flag.String("lang", "", "language code or Steam language name")
	uninstall := flag.Bool("uninstall", false, "remove the mod")
	yes := flag.Bool("yes", false, "no questions")
	out := flag.String("out", "", "write the patched file here instead of installing it")
	previewData := flag.String("preview", "", "debug: id,min,max test data shown outside the game")
	flag.Parse()
	setupConsole()

	ui := uiFor("")
	err := run(&ui, *gameDir, *lang, *uninstall, *yes, *out, *previewData)
	if err != nil {
		fmt.Println()
		fmt.Println(ui.t("error"), err)
	}
	if !*yes {
		fmt.Println()
		fmt.Print(ui.t("press_enter"))
		in.ReadString('\n')
	}
	if err != nil {
		os.Exit(1)
	}
}

func run(ui *UI, gameDir, lang string, uninstall, yes bool, out, previewData string) error {
	inst, ok := game.Locate(gameDir)
	*ui = uiFor(inst.Language)
	fmt.Printf("COTWSpottingPlus %s — %s\n", version, ui.t("credit"))
	fmt.Println(strings.Repeat("-", 78))
	fmt.Println(wrap(ui.t("preamble"), 78))
	fmt.Println()
	if !ok {
		if yes {
			return errors.New(ui.t("not_found"))
		}
		fmt.Println(ui.t("not_found"))
		for !ok {
			fmt.Print(ui.t("ask_folder"))
			line, _ := in.ReadString('\n')
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
	fmt.Println(ui.t("game"), inst.Dir)
	if lang == "" {
		lang = inst.Language
	}
	code := patch.LanguageFor(lang)
	fmt.Println(ui.t("language"), code)

	dropzone := filepath.Join(inst.Dir, "dropzone")
	target := filepath.Join(dropzone, filepath.FromSlash(patch.GamePath))
	if uninstall {
		return remove(ui, dropzone, target)
	}

	if !yes && out == "" {
		fmt.Println()
		fmt.Println(ui.t("menu"))
		fmt.Print(ui.t("choice"))
		line, _ := in.ReadString('\n')
		switch strings.TrimSpace(line) {
		case "1":
		case "2":
			return remove(ui, dropzone, target)
		default:
			return nil
		}
	}

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

	opts := patch.Options{Language: code, IgnoreSkill: true}
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

	if !ours(dropzone, target) {
		if _, err := os.Stat(target); err == nil {
			fmt.Println(ui.t("other_mod"))
			if !yes {
				fmt.Print(ui.t("confirm"))
				a, _ := in.ReadString('\n')
				if !isYes(a) {
					return nil
				}
			}
			os.Rename(target, target+".bak")
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return writeError(ui, err)
	}
	if err := os.WriteFile(target, patched, 0o644); err != nil {
		return writeError(ui, err)
	}
	info, _ := json.MarshalIndent(install{version, code, len(species), time.Now()}, "", "  ")
	os.WriteFile(filepath.Join(dropzone, marker), info, 0o644)
	fmt.Println()
	fmt.Println(ui.t("installed"))
	launchOptions(ui, inst.Steam)
	return nil
}

func launchOptions(ui *UI, steam string) {
	for _, o := range game.CurrentLaunchOptions(steam) {
		if game.HasDropzone(o) {
			fmt.Println(ui.t("launch_ok"))
			fmt.Println(ui.t("restart"))
			return
		}
	}
	fmt.Println()
	fmt.Println(ui.t("launch_todo"))
	fmt.Println()
	fmt.Println("    " + game.LaunchOptions)
	fmt.Println()
	if copyClipboard(game.LaunchOptions) {
		fmt.Println(ui.t("copied"))
	}
}

// ours reports whether the installed panel was made by COTWSpottingPlus.
func ours(dropzone, target string) bool {
	if _, err := os.Stat(filepath.Join(dropzone, marker)); err == nil {
		return true
	}
	b, err := os.ReadFile(target)
	if err != nil {
		return false
	}
	m, err := swf.Load(b)
	return err == nil && bytes.Contains(m.Body(), []byte("SpottingPlus"))
}

func remove(ui *UI, dropzone, target string) error {
	if !ours(dropzone, target) {
		fmt.Println(ui.t("not_installed"))
		return nil
	}
	os.Remove(target)
	os.Remove(filepath.Join(dropzone, marker))
	if _, err := os.Stat(target + ".bak"); err == nil {
		os.Rename(target+".bak", target)
	}
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

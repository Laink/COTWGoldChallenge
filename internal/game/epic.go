package game

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Epic Games Store version: the same game, found through the launcher's install manifests.

// epicDirs lists the game folders installed with the Epic Games Launcher.
func epicDirs() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	data := os.Getenv("ProgramData")
	if data == "" {
		data = `C:\ProgramData`
	}
	var out []string
	files, _ := filepath.Glob(filepath.Join(data, "Epic", "EpicGamesLauncher", "Data", "Manifests", "*.item"))
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var m struct {
			DisplayName      string
			InstallLocation  string
			LaunchExecutable string
		}
		if json.Unmarshal(b, &m) != nil || m.InstallLocation == "" {
			continue
		}
		if strings.Contains(strings.ToLower(m.LaunchExecutable), "thehuntercotw") ||
			strings.Contains(strings.ToLower(m.DisplayName), "call of the wild") {
			out = append(out, filepath.Clean(m.InstallLocation))
		}
	}
	// default folder, if the manifests cannot be read (Manifests\Pending holds unfinished installs)
	return append(out, `C:\Program Files\Epic Games\theHunterCallOfTheWild`)
}

// isEpicDir tells whether a game folder is an Epic Games installation.
func isEpicDir(dir string) bool {
	for _, d := range epicDirs() {
		if strings.EqualFold(filepath.Clean(d), filepath.Clean(dir)) {
			return true
		}
	}
	return false
}

// epicLanguage returns the language of the Epic Games Launcher, which it passes to the game, or
// else the Windows language, as a language code ("fr-FR").
func epicLanguage() string {
	if runtime.GOOS != "windows" {
		return ""
	}
	ini := filepath.Join(os.Getenv("LOCALAPPDATA"), "EpicGamesLauncher", "Saved", "Config", "Windows", "GameUserSettings.ini")
	if b, err := os.ReadFile(ini); err == nil {
		if m := regexp.MustCompile(`(?mi)^\s*(?:Culture|DefaultLanguage|Language)\s*=\s*([A-Za-z-]+)`).FindSubmatch(b); m != nil {
			return string(m[1])
		}
	}
	out, err := exec.Command("reg", "query", `HKCU\Control Panel\International`, "/v", "LocaleName").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) >= 3 && strings.EqualFold(f[0], "LocaleName") {
				return f[2]
			}
		}
	}
	return ""
}

// EpicLaunchOptionsSet tells whether the dropzone arguments appear in the launcher settings.
func EpicLaunchOptionsSet() bool {
	ini := filepath.Join(os.Getenv("LOCALAPPDATA"), "EpicGamesLauncher", "Saved", "Config", "Windows", "GameUserSettings.ini")
	b, err := os.ReadFile(ini)
	return err == nil && HasDropzone(string(b))
}

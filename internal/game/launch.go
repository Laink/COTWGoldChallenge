package game

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Exe is the game executable, in the game folder.
const Exe = "theHunterCotW_F.exe"

// Launch starts the game through its store, which adds the player's launch options (the ones
// that load the mod).
func Launch(in *Install) error {
	if runtime.GOOS != "windows" {
		return errors.New("Windows only")
	}
	url := "steam://rungameid/" + AppID
	if in.Epic {
		var ok bool
		if url, ok = epicLaunchURL(in.Dir); !ok {
			return errors.New("Epic Games manifest not found")
		}
	}
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

// Running tells whether the game runs.
func Running() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq "+Exe, "/NH").Output()
	return err == nil && strings.Contains(strings.ToLower(string(out)), strings.ToLower(Exe))
}

// epicLaunchURL is the launcher link that starts the game installed in dir.
func epicLaunchURL(dir string) (string, bool) {
	for _, m := range epicManifests() {
		if strings.EqualFold(filepath.Clean(m.InstallLocation), filepath.Clean(dir)) && m.AppName != "" {
			return "com.epicgames.launcher://apps/" + m.CatalogNamespace + "%3A" + m.CatalogItemID + "%3A" + m.AppName +
				"?action=launch&silent=true", true
		}
	}
	return "", false
}

// Shortcut makes a desktop shortcut that runs target with args, with the icon of the game.
func Shortcut(name, target, args string, in *Install) error {
	if runtime.GOOS != "windows" {
		return errors.New("Windows only")
	}
	// the values go through the environment: no quoting in the script
	script := `$d = [Environment]::GetFolderPath('Desktop'); ` +
		`$s = (New-Object -ComObject WScript.Shell).CreateShortcut((Join-Path $d $env:CGC_NAME)); ` +
		`$s.TargetPath = $env:CGC_TARGET; $s.Arguments = $env:CGC_ARGS; ` +
		`$s.WorkingDirectory = (Split-Path $env:CGC_TARGET); $s.IconLocation = $env:CGC_ICON; $s.Save()`
	c := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	c.Env = append(os.Environ(), "CGC_NAME="+name+".lnk", "CGC_TARGET="+target, "CGC_ARGS="+args,
		"CGC_ICON="+filepath.Join(in.Dir, Exe)+",0")
	if out, err := c.CombinedOutput(); err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}
	return nil
}

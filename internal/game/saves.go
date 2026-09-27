package game

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// SavesLink is the link from dropzone to the save folder: the mod reads the hunting log
// through it (dropzone/cotwgc_saves/hunting_log_adf).
const SavesLink = "cotwgc_saves"

// SavesDir returns the save folder of the profile played last: Documents\Avalanche Studios\
// COTW\Saves\<Steam id>.
func SavesDir() (string, error) {
	var best string
	var bestTime time.Time
	for _, docs := range documents() {
		root := filepath.Join(docs, "Avalanche Studios", "COTW", "Saves")
		dirs, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, d := range dirs {
			if !d.IsDir() {
				continue
			}
			st, err := os.Stat(filepath.Join(root, d.Name(), "hunting_log_adf"))
			if err == nil && st.ModTime().After(bestTime) {
				best, bestTime = filepath.Join(root, d.Name()), st.ModTime()
			}
		}
	}
	if best == "" {
		return "", errors.New("save folder not found")
	}
	return best, nil
}

// documents lists the possible Documents folders, the Windows one first (it can be moved,
// for example to OneDrive).
func documents() []string {
	var out []string
	if runtime.GOOS == "windows" {
		b, err := exec.Command("reg", "query", `HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer\User Shell Folders`, "/v", "Personal").Output()
		if err == nil {
			for _, line := range strings.Split(string(b), "\n") {
				f := strings.Fields(line)
				if len(f) >= 3 && strings.EqualFold(f[0], "Personal") {
					out = append(out, expandWindows(strings.Join(f[2:], " ")))
				}
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		out = append(out, filepath.Join(home, "Documents"), filepath.Join(home, "OneDrive", "Documents"))
	}
	return out
}

var winVar = regexp.MustCompile(`%([^%]+)%`)

func expandWindows(s string) string {
	return winVar.ReplaceAllStringFunc(s, func(v string) string {
		if e := os.Getenv(strings.Trim(v, "%")); e != "" {
			return e
		}
		return v
	})
}

// LinkSaves makes dropzone/cotwgc_saves point to the save folder (a directory junction on
// Windows, which needs no administrator rights).
func LinkSaves(dropzone, saves string) error {
	link := filepath.Join(dropzone, SavesLink)
	UnlinkSaves(dropzone)
	if runtime.GOOS != "windows" {
		return os.Symlink(saves, link)
	}
	out, err := exec.Command("cmd", "/c", "mklink", "/J", link, saves).CombinedOutput()
	if err != nil {
		return errors.New(strings.TrimSpace(string(out)))
	}
	return nil
}

// UnlinkSaves removes the link, never the save folder itself.
func UnlinkSaves(dropzone string) {
	link := filepath.Join(dropzone, SavesLink)
	if st, err := os.Lstat(link); err == nil && (st.Mode()&os.ModeSymlink != 0 || st.Mode()&os.ModeIrregular != 0 || isJunction(link)) {
		os.Remove(link)
	}
}

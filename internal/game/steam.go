package game

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// AppID is the Steam id of theHunter: Call of the Wild.
const AppID = "518790"

// LaunchOptions are the Steam launch options that load the dropzone folder.
const LaunchOptions = "--vfs-fs dropzone --vfs-archive archives_win64 --vfs-fs ."

// Install describes a game installation.
type Install struct {
	Dir      string // ...\steamapps\common\theHunterCotW
	Language string // Steam language name ("french") or language code ("fr-FR")
	Steam    string // Steam root folder, if known
	Epic     bool   // installed with the Epic Games Launcher
}

// IsGameDir reports whether dir contains the game.
func IsGameDir(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "archives_win64"))
	return err == nil
}

// Locate finds the game: explicit folder, program folder, Steam libraries, then Epic Games.
func Locate(explicit string) (*Install, bool) {
	all := LocateAll(explicit)
	if len(all) == 0 {
		return &Install{Steam: steamRoot()}, false
	}
	return all[0], true
}

// LocateAll lists the game installations, in the order of Locate: a player can have the Steam
// and the Epic Games versions. An explicit folder that holds the game is the only one listed.
func LocateAll(explicit string) []*Install {
	steam := steamRoot()
	newInstall := func(dir string) *Install {
		in := &Install{Dir: dir, Steam: steam}
		in.Language = manifestLanguage(dir)
		if in.Language == "" && isEpicDir(dir) {
			in.Epic = true
			in.Language = epicLanguage()
		}
		return in
	}
	if explicit != "" && IsGameDir(explicit) {
		return []*Install{newInstall(explicit)}
	}
	var cands []string
	if exe, err := os.Executable(); err == nil {
		cands = append(cands, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		cands = append(cands, wd)
	}
	for _, lib := range libraries(steam) {
		cands = append(cands, filepath.Join(lib, "steamapps", "common", "theHunterCotW"))
	}
	cands = append(cands, epicDirs()...)
	var out []*Install
	seen := map[string]bool{}
	for _, c := range cands {
		key := strings.ToLower(filepath.Clean(c))
		if seen[key] || !IsGameDir(c) {
			continue
		}
		seen[key] = true
		out = append(out, newInstall(c))
	}
	return out
}

func steamRoot() string {
	if runtime.GOOS == "windows" {
		for _, key := range []string{`HKCU\Software\Valve\Steam`, `HKLM\SOFTWARE\WOW6432Node\Valve\Steam`} {
			out, err := exec.Command("reg", "query", key).Output()
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(out), "\n") {
				f := strings.Fields(line)
				if len(f) >= 3 && (strings.EqualFold(f[0], "SteamPath") || strings.EqualFold(f[0], "InstallPath")) {
					p := strings.Join(f[2:], " ")
					return filepath.Clean(strings.ReplaceAll(p, "/", `\`))
				}
			}
		}
		return `C:\Program Files (x86)\Steam`
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".steam", "steam")
}

func libraries(steam string) []string {
	libs := []string{steam}
	b, err := os.ReadFile(filepath.Join(steam, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return libs
	}
	root := ParseVDF(string(b))
	for _, v := range root.Child("libraryfolders").Children {
		if p := v.Value("path"); p != "" {
			libs = append(libs, strings.ReplaceAll(p, `\\`, `\`))
		}
	}
	return libs
}

// manifestLanguage reads the language chosen in Steam for the game.
func manifestLanguage(gameDir string) string {
	apps := filepath.Dir(filepath.Dir(gameDir))
	b, err := os.ReadFile(filepath.Join(apps, "appmanifest_"+AppID+".acf"))
	if err != nil {
		return ""
	}
	return strings.ToLower(ParseVDF(string(b)).Child("AppState").Child("UserConfig").Value("language"))
}

// CurrentLaunchOptions returns the launch options set in Steam for the game, per user.
func CurrentLaunchOptions(steam string) []string {
	files, _ := filepath.Glob(filepath.Join(steam, "userdata", "*", "config", "localconfig.vdf"))
	var out []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		apps := ParseVDF(string(b)).Child("UserLocalConfigStore").Child("Software").Child("Valve").Child("Steam")
		app := apps.Child("apps").Child(AppID)
		if app == nil {
			app = apps.Child("Apps").Child(AppID)
		}
		if app != nil {
			out = append(out, app.Value("LaunchOptions"))
		}
	}
	return out
}

// HasDropzone reports whether launch options load the dropzone folder.
func HasDropzone(opts string) bool {
	return strings.Contains(opts, "--vfs-fs dropzone") && strings.Contains(opts, "archives_win64")
}

// VDF is a node of a Valve KeyValues text file.
type VDF struct {
	Key      string
	Val      string
	Children []*VDF
}

// Child returns the first child named key (case-insensitive).
func (n *VDF) Child(key string) *VDF {
	if n == nil {
		return nil
	}
	for _, c := range n.Children {
		if strings.EqualFold(c.Key, key) {
			return c
		}
	}
	return nil
}

// Value returns the value of the child named key.
func (n *VDF) Value(key string) string {
	if c := n.Child(key); c != nil {
		return c.Val
	}
	return ""
}

// ParseVDF parses a Valve KeyValues text file.
func ParseVDF(s string) *VDF {
	toks := vdfTokens(s)
	root := &VDF{}
	stack := []*VDF{root}
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		cur := stack[len(stack)-1]
		switch {
		case t == "}":
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case i+1 < len(toks) && toks[i+1] == "{":
			n := &VDF{Key: t}
			cur.Children = append(cur.Children, n)
			stack = append(stack, n)
			i++
		case i+1 < len(toks):
			cur.Children = append(cur.Children, &VDF{Key: t, Val: toks[i+1]})
			i++
		}
	}
	return root
}

func vdfTokens(s string) []string {
	var out []string
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case unicode.IsSpace(c):
		case c == '/' && i+1 < len(r) && r[i+1] == '/':
			for i < len(r) && r[i] != '\n' {
				i++
			}
		case c == '{' || c == '}':
			out = append(out, string(c))
		case c == '"':
			var b strings.Builder
			for i++; i < len(r) && r[i] != '"'; i++ {
				if r[i] == '\\' && i+1 < len(r) {
					i++
					switch r[i] {
					case 'n':
						b.WriteRune('\n')
					case 't':
						b.WriteRune('\t')
					default:
						b.WriteRune(r[i])
					}
					continue
				}
				b.WriteRune(r[i])
			}
			out = append(out, b.String())
		default:
			var b strings.Builder
			for ; i < len(r) && !unicode.IsSpace(r[i]) && r[i] != '{' && r[i] != '}' && r[i] != '"'; i++ {
				b.WriteRune(r[i])
			}
			i--
			out = append(out, b.String())
		}
	}
	return out
}

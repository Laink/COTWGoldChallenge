package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
	"github.com/Laink/COTWGoldChallenge/internal/game"
	"github.com/Laink/COTWGoldChallenge/internal/harvest"
	"github.com/Laink/COTWGoldChallenge/internal/icons"
	"github.com/Laink/COTWGoldChallenge/internal/maps"
	"github.com/Laink/COTWGoldChallenge/internal/patch"
	"github.com/Laink/COTWGoldChallenge/internal/safefile"
	"github.com/Laink/COTWGoldChallenge/internal/settings"
)

// colors tells whether the console shows ANSI colours (set by setupConsole).
var colors bool

// recorder copies the harvests of the hunting log into the history while the program runs:
// the game keeps only the best harvests of each species and the last ones.
var recorder struct {
	sync.Mutex
	h *harvest.History
}

// history returns the history, nil before the recorder starts.
func history() *harvest.History {
	recorder.Lock()
	defer recorder.Unlock()
	return recorder.h
}

// startRecorder starts the recorder once the mod is installed; it runs until the program ends.
func startRecorder(dropzone string) {
	if _, err := os.Stat(filepath.Join(dropzone, marker)); err != nil {
		return
	}
	recorder.Lock()
	defer recorder.Unlock()
	if recorder.h != nil {
		return
	}
	uiDir := filepath.Join(dropzone, "ui")
	h, err := harvest.Load(uiDir)
	if err != nil {
		// a history not read to the end is left as it is, not saved again without its end
		fmt.Println("COTWGoldChallenge:", err)
		return
	}
	if data, err := harvest.LoadData(uiDir); err == nil {
		h.SetAliases(data.Alias)
	}
	recorder.h = h
	go func() {
		saves := filepath.Join(dropzone, game.SavesLink)
		log := watched{path: filepath.Join(saves, "hunting_log_adf")}
		lodges := watched{path: filepath.Join(saves, harvest.LodgesFile)}
		for ; ; time.Sleep(5 * time.Second) {
			h.Flush() // a save that failed (the game reading the file) is tried again
			log.read(h, harvest.Parse)
			// the trophy lodges are never cleared by the game; they need the data files of 2.5
			lodges.read(h, func(raw []byte) ([]harvest.Entry, error) {
				data, err := harvest.LoadData(uiDir)
				if err != nil {
					return nil, err
				}
				h.SetAliases(data.Alias)
				return harvest.ParseLodges(raw, data)
			})
		}
	}()
}

// watched is a save file read again when it changes.
type watched struct {
	path string
	size int64
	mod  time.Time
}

// read merges the harvests of the file into the history, when it changed since the last read.
func (w *watched) read(h *harvest.History, parse func([]byte) ([]harvest.Entry, error)) {
	st, err := os.Stat(w.path)
	if err != nil || (st.Size() == w.size && st.ModTime().Equal(w.mod)) {
		return
	}
	raw, err := os.ReadFile(w.path)
	if err != nil {
		return
	}
	list, err := parse(raw)
	if err != nil {
		return // being written by the game: read again next time
	}
	if _, err := h.Merge(list); err == nil {
		w.size, w.mod = st.Size(), st.ModTime()
	}
}

// printBanner tells, very visibly, that the window must stay open while playing.
func printBanner(ui *UI) {
	setTitle("COTWGoldChallenge — " + ui.t("title_open"))
	text := wrap(ui.t("banner"), 74)
	if h := history(); h != nil {
		seen, manual := h.Counts()
		text += "\n" + fmt.Sprintf(ui.t("banner_count"), seen)
		if manual > 0 {
			text += fmt.Sprintf(ui.t("banner_manual"), manual)
		}
	}
	if tag := newRelease(); tag != "" {
		text += "\n\n" + fmt.Sprintf(ui.t("banner_release"), tag) + "\n" + releasesURL
	}
	fmt.Println()
	if !colors {
		line := strings.Repeat("=", 78)
		fmt.Println(line)
		for _, l := range strings.Split(text, "\n") {
			fmt.Println("  " + l)
		}
		fmt.Println(line)
		return
	}
	// black on amber, padded to the full width
	for _, l := range append(append([]string{""}, strings.Split(text, "\n")...), "") {
		pad := 76 - len([]rune(l))
		fmt.Printf("\x1b[30;43m  %s%s\x1b[0m\n", l, strings.Repeat(" ", max(pad, 0)))
	}
}

// Dashboard: trophies by reserve, as the mod counts them, and the trophies added by hand.

type trophyJSON struct {
	Hash     uint32     `json:"hash"`
	Name     string     `json:"name"`
	Class    int        `json:"cls"`
	CanGO    bool       `json:"canGo"`    // the species has a Great One
	Detected uint32     `json:"detected"` // greatOne, or 0 diamond .. 4 none; without the trophy added by hand
	Manual   uint32     `json:"manual"`
	RareFurs []string   `json:"furs"` // rare furs harvested, in the challenge shown
	Here     bool       `json:"here"` // harvested in this reserve (not by hand): something on its map
	Medals   [3]float64 `json:"t"`    // lowest scores of silver, gold and diamond
}

// speciesJSON describes a species for the album and the species sheet.
type speciesJSON struct {
	Class    int        `json:"cls"`
	GreatOne bool       `json:"go"`
	Medals   [3]float64 `json:"t"` // lowest scores of silver, gold and diamond
	Furs     []furJSON  `json:"furs"`
}

// furJSON is a rare fur of a species, from the game data.
type furJSON struct {
	Hash   uint32     `json:"h"`
	Name   string     `json:"n"`
	Rarity int        `json:"r"` // 2 rare, 3 very rare
	Odds   [2]float64 `json:"o"` // chance in percent, of a male and of a female
}

// greatOne is the rank of a Great One in the dashboard, and of the trophies added by hand (any
// rank above 4 is a Great One in the hunting log).
const greatOne = 5

func dashboardRank(rank uint32, g1 bool) uint32 {
	if g1 {
		return greatOne
	}
	return rank
}

type reserveJSON struct {
	Number  int          `json:"n"`
	Name    string       `json:"name"`
	Species []trophyJSON `json:"species"`
}

type harvestJSON struct {
	Species uint32     `json:"s"`
	Score   float32    `json:"score"`
	Rank    uint32     `json:"rank"` // greatOne, or 0 diamond .. 4 none
	Time    uint32     `json:"t"`
	Reserve int        `json:"r"` // -1: unknown
	Fur     string     `json:"fur,omitempty"`
	Rarity  int        `json:"rarity"`      // of the fur: 0 common .. 3 very rare
	Odds    [2]float64 `json:"odds"`        // chance of the fur in percent, of a male and of a female
	X       float32    `json:"x,omitempty"` // position on the map, 0 to 1 over the area played
	Y       float32    `json:"y,omitempty"`
	Region  uint32     `json:"g,omitempty"` // hash of the region of the hunting log
}

// harvestsAPI lists the harvests recorded, for the statistics; not the trophies added by hand.
func harvestsAPI(ui *UI, dropzone string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := history()
		data, err := harvest.LoadData(filepath.Join(dropzone, "ui"))
		if h == nil || err != nil {
			http.Error(w, ui.t("page_no_mod"), http.StatusConflict)
			return
		}
		out := []harvestJSON{}
		for _, e := range data.Place(h.Entries()) {
			if !e.Manual {
				fur := data.Furs[[2]uint32{e.Species, e.Fur}]
				// a region the game did not know (0xDEADBEEF in the log) is no region
				region := e.Region
				if _, known := data.Regions[region]; !known {
					region = 0
				}
				out = append(out, harvestJSON{e.Species, e.Score, dashboardRank(e.Rank, e.GreatOne), e.Time, e.Reserve, fur.Name, fur.Rarity, fur.Odds, e.X, e.Y, region})
			}
		}
		names := map[uint32]string{}
		for k, s := range data.Species {
			names[k] = s.Name
		}
		reserves := map[int]string{}
		for _, res := range data.Reserves {
			reserves[res.Number] = res.Name
		}
		// every species of the game: its medals and its rare furs (album, species sheet)
		info := map[uint32]*speciesJSON{}
		for k, s := range data.Species {
			info[k] = &speciesJSON{Class: s.Class, GreatOne: s.GreatOne, Medals: s.Thresholds, Furs: []furJSON{}}
		}
		for k, f := range data.Furs {
			if sp := info[k[0]]; sp != nil && f.Rarity >= harvest.RareFur {
				sp.Furs = append(sp.Furs, furJSON{k[1], f.Name, f.Rarity, f.Odds})
			}
		}
		for _, sp := range info {
			sort.Slice(sp.Furs, func(i, j int) bool { return sp.Furs[i].Name < sp.Furs[j].Name })
		}
		album := h.Album()
		if album == nil {
			album = [][2]uint32{}
		}
		writeJSON(w, map[string]any{"harvests": out, "species": names, "reserves": reserves, "info": info, "album": album})
	}
}

// albumAPI ticks a rare fur by hand in the album, or unticks it: {species, fur, on}.
func albumAPI(ui *UI) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := history()
		if h == nil {
			http.Error(w, ui.t("page_no_mod"), http.StatusConflict)
			return
		}
		var req struct {
			Species uint32 `json:"species"`
			Fur     uint32 `json:"fur"`
			On      bool   `json:"on"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req) != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if err := h.SetAlbum(req.Species, req.Fur, req.On, uint32(time.Now().Unix())); err != nil {
			http.Error(w, writeError(ui, err).Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]bool{"ok": true})
	}
}

// writeZones writes where the area played of each reserve is on its map, and its regions.
func writeZones(arc *apex.Archives, uiDir string, reserveOf map[uint32]int) error {
	all, err := maps.Maps(arc, reserveOf)
	if err != nil {
		return err
	}
	b, err := json.Marshal(all)
	if err != nil {
		return err
	}
	return safefile.Write(filepath.Join(uiDir, zonesFile), b)
}

// zonesAPI gives where the area played of a reserve (?r=number) is on its map, and its regions. They are written at
// installation; for a mod installed before, they are read from the game files here.
func zonesAPI(dropzone string) http.HandlerFunc {
	var once sync.Once
	return func(w http.ResponseWriter, r *http.Request) {
		uiDir := filepath.Join(dropzone, "ui")
		path := filepath.Join(uiDir, zonesFile)
		once.Do(func() {
			if _, err := os.Stat(path); err == nil {
				return
			}
			data, err := harvest.LoadData(uiDir)
			if err != nil {
				return
			}
			if arc, err := apex.Open(filepath.Join(filepath.Dir(dropzone), "archives_win64")); err == nil {
				writeZones(arc, uiDir, data.Regions)
				arc.Close()
			}
		})
		var all map[string]json.RawMessage
		b, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(b, &all) != nil || all[r.URL.Query().Get("r")] == nil {
			writeJSON(w, maps.Map{Frame: maps.DefaultFrame})
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(all[r.URL.Query().Get("r")])
	}
}

// mapAPI gives the map of a reserve (?r=number) as a JPEG image, read once from the game files.
func mapAPI(dropzone string) http.HandlerFunc {
	var mu sync.Mutex
	cache := map[int][]byte{}
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(r.URL.Query().Get("r"))
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		b, ok := cache[n]
		if !ok {
			arc, err := apex.Open(filepath.Join(filepath.Dir(dropzone), "archives_win64"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			b, err = maps.JPEG(arc, n)
			arc.Close()
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			cache[n] = b
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "max-age=3600")
		w.Write(b)
	}
}

// tileAPI gives a zoom tile of the map of a reserve (?r=number&z=level&i=index) as a JPEG image.
// The browser keeps them. The game files stay open for the next tiles, and are closed after a
// while without any, not to keep a game update from replacing them.
func tileAPI(dropzone string) http.HandlerFunc {
	var mu sync.Mutex
	var arc *apex.Archives
	var idle *time.Timer
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		n, err1 := strconv.Atoi(q.Get("r"))
		z, err2 := strconv.Atoi(q.Get("z"))
		i, err3 := strconv.Atoi(q.Get("i"))
		if err1 != nil || err2 != nil || err3 != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		mu.Lock()
		if arc == nil {
			a, err := apex.Open(filepath.Join(filepath.Dir(dropzone), "archives_win64"))
			if err != nil {
				mu.Unlock()
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			arc = a
		}
		if idle != nil {
			idle.Stop()
		}
		idle = time.AfterFunc(30*time.Second, func() {
			mu.Lock()
			defer mu.Unlock()
			if arc != nil {
				arc.Close()
				arc = nil
			}
		})
		b, err := maps.TileJPEG(arc, n, z, i)
		mu.Unlock()
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Cache-Control", "max-age=86400")
		w.Write(b)
	}
}

// iconsAPI gives the SVG icon of each species, by species hash. The icons are made at
// installation; for a mod installed before, they are made from the game files here.
func iconsAPI(dropzone string) http.HandlerFunc {
	var once sync.Once
	return func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dropzone, "ui", iconsFile)
		once.Do(func() {
			if _, err := os.Stat(path); err == nil {
				return
			}
			arc, err := apex.Open(filepath.Join(filepath.Dir(dropzone), "archives_win64"))
			if err != nil {
				return
			}
			defer arc.Close()
			if e := arc.Find(patch.GamePath); len(e) > 0 {
				if b, err := arc.Read(e[len(e)-1]); err == nil {
					if svgs, err := icons.Extract(b); err == nil {
						if j, err := json.Marshal(svgs); err == nil {
							os.WriteFile(path, j, 0o644)
						}
					}
				}
			}
		})
		var byIcon map[int]string
		b, err := os.ReadFile(path)
		data, derr := harvest.LoadData(filepath.Join(dropzone, "ui"))
		if err != nil || derr != nil || json.Unmarshal(b, &byIcon) != nil {
			writeJSON(w, map[string]string{})
			return
		}
		out := map[uint32]string{}
		for h, s := range data.Species {
			if svg, ok := byIcon[s.Icon]; ok {
				out[h] = svg
			}
		}
		writeJSON(w, out)
	}
}

// challengeStart is the time given to the trophies added by hand: the challenge start, so that
// they count in the challenge shown; 1 for all time.
func challengeStart(since string) uint32 {
	return max(harvest.Since(since), 1)
}

func trophiesAPI(ui *UI, dropzone, lang string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := history()
		if h == nil {
			http.Error(w, ui.t("page_no_mod"), http.StatusConflict)
			return
		}
		switch r.Method {
		case http.MethodGet:
			data, err := harvest.LoadData(filepath.Join(dropzone, "ui"))
			if err != nil {
				http.Error(w, ui.t("page_no_mod"), http.StatusConflict)
				return
			}
			q := r.URL.Query()
			since := harvest.Since(q.Get("since"))
			mega := q.Get("scope") == "mega"
			list := data.Place(h.Entries())
			var out []reserveJSON
			for _, res := range data.Reserves {
				rj := reserveJSON{Number: res.Number, Name: res.Name}
				for _, sp := range res.Species {
					in := res.Number
					if mega {
						in = -1
					}
					// the trophy added by hand in this reserve is the one the player can change
					mine := func(e harvest.Placed) bool { return e.Manual && e.Reserve == res.Number }
					det, g1 := harvest.Best(list, sp, since, in, mine)
					man, mg1 := harvest.Best(list, sp, since, res.Number, func(e harvest.Placed) bool { return !mine(e) })
					s := data.Species[sp]
					here := false
					for _, e := range list {
						here = here || (e.Species == sp && !e.Manual && e.Reserve == res.Number)
					}
					rj.Species = append(rj.Species, trophyJSON{sp, s.Name, s.Class, s.GreatOne, dashboardRank(det, g1), dashboardRank(man, mg1), data.RareFurs(list, sp, since, in), here, s.Thresholds})
				}
				out = append(out, rj)
			}
			// best medal of each species in any reserve, trophies by hand included: overall progress
			global := map[uint32]uint32{}
			for _, res := range data.Reserves {
				for _, sp := range res.Species {
					if _, ok := global[sp]; !ok {
						global[sp] = dashboardRank(harvest.Best(list, sp, since, -1, nil))
					}
				}
			}
			seen, manual := h.Counts()
			// reserve shown first: the one forced in the settings, else the one played last
			current := -1
			if v, err := strconv.Atoi(settings.Read(dropzone, lang)["reserve"]); err == nil {
				current = v
			} else if raw, err := os.ReadFile(filepath.Join(dropzone, game.SavesLink, "reserveworlddata_adf")); err == nil {
				if v, err := harvest.LastReserve(raw); err == nil {
					current = v
				}
			}
			writeJSON(w, map[string]any{"reserves": out, "global": global, "recorded": seen, "manual": manual, "reserve": current})
		case http.MethodPost:
			var req struct {
				Species uint32 `json:"species"`
				Reserve int    `json:"reserve"`
				Rank    uint32 `json:"rank"`
				Since   string `json:"since"`
			}
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req); err != nil || req.Rank > greatOne {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if err := h.SetManual(req.Species, req.Reserve, req.Rank, challengeStart(req.Since)); err != nil {
				http.Error(w, writeError(ui, err).Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]bool{"ok": true})
		default:
			http.Error(w, "", http.StatusMethodNotAllowed)
		}
	}
}

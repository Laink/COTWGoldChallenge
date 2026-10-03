package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
	"github.com/Laink/COTWGoldChallenge/internal/game"
	"github.com/Laink/COTWGoldChallenge/internal/harvest"
	"github.com/Laink/COTWGoldChallenge/internal/icons"
	"github.com/Laink/COTWGoldChallenge/internal/patch"
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
	h, err := harvest.Load(filepath.Join(dropzone, "ui"))
	if err != nil {
		return
	}
	recorder.h = h
	go func() {
		saves := filepath.Join(dropzone, game.SavesLink)
		log := watched{path: filepath.Join(saves, "hunting_log_adf")}
		lodges := watched{path: filepath.Join(saves, harvest.LodgesFile)}
		for ; ; time.Sleep(5 * time.Second) {
			log.read(h, harvest.Parse)
			// the trophy lodges are never cleared by the game; they need the data files of 2.5
			lodges.read(h, func(raw []byte) ([]harvest.Entry, error) {
				data, err := harvest.LoadData(filepath.Join(dropzone, "ui"))
				if err != nil {
					return nil, err
				}
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
	Hash     uint32   `json:"hash"`
	Name     string   `json:"name"`
	Class    int      `json:"cls"`
	CanGO    bool     `json:"canGo"`    // the species has a Great One
	Detected uint32   `json:"detected"` // greatOne, or 0 diamond .. 4 none; without the trophy added by hand
	Manual   uint32   `json:"manual"`
	RareFurs []string `json:"furs"` // rare furs harvested, in the challenge shown
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
	Rarity  int        `json:"rarity"` // of the fur: 0 common .. 3 very rare
	Odds    [2]float64 `json:"odds"`   // chance of the fur in percent, of a male and of a female
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
				out = append(out, harvestJSON{e.Species, e.Score, dashboardRank(e.Rank, e.GreatOne), e.Time, e.Reserve, fur.Name, fur.Rarity, fur.Odds})
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
		writeJSON(w, map[string]any{"harvests": out, "species": names, "reserves": reserves})
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
					rj.Species = append(rj.Species, trophyJSON{sp, s.Name, s.Class, s.GreatOne, dashboardRank(det, g1), dashboardRank(man, mg1), data.RareFurs(list, sp, since, in)})
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

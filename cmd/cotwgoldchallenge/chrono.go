package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/harvest"
	"github.com/Laink/COTWGoldChallenge/internal/hotkey"
	"github.com/Laink/COTWGoldChallenge/internal/safefile"
	"github.com/Laink/COTWGoldChallenge/internal/settings"
	"github.com/Laink/COTWGoldChallenge/internal/timer"
)

// TimerFile tells the mod the challenge timer to show, rewritten every second:
// "<beat> <shown 0|1> <paused 0|1> <elapsed s> <since the last species done s, -1 for none>".
// When the beat stops changing, the mod hides the timer.
const TimerFile = "cotwgc_timer.txt"

// chrono is the challenge timer, counting in the background once the mod is installed.
var chrono struct {
	sync.Mutex
	t *timer.Timer
}

func challengeTimer() *timer.Timer {
	chrono.Lock()
	defer chrono.Unlock()
	return chrono.t
}

// startTimer starts the timer; it counts while the game is in front, the timer is on in the
// settings and not paused, until the program ends.
func startTimer(dropzone, lang string) {
	uiDir := filepath.Join(dropzone, "ui")
	chrono.Lock()
	if chrono.t != nil {
		chrono.Unlock()
		return
	}
	t, err := timer.Load(uiDir)
	if err != nil {
		chrono.Unlock()
		fmt.Println("COTWGoldChallenge:", err)
		return
	}
	chrono.t = t
	chrono.Unlock()
	go func() {
		path := filepath.Join(uiDir, TimerFile)
		var s map[string]string
		var done uint32
		beat := 0
		for n := 0; ; n++ {
			if n%12 == 0 { // settings every 3 s
				s = settings.Read(dropzone, lang)
			}
			on := s["timerOn"] == "1"
			t.Tick(time.Now().Unix(), on && hotkey.GameInFront())
			if n%20 == 0 { // the last species done every 5 s
				done = lastDone(uiDir, s)
			}
			if n%4 == 0 {
				beat++
				safefile.Write(path, []byte(timerLine(t, beat, on, s["timerSub"] != "0", done)))
			}
			if n%40 == 0 {
				t.Flush()
			}
			time.Sleep(250 * time.Millisecond)
		}
	}()
}

// timerLine is the line of TimerFile.
func timerLine(t *timer.Timer, beat int, on, sub bool, done uint32) string {
	elapsed := t.Elapsed()
	since := -1.0
	if sub {
		since = elapsed - t.At(int64(done)) // from the start when no species is done yet
	}
	return fmt.Sprintf("%d %d %d %d %d\n", beat, bit(on && !t.Hidden()), bit(t.Paused()), int64(elapsed), int64(since))
}

func bit(b bool) int {
	if b {
		return 1
	}
	return 0
}

// lastDone returns the time of the last species done for the challenge of the settings, 0 when
// none is.
func lastDone(uiDir string, s map[string]string) uint32 {
	h := history()
	data, err := harvest.LoadData(uiDir)
	if h == nil || err != nil {
		return 0
	}
	target, _ := strconv.Atoi(s["tier"])
	return data.LastDone(data.Place(h.Entries()), harvest.Since(s["since"]), uint32(target), s["higher"] != "0", s["scope"] == "mega")
}

// timerAPI gives the timer (GET) and changes it (POST {action: set|estimate|reset|pause|show,
// seconds}).
func timerAPI(ui *UI, dropzone, lang string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t := challengeTimer()
		if t == nil {
			http.Error(w, ui.t("page_no_mod"), http.StatusConflict)
			return
		}
		s := settings.Read(dropzone, lang)
		if r.Method == http.MethodPost {
			var req struct {
				Action  string  `json:"action"`
				Seconds float64 `json:"seconds"`
			}
			if json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&req) != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			var err error
			switch req.Action {
			case "set":
				err = t.Set(max(req.Seconds, 0))
			case "reset":
				err = t.Reset()
			case "pause":
				err = t.TogglePause()
			case "show":
				err = t.ToggleHidden()
			case "estimate":
				var times []int64
				if h := history(); h != nil {
					for _, e := range h.Entries() {
						if !e.Manual {
							times = append(times, int64(e.Time))
						}
					}
				}
				err = t.Estimate(times, int64(harvest.Since(s["since"])))
			default:
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			if err != nil {
				http.Error(w, writeError(ui, err).Error(), http.StatusInternalServerError)
				return
			}
		}
		done := lastDone(filepath.Join(dropzone, "ui"), s)
		elapsed := t.Elapsed()
		var sessions [][3]int64
		for _, x := range t.Sessions() {
			sessions = append(sessions, [3]int64{x.Start, x.End, int64(bit(x.Estimated))})
		}
		if sessions == nil {
			sessions = [][3]int64{}
		}
		writeJSON(w, map[string]any{"elapsed": elapsed, "since": elapsed - t.At(int64(done)), "done": done,
			"paused": t.Paused(), "hidden": t.Hidden(), "sessions": sessions})
	}
}

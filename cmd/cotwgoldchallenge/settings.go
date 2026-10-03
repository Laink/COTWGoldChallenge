package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/settings"
)

//go:embed web
var web embed.FS

type fieldJSON struct {
	Key      string       `json:"key"`
	Kind     string       `json:"kind"`
	Group    string       `json:"group"`
	Advanced bool         `json:"advanced"`
	Default  string       `json:"default"`
	Label    string       `json:"label"`
	Help     string       `json:"help,omitempty"`
	Choices  []choiceJSON `json:"choices,omitempty"`
	Min      float64      `json:"min,omitempty"`
	Max      float64      `json:"max,omitempty"`
	Step     float64      `json:"step,omitempty"`
}

type choiceJSON struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// settingsURL is the address of the settings page once served.
var settingsURL string

// openSettings serves the settings page on the local machine, in the background until the
// program ends, and opens it in the browser.
func openSettings(ui *UI, dropzone, lang string) error {
	if settingsURL == "" {
		url, err := serveSettings(ui, dropzone, lang)
		if err != nil {
			return err
		}
		settingsURL = url
	}
	fmt.Println()
	fmt.Println(ui.t("settings_open"))
	fmt.Println("    " + settingsURL)
	openBrowser(settingsURL)
	return nil
}

func serveSettings(ui *UI, dropzone, lang string) (string, error) {
	tok := make([]byte, 16)
	rand.Read(tok)
	token := hex.EncodeToString(tok)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	mux := http.NewServeMux()
	static, _ := fs.Sub(web, "web")
	mux.Handle("/"+token+"/", http.StripPrefix("/"+token+"/", http.FileServer(http.FS(static))))
	mux.HandleFunc("/"+token+"/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			var fields []fieldJSON
			defaults := settings.Defaults(lang)
			for _, f := range settings.Fields {
				j := fieldJSON{f.Key, f.Kind, f.Group, f.Advanced, defaults[f.Key], f.Label.Get(lang), f.Help.Get(lang), nil, f.Min, f.Max, f.Step}
				for _, c := range f.Choices {
					j.Choices = append(j.Choices, choiceJSON{c.Value, c.Label.Get(lang)})
				}
				fields = append(fields, j)
			}
			groups := map[string]string{}
			for k, v := range settings.Groups {
				groups[k] = v.Get(lang)
			}
			writeJSON(w, map[string]any{
				"lang": lang, "version": version, "fields": fields, "groups": groups,
				"values": settings.Read(dropzone, lang), "text": pageText(ui),
			})
		case http.MethodPost:
			var values map[string]string
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&values); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if err := settings.Write(dropzone, lang, values); err != nil {
				http.Error(w, writeError(ui, err).Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]bool{"ok": true})
		default:
			http.Error(w, "", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/"+token+"/api/trophies", trophiesAPI(ui, dropzone, lang))
	mux.HandleFunc("/"+token+"/api/harvests", harvestsAPI(ui, dropzone))
	mux.HandleFunc("/"+token+"/api/icons", iconsAPI(dropzone))
	mux.HandleFunc("/"+token+"/api/close", func(w http.ResponseWriter, r *http.Request) {})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go srv.Serve(ln)
	return fmt.Sprintf("http://%s/%s/", ln.Addr(), token), nil
}

func pageText(ui *UI) map[string]string {
	m := map[string]string{}
	for _, k := range []string{"page_title", "page_intro", "page_reset_ask", "page_saved", "page_reset", "page_close",
		"page_closed", "page_advanced", "page_always", "page_preview", "page_preview_note", "page_time", "page_zoom", "page_full", "page_list_note", "page_panels", "page_zone_note", "page_key_set", "page_key_clear", "page_key_wait", "page_key_none", "page_mouse",
		"page_tab_settings", "page_tab_trophies", "page_trophies", "page_troph_intro", "page_keep_open", "page_reserve", "page_detected", "page_by_hand", "page_recorded", "page_no_mod", "page_great_one", "page_rank_none", "page_mode_mega", "page_mode_reserve",
		"page_tab_stats", "page_overview", "page_by_species", "page_all_species", "page_missing", "page_missing_none", "page_stats_note", "page_stats_since", "page_stats_all", "page_harvests", "page_species_n", "page_medals", "page_by_reserve", "page_top_species", "page_recent", "page_col_date", "page_col_species", "page_col_score", "page_col_medal", "page_col_fur", "page_rare_furs", "page_unknown", "page_no_harvest"} {
		m[k] = ui.t(k)
	}
	return m
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}

// settingsLang is the language of the settings file texts: the language of the installed mod.
func settingsLang(code string) string {
	if strings.HasPrefix(code, "fr") {
		return "fr"
	}
	return "en"
}

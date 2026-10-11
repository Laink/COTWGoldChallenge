// Package harvest reads the hunting log of the game and keeps the history of the harvests.
//
// The game keeps only the 5 best harvests of each species and the last 20: the program copies
// every harvest it sees into a history file (dropzone/ui/cotwgc_history.txt), read by the mod
// with the hunting log. The player can add trophies by hand, in the same file.
package harvest

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/safefile"
)

// HistoryFile is the history, relative to dropzone/ui.
const HistoryFile = "cotwgc_history.txt"

// Entry is one harvest. Rank: 0 diamond .. 3 bronze, 4 none, above 4 a Great One.
// Manual trophies have no score nor region, and their reserve set (otherwise -1).
type Entry struct {
	Species uint32
	Score   float32
	Rank    uint32
	Time    uint32 // Unix time
	Region  uint32
	Reserve int
	Manual  bool
	Fur     uint32  // hash of the fur name ("animal_visual_variation_albino"), 0 when unknown
	Lodge   bool    // read in the trophy lodges: no region, its reserve set when known
	Album   bool    // a rare fur ticked by hand in the album (Manual too): only its species and fur
	X, Y    float32 // position on the map of the reserve, 0 to 1 over the area played; 0, 0 when unknown
}

func (e Entry) key() string {
	return fmt.Sprintf("%d/%d/%d", e.Species, e.Time, e.Rank)
}

// adf reads a save file: a "SAVE" header and zlib data, or a plain ADF file. It returns the
// ADF data and the offset of its root instance.
func adf(raw []byte) ([]byte, int, error) {
	d := raw
	if len(raw) > 32 && string(raw[:4]) == "SAVE" {
		r, err := zlib.NewReader(bytes.NewReader(raw[32:]))
		if err != nil {
			return nil, 0, err
		}
		if d, err = io.ReadAll(r); err != nil {
			return nil, 0, err
		}
	}
	b := bytes.Index(d, []byte(" FDA"))
	if b < 0 || b+16 > len(d) {
		return nil, 0, errors.New("ADF not found")
	}
	inst := b + int(binary.LittleEndian.Uint32(d[b+12:]))
	if inst < 0 || inst+12 > len(d) {
		return nil, 0, errors.New("ADF truncated")
	}
	return d, b + int(binary.LittleEndian.Uint32(d[inst+8:])), nil
}

// LastReserve reads reserveworlddata_adf: the reserve played last.
func LastReserve(raw []byte) (int, error) {
	d, root, err := adf(raw)
	if err != nil {
		return 0, err
	}
	if root < 0 || root+8 > len(d) {
		return 0, errors.New("ADF truncated")
	}
	return int(binary.LittleEndian.Uint32(d[root+4:])), nil
}

// Parse reads hunting_log_adf, whose root holds the last harvests, then a list per species of
// its best harvests.
func Parse(raw []byte) ([]Entry, error) {
	d, root, err := adf(raw)
	if err != nil {
		return nil, err
	}
	var bad bool
	u := func(p int) int {
		if p < 0 || p+4 > len(d) {
			bad = true
			return 0
		}
		return int(binary.LittleEndian.Uint32(d[p:]))
	}
	entry := func(p int) Entry {
		if p < 0 || p+24 > len(d) {
			bad = true
			return Entry{}
		}
		return Entry{
			Species: binary.LittleEndian.Uint32(d[p:]),
			Score:   math.Float32frombits(binary.LittleEndian.Uint32(d[p+4:])),
			Rank:    binary.LittleEndian.Uint32(d[p+8:]),
			Fur:     binary.LittleEndian.Uint32(d[p+12:]),
			Time:    binary.LittleEndian.Uint32(d[p+16:]),
			Region:  binary.LittleEndian.Uint32(d[p+20:]),
			Reserve: -1,
		}
	}
	var out []Entry
	off, n := root+u(root+8), u(root+16)
	for i := 0; i < n && !bad; i++ {
		out = append(out, entry(off+i*24))
	}
	off, n = root+u(root+24), u(root+32)
	for i := 0; i < n && !bad; i++ {
		o := off + i*24
		po, pn := root+u(o+8), u(o+16)
		for j := 0; j < pn && !bad; j++ {
			out = append(out, entry(po+j*24))
		}
	}
	if bad {
		return nil, errors.New("hunting log truncated")
	}
	return out, nil
}

// History is the history file, safe for concurrent use.
type History struct {
	mu      sync.Mutex
	path    string
	entries []Entry
	index   map[string]int    // key of each harvest (not added by hand) -> its place in entries
	alias   map[uint32]uint32 // other hash of a species in the hunting log -> its hash
	dirty   bool              // changed since the last save that worked
}

// Load reads the history of a dropzone/ui folder; a missing file is an empty history.
func Load(uiDir string) (*History, error) {
	h := &History{path: filepath.Join(uiDir, HistoryFile), index: map[string]int{}}
	b, err := os.ReadFile(h.path)
	if errors.Is(err, os.ErrNotExist) {
		return h, nil
	}
	if err != nil {
		return nil, err
	}
	pos := map[string][2]float32{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		num := func(i int) uint32 {
			v, _ := strconv.ParseUint(f[i], 10, 32)
			return uint32(v)
		}
		switch {
		case len(f) == 6 && f[0] == "p": // position on the map of a harvest: species time rank x y
			x, _ := strconv.ParseFloat(f[4], 32)
			y, _ := strconv.ParseFloat(f[5], 32)
			pos[Entry{Species: num(1), Time: num(2), Rank: num(3)}.key()] = [2]float32{float32(x), float32(y)}
		case (len(f) == 6 || len(f) == 7) && f[0] == "h": // no fur before version 2.5
			s, _ := strconv.ParseFloat(f[2], 32)
			e := Entry{Species: num(1), Score: float32(s), Rank: num(3), Time: num(4), Region: num(5), Reserve: -1}
			if len(f) == 7 {
				e.Fur = num(6)
			}
			h.add(e)
		case len(f) == 5 && f[0] == "m":
			h.entries = append(h.entries, Entry{Species: num(1), Rank: num(2), Time: num(3), Reserve: int(num(4)), Manual: true})
		case len(f) == 7 && f[0] == "t":
			s, _ := strconv.ParseFloat(f[2], 32)
			r, _ := strconv.Atoi(f[5])
			h.add(Entry{Species: num(1), Score: float32(s), Rank: num(3), Time: num(4), Reserve: r, Fur: num(6), Lodge: true})
		case len(f) == 4 && f[0] == "f": // rare fur ticked by hand: species fur time
			h.entries = append(h.entries, Entry{Species: num(1), Fur: num(2), Time: num(3), Rank: 4, Reserve: -1, Manual: true, Album: true})
		}
	}
	if err := sc.Err(); err != nil {
		// not read to the end: saving it again would lose the rest
		return nil, fmt.Errorf("%s: %w", h.path, err)
	}
	for i, e := range h.entries {
		if p, ok := pos[e.key()]; ok && !e.Manual {
			h.entries[i].X, h.entries[i].Y = p[0], p[1]
		}
	}
	return h, nil
}

// HasPosition tells whether the position of the harvest on the map is known.
func (e Entry) HasPosition() bool { return e.X != 0 || e.Y != 0 }

// add adds a harvest not in the history yet. A harvest already there gets what it missed: its fur
// (not kept before version 2.5) and its position; changed is then true.
func (h *History) add(e Entry) (added, changed bool) {
	k := e.key()
	if i, ok := h.index[k]; ok {
		return false, h.complete(i, e)
	}
	// the same harvest seen in the hunting log and in the trophy lodges: the times differ by a few
	// seconds. The one of the hunting log is kept, as the mod reads the log too.
	if i := h.twin(e); i >= 0 {
		if e.Lodge {
			return false, h.complete(i, e)
		}
		old := h.entries[i]
		h.entries[i] = e
		h.complete(i, old)
		delete(h.index, old.key())
		h.index[k] = i
		return false, true
	}
	h.index[k] = len(h.entries)
	h.entries = append(h.entries, e)
	return true, true
}

// complete gives to the harvest i the fur and the position of e that it does not have.
func (h *History) complete(i int, e Entry) bool {
	o := &h.entries[i]
	changed := false
	if o.Fur == 0 && e.Fur != 0 {
		o.Fur, changed = e.Fur, true
	}
	if !o.HasPosition() && e.HasPosition() {
		o.X, o.Y, changed = e.X, e.Y, true
	}
	return changed
}

// reindex rebuilds the index of the harvests, after entries are removed.
func (h *History) reindex() {
	h.index = map[string]int{}
	for i, e := range h.entries {
		if !e.Manual {
			h.index[e.key()] = i
		}
	}
}

// SetAliases gives the other hashes of a species in the hunting log (the puma is
// "mountain lion" there), so that its harvests are found again in the trophy lodges.
func (h *History) SetAliases(alias map[uint32]uint32) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.alias = alias
}

func (h *History) species(s uint32) uint32 {
	if a, ok := h.alias[s]; ok {
		return a
	}
	return s
}

// twin finds, for a harvest of the trophy lodges, the same harvest of the hunting log, or the
// reverse: same species and rank, within a minute, the closest in time. A harvest whose position
// is another one is someone else. -1 when there is none.
func (h *History) twin(e Entry) int {
	if e.Manual {
		return -1
	}
	best, gap := -1, int64(61)
	for i, o := range h.entries {
		if o.Manual || o.Lodge == e.Lodge || h.species(o.Species) != h.species(e.Species) || o.Rank != e.Rank {
			continue
		}
		if o.HasPosition() && e.HasPosition() && (o.X != e.X || o.Y != e.Y) {
			continue
		}
		d := int64(o.Time) - int64(e.Time)
		if d < 0 {
			d = -d
		}
		if d < gap {
			best, gap = i, d
		}
	}
	return best
}

// Merge adds the harvests not in the history yet, and saves it when it changed.
func (h *History) Merge(list []Entry) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	n, changed := 0, false
	for _, e := range list {
		if !e.Lodge {
			e.Reserve = -1
		}
		e.Manual = false
		a, c := h.add(e)
		if a {
			n++
		}
		changed = changed || c
	}
	if !changed && !h.dirty {
		return 0, nil
	}
	return n, h.save()
}

// Flush saves the history again when a save failed; the recorder calls it now and then.
func (h *History) Flush() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.dirty {
		return nil
	}
	return h.save()
}

// SetManual sets the trophy added by hand for a species in a reserve, with the ranks of the
// hunting log (above 4 a Great One); rank 4 removes it.
func (h *History) SetManual(species uint32, reserve int, rank uint32, at uint32) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	kept := h.entries[:0]
	for _, e := range h.entries {
		if !(e.Manual && !e.Album && e.Species == species && e.Reserve == reserve) {
			kept = append(kept, e)
		}
	}
	h.entries = kept
	h.reindex()
	if rank != 4 {
		h.entries = append(h.entries, Entry{Species: species, Rank: rank, Time: at, Reserve: reserve, Manual: true})
	}
	return h.save()
}

// SetAlbum ticks a rare fur of a species in the album by hand, or unticks it.
func (h *History) SetAlbum(species, fur uint32, on bool, at uint32) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	kept := h.entries[:0]
	for _, e := range h.entries {
		if !(e.Album && e.Species == species && e.Fur == fur) {
			kept = append(kept, e)
		}
	}
	h.entries = kept
	h.reindex()
	if on {
		h.entries = append(h.entries, Entry{Species: species, Fur: fur, Time: at, Rank: 4, Reserve: -1, Manual: true, Album: true})
	}
	return h.save()
}

// Album returns the rare furs ticked by hand: species and fur hashes, and when they were ticked.
func (h *History) Album() [][3]uint32 {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out [][3]uint32
	for _, e := range h.entries {
		if e.Album {
			out = append(out, [3]uint32{e.Species, e.Fur, e.Time})
		}
	}
	return out
}

// Entries returns a copy of the history: the harvests and the trophies added by hand, not the
// furs of the album.
func (h *History) Entries() []Entry {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []Entry
	for _, e := range h.entries {
		if !e.Album {
			out = append(out, e)
		}
	}
	return out
}

// Counts returns the number of harvests recorded and of trophies added by hand.
func (h *History) Counts() (int, int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	n, m := 0, 0
	for _, e := range h.entries {
		switch {
		case e.Album:
		case e.Manual:
			m++
		default:
			n++
		}
	}
	return n, m
}

// save writes the history; when it fails, the history stays to be saved (dirty).
func (h *History) save() error {
	h.dirty = true
	var b strings.Builder
	b.WriteString("# COTWGoldChallenge - harvests seen by the program, and trophies added by hand\n")
	b.WriteString("# h species score rank time region fur / t (trophy lodges) species score rank time reserve fur /\n")
	b.WriteString("# m (by hand) species rank time reserve / p (position on the map) species time rank x y /\n")
	b.WriteString("# f (rare fur ticked by hand in the album) species fur time\n")
	list := append([]Entry(nil), h.entries...)
	sort.SliceStable(list, func(i, j int) bool { return list[i].Time < list[j].Time })
	f32 := func(v float32) string { return strconv.FormatFloat(float64(v), 'f', -1, 32) }
	for _, e := range list {
		switch {
		case e.Album:
			fmt.Fprintf(&b, "f %d %d %d\n", e.Species, e.Fur, e.Time)
		case e.Manual:
			fmt.Fprintf(&b, "m %d %d %d %d\n", e.Species, e.Rank, e.Time, e.Reserve)
		case e.Lodge:
			fmt.Fprintf(&b, "t %d %s %d %d %d %d\n", e.Species, f32(e.Score), e.Rank, e.Time, e.Reserve, e.Fur)
		default:
			fmt.Fprintf(&b, "h %d %s %d %d %d %d\n", e.Species, f32(e.Score), e.Rank, e.Time, e.Region, e.Fur)
		}
		// on lines of their own: the mods of 2.5 read the lines above as they are
		if !e.Manual && e.HasPosition() {
			fmt.Fprintf(&b, "p %d %d %d %s %s\n", e.Species, e.Time, e.Rank, f32(e.X), f32(e.Y))
		}
	}
	if err := safefile.Write(h.path, []byte(b.String())); err != nil {
		return err
	}
	h.dirty = false
	return nil
}

// Since reads the challenge start of the settings: "" (all time), YYYY-MM-DD or
// YYYY-MM-DD HH:MM, local time. Returns a Unix time, 0 for all time.
func Since(s string) uint32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return uint32(t.Unix())
		}
	}
	return 0
}

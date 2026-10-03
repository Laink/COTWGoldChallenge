package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"strings"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
)

// Reserve numbers, by the codename used in the codex (reserve_<codename>_name...).
var reserveNumbers = map[string]int{
	"hirschfelden": 0, "layton_lake_district": 1, "siberia": 2, "africa": 3, "patagonia": 4, "yukon": 6,
	"iberia": 8, "rockies": 9, "nz": 10, "mexico": 11, "delta": 12, "finland": 13, "nh": 14, "australia": 16,
	"nepal": 17, "sh": 18, "alberta": 19, "scotland": 20, "peru": 21,
}

// Codex holds what the mod needs from the game codex and translations.
type Codex struct {
	Regions      map[uint32]int    // hash of a region or landmark name key -> reserve number
	ReserveNames map[int]string    // reserve number -> translated name
	Names        map[string]string // species name key -> translated name
	loc          *loc
	lang         int
}

// Text returns the translation of a key, "" when there is none.
func (c *Codex) Text(key string) string { return c.loc.get(c.lang, key) }

// Language indexes of the translation table.
var locLanguages = map[string]int{"en": 0, "fr": 1, "de": 2, "es": 3, "ru": 4, "pl": 5, "ja": 6, "pt": 7, "cs": 8, "zh-hans": 9}

// LoadCodex reads the codex workbook (regions, wildlife) and the translations of a language.
func LoadCodex(a *apex.Archives, lang string) (*Codex, error) {
	book, err := findADF(a, 100<<10, 2<<20, "xls_types", func(b []byte) bool {
		s, err := xlsSheets(b)
		return err == nil && s["Regions"] != nil && s["Wildlife"] != nil
	})
	if err != nil {
		return nil, errors.New("codex not found")
	}
	sheets, _ := xlsSheets(book)
	locData, err := findADF(a, 5<<20, 1<<31-1, "StringLookup", nil)
	if err != nil {
		return nil, errors.New("translations not found")
	}
	loc, err := newLoc(locData)
	if err != nil {
		return nil, err
	}
	li, ok := locLanguages[lang]
	if !ok {
		li = 0
	}
	c := &Codex{Regions: map[uint32]int{}, ReserveNames: map[int]string{}, Names: map[string]string{}, loc: loc, lang: li}
	for _, sheet := range []string{"Regions", "Landmarks"} {
		rows := sheets[sheet]
		if len(rows) == 0 {
			continue
		}
		ni, ri := column(rows[0], "name"), column(rows[0], "reserve_id")
		for _, r := range rows[1:] {
			name, _ := cell(r, ni).(string)
			res, _ := cell(r, ri).(string)
			n, ok := reserveNumber(res)
			if name == "" || !ok {
				continue
			}
			c.Regions[apex.HashString(name)] = n
			if _, done := c.ReserveNames[n]; !done {
				c.ReserveNames[n] = loc.get(li, res)
			}
		}
	}
	w := sheets["Wildlife"]
	ni := column(w[0], "animal_name")
	for _, r := range w[1:] {
		key, _ := cell(r, ni).(string)
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		c.Names[key] = loc.get(li, key)
	}
	return c, nil
}

func reserveNumber(key string) (int, bool) {
	k := strings.TrimPrefix(key, "reserve_")
	k = strings.TrimSuffix(strings.TrimSuffix(k, "_uppercase"), "_name")
	n, ok := reserveNumbers[k]
	return n, ok
}

func column(header []any, name string) int {
	for i, v := range header {
		if s, ok := v.(string); ok && s == name {
			return i
		}
	}
	return -1
}

func cell(r []any, i int) any {
	if i < 0 || i >= len(r) {
		return nil
	}
	return r[i]
}

// findADF returns the first loose ADF file of the archives whose type name contains `kind`.
func findADF(a *apex.Archives, minSize, maxSize int, kind string, accept func([]byte) bool) ([]byte, error) {
	for _, e := range a.Entries {
		if int(e.Size) < minSize || int(e.Size) > maxSize {
			continue
		}
		h, err := a.ReadAt(e, 0, 0x60)
		if err != nil || len(h) < 0x60 || string(h[:4]) != " FDA" || !bytes.Contains(h[0x40:0x60], []byte(kind)) {
			continue
		}
		b, err := a.Read(e)
		if err != nil {
			continue
		}
		if accept == nil || accept(b) {
			return b, nil
		}
	}
	return nil, errors.New("not found")
}

// xlsSheets decodes an ADF spreadsheet: sheet name -> rows of cells (string, float64, bool or nil).
func xlsSheets(d []byte) (out map[string][][]any, err error) {
	defer func() {
		if recover() != nil {
			out, err = nil, errors.New("invalid spreadsheet")
		}
	}()
	u32 := func(o int) int { return int(binary.LittleEndian.Uint32(d[o:])) }
	base := u32(u32(0x0c) + 8)
	arr := func(rel int) (int, int) { return u32(base + rel), u32(base + rel + 8) }
	so, sc := arr(0)
	co, _ := arr(16)
	to, tc := arr(32)
	vo, vc := arr(48)
	bo, bc := arr(64)
	cstr := func(rel int) string {
		o := base + rel
		e := bytes.IndexByte(d[o:], 0)
		return string(d[o : o+e])
	}
	strs := make([]string, tc)
	for i := range strs {
		strs[i] = cstr(u32(base + to + i*8))
	}
	cellAt := func(i int) any {
		t, di := u32(base+co+i*12), u32(base+co+i*12+4)
		switch t {
		case 1:
			if di < tc {
				return strs[di]
			}
		case 2:
			if di < vc {
				return float64(math.Float32frombits(uint32(u32(base + vo + di*4))))
			}
		case 3:
			if di < bc {
				return d[base+bo+di] != 0
			}
		}
		return nil
	}
	out = map[string][][]any{}
	for s := 0; s < sc; s++ {
		o := base + so + s*32
		cols, rows, ci := u32(o), u32(o+4), u32(o+8)
		name := cstr(u32(o + 24))
		table := make([][]any, rows)
		for r := range table {
			table[r] = make([]any, cols)
			for c := range table[r] {
				table[r][c] = cellAt(u32(base + ci + (r*cols+c)*4))
			}
		}
		out[name] = table
	}
	return out, nil
}

// loc is the game translation table (ADF StringLookup).
type loc struct {
	d     []byte
	base  int
	text  int
	langs [][2]int // offset and count of each language index
}

func newLoc(d []byte) (l *loc, err error) {
	defer func() {
		if recover() != nil {
			l, err = nil, errors.New("invalid translations")
		}
	}()
	u32 := func(o int) int { return int(binary.LittleEndian.Uint32(d[o:])) }
	l = &loc{d: d}
	l.base = u32(u32(0x0c) + 8)
	lo, lc := u32(l.base), u32(l.base+8)
	l.text = u32(l.base + 16)
	for i := 0; i < lc; i++ {
		o := l.base + lo + i*32
		l.langs = append(l.langs, [2]int{u32(o), u32(o + 8)})
	}
	return l, nil
}

func (l *loc) get(lang int, key string) string {
	if lang >= len(l.langs) {
		lang = 0
	}
	u32 := func(o int) uint32 { return binary.LittleEndian.Uint32(l.d[o:]) }
	po, pc := l.langs[lang][0], l.langs[lang][1]
	h := apex.HashString(key)
	lo, hi := 0, pc-1
	for lo <= hi {
		mid := (lo + hi) / 2
		hh := u32(l.base + po + mid*8)
		switch {
		case hh < h:
			lo = mid + 1
		case hh > h:
			hi = mid - 1
		default:
			s := l.base + l.text + int(u32(l.base+po+mid*8+4))
			e := bytes.IndexByte(l.d[s:], 0)
			return string(l.d[s : s+e])
		}
	}
	return ""
}

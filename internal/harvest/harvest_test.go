package harvest

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// fakeLog builds a hunting log: the last harvests, then groups of best harvests.
func fakeLog(last []Entry, groups [][]Entry) []byte {
	var d []byte
	put := func(p int, v uint32) { binary.LittleEndian.PutUint32(d[p:], v) }
	grow := func(n int) int { p := len(d); d = append(d, make([]byte, n)...); return p }
	entries := func(list []Entry) int {
		p := grow(24 * len(list))
		for i, e := range list {
			o := p + i*24
			put(o, e.Species)
			put(o+4, math.Float32bits(e.Score))
			put(o+8, e.Rank)
			put(o+16, e.Time)
			put(o+20, e.Region)
		}
		return p
	}
	b := grow(16)
	copy(d[b:], " FDA")
	inst := grow(12)
	put(b+12, uint32(inst-b))
	root := grow(40)
	put(inst+8, uint32(root-b))
	put(root+8, uint32(entries(last)-root))
	put(root+16, uint32(len(last)))
	gs := grow(24 * len(groups))
	put(root+24, uint32(gs-root))
	put(root+32, uint32(len(groups)))
	for i, g := range groups {
		put(gs+i*24+8, uint32(entries(g)-root))
		put(gs+i*24+16, uint32(len(g)))
	}
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	w.Write(d)
	w.Close()
	return append(append([]byte("SAVE"), make([]byte, 28)...), z.Bytes()...)
}

func TestParse(t *testing.T) {
	a := Entry{Species: 1, Score: 12.5, Rank: 1, Time: 1000, Region: 7, Reserve: -1}
	b := Entry{Species: 2, Score: 3, Rank: 3, Time: 2000, Region: 8, Reserve: -1}
	got, err := Parse(fakeLog([]Entry{a}, [][]Entry{{a, b}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0] != a || got[2] != b {
		t.Fatalf("got %+v", got)
	}
	if _, err := Parse([]byte("SAVE nothing here at all, not even zlib")); err == nil {
		t.Fatal("no error on a broken log")
	}
}

// TestParseSave reads the save of this computer, when there is one.
func TestParseSave(t *testing.T) {
	home, _ := os.UserHomeDir()
	m, _ := filepath.Glob(filepath.Join(home, "Documents", "Avalanche Studios", "COTW", "Saves", "*", "hunting_log_adf"))
	if len(m) == 0 {
		t.Skip("no save")
	}
	raw, err := os.ReadFile(m[0])
	if err != nil {
		t.Skip(err)
	}
	list, err := Parse(raw)
	if err != nil || len(list) == 0 {
		t.Fatalf("%d entries, %v", len(list), err)
	}
}

func TestHistory(t *testing.T) {
	dir := t.TempDir()
	h, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := Entry{Species: 1, Score: 12.5, Rank: 1, Time: 1000, Region: 7, Reserve: -1}
	b := Entry{Species: 2, Score: 3.25, Rank: 3, Time: 2000, Region: 8, Reserve: -1}
	if n, err := h.Merge([]Entry{a, b, a}); n != 2 || err != nil {
		t.Fatalf("merge: %d %v", n, err)
	}
	if n, _ := h.Merge([]Entry{b}); n != 0 {
		t.Fatal("harvest added twice")
	}
	h.SetManual(3, 5, 2, 1)
	h.SetManual(3, 5, 0, 1) // replaces the previous one
	h.SetManual(4, 5, 1, 1)
	h.SetManual(4, 5, 4, 1) // removes it

	h, err = Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen, manual := h.Counts()
	if seen != 2 || manual != 1 {
		t.Fatalf("counts %d %d", seen, manual)
	}
	got := h.Entries()
	want := Entry{Species: 3, Rank: 0, Time: 1, Reserve: 5, Manual: true}
	if got[0] != want || got[1] != a || got[2] != b {
		t.Fatalf("got %+v", got)
	}
	if n, _ := h.Merge([]Entry{a}); n != 0 {
		t.Fatal("harvest of the file added again")
	}
}

func TestLoadData(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"cotwgc_species.txt":  "10|111|3|animal_a_name|Deer|\n11|222|9|animal_b_name|Bear|999\n",
		"cotwgc_scoring.txt":  "10|0|1|2|3|0,0,1,2,3,4,5;1,1,1,2,3,4,5\n11|1|1|2|3|0,0,1,2,3,4,5\n",
		"cotwgc_reserves.txt": "0|Forest|111,222\n",
		"cotwgc_regions.txt":  "50 0\n",
	}
	for name, s := range files {
		os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644)
	}
	d, err := LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a, b := d.Species[111], d.Species[222]; a.Name != "Deer" || b.Class != 9 || !a.GreatOne || b.GreatOne {
		t.Errorf("species %+v %+v", a, b)
	}
	if d.Alias[999] != 222 || d.Regions[50] != 0 || len(d.Reserves) != 1 || len(d.Reserves[0].Species) != 2 {
		t.Errorf("data %+v", d)
	}
}

func TestPlace(t *testing.T) {
	d := &Data{
		Alias:    map[uint32]uint32{99: 1},
		Reserves: []Reserve{{Number: 0, Species: []uint32{1, 2}}, {Number: 1, Species: []uint32{1, 3}}},
		Regions:  map[uint32]int{50: 1},
	}
	list := d.Place([]Entry{
		{Species: 1, Rank: 2, Time: 10000, Region: 50},               // region: reserve 1
		{Species: 2, Rank: 1, Time: 50000, Region: 0},                // only reserve of the species: 0
		{Species: 99, Rank: 6, Time: 11000, Region: 0},               // alias, Great One, closest harvest: 1
		{Species: 1, Rank: 3, Time: 30000, Region: 0},                // nothing within 4 hours
		{Species: 3, Rank: 0, Time: 100, Reserve: 1, Manual: true},   // by hand
		{Species: 1, Rank: 0, Time: 49000, Reserve: 0, Manual: true}, // by hand: not a trip
	})
	for i, r := range []int{1, 0, 1, -1, 1, 0} {
		if list[i].Reserve != r {
			t.Errorf("entry %d: reserve %d, want %d", i, list[i].Reserve, r)
		}
	}
	if list[2].Species != 1 || list[2].Rank != 0 || !list[2].GreatOne {
		t.Errorf("Great One: %+v", list[2])
	}
	if r, g1 := Best(list, 1, 0, 1, nil); r != 0 || !g1 {
		t.Errorf("best in reserve 1: %d %v", r, g1)
	}
	if r, _ := Best(list, 1, 12000, 1, nil); r != 4 {
		t.Errorf("best since: %d", r)
	}
	manual := func(e Placed) bool { return e.Manual }
	if r, _ := Best(list, 1, 0, -1, manual); r != 0 {
		t.Errorf("best anywhere: %d", r)
	}
	if r, _ := Best(list, 3, 0, 1, manual); r != 4 {
		t.Errorf("best without the trophies by hand: %d", r)
	}
}

func TestFurAndLodges(t *testing.T) {
	dir := t.TempDir()
	h, _ := Load(dir)
	// a harvest recorded before the fur was kept gets it from the hunting log
	h.add(Entry{Species: 1, Rank: 1, Time: 1000, Reserve: -1})
	if n, err := h.Merge([]Entry{{Species: 1, Rank: 1, Time: 1000, Fur: 77}}); n != 0 || err != nil {
		t.Fatalf("merge %d %v", n, err)
	}
	// the same harvest in the trophy lodges, 4 s apart: not added twice
	h.Merge([]Entry{{Species: 1, Rank: 1, Time: 1004, Fur: 77, Reserve: 3, Lodge: true}})
	// a harvest only in the lodges, then seen in the log: the log one replaces it
	h.Merge([]Entry{{Species: 2, Rank: 0, Time: 5000, Fur: 9, Reserve: 5, Lodge: true}})
	h.Merge([]Entry{{Species: 2, Rank: 0, Time: 4997, Region: 8}})
	// a harvest only in the lodges stays there
	h.Merge([]Entry{{Species: 3, Rank: 2, Time: 9000, Fur: 4, Reserve: 7, Lodge: true}})
	h, _ = Load(dir)
	got := h.Entries()
	want := []Entry{
		{Species: 1, Rank: 1, Time: 1000, Reserve: -1, Fur: 77},
		{Species: 2, Rank: 0, Time: 4997, Region: 8, Reserve: -1, Fur: 9},
		{Species: 3, Rank: 2, Time: 9000, Reserve: 7, Fur: 4, Lodge: true},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d: %+v, want %+v", i, got[i], want[i])
		}
	}
	d := &Data{Furs: map[[2]uint32]Fur{{3, 4}: {Rarity: 3, Name: "Albino"}, {1, 77}: {Rarity: 0, Name: "Brown"}}}
	list := d.Place(got)
	if f := d.RareFurs(list, 3, 0, 7); len(f) != 1 || f[0] != "Albino" {
		t.Errorf("rare furs: %v", f)
	}
	if f := d.RareFurs(list, 1, 0, -1); len(f) != 0 {
		t.Errorf("common fur counted: %v", f)
	}
}

// TestParseLodgesSave reads the trophy lodges of this computer, with the data files of the folder
// COTWGC_DATA (dropzone/ui of an installed mod).
func TestParseLodgesSave(t *testing.T) {
	home, _ := os.UserHomeDir()
	m, _ := filepath.Glob(filepath.Join(home, "Documents", "Avalanche Studios", "COTW", "Saves", "*", LodgesFile))
	data := os.Getenv("COTWGC_DATA")
	if len(m) == 0 || data == "" {
		t.Skip("no save or no data files")
	}
	d, err := LoadData(data)
	if err != nil {
		t.Skip("no data files of 2.5")
	}
	raw, _ := os.ReadFile(m[0])
	list, err := ParseLodges(raw, d)
	if err != nil || len(list) == 0 {
		t.Fatalf("%d trophies, %v", len(list), err)
	}
	named := 0
	for _, e := range list {
		if d.Furs[[2]uint32{e.Species, e.Fur}].Name != "" {
			named++
		}
	}
	if named != len(list) {
		t.Errorf("%d furs of %d named", named, len(list))
	}
}

func TestPositions(t *testing.T) {
	dir := t.TempDir()
	h, _ := Load(dir)
	// the last harvest of the lodges, with its position, then the same harvest in the log
	h.Merge([]Entry{{Species: 5, Rank: 1, Time: 2000, Reserve: 17, Fur: 3, X: 0.25, Y: 0.75, Lodge: true}})
	h.Merge([]Entry{{Species: 5, Rank: 1, Time: 1996, Region: 9}})
	// a harvest of the log, then its lodge twin with the position
	h.Merge([]Entry{{Species: 6, Rank: 2, Time: 3000, Region: 9}})
	if _, err := h.Merge([]Entry{{Species: 6, Rank: 2, Time: 3003, X: 0.5, Y: 0.125, Lodge: true}}); err != nil {
		t.Fatal(err)
	}
	h, _ = Load(dir)
	got := h.Entries()
	if len(got) != 2 {
		t.Fatalf("got %+v", got)
	}
	if e := got[0]; e.Lodge || e.Time != 1996 || e.X != 0.25 || e.Y != 0.75 || e.Fur != 3 {
		t.Errorf("first %+v", e)
	}
	if e := got[1]; e.Lodge || e.X != 0.5 || e.Y != 0.125 {
		t.Errorf("second %+v", e)
	}
	// a manual trophy does not break the index
	h.SetManual(7, 1, 0, 1)
	if n, _ := h.Merge([]Entry{{Species: 6, Rank: 2, Time: 3000, Region: 9}}); n != 0 {
		t.Error("harvest added twice after a manual trophy")
	}
}

// TestTwins: the lodge harvest goes to the closest harvest of the log, a puma of the log (its
// other hash) is found again in the lodges.
func TestTwins(t *testing.T) {
	h, _ := Load(t.TempDir())
	h.SetAliases(map[uint32]uint32{99: 9})
	h.Merge([]Entry{{Species: 5, Rank: 2, Time: 100, Region: 1}, {Species: 5, Rank: 2, Time: 130, Region: 1}, {Species: 99, Rank: 1, Time: 500, Region: 1}})
	h.Merge([]Entry{{Species: 5, Rank: 2, Time: 133, X: 0.5, Y: 0.5, Lodge: true}, {Species: 9, Rank: 1, Time: 504, X: 0.25, Y: 0.25, Lodge: true}})
	got := h.Entries()
	if len(got) != 3 {
		t.Fatalf("got %d harvests: %+v", len(got), got)
	}
	for _, e := range got {
		want := e.Time == 130 || e.Time == 500
		if e.HasPosition() != want {
			t.Errorf("harvest at %d: position %v %v", e.Time, e.X, e.Y)
		}
	}
}

// TestSaveAgain: a save that failed is done again by the next merge or by Flush.
func TestSaveAgain(t *testing.T) {
	dir := t.TempDir()
	h, _ := Load(dir)
	good := h.path
	h.path = filepath.Join(dir, "missing", HistoryFile)
	if _, err := h.Merge([]Entry{{Species: 5, Rank: 1, Time: 100, Region: 1}}); err == nil {
		t.Fatal("save in a missing folder")
	}
	h.path = good
	if _, err := h.Merge([]Entry{{Species: 5, Rank: 1, Time: 100, Region: 1}}); err != nil {
		t.Fatal(err)
	}
	if h2, _ := Load(dir); len(h2.Entries()) != 1 {
		t.Error("harvest not saved again by the next merge")
	}
	h.path = filepath.Join(dir, "missing", HistoryFile)
	h.SetManual(7, 1, 0, 1)
	h.path = good
	if err := h.Flush(); err != nil {
		t.Fatal(err)
	}
	if h2, _ := Load(dir); len(h2.Entries()) != 2 {
		t.Error("manual trophy not saved again by Flush")
	}
}

// TestLoadLongLine: a history not read to the end is an error, not a shorter history.
func TestLoadLongLine(t *testing.T) {
	dir := t.TempDir()
	text := "h 5 10 1 100 1 0\n" + string(bytes.Repeat([]byte("x"), 2<<20)) + "\nh 6 10 1 200 1 0\n"
	os.WriteFile(filepath.Join(dir, HistoryFile), []byte(text), 0o644)
	if _, err := Load(dir); err == nil {
		t.Error("history with a too long line read without error")
	}
}

// TestParseLodgesDamaged: a damaged lodges file is an error, never a crash.
func TestParseLodgesDamaged(t *testing.T) {
	d := &Data{Engine: map[uint32]uint32{1: 1}}
	for _, raw := range [][]byte{nil, []byte(" FDA"), append([]byte(" FDA"), make([]byte, 64)...), append([]byte(" FDA"), bytes.Repeat([]byte{0xff}, 200)...)} {
		if _, err := ParseLodges(raw, d); err == nil {
			t.Errorf("no error for %d bytes", len(raw))
		}
	}
}

// TestAlbum: rare furs ticked by hand are kept apart from the harvests and trophies.
func TestAlbum(t *testing.T) {
	dir := t.TempDir()
	h, _ := Load(dir)
	h.Merge([]Entry{{Species: 5, Rank: 1, Time: 100, Region: 1}})
	h.SetManual(5, 2, 0, 50)
	if err := h.SetAlbum(5, 77, true, 200); err != nil {
		t.Fatal(err)
	}
	h.SetAlbum(6, 78, true, 201)
	h.SetAlbum(6, 78, false, 202)
	h, _ = Load(dir)
	if a := h.Album(); len(a) != 1 || a[0] != [2]uint32{5, 77} {
		t.Errorf("album %v", a)
	}
	if n, m := h.Counts(); n != 1 || m != 1 || len(h.Entries()) != 2 {
		t.Errorf("counts %d %d, entries %d", n, m, len(h.Entries()))
	}
	// a trophy by hand changed: the album stays
	h.SetManual(5, 2, 4, 50)
	if len(h.Album()) != 1 {
		t.Error("album lost with the trophy")
	}
}

func TestLastDone(t *testing.T) {
	d := &Data{Reserves: []Reserve{{Number: 1, Species: []uint32{5, 6}}, {Number: 2, Species: []uint32{5}}}}
	list := []Placed{
		{Entry: Entry{Species: 5, Rank: 1, Time: 100, Reserve: 1}},
		{Entry: Entry{Species: 5, Rank: 1, Time: 300, Reserve: 1}}, // the same species again
		{Entry: Entry{Species: 6, Rank: 2, Time: 400, Reserve: 1}}, // silver: not done for gold
		{Entry: Entry{Species: 5, Rank: 0, Time: 200, Reserve: 2}}, // diamond, better than gold
	}
	if v := d.LastDone(list, 0, 1, true, false); v != 200 {
		t.Errorf("reserve mode: %d, want 200", v)
	}
	if v := d.LastDone(list, 0, 1, true, true); v != 100 {
		t.Errorf("100%% mode: %d, want 100", v)
	}
	if v := d.LastDone(list, 0, 1, false, false); v != 100 {
		t.Errorf("gold only: %d, want 100", v)
	}
}

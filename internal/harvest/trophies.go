package harvest

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sameTrip: harvests this close in time are in the same reserve (as the mod).
const sameTrip = 4 * 3600

// Species is a species of the data files written at installation.
type Species struct {
	Hash     uint32
	Name     string
	Class    int
	GreatOne bool // the species has a Great One
	Icon     int  // frame of the icons clip, minus one
}

// Reserve is a reserve of the data files, with its species in the order of the overlay.
type Reserve struct {
	Number  int
	Name    string
	Species []uint32
}

// Data are the data files read by the mod in dropzone/ui.
type Data struct {
	Species  map[uint32]Species
	Alias    map[uint32]uint32 // other hash used by the hunting log -> species hash
	Reserves []Reserve
	Regions  map[uint32]int       // region hash -> reserve number
	Furs     map[[2]uint32]Fur    // species hash, fur hash -> fur
	Engine   map[uint32]uint32    // hash of the engine name (trophy lodges) -> species hash
	FurIndex map[[2]uint32]uint32 // species hash, variation index (trophy lodges) -> fur hash
}

// Fur is a fur of a species. Rarity: 0 common, 1 uncommon, 2 rare, 3 very rare.
type Fur struct {
	Rarity int
	Name   string
	Odds   [2]float64 // chance in percent of a male and of a female (0: not of that gender)
}

// RareFur is the lowest rarity counted as a rare fur.
const RareFur = 2

// LoadData reads cotwgc_species.txt, cotwgc_reserves.txt and cotwgc_regions.txt.
func LoadData(uiDir string) (*Data, error) {
	d := &Data{Species: map[uint32]Species{}, Alias: map[uint32]uint32{}, Regions: map[uint32]int{}, Furs: map[[2]uint32]Fur{},
		Engine: map[uint32]uint32{}, FurIndex: map[[2]uint32]uint32{}}
	read := func(name string) ([]string, error) {
		b, err := os.ReadFile(filepath.Join(uiDir, name))
		if err != nil {
			return nil, err
		}
		return strings.Split(strings.ReplaceAll(string(b), "\r", ""), "\n"), nil
	}
	num := func(s string) uint32 {
		v, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 32)
		return uint32(v)
	}
	lines, err := read("cotwgc_species.txt")
	if err != nil {
		return nil, err
	}
	// icon|hash|class|name key|name|other hashes
	byIcon := map[string]uint32{}
	for _, l := range lines {
		f := strings.Split(l, "|")
		if len(f) < 5 {
			continue
		}
		h := num(f[1])
		cls, _ := strconv.Atoi(f[2])
		icon, _ := strconv.Atoi(f[0])
		d.Species[h] = Species{Hash: h, Name: f[4], Class: cls, Icon: icon}
		byIcon[f[0]] = h
		if len(f) > 6 && f[6] != "" {
			d.Engine[num(f[6])] = h
		}
		if len(f) > 5 {
			for _, a := range strings.Split(f[5], ",") {
				if a != "" {
					d.Alias[num(a)] = h
				}
			}
		}
	}
	// icon|weight based|silver|gold|diamond|gender,great one,...;...
	if lines, err = read("cotwgc_scoring.txt"); err != nil {
		return nil, err
	}
	for _, l := range lines {
		f := strings.Split(l, "|")
		s, ok := d.Species[byIcon[f[0]]]
		if len(f) < 6 || !ok {
			continue
		}
		for _, dist := range strings.Split(f[5], ";") {
			if g := strings.Split(dist, ","); len(g) > 1 && g[1] == "1" {
				s.GreatOne = true
			}
		}
		d.Species[s.Hash] = s
	}
	if lines, err = read("cotwgc_reserves.txt"); err != nil {
		return nil, err
	}
	// number|name|species hashes
	for _, l := range lines {
		f := strings.Split(l, "|")
		if len(f) < 3 {
			continue
		}
		n, _ := strconv.Atoi(f[0])
		r := Reserve{Number: n, Name: f[1]}
		for _, h := range strings.Split(f[2], ",") {
			if h != "" {
				r.Species = append(r.Species, num(h))
			}
		}
		d.Reserves = append(d.Reserves, r)
	}
	if lines, err = read("cotwgc_regions.txt"); err != nil {
		return nil, err
	}
	for _, l := range lines {
		if f := strings.Fields(l); len(f) == 2 {
			n, _ := strconv.Atoi(f[1])
			d.Regions[num(f[0])] = n
		}
	}
	// species hash|fur hash|rarity|name; missing for a mod installed before version 2.5
	lines, _ = read("cotwgc_furs.txt")
	for _, l := range lines {
		if f := strings.Split(l, "|"); len(f) >= 4 {
			r, _ := strconv.Atoi(f[2])
			fur := Fur{Rarity: r, Name: f[3]}
			if len(f) > 5 {
				for i, s := range strings.SplitN(f[5], ",", 2) {
					fur.Odds[i], _ = strconv.ParseFloat(s, 64)
				}
			}
			d.Furs[[2]uint32{num(f[0]), num(f[1])}] = fur
			if len(f) > 4 {
				for _, i := range strings.Split(f[4], ",") {
					if i != "" {
						d.FurIndex[[2]uint32{num(f[0]), num(i)}] = num(f[1])
					}
				}
			}
		}
	}
	if len(d.Species) == 0 || len(d.Reserves) == 0 {
		return nil, errors.New("empty data files")
	}
	return d, nil
}

// Place returns the harvests with their species hash resolved, their rank 0 diamond .. 4 none
// (a Great One counts as a diamond, GreatOne set), and their reserve, as the mod: from the
// region; otherwise the only reserve of the species; otherwise the reserve of the closest
// harvest in time, within 4 hours. Manual trophies keep theirs.
func (d *Data) Place(list []Entry) []Placed {
	only := map[uint32]int{}
	for _, r := range d.Reserves {
		for _, h := range r.Species {
			if _, ok := only[h]; ok {
				only[h] = -1
			} else {
				only[h] = r.Number
			}
		}
	}
	out := make([]Placed, len(list))
	var known []int
	for i, e := range list {
		p := Placed{Entry: e}
		if a, ok := d.Alias[p.Species]; ok {
			p.Species = a
		}
		if p.Rank > 4 {
			p.Rank, p.GreatOne = 0, true
		}
		if !e.Manual {
			if e.Lodge && e.Reserve >= 0 {
				// the lodges know the reserve
			} else if r, ok := d.Regions[e.Region]; ok {
				p.Reserve = r
			} else if r, ok := only[p.Species]; ok && r >= 0 {
				p.Reserve = r
			} else {
				p.Reserve = -1
			}
			if p.Reserve >= 0 {
				known = append(known, i)
			}
		}
		out[i] = p
	}
	for i := range out {
		if out[i].Manual || out[i].Reserve >= 0 {
			continue
		}
		gap := int64(sameTrip)
		for _, k := range known {
			g := int64(out[k].Time) - int64(out[i].Time)
			if g < 0 {
				g = -g
			}
			if g < gap {
				gap, out[i].Reserve = g, out[k].Reserve
			}
		}
	}
	return out
}

// Placed is a harvest ready for the challenge: see Data.Place.
type Placed struct {
	Entry
	GreatOne bool
}

// Best returns the best rank (0 diamond .. 4 none) of a species since a time, in a reserve
// (any reserve when reserve < 0), and whether a Great One is among them. skip, when not nil,
// leaves out some harvests.
func Best(list []Placed, species uint32, since uint32, reserve int, skip func(Placed) bool) (uint32, bool) {
	rank, g1 := uint32(4), false
	for _, e := range list {
		if e.Species != species || e.Time < since || (reserve >= 0 && e.Reserve != reserve) || (skip != nil && skip(e)) {
			continue
		}
		rank = min(rank, e.Rank)
		g1 = g1 || e.GreatOne
	}
	return rank, g1
}

// RareFurs returns the names of the rare furs of a species harvested since a time, in a reserve
// (any reserve when reserve < 0).
func (d *Data) RareFurs(list []Placed, species uint32, since uint32, reserve int) []string {
	var out []string
	seen := map[uint32]bool{}
	for _, e := range list {
		if e.Species != species || e.Manual || e.Time < since || (reserve >= 0 && e.Reserve != reserve) || seen[e.Fur] {
			continue
		}
		if f, ok := d.Furs[[2]uint32{species, e.Fur}]; ok && f.Rarity >= RareFur {
			seen[e.Fur] = true
			out = append(out, f.Name)
		}
	}
	return out
}

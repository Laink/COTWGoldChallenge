// Package game reads trophy data and settings of theHunter: Call of the Wild.
package game

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
)

// RTPC field hashes.
const (
	pName         = 0xd31ab684
	pClass        = 0x1473b179
	pIconIndex    = 0x19b8918c
	pVariantScore = 0x2366e14f
	pAnimation    = 0x475d8996
	pNameKey      = 0x766b788e
	pTrophyMin    = 0x140687d8
	pTrophyMax    = 0xe3062a0e
	pGreatOne     = 0xdaa06641
	pActive       = 0xbaace199
	pGeneration   = 0x7d6f07cc
	pXMLFile      = 0xba5330a0
	pWeaponClass  = 0x27808d4a
	pTrophyType   = 0x3cb71423
	pGender       = 0x69e88c57 // "gender"
	pWeightMin    = 0x87ea42c8 // "weight_min"
	pWeightMax    = 0x29f241f4 // "weight_max"
	pScoreDev     = 0xac99ddc9 // "score_deviation"
	pFurName      = 0xf336b29f // fur of a visual variation, e.g. "animal_visual_variation_albino"
	pFurRarity    = 0xc82af7b4 // 0 common, 1 uncommon, 2 rare, 3 very rare
	pFurIndex     = 0x73ead679 // index of a visual variation (VariationIndex of the trophy lodges)
	pFurWeight    = 0xd8a03db0 // "probability": weight of a visual variation
)

// Dist is one scoring distribution of a species: the weight and score ranges of one gender,
// of the regular animals or of the Great Ones.
type Dist struct {
	Gender   int // 0 male, 1 female
	GreatOne bool
	WMin     float64
	WMax     float64
	SMin     float64
	SMax     float64
	Dev      float64 // score deviation, as a fraction of the score range
}

// Species holds the trophy scale of one species.
type Species struct {
	Key     string  // engine name, e.g. "red_fox"
	NameKey string  // localisation key, e.g. "animal_redfox_name"
	Icon    int     // animal_id sent to the HUD
	Class   int     // hunting class, 1 to 9
	Min     float64 // lowest possible trophy score
	Max     float64 // highest possible trophy score
	Silver  float64
	Gold    float64
	Diamond float64
	TruRACS bool
	Weight  bool // the trophy score follows the weight
	Dists   []Dist
	// Furs gives the rarity of each fur, by its name ("animal_visual_variation_albino", whose hash
	// is the fur of the hunting log): 0 common .. 3 very rare. A fur of several rarities (by gender)
	// keeps the lowest.
	Furs map[string]int
	// FurIndex gives the fur of each visual variation index (the trophies of the lodges).
	FurIndex map[int]string
	// FurOdds gives the chance of each fur, in percent, for a male and for a female (0 when the
	// fur is not of that gender).
	FurOdds map[string][2]float64
}

func name(n *apex.Node) string {
	if s, ok := n.Props[pName].(string); ok && s != "" {
		return s
	}
	s, _ := n.Props[pClass].(string)
	return s
}

func f64(v any) (float64, bool) {
	switch x := v.(type) {
	case float32:
		return float64(x), true
	case uint32:
		return float64(x), true
	}
	return 0, false
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

type antlerSet struct {
	active     bool
	generation uint32
}

// LoadSpecies computes the medal thresholds of every species from the game files,
// the same way the game does: silver/gold/diamond = min + 20/60/90 % of the range.
func LoadSpecies(a *apex.Archives, progress func(step string, done, total int)) (map[int]*Species, error) {
	types, err := a.FindInContainers("global/global_animal_types.blo", func(d, t int) {
		if progress != nil {
			progress("types", d, t)
		}
	})
	if err != nil {
		return nil, err
	}
	root, err := apex.ParseRTPC(types)
	if err != nil {
		return nil, err
	}
	var list *apex.Node
	for _, c := range root.Children {
		if name(c) == "AnimalTypesList" {
			list = c
		}
	}
	if list == nil {
		return nil, errors.New("AnimalTypesList not found")
	}

	var tables map[string][2]float64
	out := map[int]*Species{}
	for _, sp := range list.Children {
		icon, ok := sp.Props[pIconIndex].(uint32)
		if !ok {
			continue
		}
		s := &Species{Key: name(sp), Icon: int(icon), Min: math.Inf(1), Max: math.Inf(-1)}
		s.NameKey, _ = sp.Props[pNameKey].(string)
		if c, ok := sp.Props[pWeaponClass].(uint32); ok {
			s.Class = int(c)
		}
		if t, _ := sp.Props[pTrophyType].(string); t == "harvest_trophy_type_weight" {
			s.Weight = true
		}
		var sets []antlerSet
		for _, c := range sp.Children {
			if name(c) == "VisualVariationSettings" {
				s.Furs, s.FurIndex, s.FurOdds = furs(c)
			}
			if name(c) != "ScoringSettings" {
				continue
			}
			for _, sc := range c.Children {
				if mx, ok := f64(sc.Props[pTrophyMax]); ok {
					g, _ := sc.Props[pGreatOne].(uint32)
					d := Dist{GreatOne: g != 0, SMax: mx}
					d.SMin, _ = f64(sc.Props[pTrophyMin])
					d.WMin, _ = f64(sc.Props[pWeightMin])
					d.WMax, _ = f64(sc.Props[pWeightMax])
					d.Dev, _ = f64(sc.Props[pScoreDev])
					if gd, ok := sc.Props[pGender].(uint32); ok {
						d.Gender = int(gd)
					}
					s.Dists = append(s.Dists, d)
					if g != 0 || mx <= 0 {
						continue
					}
					mn, _ := f64(sc.Props[pTrophyMin])
					s.Min = math.Min(s.Min, mn)
					s.Max = math.Max(s.Max, mx)
				} else if _, ok := sc.Props[pXMLFile]; ok {
					act, _ := sc.Props[pActive].(uint32)
					gen, _ := sc.Props[pGeneration].(uint32)
					sets = append(sets, antlerSet{act != 0, gen})
				}
			}
		}
		if math.IsInf(s.Min, 0) {
			continue
		}
		// TruRACS species: the active generation-2 antler/horn tables are authoritative.
		maxGen := uint32(0)
		for _, st := range sets {
			if st.active {
				g := st.generation
				if g == 0 {
					g = 1
				}
				if g > maxGen {
					maxGen = g
				}
			}
		}
		if maxGen >= 2 {
			if tables == nil {
				if tables, err = loadAntlerTables(a, progress); err != nil {
					return nil, err
				}
			}
			prefix := strings.ReplaceAll(strings.ReplaceAll(s.Key, "_", ""), "desert", "")
			lo, hi := math.Inf(1), math.Inf(-1)
			for k, v := range tables {
				if strings.Contains(k, "truracs_gen02") && strings.HasPrefix(strings.ReplaceAll(k, "_", ""), prefix) {
					lo, hi = math.Min(lo, v[0]), math.Max(hi, v[1])
				}
			}
			if !math.IsInf(lo, 0) {
				s.Min, s.Max, s.TruRACS = lo, hi, true
			}
		}
		r := s.Max - s.Min
		s.Silver, s.Gold, s.Diamond = round2(s.Min+.2*r), round2(s.Min+.6*r), round2(s.Min+.9*r)
		out[s.Icon] = s
	}
	return out, nil
}

// furs reads the visual variations of a species: fur name -> rarity, index -> fur name, and fur
// name -> chance in percent for a male and a female. A variation of gender 0 is of both genders,
// 1 male, 2 female; its weight is shared with the variations of the same gender.
func furs(settings *apex.Node) (map[string]int, map[int]string, map[string][2]float64) {
	out, byIndex, odds := map[string]int{}, map[int]string{}, map[string][2]float64{}
	var total [2]float64
	for _, v := range settings.Children {
		w, _ := v.Props[pFurWeight].(uint32)
		g, _ := v.Props[pGender].(uint32)
		for i := range total {
			if g == 0 || int(g) == i+1 {
				total[i] += float64(w)
			}
		}
	}
	for _, v := range settings.Children {
		fur, _ := v.Props[pFurName].(string)
		r, ok := v.Props[pFurRarity].(uint32)
		if fur == "" || !ok {
			continue
		}
		if i, ok := v.Props[pFurIndex].(uint32); ok {
			byIndex[int(i)] = fur
		}
		if old, seen := out[fur]; !seen || int(r) < old {
			out[fur] = int(r)
		}
		w, _ := v.Props[pFurWeight].(uint32)
		g, _ := v.Props[pGender].(uint32)
		o := odds[fur]
		for i := range total {
			if (g == 0 || int(g) == i+1) && total[i] > 0 {
				o[i] += 100 * float64(w) / total[i] // a fur can have several variations
			}
		}
		odds[fur] = o
	}
	return out, byIndex, odds
}

// loadAntlerTables reads the TruRACS antler/horn variant tables: name -> full trophy range.
// Each variant stores half of the trophy score.
func loadAntlerTables(a *apex.Archives, progress func(string, int, int)) (map[string][2]float64, error) {
	out := map[string][2]float64{}
	for i, e := range a.Entries {
		if progress != nil && i%2000 == 0 {
			progress("antlers", i, len(a.Entries))
		}
		if e.Size < 20 || e.Size > 50<<20 {
			continue
		}
		path, ok := animationPath(a, e)
		if !ok || !strings.HasSuffix(path, ".al") || !strings.Contains(path, "truracs_gen02") {
			continue
		}
		key := path[strings.LastIndex(path, "/")+1 : len(path)-3]
		full, err := a.Read(e)
		if err != nil {
			continue
		}
		root, err := apex.ParseRTPC(full)
		if err != nil {
			continue
		}
		lo, hi := math.Inf(1), math.Inf(-1)
		stack := []*apex.Node{root}
		for len(stack) > 0 {
			n := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if v, ok := n.Props[pVariantScore].(float32); ok {
				lo, hi = math.Min(lo, float64(v)*2), math.Max(hi, float64(v)*2)
			}
			stack = append(stack, n.Children...)
		}
		if math.IsInf(lo, 0) {
			continue
		}
		if cur, ok := out[key]; ok {
			lo, hi = math.Min(lo, cur[0]), math.Max(hi, cur[1])
		}
		out[key] = [2]float64{lo, hi}
	}
	return out, nil
}

// animationPath reads the animation path of an RTPC root node, without loading the file.
func animationPath(a *apex.Archives, e apex.Entry) (string, bool) {
	h, err := a.ReadAt(e, 0, 20)
	if err != nil || len(h) < 20 || string(h[:4]) != "RTPC" {
		return "", false
	}
	data := int(binary.LittleEndian.Uint32(h[12:]))
	np := int(binary.LittleEndian.Uint16(h[16:]))
	if np == 0 || np > 4096 {
		return "", false
	}
	props, err := a.ReadAt(e, data, 9*np)
	if err != nil || len(props) < 9*np {
		return "", false
	}
	for i := 0; i < np; i++ {
		o := 9 * i
		if binary.LittleEndian.Uint32(props[o:]) != pAnimation || props[o+8] != 3 {
			continue
		}
		str, err := a.ReadAt(e, int(binary.LittleEndian.Uint32(props[o+4:])), 512)
		if err != nil {
			return "", false
		}
		if n := bytes.IndexByte(str, 0); n >= 0 {
			return string(str[:n]), true
		}
	}
	return "", false
}

// SortedIcons returns the icon ids in ascending order.
func SortedIcons(m map[int]*Species) []int {
	ids := make([]int, 0, len(m))
	for k := range m {
		ids = append(ids, k)
	}
	sort.Ints(ids)
	return ids
}

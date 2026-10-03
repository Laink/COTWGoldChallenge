// Package moddata writes the data files read by the mod in dropzone/ui, from the game files.
package moddata

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
	"github.com/Laink/COTWGoldChallenge/internal/game"
)

// Files are the data files, relative to dropzone/ui.
var Files = []string{"cotwgc_species.txt", "cotwgc_scoring.txt", "cotwgc_reserves.txt", "cotwgc_regions.txt", "cotwgc_furs.txt"}

// Other names used by the hunting log for some species.
var aliases = map[string][]string{"animal_puma_name": {"animal_mountainlion_name"}}

// f32 formats a value read as a float32 without the float64 noise.
func f32(v float64) string { return strconv.FormatFloat(v, 'g', -1, 32) }

// pct formats a chance in percent with 3 significant digits.
func pct(v float64) string { return strconv.FormatFloat(v, 'g', 3, 64) }

// Write writes the data files into dir. cx gives the names in the game language.
func Write(dir string, sp map[int]*game.Species, cx *game.Codex) error {
	var b bytes.Buffer
	// icon|name hash|class|name key|name|other hashes of the hunting log|hash of the engine name
	// (the species of the trophy lodges)
	for _, id := range game.SortedIcons(sp) {
		s := sp[id]
		var al []string
		for _, a := range aliases[s.NameKey] {
			al = append(al, fmt.Sprint(apex.HashString(a)))
		}
		name := strings.ReplaceAll(cx.Names[s.NameKey], "|", " ")
		fmt.Fprintf(&b, "%d|%d|%d|%s|%s|%s|%d\n", id, apex.HashString(s.NameKey), s.Class, s.NameKey, name, strings.Join(al, ","), apex.HashString(s.Key))
	}
	files := map[string][]byte{Files[0]: append([]byte{}, b.Bytes()...)}
	b.Reset()
	// icon|weight based|silver|gold|diamond|gender,great one,wmin,wmax,smin,smax,dev;...
	for _, id := range game.SortedIcons(sp) {
		s := sp[id]
		var ds []string
		for _, d := range s.Dists {
			g1 := 0
			if d.GreatOne {
				g1 = 1
			}
			ds = append(ds, fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s", d.Gender, g1, f32(d.WMin), f32(d.WMax), f32(d.SMin), f32(d.SMax), f32(d.Dev)))
		}
		w := 0
		if s.Weight {
			w = 1
		}
		fmt.Fprintf(&b, "%d|%d|%g|%g|%g|%s\n", id, w, s.Silver, s.Gold, s.Diamond, strings.Join(ds, ";"))
	}
	files[Files[1]] = append([]byte{}, b.Bytes()...)
	b.Reset()
	// reserve number|name|species name hashes
	var nums []int
	for n := range game.ReserveSpecies {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		var hs []string
		for _, k := range game.ReserveSpecies[n] {
			hs = append(hs, fmt.Sprint(apex.HashString("animal_"+k+"_name")))
		}
		fmt.Fprintf(&b, "%d|%s|%s\n", n, strings.ReplaceAll(cx.ReserveNames[n], "|", " "), strings.Join(hs, ","))
	}
	files[Files[2]] = append([]byte{}, b.Bytes()...)
	b.Reset()
	// region hash, reserve number
	var hs []uint32
	for h := range cx.Regions {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i] < hs[j] })
	for _, h := range hs {
		fmt.Fprintf(&b, "%d %d\n", h, cx.Regions[h])
	}
	files[Files[3]] = append([]byte{}, b.Bytes()...)
	b.Reset()
	// species name hash|fur hash (the fur of the hunting log)|rarity 0 common .. 3 very rare|name|
	// variation indexes (the fur of the trophy lodges)|chance in percent of a male,of a female
	for _, id := range game.SortedIcons(sp) {
		s := sp[id]
		var keys []string
		for k := range s.Furs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			name := cx.Text(k)
			if name == "" {
				name = strings.ReplaceAll(strings.TrimPrefix(k, "animal_visual_variation_"), "_", " ")
			}
			var idx []int
			for i, f := range s.FurIndex {
				if f == k {
					idx = append(idx, i)
				}
			}
			sort.Ints(idx)
			odds := s.FurOdds[k]
			fmt.Fprintf(&b, "%d|%d|%d|%s|%s|%s,%s\n", apex.HashString(s.NameKey), apex.HashString(k), s.Furs[k], strings.ReplaceAll(name, "|", " "),
				strings.Trim(strings.ReplaceAll(fmt.Sprint(idx), " ", ","), "[]"), pct(odds[0]), pct(odds[1]))
		}
	}
	files[Files[4]] = append([]byte{}, b.Bytes()...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range Files {
		if err := os.WriteFile(filepath.Join(dir, name), files[name], 0o644); err != nil {
			return err
		}
	}
	return nil
}

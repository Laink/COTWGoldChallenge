package maps

import (
	"bytes"
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
)

// Zone is a region of a reserve: its outline and the place of its name on the map, 0 to 1 over
// the area played, as the positions of the harvests.
type Zone struct {
	Hash    uint32       `json:"h"` // hash of its name key: the region of the hunting log
	Level   int          `json:"l"` // 0 region, 1 part of a region, 2 place
	Outline [][2]float32 `json:"p"`
	Label   [2]float32   `json:"c"`
}

// Map is what the page draws on the map of a reserve: where the area played is on the image, and
// the regions.
type Map struct {
	// Frame is left, top and scale: a position x, y (0 to 1) is at left + scale*x, top + scale*y
	// on the image (0 to 1).
	Frame [3]float64 `json:"f"`
	Zones []Zone     `json:"z"`
}

// Side is the side of the area played, in metres: that of the habitat grids of the game, 256 cells
// of 32 m. The positions of the harvests are over it.
const Side = 8192

// DefaultFrame is the frame of the maps of most reserves: the image is twice the area played, which
// is in its middle.
var DefaultFrame = [3]float64{0.25, 0.25, 0.5}

// imageOrigin gives the corner of the map image of the reserves whose image is not on the usual
// grid of 4096 m (found by the lakes of the game on the image), in metres.
var imageOrigin = map[int][2]float64{9: {-12993, -17189}}

// RTPC properties of the regions (CRegion).
const (
	pClass   = 0x1473b179
	pMatrix  = 0x6ca6d4b9 // 4x4, row vectors: translation in 12..14
	pOutline = 0x1c1d51a9 // x, y, z, w per point, moved by the matrix
	pKey     = 0xa23e3cac // name key, e.g. "subregion_tichenau_name_uppercase"
	pLevel   = 0x2f6488d6
	pLabel   = 0x4e1536d6 // x, z of the name on the map
)

var (
	worldFile = regexp.MustCompile(`^worlds/[^/]+/global/global\.blo$`)
	spawnFile = regexp.MustCompile(`^settings/hp_settings/hp_ai_textures/spawn_maps/(reserve_(\d+)_)?[a-z0-9_]+\.bmp_datac$`)
)

// region is a region in world coordinates.
type region struct {
	zone    Zone
	outline [][2]float64
	label   [2]float64
}

// Maps reads the regions of every reserve from the game files, and places the area played on the
// map image. reserveOf gives the reserve of a region, by the hash of its name key (the regions of
// the codex).
func Maps(a *apex.Archives, reserveOf map[uint32]int) (map[int]Map, error) {
	var regions []region
	// corners of the area played given by the habitat grids, with their votes: by reserve, and
	// for the old files of no reserve number
	votes := map[int]map[[2]float64]int{}
	unnamedVotes := map[[2]float64]int{}
	files := map[int]int{}
	a.EachInContainers(func(name string) bool {
		if worldFile.MatchString(name) {
			return true
		}
		m := spawnFile.FindStringSubmatch(name)
		if m == nil {
			return false
		}
		if m[2] == "" {
			return true
		}
		n, _ := strconv.Atoi(m[2])
		files[n]++
		return files[n] <= 6 // enough species to outvote a small habitat
	}, func(name string, b []byte) {
		if worldFile.MatchString(name) {
			regions = append(regions, worldRegions(b)...)
			return
		}
		v := gridVotes(b)
		if m := spawnFile.FindStringSubmatch(name); m[2] != "" {
			n, _ := strconv.Atoi(m[2])
			if votes[n] == nil {
				votes[n] = map[[2]float64]int{}
			}
			for o, k := range v {
				votes[n][o] += k
			}
		} else {
			for o, k := range v {
				unnamedVotes[o] += k
			}
		}
	})
	if len(regions) == 0 {
		return nil, errors.New("no region")
	}
	origins := map[int][2]float64{} // reserve -> corner of its area played
	for n, v := range votes {
		if o, ok := mostVoted(v, 1); ok {
			origins[n] = o
		}
	}
	// grids of the old files: the reserve whose regions are inside
	var unnamed [][2]float64
	for o, k := range unnamedVotes {
		if k >= 500 {
			unnamed = append(unnamed, o)
		}
	}
	sort.Slice(unnamed, func(i, j int) bool {
		a, b := unnamed[i], unnamed[j]
		if unnamedVotes[a] != unnamedVotes[b] {
			return unnamedVotes[a] > unnamedVotes[b]
		}
		return a[0] < b[0] || a[0] == b[0] && a[1] < b[1]
	})
	for _, o := range unnamed {
		count := map[int]int{}
		for _, r := range regions {
			n, ok := reserveOf[r.zone.Hash]
			if _, known := origins[n]; ok && !known && r.label[0] >= o[0] && r.label[0] < o[0]+Side && r.label[1] >= o[1] && r.label[1] < o[1]+Side {
				count[n]++
			}
		}
		best, most := -1, 0
		for n, c := range count {
			if c > most || c == most && n < best {
				best, most = n, c
			}
		}
		if best >= 0 {
			origins[best] = o
		}
	}
	out := map[int]Map{}
	xs, zs := map[int][]float64{}, map[int][]float64{}
	for _, r := range regions {
		n, ok := reserveOf[r.zone.Hash]
		o, ok2 := origins[n]
		if !ok || !ok2 {
			continue
		}
		z := r.zone
		at := func(p [2]float64) [2]float32 {
			return [2]float32{round4((p[0] - o[0]) / Side), round4((p[1] - o[1]) / Side)}
		}
		for _, p := range r.outline {
			z.Outline = append(z.Outline, at(p))
			xs[n], zs[n] = append(xs[n], p[0]), append(zs[n], p[1])
		}
		z.Label = at(r.label)
		m := out[n]
		m.Zones = append(m.Zones, z)
		out[n] = m
	}
	for n, m := range out {
		m.Frame = frame(origins[n], xs[n], zs[n], imageOrigin[n])
		out[n] = m
	}
	return out, nil
}

// frame places the area played (corner o) on the map image. The image is twice the area played,
// on the grid of 4096 m of the world, around the middle of the regions (some reserves, as
// Salzwiesen, use only a part of the area played); or given.
func frame(o [2]float64, xs, zs []float64, given [2]float64) [3]float64 {
	if given == ([2]float64{}) {
		given = [2]float64{imageCorner(xs), imageCorner(zs)}
	}
	return [3]float64{round4f((o[0] - given[0]) / (2 * Side)), round4f((o[1] - given[1]) / (2 * Side)), 0.5}
}

// imageCorner is the corner of the image on one axis: from the middle of the regions, the few
// points far out left out.
func imageCorner(v []float64) float64 {
	sort.Float64s(v)
	mid := (v[len(v)/50] + v[len(v)-1-len(v)/50]) / 2
	return math.Round((mid-Side)/4096) * 4096
}

func round4f(v float64) float64 { return math.Round(v*10000) / 10000 }

func round4(v float64) float32 { return float32(math.Round(v*10000) / 10000) }

// worldRegions reads the regions of global.blo of a world.
func worldRegions(b []byte) (out []region) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	root, err := apex.ParseRTPC(b)
	if err != nil {
		return nil
	}
	var walk func(n *apex.Node)
	walk = func(n *apex.Node) {
		if c, _ := n.Props[pClass].(string); c == "CRegion" {
			key, _ := n.Props[pKey].(string)
			m, _ := n.Props[pMatrix].([]float32)
			pts, _ := n.Props[pOutline].([]float32)
			lv, _ := n.Props[pLevel].(uint32)
			if key != "" && len(m) == 16 && len(pts) >= 12 {
				r := region{zone: Zone{Hash: apex.HashString(key), Level: int(lv)}}
				f := func(i int) float64 { return float64(m[i]) }
				var sx, sz float64
				for i := 0; i+3 < len(pts); i += 4 {
					x, y, z := float64(pts[i]), float64(pts[i+1]), float64(pts[i+2])
					p := [2]float64{x*f(0) + y*f(4) + z*f(8) + f(12), x*f(2) + y*f(6) + z*f(10) + f(14)}
					r.outline = append(r.outline, p)
					sx, sz = sx+p[0], sz+p[1]
				}
				// the name: its place on the map, else the middle of the outline
				r.label = [2]float64{sx / float64(len(r.outline)), sz / float64(len(r.outline))}
				if l, ok := n.Props[pLabel].([]float32); ok && len(l) == 2 && (l[0] != 0 || l[1] != 0) {
					r.label = [2]float64{float64(l[0]), float64(l[1])}
				}
				out = append(out, r)
			}
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}

// gridVotes reads a habitat grid of the game (BitmapData: areas of cells of 32 m, with their
// cells and their bounds in metres): each area votes for the corner of the grid it implies.
func gridVotes(b []byte) (votes map[[2]float64]int) {
	votes = map[[2]float64]int{}
	defer func() {
		if recover() != nil {
			votes = map[[2]float64]int{}
		}
	}()
	start := bytes.Index(b, []byte(" FDA"))
	if start < 0 {
		return votes
	}
	root, err := apex.ReadADF(b[start:])
	if err != nil {
		return votes
	}
	w := int(root.U32("Width"))
	layers, ok := root.Field("Layers")
	if !ok || w <= 0 {
		return votes
	}
	for _, layer := range layers.Items() {
		areas, _ := layer.Field("Areas")
		for _, area := range areas.Items() {
			tiles, _ := area.Field("Tiles")
			lo, _ := area.Field("AabbMin")
			hi, _ := area.Field("AabbMax")
			minC, maxC, minR, maxR := math.MaxInt, -1, math.MaxInt, -1
			for _, t := range tiles.Items() {
				c, r := int(t.Uint())%w, int(t.Uint())/w
				minC, maxC, minR, maxR = min(minC, c), max(maxC, c), min(minR, r), max(maxR, r)
			}
			if maxC < 0 {
				continue
			}
			l, h := lo.Floats(3), hi.Floats(3)
			cx := float64(h[0]-l[0]) / float64(maxC-minC+1)
			cz := float64(h[2]-l[2]) / float64(maxR-minR+1)
			if math.Abs(cx-Side/float64(w)) > 0.5 || math.Abs(cz-Side/float64(w)) > 0.5 {
				continue // an area whose bounds are not those of its cells
			}
			votes[[2]float64{math.Round(float64(l[0]) - float64(minC)*cx), math.Round(float64(l[2]) - float64(minR)*cz)}]++
		}
	}
	return votes
}

// mostVoted returns the corner with the most votes, if it has at least min.
func mostVoted(votes map[[2]float64]int, min int) ([2]float64, bool) {
	var best [2]float64
	most := 0
	for o, n := range votes {
		if n > most || n == most && (o[0] < best[0] || o[0] == best[0] && o[1] < best[1]) {
			best, most = o, n
		}
	}
	return best, most >= min
}

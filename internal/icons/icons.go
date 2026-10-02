// Package icons turns the species icons of the game (vector shapes of clue_hud.gfx) into SVG, for
// the settings page. It runs on the player's computer, from the player's game files.
package icons

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Laink/COTWGoldChallenge/internal/swf"
)

// Extract returns the SVG of each frame of the icons clip: the icon of species icon n is frame n+1,
// so the key is n.
// A movie it cannot read gives an error, never a panic: the icons are optional.
func Extract(clueHud []byte) (out map[int]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			out, err = nil, fmt.Errorf("icons: %v", r)
		}
	}()
	m, err := swf.Load(clueHud)
	if err != nil {
		return nil, err
	}
	defs := map[int]*swf.Tag{}
	var sprite *swf.Tag
	for _, t := range m.Tags {
		switch t.Code {
		case 2, 22, 32, 83:
			defs[int(binary.LittleEndian.Uint16(t.Data))] = t
		case 39:
			if len(t.Data) >= 4 && (sprite == nil || binary.LittleEndian.Uint16(t.Data[2:]) > binary.LittleEndian.Uint16(sprite.Data[2:])) {
				sprite = t
			}
		}
	}
	if sprite == nil || binary.LittleEndian.Uint16(sprite.Data[2:]) < 100 {
		return nil, errors.New("icons not found")
	}
	inner, err := swf.LoadTags(sprite.Data[4:])
	if err != nil {
		return nil, err
	}
	shapes := map[int]*shape{}
	getShape := func(id int) *shape {
		if s, ok := shapes[id]; ok {
			return s
		}
		var s *shape
		if t, ok := defs[id]; ok {
			s, _ = parseShape(t)
		}
		shapes[id] = s
		return s
	}
	type object struct {
		char int
		m    matrix
	}
	list := map[int]*object{}
	out = map[int]string{}
	frame := 0
	for _, t := range inner {
		switch t.Code {
		case 26: // PlaceObject2
			if len(t.Data) < 3 {
				continue
			}
			r := &bits{b: t.Data}
			flags := r.u8()
			depth := int(r.u16())
			o := list[depth]
			if o == nil || flags&1 == 0 {
				o = &object{m: identity}
				list[depth] = o
			}
			if flags&2 != 0 {
				o.char = int(r.u16())
			}
			if flags&4 != 0 {
				o.m = r.matrix()
			}
		case 28: // RemoveObject2
			if len(t.Data) >= 2 {
				delete(list, int(binary.LittleEndian.Uint16(t.Data)))
			}
		case 5: // RemoveObject
			if len(t.Data) >= 4 {
				delete(list, int(binary.LittleEndian.Uint16(t.Data[2:])))
			}
		case 1: // ShowFrame
			var depths []int
			for d := range list {
				depths = append(depths, d)
			}
			sort.Ints(depths)
			var parts []placed
			for _, d := range depths {
				if s := getShape(list[d].char); s != nil {
					parts = append(parts, placed{s, list[d].m})
				}
			}
			if svg := render(parts); svg != "" {
				out[frame] = svg
			}
			frame++
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no icon")
	}
	return out, nil
}

type matrix struct{ a, b, c, d, tx, ty float64 }

var identity = matrix{a: 1, d: 1}

func (m matrix) apply(x, y float64) (float64, float64) {
	return m.a*x + m.c*y + m.tx, m.b*x + m.d*y + m.ty
}

type bits struct {
	b   []byte
	pos int // in bits
}

func (r *bits) align() { r.pos = (r.pos + 7) &^ 7 }

func (r *bits) ub(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v <<= 1
		if p := r.pos >> 3; p < len(r.b) && r.b[p]&(0x80>>(r.pos&7)) != 0 {
			v |= 1
		}
		r.pos++
	}
	return v
}

func (r *bits) sb(n int) int32 {
	v := r.ub(n)
	if n > 0 && v&(1<<(n-1)) != 0 {
		v |= ^uint32(0) << n
	}
	return int32(v)
}

func (r *bits) u8() uint32 {
	r.align()
	return r.ub(8)
}

func (r *bits) u16() uint32 {
	lo := r.u8()
	return lo | r.u8()<<8
}

func (r *bits) eof() bool { return r.pos>>3 >= len(r.b) }

func (r *bits) matrix() matrix {
	r.align()
	m := identity
	if r.ub(1) == 1 {
		n := int(r.ub(5))
		m.a, m.d = float64(r.sb(n))/65536, float64(r.sb(n))/65536
	}
	if r.ub(1) == 1 {
		n := int(r.ub(5))
		m.b, m.c = float64(r.sb(n))/65536, float64(r.sb(n))/65536
	}
	n := int(r.ub(5))
	m.tx, m.ty = float64(r.sb(n)), float64(r.sb(n))
	r.align()
	return m
}

func (r *bits) rect() {
	r.align()
	n := int(r.ub(5))
	r.ub(4 * n)
	r.align()
}

// color reads an RGB or RGBA colour as an SVG colour and opacity.
func (r *bits) color(alpha bool) (string, float64) {
	c := fmt.Sprintf("#%02x%02x%02x", r.u8(), r.u8(), r.u8())
	a := 1.0
	if alpha {
		a = float64(r.u8()) / 255
	}
	return c, a
}

type style struct {
	color   string // "" when not drawn
	opacity float64
	width   float64 // lines
}

type edge struct {
	x0, y0, cx, cy, x1, y1 int32
	curve                  bool
}

func (e edge) reverse() edge {
	return edge{e.x1, e.y1, e.cx, e.cy, e.x0, e.y0, e.curve}
}

// path is a set of edges drawn with one style.
type path struct {
	st    style
	line  bool
	edges []edge
}

type shape struct{ paths []path }

func parseShape(t *swf.Tag) (*shape, error) {
	v := map[int]int{2: 1, 22: 2, 32: 3, 83: 4}[t.Code]
	r := &bits{b: t.Data, pos: 16}
	r.rect()
	if v == 4 {
		r.rect()
		r.u8()
	}
	fills, lines := styles(r, v)
	r.align()
	nf, nl := int(r.ub(4)), int(r.ub(4))
	s := &shape{}
	fillEdges, lineEdges := map[int][]edge{}, map[int][]edge{}
	flush := func() {
		for i := 1; i <= len(fills); i++ {
			if len(fillEdges[i]) > 0 && fills[i-1].color != "" {
				s.paths = append(s.paths, path{st: fills[i-1], edges: fillEdges[i]})
			}
		}
		for i := 1; i <= len(lines); i++ {
			if len(lineEdges[i]) > 0 && lines[i-1].color != "" {
				s.paths = append(s.paths, path{st: lines[i-1], line: true, edges: lineEdges[i]})
			}
		}
		fillEdges, lineEdges = map[int][]edge{}, map[int][]edge{}
	}
	var x, y int32
	f0, f1, ln := 0, 0, 0
	add := func(e edge) {
		if f1 > 0 {
			fillEdges[f1] = append(fillEdges[f1], e)
		}
		if f0 > 0 {
			fillEdges[f0] = append(fillEdges[f0], e.reverse())
		}
		if ln > 0 {
			lineEdges[ln] = append(lineEdges[ln], e)
		}
		x, y = e.x1, e.y1
	}
	for !r.eof() {
		if r.ub(1) == 0 {
			flags := r.ub(5)
			if flags == 0 {
				break
			}
			if flags&1 != 0 {
				n := int(r.ub(5))
				x, y = r.sb(n), r.sb(n)
			}
			if flags&2 != 0 {
				f0 = int(r.ub(nf))
			}
			if flags&4 != 0 {
				f1 = int(r.ub(nf))
			}
			if flags&8 != 0 {
				ln = int(r.ub(nl))
			}
			if flags&16 != 0 {
				flush()
				fills, lines = styles(r, v)
				r.align()
				nf, nl = int(r.ub(4)), int(r.ub(4))
			}
			continue
		}
		straight := r.ub(1) == 1
		n := int(r.ub(4)) + 2
		if straight {
			var dx, dy int32
			if r.ub(1) == 1 {
				dx, dy = r.sb(n), r.sb(n)
			} else if r.ub(1) == 1 {
				dy = r.sb(n)
			} else {
				dx = r.sb(n)
			}
			add(edge{x0: x, y0: y, x1: x + dx, y1: y + dy})
		} else {
			cx, cy := x+r.sb(n), y+r.sb(n)
			ax, ay := cx+r.sb(n), cy+r.sb(n)
			add(edge{x, y, cx, cy, ax, ay, true})
		}
	}
	flush()
	return s, nil
}

// styles reads the fill and line style arrays. Gradients are drawn with their first colour;
// bitmaps are not drawn.
func styles(r *bits, v int) ([]style, []style) {
	count := func() int {
		n := int(r.u8())
		if n == 0xFF && v >= 2 {
			n = int(r.u16())
		}
		return n
	}
	fill := func() style {
		switch k := r.u8(); {
		case k == 0:
			c, a := r.color(v >= 3)
			return style{color: c, opacity: a}
		case k == 0x10 || k == 0x12 || k == 0x13:
			r.matrix()
			n := int(r.u8() & 15)
			var st style
			for i := 0; i < n; i++ {
				r.u8()
				c, a := r.color(v >= 3)
				if i == 0 {
					st = style{color: c, opacity: a}
				}
			}
			if k == 0x13 {
				r.u16()
			}
			return st
		default:
			r.u16()
			r.matrix()
			return style{}
		}
	}
	fills := make([]style, count())
	for i := range fills {
		fills[i] = fill()
	}
	lines := make([]style, count())
	for i := range lines {
		w := float64(r.u16())
		if v == 4 {
			flags := r.u16()
			if flags&0x30 == 0x20 { // miter join
				r.u16()
			}
			if flags&0x08 != 0 {
				st := fill()
				st.width = w
				lines[i] = st
				continue
			}
		}
		c, a := r.color(v >= 3)
		lines[i] = style{color: c, opacity: a, width: w}
	}
	return fills, lines
}

type placed struct {
	s *shape
	m matrix
}

// render draws the shapes of a frame, in twips, cropped to their bounds.
func render(parts []placed) string {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	var body strings.Builder
	num := func(v float64) string { return strings.TrimSuffix(fmt.Sprintf("%.0f", v), ".0") }
	for _, p := range parts {
		for _, pa := range p.s.paths {
			var d strings.Builder
			pt := func(x, y int32) string {
				px, py := p.m.apply(float64(x), float64(y))
				minX, minY, maxX, maxY = math.Min(minX, px), math.Min(minY, py), math.Max(maxX, px), math.Max(maxY, py)
				return num(px) + " " + num(py)
			}
			for _, contour := range chain(pa.edges, !pa.line) {
				for i, e := range contour {
					if i == 0 || (e.x0 != contour[i-1].x1 || e.y0 != contour[i-1].y1) {
						d.WriteString("M" + pt(e.x0, e.y0))
					}
					if e.curve {
						d.WriteString("Q" + pt(e.cx, e.cy) + " " + pt(e.x1, e.y1))
					} else {
						d.WriteString("L" + pt(e.x1, e.y1))
					}
				}
				if !pa.line {
					d.WriteString("Z")
				}
			}
			op := ""
			if pa.st.opacity < 1 {
				op = fmt.Sprintf(` opacity="%.2f"`, pa.st.opacity)
			}
			if pa.line {
				w := math.Max(pa.st.width*math.Hypot(p.m.a, p.m.b), 20)
				fmt.Fprintf(&body, `<path d="%s" fill="none" stroke="%s" stroke-width="%s"%s/>`, d.String(), pa.st.color, num(w), op)
			} else {
				fmt.Fprintf(&body, `<path d="%s" fill="%s" fill-rule="evenodd"%s/>`, d.String(), pa.st.color, op)
			}
		}
	}
	if body.Len() == 0 || maxX <= minX || maxY <= minY {
		return ""
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s">%s</svg>`,
		num(minX), num(minY), num(maxX-minX), num(maxY-minY), body.String())
}

// chain links the edges of a fill into contours, each edge starting where the previous one ends.
func chain(edges []edge, closed bool) [][]edge {
	if !closed {
		return [][]edge{edges}
	}
	type pt struct{ x, y int32 }
	from := map[pt][]int{}
	for i, e := range edges {
		from[pt{e.x0, e.y0}] = append(from[pt{e.x0, e.y0}], i)
	}
	used := make([]bool, len(edges))
	next := func(p pt) int {
		for _, i := range from[p] {
			if !used[i] {
				return i
			}
		}
		return -1
	}
	var out [][]edge
	for i := range edges {
		if used[i] {
			continue
		}
		var c []edge
		for j := i; j >= 0; j = next(pt{edges[j].x1, edges[j].y1}) {
			used[j] = true
			c = append(c, edges[j])
		}
		out = append(out, c)
	}
	return out
}

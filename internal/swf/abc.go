package swf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type reader struct {
	b []byte
	p int
}

func (r *reader) u8() int {
	v := r.b[r.p]
	r.p++
	return int(v)
}

func (r *reader) u30() int {
	v := 0
	for i := 0; i < 5; i++ {
		c := r.u8()
		v |= (c & 0x7f) << (7 * i)
		if c&0x80 == 0 {
			break
		}
	}
	return v
}

func (r *reader) s24() int {
	v := int(r.b[r.p]) | int(r.b[r.p+1])<<8 | int(r.b[r.p+2])<<16
	r.p += 3
	if v&0x800000 != 0 {
		v -= 1 << 24
	}
	return v
}

// Namespace kinds.
const (
	nsPackage         = 0x16
	nsPackageInternal = 0x17
)

// Multiname kinds.
const (
	mnQName     = 0x07
	mnMultiname = 0x09
)

// Multiname is a constant pool name.
type Multiname struct {
	Kind int
	Name string
	NS   int // namespace index (QName) or namespace set index (Multiname)
}

// Body is a method body with its location in the bytecode.
type Body struct {
	Method, MaxStack, Locals, InitScope, MaxScope int
	Code                                          []byte
	Start, End                                    int
	tail                                          []byte // exceptions and traits
}

// ABC is a parsed ActionScript 3 bytecode block.
type ABC struct {
	raw        []byte
	Strings    []string
	NSKinds    []int
	NSNames    []string
	Multinames []Multiname
	Methods    map[string]int // "Class.method", "Class.<init>"
	Bodies     map[int]*Body

	dblPos, dblEnd, nDbl int
	strPos, strEnd, nStr int
	mnPos, mnEnd, nMn    int

	newStrings []string
	newDoubles []float64
	newNames   [][2]int // namespace, string
	changed    map[int]*Body
}

// ParseABC parses bytecode.
func ParseABC(b []byte) (a *ABC, err error) {
	defer func() {
		if e := recover(); e != nil {
			a, err = nil, fmt.Errorf("corrupt bytecode: %v", e)
		}
	}()
	a = &ABC{raw: b, Methods: map[string]int{}, Bodies: map[int]*Body{}, changed: map[int]*Body{}}
	r := &reader{b: b, p: 4}
	for k := 0; k < 2; k++ { // int and uint pools
		n := r.u30()
		for i := 1; i < n; i++ {
			r.u30()
		}
	}
	a.dblPos = r.p
	a.nDbl = r.u30()
	if a.nDbl > 0 {
		r.p += 8 * (a.nDbl - 1)
	}
	a.dblEnd = r.p
	a.strPos = r.p
	a.nStr = r.u30()
	a.Strings = []string{""}
	for i := 1; i < a.nStr; i++ {
		l := r.u30()
		a.Strings = append(a.Strings, string(b[r.p:r.p+l]))
		r.p += l
	}
	a.strEnd = r.p
	n := r.u30()
	a.NSKinds, a.NSNames = []int{0}, []string{""}
	for i := 1; i < n; i++ {
		a.NSKinds = append(a.NSKinds, r.u8())
		a.NSNames = append(a.NSNames, a.Strings[r.u30()])
	}
	n = r.u30()
	for i := 1; i < n; i++ {
		c := r.u30()
		for j := 0; j < c; j++ {
			r.u30()
		}
	}
	a.mnPos = r.p
	a.nMn = r.u30()
	a.Multinames = []Multiname{{}}
	for i := 1; i < a.nMn; i++ {
		k := r.u8()
		m := Multiname{Kind: k}
		switch k {
		case 0x07, 0x0D:
			m.NS = r.u30()
			m.Name = a.Strings[r.u30()]
		case 0x09, 0x0E:
			m.Name = a.Strings[r.u30()]
			m.NS = r.u30()
		case 0x0F, 0x10:
			m.Name = a.Strings[r.u30()]
		case 0x11, 0x12:
		case 0x1B, 0x1C:
			r.u30()
		case 0x1D:
			r.u30()
			c := r.u30()
			for j := 0; j < c; j++ {
				r.u30()
			}
		default:
			return nil, fmt.Errorf("unknown multiname kind %#x", k)
		}
		a.Multinames = append(a.Multinames, m)
	}
	a.mnEnd = r.p
	n = r.u30()
	for i := 0; i < n; i++ { // methods
		pc := r.u30()
		r.u30()
		for j := 0; j < pc; j++ {
			r.u30()
		}
		r.u30()
		fl := r.u8()
		if fl&0x08 != 0 {
			oc := r.u30()
			for j := 0; j < oc; j++ {
				r.u30()
				r.u8()
			}
		}
		if fl&0x80 != 0 {
			for j := 0; j < pc; j++ {
				r.u30()
			}
		}
	}
	n = r.u30()
	for i := 0; i < n; i++ { // metadata
		r.u30()
		c := r.u30()
		for j := 0; j < 2*c; j++ {
			r.u30()
		}
	}
	nc := r.u30()
	classes := make([]string, nc)
	for i := 0; i < nc; i++ { // instances
		classes[i] = a.Multinames[r.u30()].Name
		r.u30()
		fl := r.u8()
		if fl&0x08 != 0 {
			r.u30()
		}
		c := r.u30()
		for j := 0; j < c; j++ {
			r.u30()
		}
		a.Methods[classes[i]+".<init>"] = r.u30()
		a.traits(r, classes[i]+".")
	}
	for i := 0; i < nc; i++ { // classes
		r.u30()
		a.traits(r, classes[i]+".static ")
	}
	n = r.u30()
	for i := 0; i < n; i++ { // scripts
		r.u30()
		a.traits(r, "")
	}
	n = r.u30()
	for i := 0; i < n; i++ {
		bd := &Body{Start: r.p}
		bd.Method, bd.MaxStack, bd.Locals, bd.InitScope, bd.MaxScope = r.u30(), r.u30(), r.u30(), r.u30(), r.u30()
		cl := r.u30()
		bd.Code = b[r.p : r.p+cl]
		r.p += cl
		tail := r.p
		ec := r.u30()
		for j := 0; j < 5*ec; j++ {
			r.u30()
		}
		a.traits(r, "")
		bd.tail = b[tail:r.p]
		bd.End = r.p
		a.Bodies[bd.Method] = bd
	}
	return a, nil
}

func (a *ABC) traits(r *reader, prefix string) {
	n := r.u30()
	for i := 0; i < n; i++ {
		name := a.Multinames[r.u30()].Name
		k := r.u8()
		switch k & 0x0f {
		case 0, 6:
			r.u30()
			r.u30()
			if r.u30() != 0 {
				r.u8()
			}
		case 1, 2, 3:
			r.u30()
			m := r.u30()
			if prefix != "" && k&0x0f == 1 {
				a.Methods[prefix+name] = m
			}
		case 4, 5:
			r.u30()
			r.u30()
		}
		if k>>4&0x04 != 0 {
			c := r.u30()
			for j := 0; j < c; j++ {
				r.u30()
			}
		}
	}
}

// Body returns the body of a named method.
func (a *ABC) Body(name string) (*Body, error) {
	m, ok := a.Methods[name]
	if !ok {
		return nil, fmt.Errorf("method %s not found", name)
	}
	bd, ok := a.Bodies[m]
	if !ok {
		return nil, fmt.Errorf("method %s has no body", name)
	}
	return bd, nil
}

// String adds a string constant and returns its index. New strings are deduplicated.
func (a *ABC) String(s string) int {
	for i, x := range a.newStrings {
		if x == s {
			return max(1, a.nStr) + i
		}
	}
	a.newStrings = append(a.newStrings, s)
	return max(1, a.nStr) + len(a.newStrings) - 1
}

// Double adds a number constant and returns its index.
func (a *ABC) Double(v float64) int {
	for i, x := range a.newDoubles {
		if x == v {
			return max(1, a.nDbl) + i
		}
	}
	a.newDoubles = append(a.newDoubles, v)
	return max(1, a.nDbl) + len(a.newDoubles) - 1
}

// Name returns a multiname usable to reach a public property, method or class.
func (a *ABC) Name(s string) int {
	for i, m := range a.Multinames {
		if i > 0 && m.Name == s && m.Kind == mnQName && (a.NSKinds[m.NS] == nsPackage || a.NSKinds[m.NS] == nsPackageInternal) {
			return i
		}
	}
	for i, m := range a.Multinames {
		if i > 0 && m.Name == s && m.Kind == mnMultiname {
			return i
		}
	}
	public := -1
	for i := range a.NSKinds {
		if a.NSKinds[i] == nsPackage && a.NSNames[i] == "" {
			public = i
			break
		}
	}
	idx := a.String(s)
	for i, x := range a.newNames {
		if x == [2]int{public, idx} {
			return max(1, a.nMn) + i
		}
	}
	a.newNames = append(a.newNames, [2]int{public, idx})
	return max(1, a.nMn) + len(a.newNames) - 1
}

// QName returns a multiname for a name in an existing package namespace, e.g. "flash.net".
func (a *ABC) QName(pkg, s string) (int, error) {
	ns := -1
	for i := range a.NSKinds {
		if a.NSKinds[i] == nsPackage && a.NSNames[i] == pkg {
			ns = i
			break
		}
	}
	if ns < 0 {
		return 0, fmt.Errorf("namespace %s not found", pkg)
	}
	for i, m := range a.Multinames {
		if i > 0 && m.Kind == mnQName && m.NS == ns && m.Name == s {
			return i, nil
		}
	}
	idx := a.String(s)
	for i, x := range a.newNames {
		if x == [2]int{ns, idx} {
			return max(1, a.nMn) + i, nil
		}
	}
	a.newNames = append(a.newNames, [2]int{ns, idx})
	return max(1, a.nMn) + len(a.newNames) - 1, nil
}

// Replace sets new code for a method body.
func (a *ABC) Replace(bd *Body, code []byte, maxStack, locals int) {
	c := *bd
	c.Code, c.MaxStack, c.Locals = code, maxStack, locals
	a.changed[bd.Method] = &c
}

func u30(v int) []byte {
	var out []byte
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			out = append(out, c|0x80)
		} else {
			return append(out, c)
		}
	}
}

// Bytes encodes the bytecode with the added constants and replaced bodies.
func (a *ABC) Bytes() ([]byte, error) {
	b := a.raw
	if a.newNames != nil && a.NSKinds == nil {
		return nil, errors.New("no public namespace")
	}
	var out []byte
	out = append(out, b[:a.dblPos]...)
	out = append(out, u30(max(1, a.nDbl)+len(a.newDoubles))...)
	out = append(out, b[a.dblPos+len(u30(a.nDbl)):a.dblEnd]...)
	for _, v := range a.newDoubles {
		out = binary.LittleEndian.AppendUint64(out, math.Float64bits(v))
	}
	out = append(out, u30(max(1, a.nStr)+len(a.newStrings))...)
	out = append(out, b[a.strPos+len(u30(a.nStr)):a.strEnd]...)
	for _, s := range a.newStrings {
		out = append(out, u30(len(s))...)
		out = append(out, s...)
	}
	out = append(out, b[a.strEnd:a.mnPos]...)
	out = append(out, u30(max(1, a.nMn)+len(a.newNames))...)
	out = append(out, b[a.mnPos+len(u30(a.nMn)):a.mnEnd]...)
	for _, x := range a.newNames {
		out = append(out, mnQName)
		out = append(out, u30(x[0])...)
		out = append(out, u30(x[1])...)
	}
	p := a.mnEnd
	for _, bd := range sortedBodies(a.changed) {
		out = append(out, b[p:bd.Start]...)
		out = append(out, u30(bd.Method)...)
		out = append(out, u30(bd.MaxStack)...)
		out = append(out, u30(bd.Locals)...)
		out = append(out, u30(bd.InitScope)...)
		out = append(out, u30(bd.MaxScope)...)
		out = append(out, u30(len(bd.Code))...)
		out = append(out, bd.Code...)
		out = append(out, bd.tail...)
		p = bd.End
	}
	return append(out, b[p:]...), nil
}

func sortedBodies(m map[int]*Body) []*Body {
	var out []*Body
	for _, bd := range m {
		out = append(out, bd)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Start < out[j-1].Start; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// DoubleAt returns a number constant of the original pool.
func (a *ABC) DoubleAt(i int) float64 {
	r := &reader{b: a.raw, p: a.dblPos}
	r.u30()
	p := r.p + 8*(i-1)
	return math.Float64frombits(binary.LittleEndian.Uint64(a.raw[p:]))
}

// HasExceptions reports whether the body has exception handlers.
func (bd *Body) HasExceptions() bool { return len(bd.tail) > 0 && bd.tail[0] != 0 }

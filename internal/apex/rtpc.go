package apex

import (
	"encoding/binary"
	"errors"
	"math"
)

// Node is a node of an RTPC property container.
type Node struct {
	Hash     uint32
	Props    map[uint32]any
	Children []*Node
}

// Property value types: uint32, float32, string, []float32, []uint32, []byte, uint64, []uint64.

// ParseRTPC decodes an RTPC file.
func ParseRTPC(b []byte) (n *Node, err error) {
	if len(b) < 20 || string(b[:4]) != "RTPC" {
		return nil, errors.New("not an RTPC file")
	}
	defer func() {
		if recover() != nil {
			n, err = nil, errors.New("corrupt RTPC file")
		}
	}()
	return rtpcNode(b, 8, 0), nil
}

func rtpcNode(b []byte, off int, depth int) *Node {
	if depth > 64 {
		panic("too deep")
	}
	le := binary.LittleEndian
	n := &Node{Hash: le.Uint32(b[off:]), Props: map[uint32]any{}}
	data := int(le.Uint32(b[off+4:]))
	np := int(le.Uint16(b[off+8:]))
	nc := int(le.Uint16(b[off+10:]))
	for i := 0; i < np; i++ {
		o := data + 9*i
		n.Props[le.Uint32(b[o:])] = rtpcValue(b, le.Uint32(b[o+4:]), b[o+8])
	}
	co := (data + 9*np + 3) &^ 3
	for i := 0; i < nc; i++ {
		n.Children = append(n.Children, rtpcNode(b, co+12*i, depth+1))
	}
	return n
}

func rtpcValue(b []byte, d uint32, t byte) any {
	le := binary.LittleEndian
	p := int(d)
	floats := func(n, at int) []float32 {
		out := make([]float32, n)
		for i := range out {
			out[i] = math.Float32frombits(le.Uint32(b[at+4*i:]))
		}
		return out
	}
	switch t {
	case 1:
		return d
	case 2:
		return math.Float32frombits(d)
	case 3:
		e := p
		for b[e] != 0 {
			e++
		}
		return string(b[p:e])
	case 4, 5, 6, 7, 8:
		return floats(map[byte]int{4: 2, 5: 3, 6: 4, 7: 9, 8: 16}[t], p)
	case 9:
		n := int(le.Uint32(b[p:]))
		out := make([]uint32, n)
		for i := range out {
			out[i] = le.Uint32(b[p+4+4*i:])
		}
		return out
	case 10:
		return floats(int(le.Uint32(b[p:])), p+4)
	case 11:
		n := int(le.Uint32(b[p:]))
		return b[p+4 : p+4+n]
	case 13:
		return le.Uint64(b[p:])
	case 14:
		n := int(le.Uint32(b[p:]))
		out := make([]uint64, n)
		for i := range out {
			out[i] = le.Uint64(b[p+4+8*i:])
		}
		return out
	}
	return nil
}

// RootString reads a string property of the root node without decoding the whole file.
func RootString(b []byte, prop uint32) (string, bool) {
	if len(b) < 20 || string(b[:4]) != "RTPC" {
		return "", false
	}
	le := binary.LittleEndian
	data := int(le.Uint32(b[12:]))
	np := int(le.Uint16(b[16:]))
	for i := 0; i < np; i++ {
		o := data + 9*i
		if o+9 > len(b) {
			return "", false
		}
		if le.Uint32(b[o:]) != prop || b[o+8] != 3 {
			continue
		}
		p := int(le.Uint32(b[o+4:]))
		for e := p; e < len(b); e++ {
			if b[e] == 0 {
				return string(b[p:e]), true
			}
		}
		return "", false
	}
	return "", false
}

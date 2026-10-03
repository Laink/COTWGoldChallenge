package harvest

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// adfFile reads an ADF file with the type definitions it holds: members are found by name, so
// that a new version of a structure does not break the reading.
type adfFile struct {
	d     []byte
	types map[uint32]*adfType
}

type adfType struct {
	kind    uint32 // 1 structure, 3 array
	size    uint32
	elem    uint32
	members map[string]adfMember
}

type adfMember struct {
	typ    uint32
	offset int
}

// adfValue is a value of an ADF file: its type and position.
type adfValue struct {
	f    *adfFile
	typ  uint32
	pos  int
	inst int // start of the instance, which array offsets are relative to
}

// readADF reads the type definitions of an ADF file (d starts at " FDA") and returns its first
// instance.
func readADF(d []byte) (v adfValue, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("invalid ADF: %v", r)
		}
	}()
	u32 := func(p int) uint32 { return binary.LittleEndian.Uint32(d[p:]) }
	f := &adfFile{d: d, types: map[uint32]*adfType{}}
	instCount, instOff := int(u32(8)), int(u32(12))
	typeCount, typeOff := int(u32(16)), int(u32(20))
	nameCount, nameOff := int(u32(32)), int(u32(36))
	var names []string
	p := nameOff + nameCount
	for i := 0; i < nameCount; i++ {
		l := int(d[nameOff+i])
		names = append(names, string(d[p:p+l]))
		p += l + 1
	}
	name := func(p int) string {
		if i := int(u32(p)); i < len(names) {
			return names[i]
		}
		return ""
	}
	p = typeOff
	for i := 0; i < typeCount; i++ {
		t := &adfType{kind: u32(p), size: u32(p + 4), elem: u32(p + 28), members: map[string]adfMember{}}
		hash, n := u32(p+12), int(u32(p+36))
		p += 40
		switch t.kind {
		case 1: // structure: 32-byte members
			for j := 0; j < n; j++ {
				t.members[name(p)] = adfMember{u32(p + 8), int(u32(p+16) & 0xFFFFFF)}
				p += 32
			}
		case 8: // enumeration: 12-byte members
			p += 12 * n
		}
		f.types[hash] = t
	}
	if instCount == 0 {
		return v, errors.New("no instance")
	}
	inst := int(u32(instOff + 8))
	return adfValue{f, u32(instOff + 4), inst, inst}, nil
}

// field returns a member of a structure.
func (v adfValue) field(name string) (adfValue, bool) {
	t := v.f.types[v.typ]
	if t == nil || t.kind != 1 {
		return v, false
	}
	m, ok := t.members[name]
	if !ok {
		return v, false
	}
	return adfValue{v.f, m.typ, v.pos + m.offset, v.inst}, true
}

// items returns the elements of an array.
func (v adfValue) items() []adfValue {
	t := v.f.types[v.typ]
	if t == nil || t.kind != 3 {
		return nil
	}
	off, n := int(binary.LittleEndian.Uint32(v.f.d[v.pos:])), int(binary.LittleEndian.Uint32(v.f.d[v.pos+8:]))
	size := 4
	if et := v.f.types[t.elem]; et != nil {
		size = int(et.size)
	}
	if size <= 0 || off < 0 || v.inst+off+n*size > len(v.f.d) {
		return nil
	}
	out := make([]adfValue, n)
	for i := range out {
		out[i] = adfValue{v.f, t.elem, v.inst + off + i*size, v.inst}
	}
	return out
}

// Hashes of the 8- and 16-bit primitive types.
const (
	adfUint8  = 0x0ca2821d
	adfInt8   = 0x0ae8e5a9
	adfUint16 = 0x86d152bd
	adfInt16  = 0xd13fcf93
)

// u32 reads an integer member of 8 to 32 bits, or the bits of a float; 0 when missing.
func (v adfValue) u32(name string) uint32 {
	m, ok := v.field(name)
	if !ok {
		return 0
	}
	switch m.typ {
	case adfUint8, adfInt8:
		if m.pos < len(v.f.d) {
			return uint32(v.f.d[m.pos])
		}
	case adfUint16, adfInt16:
		if m.pos+2 <= len(v.f.d) {
			return uint32(binary.LittleEndian.Uint16(v.f.d[m.pos:]))
		}
	default:
		if m.pos+4 <= len(v.f.d) {
			return binary.LittleEndian.Uint32(v.f.d[m.pos:])
		}
	}
	return 0
}

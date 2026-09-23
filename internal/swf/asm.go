package swf

import "fmt"

// Operand formats: m multiname, s string, i int, u uint, d double, n u30, b u8, j branch, L lookupswitch.
var opcodes = map[byte][2]string{
	0x01: {"bkpt", ""}, 0x02: {"nop", ""}, 0x03: {"throw", ""}, 0x04: {"getsuper", "m"}, 0x05: {"setsuper", "m"},
	0x06: {"dxns", "s"}, 0x08: {"kill", "n"}, 0x09: {"label", ""}, 0x0C: {"ifnlt", "j"}, 0x0D: {"ifnle", "j"},
	0x0E: {"ifngt", "j"}, 0x0F: {"ifnge", "j"}, 0x10: {"jump", "j"}, 0x11: {"iftrue", "j"}, 0x12: {"iffalse", "j"},
	0x13: {"ifeq", "j"}, 0x14: {"ifne", "j"}, 0x15: {"iflt", "j"}, 0x16: {"ifle", "j"}, 0x17: {"ifgt", "j"},
	0x18: {"ifge", "j"}, 0x19: {"ifstricteq", "j"}, 0x1A: {"ifstrictne", "j"}, 0x1B: {"lookupswitch", "L"},
	0x1C: {"pushwith", ""}, 0x1D: {"popscope", ""}, 0x1E: {"nextname", ""}, 0x1F: {"hasnext", ""},
	0x20: {"pushnull", ""}, 0x21: {"pushundefined", ""}, 0x23: {"nextvalue", ""}, 0x24: {"pushbyte", "b"},
	0x25: {"pushshort", "n"}, 0x26: {"pushtrue", ""}, 0x27: {"pushfalse", ""}, 0x28: {"pushnan", ""},
	0x29: {"pop", ""}, 0x2A: {"dup", ""}, 0x2B: {"swap", ""}, 0x2C: {"pushstring", "s"}, 0x2D: {"pushint", "i"},
	0x2E: {"pushuint", "u"}, 0x2F: {"pushdouble", "d"}, 0x30: {"pushscope", ""}, 0x31: {"pushnamespace", "n"},
	0x32: {"hasnext2", "nn"}, 0x40: {"newfunction", "n"}, 0x41: {"call", "n"}, 0x42: {"construct", "n"},
	0x43: {"callmethod", "nn"}, 0x44: {"callstatic", "nn"}, 0x45: {"callsuper", "mn"}, 0x46: {"callproperty", "mn"},
	0x47: {"returnvoid", ""}, 0x48: {"returnvalue", ""}, 0x49: {"constructsuper", "n"}, 0x4A: {"constructprop", "mn"},
	0x4C: {"callproplex", "mn"}, 0x4E: {"callsupervoid", "mn"}, 0x4F: {"callpropvoid", "mn"}, 0x53: {"applytype", "n"},
	0x55: {"newobject", "n"}, 0x56: {"newarray", "n"}, 0x57: {"newactivation", ""}, 0x58: {"newclass", "n"},
	0x59: {"getdescendants", "m"}, 0x5A: {"newcatch", "n"}, 0x5D: {"findpropstrict", "m"}, 0x5E: {"findproperty", "m"},
	0x5F: {"finddef", "m"}, 0x60: {"getlex", "m"}, 0x61: {"setproperty", "m"}, 0x62: {"getlocal", "n"},
	0x63: {"setlocal", "n"}, 0x64: {"getglobalscope", ""}, 0x65: {"getscopeobject", "b"}, 0x66: {"getproperty", "m"},
	0x68: {"initproperty", "m"}, 0x6A: {"deleteproperty", "m"}, 0x6C: {"getslot", "n"}, 0x6D: {"setslot", "n"},
	0x6E: {"getglobalslot", "n"}, 0x6F: {"setglobalslot", "n"}, 0x70: {"convert_s", ""}, 0x73: {"convert_i", ""},
	0x74: {"convert_u", ""}, 0x75: {"convert_d", ""}, 0x76: {"convert_b", ""}, 0x77: {"convert_o", ""},
	0x78: {"checkfilter", ""}, 0x80: {"coerce", "m"}, 0x81: {"coerce_b", ""}, 0x82: {"coerce_a", ""},
	0x83: {"coerce_i", ""}, 0x84: {"coerce_d", ""}, 0x85: {"coerce_s", ""}, 0x86: {"astype", "m"},
	0x87: {"astypelate", ""}, 0x88: {"coerce_u", ""}, 0x89: {"coerce_o", ""}, 0x90: {"negate", ""},
	0x91: {"increment", ""}, 0x92: {"inclocal", "n"}, 0x93: {"decrement", ""}, 0x94: {"declocal", "n"},
	0x95: {"typeof", ""}, 0x96: {"not", ""}, 0x97: {"bitnot", ""}, 0xA0: {"add", ""}, 0xA1: {"subtract", ""},
	0xA2: {"multiply", ""}, 0xA3: {"divide", ""}, 0xA4: {"modulo", ""}, 0xA5: {"lshift", ""}, 0xA6: {"rshift", ""},
	0xA7: {"urshift", ""}, 0xA8: {"bitand", ""}, 0xA9: {"bitor", ""}, 0xAA: {"bitxor", ""}, 0xAB: {"equals", ""},
	0xAC: {"strictequals", ""}, 0xAD: {"lessthan", ""}, 0xAE: {"lessequals", ""}, 0xAF: {"greaterthan", ""},
	0xB0: {"greaterequals", ""}, 0xB1: {"instanceof", ""}, 0xB2: {"istype", "m"}, 0xB3: {"istypelate", ""},
	0xB4: {"in", ""}, 0xC0: {"increment_i", ""}, 0xC1: {"decrement_i", ""}, 0xC2: {"inclocal_i", "n"},
	0xC3: {"declocal_i", "n"}, 0xC4: {"negate_i", ""}, 0xC5: {"add_i", ""}, 0xC6: {"subtract_i", ""},
	0xC7: {"multiply_i", ""}, 0xD0: {"getlocal0", ""}, 0xD1: {"getlocal1", ""}, 0xD2: {"getlocal2", ""},
	0xD3: {"getlocal3", ""}, 0xD4: {"setlocal0", ""}, 0xD5: {"setlocal1", ""}, 0xD6: {"setlocal2", ""},
	0xD7: {"setlocal3", ""}, 0xEF: {"debug", "bnbn"}, 0xF0: {"debugline", "n"}, 0xF1: {"debugfile", "s"},
}

var opByName = func() map[string]byte {
	m := map[string]byte{}
	for k, v := range opcodes {
		m[v[0]] = k
	}
	return m
}()

// Instr is a decoded instruction.
type Instr struct {
	Pos  int
	Op   string
	Args []int
}

// Disassemble decodes a method body.
func Disassemble(code []byte) ([]Instr, error) {
	r := &reader{b: code}
	var out []Instr
	for r.p < len(code) {
		pos := r.p
		op, ok := opcodes[code[r.p]]
		if !ok {
			return out, fmt.Errorf("unknown opcode %#x at %d", code[r.p], pos)
		}
		r.p++
		in := Instr{Pos: pos, Op: op[0]}
		for _, f := range op[1] {
			switch f {
			case 'b':
				in.Args = append(in.Args, r.u8())
			case 'j':
				o := r.s24()
				in.Args = append(in.Args, r.p+o)
			case 'L':
				r.s24()
				c := r.u30()
				for i := 0; i <= c; i++ {
					r.s24()
				}
			default:
				in.Args = append(in.Args, r.u30())
			}
		}
		out = append(out, in)
	}
	return out, nil
}

type item struct {
	label string
	op    string
	args  []any // int, or string label for branches
}

// Asm assembles code appended at a given offset of a method.
type Asm struct {
	base  int
	items []item
	n     int
}

// NewAsm starts code located at base in the method body.
func NewAsm(base int) *Asm { return &Asm{base: base} }

// Op appends an instruction. Branch targets are label names.
func (a *Asm) Op(op string, args ...any) *Asm {
	if _, ok := opByName[op]; !ok {
		panic("unknown op " + op)
	}
	a.items = append(a.items, item{op: op, args: args})
	return a
}

// Label marks the current position.
func (a *Asm) Label(name string) *Asm {
	a.items = append(a.items, item{label: name})
	return a
}

// NewLabel returns a unique label name.
func (a *Asm) NewLabel() string {
	a.n++
	return fmt.Sprintf("L%d", a.n)
}

func (a *Asm) size(it item) int {
	if it.label != "" {
		return 0
	}
	if it.op == "lookupswitch" {
		cases := it.args[1].([]string)
		return 1 + 3 + len(u30(len(cases)-1)) + 3*len(cases)
	}
	n := 1
	f := opcodes[opByName[it.op]][1]
	for i, c := range f {
		switch c {
		case 'j':
			n += 3
		case 'b':
			n++
		default:
			n += len(u30(it.args[i].(int)))
		}
	}
	return n
}

func s24(v int) []byte { return []byte{byte(v), byte(v >> 8), byte(v >> 16)} }

// Assemble encodes the instructions.
func (a *Asm) Assemble() ([]byte, error) {
	labels := map[string]int{}
	adr := a.base
	for _, it := range a.items {
		if it.label != "" {
			labels[it.label] = adr
		}
		adr += a.size(it)
	}
	target := func(l any) (int, error) {
		t, ok := labels[l.(string)]
		if !ok {
			return 0, fmt.Errorf("undefined label %v", l)
		}
		return t, nil
	}
	var out []byte
	adr = a.base
	for _, it := range a.items {
		if it.label != "" {
			continue
		}
		start, sz := adr, a.size(it)
		out = append(out, opByName[it.op])
		if it.op == "lookupswitch" {
			def, err := target(it.args[0])
			if err != nil {
				return nil, err
			}
			cases := it.args[1].([]string)
			out = append(out, s24(def-start)...)
			out = append(out, u30(len(cases)-1)...)
			for _, c := range cases {
				t, err := target(c)
				if err != nil {
					return nil, err
				}
				out = append(out, s24(t-start)...)
			}
		} else {
			for i, c := range opcodes[opByName[it.op]][1] {
				switch c {
				case 'j':
					t, err := target(it.args[i])
					if err != nil {
						return nil, err
					}
					out = append(out, s24(t-(start+sz))...)
				case 'b':
					out = append(out, byte(it.args[i].(int)))
				default:
					out = append(out, u30(it.args[i].(int))...)
				}
			}
		}
		adr += sz
	}
	return out, nil
}

// Package apex reads theHunter: Call of the Wild (Apex engine) archives.
package apex

import "encoding/binary"

func rot(x uint32, k uint) uint32 { return x<<k | x>>(32-k) }

// Hash is Bob Jenkins' lookup3 hashlittle, used by the engine for file and field names.
func Hash(data []byte) uint32 {
	n := len(data)
	a := uint32(0xdeadbeef) + uint32(n)
	b, c := a, a
	i := 0
	for n-i > 12 {
		a += binary.LittleEndian.Uint32(data[i:])
		b += binary.LittleEndian.Uint32(data[i+4:])
		c += binary.LittleEndian.Uint32(data[i+8:])
		a -= c
		a ^= rot(c, 4)
		c += b
		b -= a
		b ^= rot(a, 6)
		a += c
		c -= b
		c ^= rot(b, 8)
		b += a
		a -= c
		a ^= rot(c, 16)
		c += b
		b -= a
		b ^= rot(a, 19)
		a += c
		c -= b
		c ^= rot(b, 4)
		b += a
		i += 12
	}
	if n == 0 {
		return c
	}
	var pad [12]byte
	copy(pad[:], data[i:])
	a += binary.LittleEndian.Uint32(pad[0:])
	b += binary.LittleEndian.Uint32(pad[4:])
	c += binary.LittleEndian.Uint32(pad[8:])
	c ^= b
	c -= rot(b, 14)
	a ^= c
	a -= rot(c, 11)
	b ^= a
	b -= rot(a, 25)
	c ^= b
	c -= rot(b, 16)
	a ^= c
	a -= rot(c, 4)
	b ^= a
	b -= rot(a, 14)
	c ^= b
	c -= rot(b, 24)
	return c
}

// HashString hashes a path such as "ui/clue_hud.gfx".
func HashString(s string) uint32 { return Hash([]byte(s)) }

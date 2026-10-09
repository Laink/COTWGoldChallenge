// Package maps reads the map of a reserve from the game files (textures/ui/map_reserve_N/world_map,
// an AVTX texture in BC1 or BC3, and its zoom tiles), for the settings page. It runs on the
// player's computer.
package maps

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
)

// TileLevels is the number of zoom levels of the maps: level L has 2^(L+1) x 2^(L+1) tiles of 512
// pixels, row by row, over the whole map image.
const TileLevels = 3

// JPEG returns the map of a reserve as a JPEG image.
func JPEG(a *apex.Archives, reserve int) ([]byte, error) {
	return jpegOf(a, fmt.Sprintf("textures/ui/map_reserve_%d/world_map.ddsc", reserve))
}

// TileJPEG returns a zoom tile of the map of a reserve as a JPEG image: level 1 to TileLevels,
// index from 0, row by row.
func TileJPEG(a *apex.Archives, reserve, level, index int) ([]byte, error) {
	if level < 1 || level > TileLevels || index < 0 || index >= 1<<(2*level+2) {
		return nil, errors.New("no tile")
	}
	return jpegOf(a, fmt.Sprintf("textures/ui/map_reserve_%d/zoom%d/%d.ddsc", reserve, level, index))
}

func jpegOf(a *apex.Archives, name string) ([]byte, error) {
	img, err := load(a, name)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 88}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// load reads a texture of the game.
func load(a *apex.Archives, name string) (image.Image, error) {
	e := a.Find(name)
	if len(e) == 0 {
		return nil, errors.New("no map")
	}
	b, err := a.Read(e[len(e)-1])
	if err != nil {
		return nil, err
	}
	return Decode(b)
}

// Decode reads the largest image of an AVTX texture in BC1 or BC3.
func Decode(b []byte) (img image.Image, err error) {
	defer func() {
		if r := recover(); r != nil {
			img, err = nil, fmt.Errorf("invalid texture: %v", r)
		}
	}()
	if len(b) < 128 || string(b[:4]) != "AVTX" {
		return nil, errors.New("not a texture")
	}
	format := b[8]
	w, h := int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:]))
	p := int(binary.LittleEndian.Uint32(b[32:])) // data of the largest image first
	var block int
	switch format {
	case 71, 72: // BC1
		block = 8
	case 77, 78: // BC3
		block = 16
	default:
		return nil, fmt.Errorf("texture format %d", format)
	}
	if w%4 != 0 || h%4 != 0 || p+w*h/16*block > len(b) {
		return nil, errors.New("truncated texture")
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for by := 0; by < h/4; by++ {
		for bx := 0; bx < w/4; bx++ {
			c := b[p : p+block]
			p += block
			if block == 16 {
				c = c[8:] // the alpha block is left out: the maps are opaque
			}
			colorBlock(out, bx*4, by*4, c, block == 16)
		}
	}
	return out, nil
}

// colorBlock draws a colour block of 4x4 pixels. In BC1 a first colour not above the second means
// 3 colours (and black); BC3 blocks always have 4 colours.
func colorBlock(img *image.RGBA, x, y int, c []byte, four bool) {
	c0, c1 := binary.LittleEndian.Uint16(c), binary.LittleEndian.Uint16(c[2:])
	rgb := func(v uint16) [3]int {
		return [3]int{int(v>>11&31) * 255 / 31, int(v>>5&63) * 255 / 63, int(v&31) * 255 / 31}
	}
	var cs [4][3]int
	cs[0], cs[1] = rgb(c0), rgb(c1)
	for i := 0; i < 3; i++ {
		if c0 > c1 || four {
			cs[2][i] = (2*cs[0][i] + cs[1][i]) / 3
			cs[3][i] = (cs[0][i] + 2*cs[1][i]) / 3
		} else {
			cs[2][i] = (cs[0][i] + cs[1][i]) / 2
		}
	}
	bits := binary.LittleEndian.Uint32(c[4:])
	for i := 0; i < 16; i++ {
		v := cs[bits>>(2*i)&3]
		img.SetRGBA(x+i%4, y+i/4, color.RGBA{uint8(v[0]), uint8(v[1]), uint8(v[2]), 255})
	}
}

package maps

import (
	"bytes"
	"encoding/binary"
	"image/jpeg"
	"path/filepath"
	"testing"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
	"github.com/Laink/COTWGoldChallenge/internal/game"
)

// TestJPEG reads the maps of the game installed on this computer, when there is one.
func TestJPEG(t *testing.T) {
	a, err := apex.Open(filepath.Join(`C:\Program Files (x86)\Steam\steamapps\common\theHunterCotW`, "archives_win64"))
	if err != nil {
		t.Skip("no game")
	}
	defer a.Close()
	for _, r := range []int{0, 17, 21} {
		b, err := JPEG(a, r)
		if err != nil {
			t.Fatalf("reserve %d: %v", r, err)
		}
		img, err := jpeg.Decode(bytes.NewReader(b))
		if err != nil || img.Bounds().Dx() < 512 {
			t.Fatalf("reserve %d: %v", r, err)
		}
	}
	if _, err := JPEG(a, 999); err == nil {
		t.Error("map of an unknown reserve")
	}
	// the last tile of each level, and none after
	for level := 1; level <= TileLevels; level++ {
		last := 1<<(2*level+2) - 1
		b, err := TileJPEG(a, 17, level, last)
		if err != nil {
			t.Fatalf("level %d: %v", level, err)
		}
		if img, err := jpeg.Decode(bytes.NewReader(b)); err != nil || img.Bounds().Dx() != 512 {
			t.Fatalf("level %d: tile %v", level, err)
		}
		if _, err := TileJPEG(a, 17, level, last+1); err == nil {
			t.Errorf("level %d: tile after the last one", level)
		}
	}
}

func TestDecodeErrors(t *testing.T) {
	if _, err := Decode([]byte("AVTX")); err == nil {
		t.Error("short texture")
	}
	b := make([]byte, 128)
	copy(b, "AVTX")
	b[8], b[12], b[14] = 77, 4, 4
	b[32] = 128
	if _, err := Decode(b); err == nil {
		t.Error("truncated texture")
	}
}

// TestFrame places an area played on the grid of 4096 m, in the middle of its image, and a small
// one whose regions use only a quarter of it.
func TestFrame(t *testing.T) {
	f := frame([2]float64{-12288, 0}, []float64{-12288, -4096}, []float64{0, 8192}, [2]float64{})
	if f != DefaultFrame {
		t.Errorf("frame %v", f)
	}
	f = frame([2]float64{6144, -10240}, []float64{6144, 10240}, []float64{-10240, -6144}, [2]float64{})
	if f != [3]float64{0.375, 0.375, 0.5} {
		t.Errorf("small reserve: frame %v", f)
	}
}

// TestZones reads the regions of the game installed on this computer, when there is one: every
// reserve with a map has regions, inside its area played; the maps checked by the lakes of the game
// have the usual frame.
func TestZones(t *testing.T) {
	a, err := apex.Open(filepath.Join(`C:\Program Files (x86)\Steam\steamapps\common\theHunterCotW`, "archives_win64"))
	if err != nil {
		t.Skip("no game")
	}
	defer a.Close()
	cx, err := game.LoadCodex(a, "en")
	if err != nil {
		t.Fatal(err)
	}
	all, err := Maps(a, cx.Regions)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []int{16, 17, 21} {
		if all[r].Frame != DefaultFrame {
			t.Errorf("reserve %d: frame %v", r, all[r].Frame)
		}
	}
	for _, r := range []int{0, 1, 6, 17, 21} {
		zones := all[r].Zones
		if len(zones) < 4 {
			t.Errorf("reserve %d: %d zones", r, len(zones))
		}
		for _, z := range zones {
			if z.Label[0] < -0.1 || z.Label[0] > 1.1 || z.Label[1] < -0.1 || z.Label[1] > 1.1 {
				t.Errorf("reserve %d: zone %d out of the map at %v", r, z.Hash, z.Label)
			}
		}
	}
}

// TestDecodeBC3Colours: the colour blocks of BC3 have 4 colours even when the first one is not
// above the second (unlike BC1).
func TestDecodeBC3Colours(t *testing.T) {
	b := make([]byte, 128+16)
	copy(b, "AVTX")
	b[8], b[12], b[14], b[32] = 77, 4, 4, 128
	blk := b[128+8:]
	binary.LittleEndian.PutUint16(blk, 0)          // black
	binary.LittleEndian.PutUint16(blk[2:], 0xffff) // white
	binary.LittleEndian.PutUint32(blk[4:], 2)      // first pixel: colour 2
	img, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, _, _ := img.At(0, 0).RGBA(); r>>8 != 85 {
		t.Errorf("colour 2 = %d, want 85 (a third of the way)", r>>8)
	}
}

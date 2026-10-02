package icons

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Laink/COTWGoldChallenge/internal/apex"
	"github.com/Laink/COTWGoldChallenge/internal/patch"
)

// TestExtract converts the icons of the game installed on this computer, when there is one.
func TestExtract(t *testing.T) {
	arc, err := apex.Open(filepath.Join(`C:\Program Files (x86)\Steam\steamapps\common\theHunterCotW`, "archives_win64"))
	if err != nil {
		t.Skip("no game")
	}
	defer arc.Close()
	e := arc.Find(patch.GamePath)
	if len(e) == 0 {
		t.Skip("no clue_hud.gfx")
	}
	b, err := arc.Read(e[len(e)-1])
	if err != nil {
		t.Fatal(err)
	}
	svgs, err := Extract(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(svgs) < 100 || !strings.HasPrefix(svgs[0], "<svg") || strings.Count(svgs[0], "<path") < 2 {
		t.Fatalf("%d icons, first: %.200s", len(svgs), svgs[0])
	}
}

func TestBits(t *testing.T) {
	r := &bits{b: []byte{0b11100000, 0xFF}}
	if r.ub(1) != 1 || r.sb(3) != -2 || r.u8() != 0xFF || !r.eof() {
		t.Fatal("bit reader")
	}
	if _, err := Extract([]byte("not a movie")); err == nil {
		t.Fatal("no error")
	}
}

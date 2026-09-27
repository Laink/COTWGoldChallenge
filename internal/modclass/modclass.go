// Package modclass holds the ActionScript class of the mod (as3/, compiled by build_as3.sh into
// mod.swc), as DoABC tags to add to the game movies.
package modclass

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"errors"
	"io"

	"github.com/Laink/COTWGoldChallenge/internal/swf"
)

//go:embed mod.swc
var swc []byte

// Tags returns the DoABC tag bodies of the class, not lazy so that the classes are defined when
// the tags run.
func Tags() ([][]byte, error) {
	z, err := zip.NewReader(bytes.NewReader(swc), int64(len(swc)))
	if err != nil {
		return nil, err
	}
	for _, f := range z.File {
		if f.Name != "library.swf" {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			return nil, err
		}
		m, err := swf.Load(b)
		if err != nil {
			return nil, err
		}
		var out [][]byte
		for _, t := range m.ABCTags() {
			if t.Code == 72 {
				out = append(out, append([]byte{0, 0, 0, 0, 0}, t.Data...))
				continue
			}
			d := append([]byte{}, t.Data...)
			d[0] = 0
			out = append(out, d)
		}
		if len(out) == 0 {
			return nil, errors.New("mod class: no ActionScript")
		}
		return out, nil
	}
	return nil, errors.New("mod class: no library.swf")
}

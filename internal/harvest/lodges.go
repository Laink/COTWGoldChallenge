package harvest

import (
	"github.com/Laink/COTWGoldChallenge/internal/apex"

	"bytes"
	"errors"
	"fmt"
	"math"
)

// LodgesFile is the save of the trophy lodges: the trophies on display, the harvests kept to be
// mounted later and the last harvest. Unlike the hunting log, the game never removes them.
const LodgesFile = "trophy_lodges_adf"

// ParseLodges reads the harvests of trophy_lodges_adf. The species and the furs come from the data
// files: harvests of an unknown species are left out.
func ParseLodges(raw []byte, d *Data) (out []Entry, err error) {
	defer func() {
		if r := recover(); r != nil {
			out, err = nil, fmt.Errorf("invalid trophy lodges: %v", r)
		}
	}()
	if len(d.Engine) == 0 {
		// data files written before version 2.5: read again once the mod is installed again
		return nil, errors.New("no engine names in the data files")
	}
	b, _, err := adf(raw)
	if err != nil {
		return nil, err
	}
	start := bytes.Index(b, []byte(" FDA"))
	if start < 0 {
		return nil, errors.New("ADF not found")
	}
	root, err := apex.ReadADF(b[start:])
	if err != nil {
		return nil, err
	}
	animals, ok := root.Field("TrophyAnimals")
	if !ok {
		return nil, errors.New("no trophy animals")
	}
	var list []apex.ADFValue
	if t, ok := animals.Field("Trophies"); ok {
		for _, e := range t.Items() {
			if a, ok := e.Field("TrophyAnimal"); ok {
				list = append(list, a)
			}
		}
	}
	if s, ok := animals.Field("SavedHarvests"); ok {
		list = append(list, s.Items()...)
	}
	if l, ok := animals.Field("LastHarvest"); ok {
		list = append(list, l)
	}
	for _, a := range list {
		sp, ok := d.Engine[a.U32("Type")]
		t := a.U32("HarvestedAt")
		if !ok || t == 0 {
			continue
		}
		e := Entry{
			Species: sp,
			Score:   math.Float32frombits(a.U32("TrophyScore")),
			Rank:    a.U32("ScoreRank"),
			Time:    t,
			Reserve: -1,
			Fur:     d.FurIndex[[2]uint32{sp, a.U32("VariationIndex")}],
			Lodge:   true,
		}
		if r := a.U32("HarvestReserve"); r < 1000 {
			e.Reserve = int(r)
		}
		if p, ok := a.Field("HarvestMapPosition"); ok {
			e.X, e.Y = math.Float32frombits(p.U32("X")), math.Float32frombits(p.U32("Y"))
		}
		out = append(out, e)
	}
	return out, nil
}

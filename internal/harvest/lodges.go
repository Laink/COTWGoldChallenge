package harvest

import (
	"bytes"
	"errors"
	"math"
)

// LodgesFile is the save of the trophy lodges: the trophies on display, the harvests kept to be
// mounted later and the last harvest. Unlike the hunting log, the game never removes them.
const LodgesFile = "trophy_lodges_adf"

// ParseLodges reads the harvests of trophy_lodges_adf. The species and the furs come from the data
// files: harvests of an unknown species are left out.
func ParseLodges(raw []byte, d *Data) ([]Entry, error) {
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
	root, err := readADF(b[start:])
	if err != nil {
		return nil, err
	}
	animals, ok := root.field("TrophyAnimals")
	if !ok {
		return nil, errors.New("no trophy animals")
	}
	var list []adfValue
	if t, ok := animals.field("Trophies"); ok {
		for _, e := range t.items() {
			if a, ok := e.field("TrophyAnimal"); ok {
				list = append(list, a)
			}
		}
	}
	if s, ok := animals.field("SavedHarvests"); ok {
		list = append(list, s.items()...)
	}
	if l, ok := animals.field("LastHarvest"); ok {
		list = append(list, l)
	}
	var out []Entry
	for _, a := range list {
		sp, ok := d.Engine[a.u32("Type")]
		t := a.u32("HarvestedAt")
		if !ok || t == 0 {
			continue
		}
		e := Entry{
			Species: sp,
			Score:   math.Float32frombits(a.u32("TrophyScore")),
			Rank:    a.u32("ScoreRank"),
			Time:    t,
			Reserve: -1,
			Fur:     d.FurIndex[[2]uint32{sp, a.u32("VariationIndex")}],
			Lodge:   true,
		}
		if r := a.u32("HarvestReserve"); r < 1000 {
			e.Reserve = int(r)
		}
		out = append(out, e)
	}
	return out, nil
}

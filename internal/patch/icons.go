package patch

import (
	"encoding/binary"
	"errors"

	"github.com/Laink/COTWGoldChallenge/internal/swf"
)

// IconClass is the class of the species icons movie clip added to the HUD.
const IconClass = "COTWGoldChallengeIcons"

// Tags that define a character, with the character id as their first field.
var defineTags = map[int]bool{2: true, 6: true, 10: true, 11: true, 14: true, 20: true, 21: true, 22: true, 32: true,
	33: true, 34: true, 35: true, 36: true, 37: true, 39: true, 46: true, 48: true, 60: true, 75: true, 83: true,
	84: true, 87: true, 90: true, 91: true}

func charID(t *swf.Tag) int { return int(binary.LittleEndian.Uint16(t.Data)) }

// spriteTags decodes the tags of a DefineSprite.
func spriteTags(t *swf.Tag) ([]*swf.Tag, error) {
	m, err := swf.LoadTags(t.Data[4:])
	if err != nil {
		return nil, err
	}
	return m, nil
}

// placedChar returns the offset of the character id in a PlaceObject/PlaceObject2/PlaceObject3 tag, or -1.
func placedChar(t *swf.Tag) int {
	switch t.Code {
	case 4:
		return 0
	case 26:
		if t.Data[0]&0x02 != 0 {
			return 3
		}
	case 70:
		if t.Data[0]&0x02 != 0 {
			return 4
		}
	}
	return -1
}

// iconSprite finds the species icons clip of the binoculars panel: the sprite with the most frames,
// made of shapes only.
func iconSprite(clue *swf.Movie) (*swf.Tag, map[int]*swf.Tag, error) {
	defs := map[int]*swf.Tag{}
	var best *swf.Tag
	for _, t := range clue.Tags {
		if defineTags[t.Code] && len(t.Data) >= 2 {
			defs[charID(t)] = t
		}
		if t.Code == 39 && len(t.Data) >= 4 {
			if best == nil || binary.LittleEndian.Uint16(t.Data[2:]) > binary.LittleEndian.Uint16(best.Data[2:]) {
				best = t
			}
		}
	}
	if best == nil || binary.LittleEndian.Uint16(best.Data[2:]) < 100 {
		return nil, nil, errors.New("icons not found")
	}
	return best, defs, nil
}

// copyIcons copies the species icons of clue_hud.gfx into another movie, before the tag at index `at`,
// and links them to IconClass. It returns the number of tags inserted.
func copyIcons(clue, dst *swf.Movie, at int) (int, error) {
	sprite, defs, err := iconSprite(clue)
	if err != nil {
		return 0, err
	}
	return copySprite(sprite, defs, dst, at, IconClass)
}

// linkIcons links the species icons of a clue_hud.gfx movie to IconClass: a second sprite with the
// same content, right after the original so that its shapes are already defined.
func linkIcons(movie *swf.Movie) error {
	sprite, _, err := iconSprite(movie)
	if err != nil {
		return err
	}
	next := 0
	at := -1
	for i, t := range movie.Tags {
		if defineTags[t.Code] && len(t.Data) >= 2 && charID(t) >= next {
			next = charID(t) + 1
		}
		if t == sprite {
			at = i + 1
		}
	}
	sd := append([]byte{}, sprite.Data...)
	binary.LittleEndian.PutUint16(sd, uint16(next))
	sc := []byte{1, 0, 0, 0}
	binary.LittleEndian.PutUint16(sc[2:], uint16(next))
	sc = append(append(sc, IconClass...), 0)
	out := []*swf.Tag{{Code: 39, Data: sd}, {Code: 76, Data: sc}}
	movie.Tags = append(movie.Tags[:at], append(out, movie.Tags[at:]...)...)
	return nil
}

// copySprite copies a sprite made of shapes into another movie, before the tag at index `at`,
// and links it to a class.
func copySprite(sprite *swf.Tag, defs map[int]*swf.Tag, dst *swf.Movie, at int, class string) (int, error) {
	inner, err := spriteTags(sprite)
	if err != nil {
		return 0, err
	}
	next := 0
	for _, t := range dst.Tags {
		if defineTags[t.Code] && len(t.Data) >= 2 && charID(t) >= next {
			next = charID(t) + 1
		}
	}
	remap := map[int]int{}
	var out []*swf.Tag
	for _, t := range inner {
		o := placedChar(t)
		if o < 0 {
			continue
		}
		id := int(binary.LittleEndian.Uint16(t.Data[o:]))
		if _, done := remap[id]; done {
			continue
		}
		d, ok := defs[id]
		if !ok || !(d.Code == 2 || d.Code == 22 || d.Code == 32 || d.Code == 83) {
			return 0, errors.New("unexpected icon content")
		}
		remap[id] = next
		c := &swf.Tag{Code: d.Code, Data: append([]byte{}, d.Data...)}
		binary.LittleEndian.PutUint16(c.Data, uint16(next))
		out = append(out, c)
		next++
	}
	var body []byte
	for _, t := range inner {
		c := &swf.Tag{Code: t.Code, Data: append([]byte{}, t.Data...)}
		if o := placedChar(c); o >= 0 {
			binary.LittleEndian.PutUint16(c.Data[o:], uint16(remap[int(binary.LittleEndian.Uint16(c.Data[o:]))]))
		}
		body = append(body, c.Encode()...)
	}
	spriteID := next
	sd := make([]byte, 4, 4+len(body))
	binary.LittleEndian.PutUint16(sd, uint16(spriteID))
	copy(sd[2:4], sprite.Data[2:4])
	out = append(out, &swf.Tag{Code: 39, Data: append(sd, body...)})
	// SymbolClass
	sc := []byte{1, 0, 0, 0}
	binary.LittleEndian.PutUint16(sc[2:], uint16(spriteID))
	sc = append(append(sc, class...), 0)
	out = append(out, &swf.Tag{Code: 76, Data: sc})
	dst.Tags = append(dst.Tags[:at], append(out, dst.Tags[at:]...)...)
	return len(out), nil
}

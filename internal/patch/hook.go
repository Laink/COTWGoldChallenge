package patch

import (
	"errors"
	"fmt"

	"github.com/Laink/COTWGoldChallenge/internal/swf"
)

// HUDPath is the always visible HUD, relative to the archives and to dropzone.
const HUDPath = "ui/hud.gfx"

// ModClass is the ActionScript class added by the mod.
const ModClass = "COTWGoldChallengeMod"

// hook makes a method call ModClass.fn(args...) at its start (after "getlocal0; pushscope")
// or just before its final returnvoid. Arguments are local register numbers.
func hook(abc *swf.ABC, method string, atStart bool, fn string, args ...int) error {
	body, err := abc.Body(method)
	if err != nil {
		return err
	}
	code := body.Code
	call := []byte{0x60} // getlex
	call = append(call, u30(abc.Name(ModClass))...)
	for _, r := range args {
		if r <= 3 {
			call = append(call, byte(0xD0+r))
		} else {
			call = append(call, 0x62)
			call = append(call, u30(r)...)
		}
	}
	call = append(call, 0x4F) // callpropvoid
	call = append(call, u30(abc.Name(fn))...)
	call = append(call, u30(len(args))...)

	var out []byte
	if atStart {
		if len(code) < 2 || code[0] != 0xD0 || code[1] != 0x30 {
			return fmt.Errorf("%s: unexpected prologue", method)
		}
		if body.HasExceptions() {
			return fmt.Errorf("%s: exception handlers", method)
		}
		out = append(append(append(out, code[:2]...), call...), code[2:]...)
	} else {
		if len(code) == 0 || code[len(code)-1] != 0x47 {
			return fmt.Errorf("%s: unexpected end", method)
		}
		out = append(append(append(out, code[:len(code)-1]...), call...), 0x47)
	}
	abc.Replace(body, out, max(body.MaxStack, len(args)+1), body.Locals)
	return nil
}

// MenuPaths are the movies with the reserve selection, relative to the archives and to dropzone.
var MenuPaths = []string{"ui/main_menu.gfx", "ui/change_reserve.gfx"}

type hookSpec struct {
	method  string
	atStart bool
	fn      string
	args    []int
	prop    string // instead of args: this and this.prop
}

// hookProp makes a method call ModClass.fn(this, this.prop) just before its final returnvoid. The
// property is read with the multiname the method itself uses, so private properties work too.
func hookProp(abc *swf.ABC, method, fn, prop string) error {
	body, err := abc.Body(method)
	if err != nil {
		return err
	}
	ins, err := swf.Disassemble(body.Code)
	if err != nil {
		return err
	}
	mn := -1
	for _, in := range ins {
		if (in.Op == "getproperty" || in.Op == "setproperty" || in.Op == "initproperty") && abc.Multinames[in.Args[0]].Name == prop {
			mn = in.Args[0]
			break
		}
	}
	if mn < 0 {
		return fmt.Errorf("%s: %s not found", method, prop)
	}
	code := body.Code
	if len(code) == 0 || code[len(code)-1] != 0x47 {
		return fmt.Errorf("%s: unexpected end", method)
	}
	call := append([]byte{0x60}, u30(abc.Name(ModClass))...)
	call = append(call, 0xD0, 0xD0, 0x66) // getlocal0, getlocal0, getproperty
	call = append(call, u30(mn)...)
	call = append(call, 0x4F)
	call = append(call, u30(abc.Name(fn))...)
	call = append(call, 2)
	out := append(append(append([]byte{}, code[:len(code)-1]...), call...), 0x47)
	abc.Replace(body, out, max(body.MaxStack, 4), body.Locals)
	return nil
}

// ApplyHUD adds the mod class and the species icons of clue_hud.gfx to hud.gfx, and calls
// the class when the HUD is created and when the player enters a region.
func ApplyHUD(original, clueHud []byte, extra [][]byte) ([]byte, error) {
	return applyMod(original, clueHud, extra, []hookSpec{
		{"hud.<init>", false, "hudCreated", []int{0}, ""},
		{"hud.EnteredRegion", true, "regionEntered", []int{0, 2}, ""},
	})
}

// ApplyMenu adds the mod class and the species icons to a movie with the reserve selection, and
// calls the class each time a reserve is shown.
func ApplyMenu(original, clueHud []byte, extra [][]byte) ([]byte, error) {
	return applyMod(original, clueHud, extra, []hookSpec{
		{"ChangeReserveParent.UpdateReserveData", false, "reserveShown", nil, "m_selectedReserveIndex"},
	})
}

func applyMod(original, clueHud []byte, extra [][]byte, hooks []hookSpec) ([]byte, error) {
	movie, err := swf.Load(original)
	if err != nil {
		return nil, err
	}
	tag, off, err := movie.ABCTag()
	if err != nil {
		return nil, err
	}
	abc, err := swf.ParseABC(tag.Data[off:])
	if err != nil {
		return nil, err
	}
	for _, h := range hooks {
		var err error
		if h.prop != "" {
			err = hookProp(abc, h.method, h.fn, h.prop)
		} else {
			err = hook(abc, h.method, h.atStart, h.fn, h.args...)
		}
		if err != nil {
			return nil, err
		}
	}
	b, err := abc.Bytes()
	if err != nil {
		return nil, err
	}
	tag.Data = append(append([]byte{}, tag.Data[:off]...), b...)
	if len(extra) == 0 {
		return nil, errors.New("no mod class")
	}
	for _, x := range extra {
		insertBefore(movie, tag, x)
	}
	clue, err := swf.Load(clueHud)
	if err != nil {
		return nil, err
	}
	for i, t := range movie.Tags {
		if t == tag {
			if _, err := copyIcons(clue, movie, i); err != nil {
				return nil, err
			}
			break
		}
	}
	return movie.Bytes(), nil
}

func insertBefore(movie *swf.Movie, tag *swf.Tag, data []byte) {
	for i, t := range movie.Tags {
		if t == tag {
			movie.Tags = append(movie.Tags[:i], append([]*swf.Tag{{Code: 82, Data: data}}, movie.Tags[i:]...)...)
			return
		}
	}
}

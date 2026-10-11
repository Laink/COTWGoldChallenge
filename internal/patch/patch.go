// Package patch adds the medal potential gauge to the binoculars panel (ui/clue_hud.gfx).
package patch

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Laink/COTWGoldChallenge/internal/game"
	"github.com/Laink/COTWGoldChallenge/internal/swf"
)

// GamePath is the patched file, relative to the archives and to dropzone.
const GamePath = "ui/clue_hud.gfx"

// SpriteName marks the gauge added to the panel.
const SpriteName = "COTWGoldChallenge"

// Options of the patch.
type Options struct {
	Language string // language code or Steam language name
	Preview  *[3]int
	Extra    [][]byte // DoABC tags added to the movie
	Hook     string   // mod class: its static gauge(...) draws the gauge, update(panel, data) ends SetData
	Icons    bool     // link the species icons to IconClass
}

// Place of the gauge, in panel units (the panel is 352.1 wide).
const (
	panelX      = -352.1
	panelBottom = 159.0
	gap         = 10.0
	anomalyLine = 53.0
)

// Apply patches the original clue_hud.gfx.
func Apply(original []byte, species map[int]*game.Species, o Options) ([]byte, error) {
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
	body, err := abc.Body("SpottingDisplayData.SetData")
	if err != nil {
		return nil, err
	}
	code := body.Code
	if len(code) == 0 || code[len(code)-1] != 0x47 {
		return nil, errors.New("unexpected end of SetData")
	}
	if o.Hook == "" {
		return nil, errors.New("the gauge needs the mod class")
	}
	g := &gen{abc: abc, asm: swf.NewAsm(len(code) - 1), locals: body.Locals, regs: map[string]int{}, t: TextFor(o.Language), hook: o.Hook}
	g.gauge(species)
	added, err := g.asm.Assemble()
	if err != nil {
		return nil, err
	}
	abc.Replace(body, append(append([]byte{}, code[:len(code)-1]...), added...), max(body.MaxStack, 12), body.Locals+len(g.regs))
	// the wall also shows the species of an animal heard (audio clue panel)
	if err := hook(abc, "ClueDisplayData.AudioClues", false, "heard", 0, 1); err != nil {
		return nil, err
	}
	if o.Preview != nil {
		if err := preview(abc, *o.Preview); err != nil {
			return nil, err
		}
	}
	b, err := abc.Bytes()
	if err != nil {
		return nil, err
	}
	tag.Data = append(append([]byte{}, tag.Data[:off]...), b...)
	for _, x := range o.Extra {
		insertBefore(movie, tag, x)
	}
	if o.Icons {
		if err := linkIcons(movie); err != nil {
			return nil, err
		}
	}
	return movie.Bytes(), nil
}

type gen struct {
	abc    *swf.ABC
	asm    *swf.Asm
	locals int
	regs   map[string]int
	t      Text
	hook   string
}

func (g *gen) reg(name string) int {
	if r, ok := g.regs[name]; ok {
		return r
	}
	g.regs[name] = g.locals + len(g.regs)
	return g.regs[name]
}

func (g *gen) op(op string, args ...any) *gen { g.asm.Op(op, args...); return g }
func (g *gen) label(l string) *gen            { g.asm.Label(l); return g }
func (g *gen) p(name string) int              { return g.abc.Name(name) }
func (g *gen) s(v string) int                 { return g.abc.String(v) }
func (g *gen) get(r int) *gen                 { return g.op("getlocal", r) }
func (g *gen) set(r int) *gen                 { return g.op("setlocal", r) }

func (g *gen) num(v float64) *gen {
	if v == math.Trunc(v) && v >= -128 && v <= 127 {
		return g.op("pushbyte", int(v)&0xff)
	}
	return g.op("pushdouble", g.abc.Double(v))
}

// gauge adds the medal gauge under the panel: a sprite that the mod class draws (gauge in the AS3
// class), with the medal thresholds of the species and the texts in the game language.
func (g *gen) gauge(species map[int]*game.Species) {
	t := g.t
	rSP := g.reg("SP")

	// Remove the previous gauge.
	g.op("getlocal0").op("pushstring", g.s(SpriteName)).op("callproperty", g.p("getChildByName"), 1).op("coerce_a").set(rSP)
	g.get(rSP).op("iffalse", "no_old")
	g.op("getlocal0").get(rSP).op("callpropvoid", g.p("removeChild"), 1)
	g.label("no_old")

	// Medal thresholds of the species.
	rS, rG, rD := g.reg("S"), g.reg("G"), g.reg("D")
	g.op("getlocal1").op("getproperty", g.p("animal_id")).op("convert_i")
	// one case per icon id, up to the highest one (new species of a game update included)
	ids := 1
	for id := range species {
		ids = max(ids, id+1)
	}
	cases := make([]string, ids)
	for i := range cases {
		if _, ok := species[i]; ok {
			cases[i] = fmt.Sprintf("c%d", i)
		} else {
			cases[i] = "unknown"
		}
	}
	g.op("lookupswitch", "unknown", cases)
	for _, id := range game.SortedIcons(species) {
		if id < 0 {
			continue
		}
		sp := species[id]
		g.label(fmt.Sprintf("c%d", id))
		for _, v := range []float64{sp.Silver, sp.Gold, sp.Diamond} {
			g.op("pushdouble", g.abc.Double(v))
		}
		g.op("jump", "known")
	}
	g.label("known")
	g.set(rD).set(rG).set(rS)

	// Sprite under the panel, lower when the anomaly line shows.
	g.op("findpropstrict", g.p("Sprite")).op("constructprop", g.p("Sprite"), 0).op("coerce_a").set(rSP)
	g.get(rSP).op("pushstring", g.s(SpriteName)).op("setproperty", g.p("name"))
	g.get(rSP).num(panelX).op("setproperty", g.p("x"))
	g.get(rSP).num(panelBottom + gap)
	g.op("getlocal0").op("getproperty", g.p("MCI_AnomalyType")).op("getproperty", g.p("visible")).op("iffalse", "y_ok")
	g.num(anomalyLine).op("add")
	g.label("y_ok").op("setproperty", g.p("y"))

	// gauge(panel, sprite, data, silver, gold, diamond, texts): texts joined with "|".
	words := []string{t.Title, t.NoTrophy}
	for _, a := range [][4]string{t.Sure, t.Possible, t.Impossible, t.TiersShort} {
		words = append(words, a[:]...)
	}
	words = append(words, t.Percent, t.SkillMissing)
	g.op("getlex", g.abc.Name(g.hook)).op("getlocal0").get(rSP).op("getlocal1").get(rS).get(rG).get(rD)
	g.op("pushstring", g.s(strings.Join(words, "|"))).op("callpropvoid", g.p("gauge"), 7)
	g.op("getlocal0").get(rSP).op("callpropvoid", g.p("addChild"), 1)
	g.op("jump", "end")
	g.label("unknown")
	g.label("end")
	g.op("getlex", g.p(g.hook)).op("getlocal0").op("getlocal1").op("callpropvoid", g.p("update"), 2)
	g.op("returnvoid")
}

// preview makes the panel show test data outside the game (Flash players, Ruffle).
func preview(abc *swf.ABC, data [3]int) error {
	init, err := abc.Body("clue_hud.<init>")
	if err != nil {
		return err
	}
	code := append([]byte{}, init.Code...)
	ins, err := swf.Disassemble(code)
	if err != nil {
		return err
	}
	start, end := -1, -1
	for i, in := range ins {
		if start < 0 && in.Op == "getlex" && abc.Multinames[in.Args[0]].Name == "Capabilities" {
			start = in.Pos
		}
		if start >= 0 && in.Op == "iffalse" {
			end = ins[i].Pos + 4
			break
		}
	}
	if start < 0 || end < 0 {
		return errors.New("test data block not found")
	}
	for i := start; i < end; i++ {
		code[i] = 0x02
	}
	fields := map[string]int{"animal_id": data[0], "animal_score_min": data[1], "animal_score_max": data[2]}
	for i := 0; i+1 < len(ins); i++ {
		if ins[i].Op == "pushbyte" && ins[i+1].Op == "setproperty" && ins[i].Pos > 330 {
			if v, ok := fields[abc.Multinames[ins[i+1].Args[0]].Name]; ok {
				code[ins[i].Pos+1] = byte(v)
			}
		}
	}
	abc.Replace(init, code, init.MaxStack, init.Locals)
	loc := abc.String("Loc")
	for name, m := range abc.Methods {
		if len(name) > 19 && name[len(name)-19:] == ".GetLocalizedString" {
			if bd, ok := abc.Bodies[m]; ok {
				abc.Replace(bd, append([]byte{0xD0, 0x30, 0x2C}, append(u30(loc), 0x48)...), 2, bd.Locals)
			}
		}
	}
	return nil
}

func u30(v int) []byte {
	var out []byte
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			out = append(out, c|0x80)
		} else {
			return append(out, c)
		}
	}
}

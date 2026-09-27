// Package patch adds the medal potential gauge to the binoculars panel (ui/clue_hud.gfx).
package patch

import (
	"errors"
	"fmt"
	"math"
	"strconv"
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
	Hook     string   // class whose static update(panel, data) ends SetData
	Icons    bool     // link the species icons to IconClass
}

// Layout, in panel units (the panel is 352.1 wide).
const (
	panelWidth   = 352.1
	bannerHeight = 30
	bodyHeight   = 62
	panelX       = -352.1
	panelBottom  = 159.0
	gap          = 10.0
	anomalyLine  = 53.0
	barX0        = 8.0
	barX1        = 344.0
	barY         = 40.0
	barH         = 10.0
	skillLine    = 16.0 // added to the body for the missing skill line

	orange  = 0xFF9800
	black   = 0x000000
	white   = 0xFFFFFF
	bronze  = 0xC87A35
	silver  = 0xE2E2E2
	gold    = 0xFFC42E
	diamond = 0x8FE3FF
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
	g := &gen{abc: abc, asm: swf.NewAsm(len(code) - 1), locals: body.Locals, regs: map[string]int{}, t: TextFor(o.Language), hook: o.Hook}
	g.gauge(species, o.Preview != nil)
	added, err := g.asm.Assemble()
	if err != nil {
		return nil, err
	}
	abc.Replace(body, append(append([]byte{}, code[:len(code)-1]...), added...), max(body.MaxStack, 12), body.Locals+len(g.regs))
	if o.Hook != "" {
		// the wall also shows the species of an animal heard (audio clue panel)
		if err := hook(abc, "ClueDisplayData.AudioClues", false, "heard", 0, 1); err != nil {
			return nil, err
		}
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

// pushOperand pushes a register (r >= 0) or the constant -r-1.
func (g *gen) pushOperand(r int) {
	if r >= 0 {
		g.get(r)
	} else {
		g.num(float64(-r - 1))
	}
}

func (g *gen) gauge(species map[int]*game.Species, trace bool) {
	t := g.t
	rSP, rGR, rTF, rFMT := g.reg("SP"), g.reg("GR"), g.reg("TF"), g.reg("FMT")

	// Remove the previous gauge.
	g.op("getlocal0").op("pushstring", g.s(SpriteName)).op("callproperty", g.p("getChildByName"), 1).op("coerce_a").set(rSP)
	g.get(rSP).op("iffalse", "no_old")
	g.op("getlocal0").get(rSP).op("callpropvoid", g.p("removeChild"), 1)
	g.label("no_old")

	// Trophy range and species scale.
	rA, rB, rS, rG, rD, rLo0, rHi0 := g.reg("A"), g.reg("B"), g.reg("S"), g.reg("G"), g.reg("D"), g.reg("L0"), g.reg("H0")
	g.op("getlocal1").op("getproperty", g.p("animal_score_min")).op("convert_d").set(rA)
	g.op("getlocal1").op("getproperty", g.p("animal_score_max")).op("convert_d").set(rB)
	// The game truncates the range bounds to integers: the true bounds are up to 1 higher.
	rB0 := g.reg("B0")
	g.get(rB).set(rB0)
	g.get(rB).num(1).op("add").op("convert_d").set(rB)
	g.op("getlocal1").op("getproperty", g.p("animal_id")).op("convert_i")
	const ids = 140
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
		if id >= ids {
			continue
		}
		sp := species[id]
		g.label(fmt.Sprintf("c%d", id))
		for _, v := range []float64{sp.Silver, sp.Gold, sp.Diamond, sp.Min, sp.Max} {
			g.op("pushdouble", g.abc.Double(v))
		}
		// Threshold labels are formatted here: in the game, rounding to 2 decimals gives values like 84.0999.
		for _, v := range []float64{sp.Silver, sp.Gold, sp.Diamond} {
			g.op("pushstring", g.s(strings.Replace(strconv.FormatFloat(v, 'f', -1, 64), ".", t.Decimal, 1)))
		}
		g.op("jump", "known")
	}
	g.label("known")
	rLS, rLG, rLD := g.reg("LS"), g.reg("LG"), g.reg("LD")
	for _, r := range []int{rLD, rLG, rLS, rHi0, rLo0, rD, rG, rS} {
		g.set(r)
	}

	// Probabilities, assuming a uniform score within the range.
	chance := func(dst, threshold int) {
		l1, l2, l3 := g.asm.NewLabel(), g.asm.NewLabel(), g.asm.NewLabel()
		g.get(rA).get(threshold).op("ifnge", l1).num(1).op("convert_d").set(dst).op("jump", l3)
		g.label(l1).get(rB).get(threshold).op("ifnlt", l2).num(0).op("convert_d").set(dst).op("jump", l3)
		g.label(l2).get(rB).get(threshold).op("subtract").get(rB).get(rA).op("subtract").op("divide").set(dst)
		g.label(l3)
	}
	rPS, rPG, rPD := g.reg("PA"), g.reg("PG"), g.reg("PD")
	chance(rPS, rS)
	chance(rPG, rG)
	chance(rPD, rD)
	if g.hook != "" {
		// The mod class refines the estimate with the weight and the game's estimation rules when it
		// can: {pS, pG, pD: chances of silver, gold, diamond; lo, hi: score range}, or null.
		rE := g.reg("E")
		g.op("getlex", g.abc.Name(g.hook)).op("getlocal1").op("callproperty", g.p("estimate"), 1).op("coerce_a").set(rE)
		g.get(rE).op("iffalse", "est_none")
		for _, f := range []struct {
			name string
			r    int
		}{{"pS", rPS}, {"pG", rPG}, {"pD", rPD}, {"lo", rA}, {"hi", rB}} {
			g.get(rE).op("getproperty", g.p(f.name)).op("convert_d").set(f.r)
		}
		g.label("est_none")
	}
	percent := func(r int) {
		g.op("getlex", g.p("Math")).get(r).num(100).op("multiply").op("pushdouble", g.abc.Double(0.5)).op("add")
		g.op("callproperty", g.p("floor"), 1).op("pushstring", g.s(t.Percent)).op("add")
	}

	// Verdict (V), detail (SUB), check or cross (OK), no trophy (NONE).
	rV, rSub, rOK, rNone := g.reg("V"), g.reg("SUB"), g.reg("OK"), g.reg("SANS")
	setStr := func(r int, s string) { g.op("pushstring", g.s(s)).set(r) }
	setStr(rSub, "")
	g.num(0).set(rOK).num(0).set(rNone)
	// Without the Spotting Knowledge skill (level 3) the game gives no trophy estimate: the scale is
	// drawn without the animal's range.
	rSkill := g.reg("SK")
	g.op("getlocal1").op("getproperty", g.p("animal_score_visible")).op("convert_b").set(rSkill)
	g.get(rSkill).op("iftrue", "v_skill")
	setStr(rV, "")
	g.num(1).set(rNone).op("jump", "v_end")
	g.label("v_skill")
	diamondDetail := func() {
		l := g.asm.NewLabel()
		g.get(rPD).num(0).op("ifngt", l)
		g.op("pushstring", g.s(t.Diamond))
		percent(rPD)
		g.op("add").op("coerce_s").set(rSub)
		g.label(l)
	}
	g.get(rB0).num(0).op("ifgt", "v_pos")
	setStr(rV, t.NoTrophy)
	g.num(1).set(rNone).op("jump", "v_end")
	g.label("v_pos")
	g.get(rPD).num(1).op("ifnge", "v_1")
	setStr(rV, t.DiamondSure)
	g.num(1).set(rOK).op("jump", "v_end")
	g.label("v_1")
	g.get(rPG).num(1).op("ifnge", "v_2")
	setStr(rV, t.GoldSure)
	g.num(1).set(rOK)
	diamondDetail()
	g.op("jump", "v_end")
	g.label("v_2")
	g.get(rPG).num(0).op("ifngt", "v_3")
	g.op("pushstring", g.s(t.GoldPossible))
	percent(rPG)
	g.op("add").op("coerce_s").set(rV)
	g.num(1).set(rOK)
	diamondDetail()
	g.op("jump", "v_end")
	g.label("v_3")
	setStr(rV, t.GoldImpossible)
	g.get(rPS).num(1).op("ifnge", "v_4")
	setStr(rSub, t.SilverSure)
	g.op("jump", "v_end")
	g.label("v_4")
	g.get(rPS).num(0).op("ifngt", "v_5")
	g.op("pushstring", g.s(t.Silver))
	percent(rPS)
	g.op("add").op("coerce_s").set(rSub).op("jump", "v_end")
	g.label("v_5")
	setStr(rSub, t.BronzeAtBest)
	g.label("v_end")

	// Scale: species range, widened to the observed range.
	rLo, rHi, rK := g.reg("LO"), g.reg("HI"), g.reg("K")
	g.get(rLo0).set(rLo).get(rHi0).set(rHi)
	g.get(rNone).op("iftrue", "scale_ok")
	g.op("getlex", g.p("Math")).get(rLo0).get(rA).op("callproperty", g.p("min"), 2).op("convert_d").set(rLo)
	g.op("getlex", g.p("Math")).get(rHi0).get(rB).op("callproperty", g.p("max"), 2).op("convert_d").set(rHi)
	g.label("scale_ok")
	g.num(barX1 - barX0).get(rHi).get(rLo).op("subtract").op("divide").set(rK)
	rXS, rXG, rXD, rXA, rXB := g.reg("XS"), g.reg("XG"), g.reg("XD"), g.reg("XA"), g.reg("XB")
	for _, pr := range [][2]int{{rXS, rS}, {rXG, rG}, {rXD, rD}, {rXA, rA}, {rXB, rB}} {
		g.get(pr[1]).get(rLo).op("subtract").get(rK).op("multiply").num(barX0).op("add").op("convert_d").set(pr[0])
	}

	// Sprite, banner and body.
	g.op("findpropstrict", g.p("Sprite")).op("constructprop", g.p("Sprite"), 0).op("coerce_a").set(rSP)
	g.get(rSP).op("pushstring", g.s(SpriteName)).op("setproperty", g.p("name"))
	g.get(rSP).num(panelX).op("setproperty", g.p("x"))
	g.get(rSP).num(panelBottom + gap)
	g.op("getlocal0").op("getproperty", g.p("MCI_AnomalyType")).op("getproperty", g.p("visible")).op("iffalse", "y_ok")
	g.num(anomalyLine).op("add")
	g.label("y_ok").op("setproperty", g.p("y"))
	g.get(rSP).op("getproperty", g.p("graphics")).op("coerce_a").set(rGR)
	rect := func(color, alpha float64, x, y, w, h func()) {
		g.get(rGR).num(color).num(alpha).op("callpropvoid", g.p("beginFill"), 2)
		g.get(rGR)
		x()
		y()
		w()
		h()
		g.op("callpropvoid", g.p("drawRect"), 4)
		g.get(rGR).op("callpropvoid", g.p("endFill"), 0)
	}
	c := func(v float64) func() { return func() { g.num(v) } }
	rect(orange, 1, c(0), c(0), c(panelWidth), c(bannerHeight))
	rect(black, 0.65, c(0), c(bannerHeight), c(panelWidth), func() {
		l := g.asm.NewLabel()
		g.num(bodyHeight).get(rSkill).op("iftrue", l).num(skillLine).op("add")
		g.label(l)
	})

	// Scale segments. Negative operands are constants: -v-1.
	x0, x1 := -int(barX0)-1, -int(barX1)-1
	segs := [4][2]int{{x0, rXS}, {rXS, rXG}, {rXG, rXD}, {rXD, x1}}
	colors := [4]float64{bronze, silver, gold, diamond}
	segWidth := func(s [2]int) { g.pushOperand(s[1]); g.pushOperand(s[0]); g.op("subtract") }
	for i, s := range segs {
		s := s
		rect(colors[i], 1, func() { g.pushOperand(s[0]) }, c(barY), func() { segWidth(s) }, c(barH))
	}

	// Observed range: dark outline, then white.
	g.get(rNone).op("iftrue", "no_range")
	for _, st := range [2][3]float64{{4, black, 0.6}, {2, white, 1}} {
		g.get(rGR).num(st[0]).num(st[1]).num(st[2]).op("callpropvoid", g.p("lineStyle"), 3)
		g.get(rGR).get(rXA).num(1.5).op("subtract")
		g.num(barY - 3)
		g.op("getlex", g.p("Math")).get(rXB).get(rXA).op("subtract").num(2).op("callproperty", g.p("max"), 2).num(3).op("add")
		g.num(barH+6).op("callpropvoid", g.p("drawRect"), 4)
	}
	g.get(rGR).op("callpropvoid", g.p("lineStyle"), 0)
	g.label("no_range")

	// Texts reuse the game fonts and effects of the panel.
	bannerFx := func() { g.op("getlocal0").op("getproperty", g.p("tf_species_name")) }
	scoreFx := func() { g.op("getlocal0").op("getproperty", g.p("MCI_score")).op("getproperty", g.p("tf_score")) }
	text := func(source string, push func(), size float64, color float64, y float64) {
		fx := scoreFx
		if source == "tf_species_name" {
			fx = bannerFx
		}
		g.op("getlocal0").op("getlocal0").op("getproperty", g.p(source)).op("callproperty", g.p("copyTextField"), 1).op("coerce_a").set(rTF)
		g.get(rTF)
		fx()
		g.op("getproperty", g.p("filters")).op("setproperty", g.p("filters"))
		g.get(rTF).op("callproperty", g.p("getTextFormat"), 0).op("coerce_a").set(rFMT)
		if size != 0 {
			g.get(rFMT).num(size).op("setproperty", g.p("size"))
		}
		if color >= 0 {
			g.get(rFMT).num(color).op("setproperty", g.p("color"))
		}
		g.get(rTF).get(rFMT).op("setproperty", g.p("defaultTextFormat"))
		g.get(rTF)
		push()
		g.op("setproperty", g.p("text"))
		g.get(rTF).get(rFMT).op("callpropvoid", g.p("setTextFormat"), 1)
		g.get(rTF).get(rTF).op("getproperty", g.p("textWidth")).num(6).op("add").op("setproperty", g.p("width"))
		g.get(rTF).num(y).op("setproperty", g.p("y"))
		g.get(rSP).get(rTF).op("callpropvoid", g.p("addChild"), 1)
	}
	setX := func(push func()) { g.get(rTF); push(); g.op("setproperty", g.p("x")) }
	textWidth := func() { g.get(rTF).op("getproperty", g.p("width")) }
	str := func(s string) func() { return func() { g.op("pushstring", g.s(s)) } }

	// Banner: title left, verdict and detail right, check or cross.
	text("tf_species_name", str(t.Title), 0, -1, 3.2)
	setX(c(6.5))
	rTitle := g.reg("TT")
	g.get(rTF).set(rTitle)
	rW, rX := g.reg("W1"), g.reg("X1")
	rVerdict := g.reg("TFV")
	text("tf_species_name", func() { g.get(rV) }, 0, -1, 3.2)
	g.get(rTF).set(rVerdict)
	textWidth()
	g.set(rW)
	g.num(panelWidth - 6).get(rW).op("subtract").set(rX)
	g.get(rSub).op("pushstring", g.s("")).op("ifeq", "no_detail")
	text("tf_species_name", func() { g.op("pushstring", g.s("· ")).get(rSub).op("add") }, 15, -1, 8.5)
	g.num(panelWidth - 6)
	textWidth()
	g.op("subtract").op("convert_d").set(rX)
	setX(func() { g.get(rX) })
	g.get(rX).get(rW).op("subtract").num(2).op("add").set(rX)
	g.label("no_detail")
	g.get(rVerdict).get(rX).op("setproperty", g.p("x"))
	rIX := g.reg("IX")
	g.get(rX).num(20).op("subtract").set(rIX)
	g.get(rSkill).op("iftrue", "ix_ok")
	g.get(rX).set(rIX) // no check or cross without the skill
	g.label("ix_ok")
	g.get(rIX).num(6.5+4).get(rTitle).op("getproperty", g.p("width")).op("add").op("ifnlt", "title_ok")
	g.get(rTitle).op("pushfalse").op("setproperty", g.p("visible"))
	g.label("title_ok")
	rShape, rGI := g.reg("SH"), g.reg("GI")
	g.op("findpropstrict", g.p("Shape")).op("constructprop", g.p("Shape"), 0).op("coerce_a").set(rShape)
	g.get(rShape)
	bannerFx()
	g.op("getproperty", g.p("filters")).op("setproperty", g.p("filters"))
	g.get(rShape).op("getproperty", g.p("graphics")).op("coerce_a").set(rGI)
	g.get(rGI).num(2.5).num(white).num(1).op("callpropvoid", g.p("lineStyle"), 3)
	stroke := func(pts [][2]float64) {
		g.get(rGI).get(rIX).num(pts[0][0]).op("add").num(pts[0][1]).op("callpropvoid", g.p("moveTo"), 2)
		for _, pt := range pts[1:] {
			g.get(rGI).get(rIX).num(pt[0]).op("add").num(pt[1]).op("callpropvoid", g.p("lineTo"), 2)
		}
	}
	g.get(rSkill).op("iffalse", "icon_ok")
	g.get(rOK).op("iffalse", "cross")
	stroke([][2]float64{{3, 15.5}, {7, 19.5}, {14.5, 10.5}})
	g.op("jump", "icon_ok")
	g.label("cross")
	stroke([][2]float64{{4, 10}, {13, 19}})
	stroke([][2]float64{{13, 10}, {4, 19}})
	g.label("icon_ok")
	g.get(rSP).get(rShape).op("callpropvoid", g.p("addChild"), 1)

	// Tier names centred on each segment, shortened or hidden when narrow.
	rSegW, rSegC := g.reg("SEGW"), g.reg("SEGC")
	for i, s := range segs {
		segWidth(s)
		g.op("convert_d").set(rSegW)
		g.pushOperand(s[0])
		g.get(rSegW).num(2).op("divide").op("add").set(rSegC)
		skip, short, done := g.asm.NewLabel(), g.asm.NewLabel(), g.asm.NewLabel()
		g.get(rSegW).num(24).op("ifngt", skip)
		g.get(rSegW).num(float64(width(t.Tiers[i])+6)).op("ifngt", short)
		text("tf_difficulty", str(t.Tiers[i]), 13, colors[i], barY+barH+1)
		g.op("jump", done)
		g.label(short)
		text("tf_difficulty", str(t.TiersShort[i]), 13, colors[i], barY+barH+1)
		g.label(done)
		setX(func() { g.get(rSegC); textWidth(); g.num(2).op("divide").op("subtract") })
		g.label(skip)
	}

	// Threshold values under the segment boundaries.
	for _, pr := range [][2]int{{rXS, rLS}, {rXG, rLG}, {rXD, rLD}} {
		rx, rl := pr[0], pr[1]
		text("tf_difficulty", func() { g.get(rl) }, 12, white, barY+barH+17)
		setX(func() {
			g.op("getlex", g.p("Math")).num(2)
			g.op("getlex", g.p("Math")).get(rx)
			textWidth()
			g.num(2).op("divide").op("subtract")
			g.num(panelWidth - 2)
			textWidth()
			g.op("subtract")
			g.op("callproperty", g.p("min"), 2).op("callproperty", g.p("max"), 2)
		})
	}

	// Name of the missing skill, under the scale.
	g.get(rSkill).op("iftrue", "skill_ok")
	text("tf_difficulty", func() {
		g.op("pushstring", g.s(t.SkillMissing))
		if g.hook != "" {
			g.op("getlex", g.abc.Name(g.hook)).op("getlocal0").op("callproperty", g.p("skillName"), 1)
		} else {
			g.op("pushstring", g.s(SkillName))
		}
		g.op("add")
	}, 13, white, barY+barH+32)
	g.get(rTF).op("pushfalse").op("setproperty", g.p("wordWrap"))
	g.get(rTF).get(rTF).op("getproperty", g.p("textWidth")).num(6).op("add").op("setproperty", g.p("width"))
	setX(c(barX0))
	g.label("skill_ok")

	g.op("getlocal0").get(rSP).op("callpropvoid", g.p("addChild"), 1)
	if trace {
		g.op("findpropstrict", g.p("trace")).op("pushstring", g.s("COTWGOLDCHALLENGE ")).get(rV).op("add")
		g.op("pushstring", g.s(" | ")).op("add").get(rSub).op("add").op("callpropvoid", g.p("trace"), 1)
	}
	g.op("jump", "end")
	g.label("unknown")
	g.label("end")
	if g.hook != "" {
		g.op("getlex", g.p(g.hook)).op("getlocal0").op("getlocal1").op("callpropvoid", g.p("update"), 2)
	}
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

// Package settings describes the settings of the in-game wall (cotwgc_settings.txt, read by the mod
// when the game starts) and reads and writes that file.
package settings

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Laink/COTWGoldChallenge/internal/safefile"
)

// Path is the settings file, relative to the dropzone folder.
const Path = "ui/cotwgc_settings.txt"

// Text is a text in each program language.
type Text map[string]string

// Get returns the text in a language, English by default.
func (t Text) Get(lang string) string {
	if s, ok := t[lang]; ok {
		return s
	}
	return t["en"]
}

// Choice is one value of a list.
type Choice struct {
	Value string
	Label Text
}

// Field is one setting.
type Field struct {
	Key      string
	Kind     string // date, choice, bool, number, slider (number with a slider), range, text, key ("code:name")
	Group    string // key of Groups
	Advanced bool
	Default  Text // per language for texts, "en" otherwise
	Label    Text
	Help     Text
	Choices  []Choice
	Min      float64
	Max      float64
	Step     float64
}

func t(en, fr string) Text { return Text{"en": en, "fr": fr} }
func d(v string) Text      { return Text{"en": v} }

var yesNo = []Choice{{"1", t("Yes", "Oui")}, {"0", t("No", "Non")}}

// Fields lists the settings in display order.
var Fields = []Field{
	{Key: "since", Kind: "date", Group: "challenge", Default: d(""),
		Label: t("Challenge start", "Début du défi"),
		Help:  t("Only harvests made since this date count. \"All time\" counts every harvest you ever made.", "Seuls les prélèvements faits depuis cette date comptent. « Depuis toujours » : tous vos prélèvements comptent.")},
	{Key: "scope", Kind: "choice", Group: "challenge", Default: d("reserve"),
		Label: t("Challenge mode", "Mode du défi"),
		Choices: []Choice{
			{"reserve", t("Reserve: only harvests made in the reserve you play", "Réserve : seulement les prélèvements faits dans la réserve jouée")},
			{"mega", t("100%: every species, harvested in any reserve", "100% : toutes les espèces, prélevées dans n'importe quelle réserve")},
		}},
	{Key: "tier", Kind: "choice", Group: "challenge", Default: d("1"),
		Label: t("Medal to get", "Médaille à obtenir"),
		Choices: []Choice{
			{"0", t("Diamond", "Diamant")}, {"1", t("Gold", "Or")}, {"2", t("Silver", "Argent")}, {"3", t("Bronze", "Bronze")},
		}},
	{Key: "higher", Kind: "bool", Group: "challenge", Default: d("1"),
		Label:   t("Better medals count too", "Les médailles supérieures comptent aussi"),
		Help:    t("For a gold challenge, a diamond counts as done.", "Pour un défi or, un diamant compte comme réussi."),
		Choices: yesNo},

	{Key: "wall", Kind: "choice", Group: "wall", Default: d("grid"),
		Label: t("Overlay", "Overlay"),
		Choices: []Choice{
			{"grid", t("Icons", "Icônes")}, {"list", t("List with names (reserve mode)", "Liste avec les noms (mode réserve)")}, {"0", t("Hidden", "Masqué")},
		}},
	{Key: "toggleKey", Kind: "key", Group: "wall", Default: d("119:F8"),
		Label: t("Key to show or hide the overlay", "Touche pour afficher ou masquer l'overlay"),
		Help: t("Works while COTWGoldChallenge is open (menu 4 turns the keys off). The game still receives the key.",
			"Fonctionne tant que COTWGoldChallenge est ouvert (le menu 4 désactive les touches). Le jeu reçoit quand même la touche."),
	},
	{Key: "namesKey", Kind: "key", Group: "wall", Default: d("120:F9"),
		Label: t("Key held to show the names of the missing species", "Touche maintenue pour afficher les noms des espèces manquantes"),
		Help: t("While held, the overlay lists the missing species of the reserve played, with their names. Works while COTWGoldChallenge is open.",
			"Tant qu'elle est maintenue, l'overlay liste les espèces manquantes de la réserve jouée, avec leurs noms. Fonctionne tant que COTWGoldChallenge est ouvert."),
	},
	{Key: "opaqueKey", Kind: "key", Group: "wall", Default: d("121:F10"),
		Label: t("Key held to show the species outside the reserve at full opacity", "Touche maintenue pour afficher les espèces hors réserve à 100 % d'opacité"),
		Help:  t("100% mode. Works while COTWGoldChallenge is open.", "Mode 100%. Fonctionne tant que COTWGoldChallenge est ouvert."),
	},
	{Key: "megaWall", Kind: "bool", Group: "wall", Default: d("1"),
		Label:   t("100% mode: show every species", "Mode 100% : afficher toutes les espèces"),
		Help:    t("Otherwise, only the species of the reserve you play.", "Sinon, seulement les espèces de la réserve jouée."),
		Choices: yesNo},
	{Key: "missing", Kind: "bool", Group: "wall", Default: d("0"),
		Label:   t("Reserve mode: show only the missing species", "Mode réserve : afficher seulement les espèces manquantes"),
		Choices: yesNo},
	{Key: "classes", Kind: "bool", Group: "wall", Default: d("1"),
		Label:   t("Class numbers", "Numéros de classe"),
		Choices: yesNo},
	{Key: "attenue", Kind: "range", Group: "wall", Default: d("0.55"), Min: 0.1, Max: 1, Step: 0.05,
		Label: t("Opacity of the species outside the reserve", "Opacité des espèces hors réserve")},

	{Key: "line", Kind: "choice", Group: "binoculars", Default: d("auto"),
		Label: t("\"Harvested\" line under the gauge", "Ligne « déjà obtenu » sous la jauge"),
		Choices: []Choice{
			{"auto", t("Only when the overlay does not show the species (overlay hidden, or species done with \"missing only\")", "Seulement si l'overlay ne montre pas l'espèce (overlay masqué, ou espèce réussie avec « seulement les manquantes »)")},
			{"1", t("Always", "Toujours")}, {"0", t("Never", "Jamais")},
		}},
	{Key: "aimScale", Kind: "range", Group: "binoculars", Default: d("1.35"), Min: 1, Max: 2, Step: 0.05,
		Label: t("Size of the aimed or heard species in the overlay", "Taille de l'espèce visée ou entendue dans l'overlay")},
	{Key: "aimAnim", Kind: "number", Group: "binoculars", Default: d("14"), Min: 0, Max: 60, Step: 1,
		Label: t("Animation length (frames, 0 = none)", "Durée de l'animation (images, 0 = aucune)")},

	{Key: "timerOn", Kind: "bool", Group: "timer", Default: d("0"),
		Label: t("Show the challenge timer in the game", "Afficher le chrono du challenge en jeu"),
		Help: t("It counts while the game window is in front and COTWGoldChallenge is open, unless you pause it. The time already played can be set or estimated below.",
			"Il tourne tant que la fenêtre du jeu est au premier plan et que COTWGoldChallenge est ouvert, sauf si vous le mettez en pause. Le temps déjà joué se règle ou s'estime plus bas."),
		Choices: yesNo},
	{Key: "timerSub", Kind: "bool", Group: "timer", Default: d("1"),
		Label: t("Under it: time since the last species done", "En dessous : temps depuis la dernière espèce validée"),
		Help: t("In challenge time, since the last species that reached the medal to get for the first time.",
			"En temps de challenge, depuis la dernière espèce qui a atteint pour la première fois la médaille visée."),
		Choices: yesNo},
	{Key: "pauseKey", Kind: "key", Group: "timer", Default: d("19:Pause"),
		Label: t("Key to pause or resume the timer", "Touche pour mettre en pause ou relancer le chrono"),
		Help: t("Works while COTWGoldChallenge is open. The game still receives the key.",
			"Fonctionne tant que COTWGoldChallenge est ouvert. Le jeu reçoit quand même la touche.")},
	{Key: "timerKey", Kind: "key", Group: "timer", Default: d("45:Insert"),
		Label: t("Key to show or hide the timer", "Touche pour afficher ou masquer le chrono")},
	{Key: "timerCorner", Kind: "choice", Group: "timer", Default: d("bl"),
		Label: t("Corner of the timer", "Coin du chrono"),
		Help: t("The distances below are from this corner, whatever the size of the timer.",
			"Les distances ci-dessous partent de ce coin, quelle que soit la taille du chrono."),
		Choices: []Choice{
			{"tl", t("Top left", "En haut à gauche")}, {"tr", t("Top right", "En haut à droite")},
			{"bl", t("Bottom left", "En bas à gauche")}, {"br", t("Bottom right", "En bas à droite")},
		}},
	{Key: "timerX", Kind: "slider", Group: "timer", Default: d("10"), Min: 0, Max: 1240, Step: 1,
		Label: t("Timer: distance from the left or right edge", "Chrono : distance du bord gauche ou droit")},
	{Key: "timerY", Kind: "slider", Group: "timer", Default: d("10"), Min: 0, Max: 700, Step: 1,
		Label: t("Timer: distance from the top or bottom edge", "Chrono : distance du bord haut ou bas")},
	{Key: "timerScale", Kind: "range", Group: "timer", Default: d("0.6"), Min: 0.5, Max: 2.5, Step: 0.05,
		Label: t("Timer size", "Taille du chrono")},

	{Key: "wallX", Kind: "slider", Group: "position", Advanced: true, Default: d("40"), Min: 0, Max: 1200, Step: 1,
		Label: t("Distance from the left edge", "Distance du bord gauche")},
	{Key: "wallY", Kind: "slider", Group: "position", Advanced: true, Default: d("40"), Min: 0, Max: 700, Step: 1,
		Label: t("Distance from the top edge", "Distance du bord haut")},
	{Key: "megaSize", Kind: "number", Group: "mega", Advanced: true, Default: d("20"), Min: 10, Max: 60, Step: 1,
		Label: t("Icon size (100% mode)", "Taille des icônes (mode 100 %)")},
	{Key: "megaGap", Kind: "number", Group: "mega", Advanced: true, Default: d("3"), Min: 0, Max: 20, Step: 1,
		Label: t("Space between icons", "Espace entre les icônes")},
	{Key: "megaRows", Kind: "number", Group: "mega", Advanced: true, Default: d("5"), Min: 1, Max: 20, Step: 1,
		Label: t("Rows (100% mode)", "Nombre de lignes (mode 100 %)")},
	{Key: "reserveSize", Kind: "number", Group: "mega", Advanced: true, Default: d("30"), Min: 10, Max: 80, Step: 1,
		Label: t("Icon size (reserve mode)", "Taille des icônes (mode réserve)")},
	{Key: "perRow", Kind: "number", Group: "mega", Advanced: true, Default: d("10"), Min: 1, Max: 40, Step: 1,
		Label: t("Icons per row (reserve mode)", "Icônes par ligne (mode réserve)")},
	{Key: "gaugeSize", Kind: "number", Group: "mega", Advanced: true, Default: d("52"), Min: 20, Max: 150, Step: 1,
		Label: t("Size of the % badge (both modes)", "Taille du badge % (les deux modes)")},
	{Key: "iconsAlign", Kind: "choice", Group: "mega", Advanced: true, Default: d("auto"),
		Label: t("Icons beside the % badge", "Icônes à côté du badge %"),
		Choices: []Choice{
			{"auto", t("Automatic: top in 100% mode, centred in reserve mode", "Automatique : en haut en mode 100 %, centrées en mode réserve")},
			{"top", t("Top", "En haut")}, {"center", t("Centred", "Centrées")}, {"bottom", t("Bottom", "En bas")},
		}},
	{Key: "nameSize", Kind: "number", Group: "mega", Advanced: true, Default: d("6"), Min: 4, Max: 20, Step: 1,
		Label: t("Reserve name size", "Taille du nom de réserve")},
	{Key: "classSize", Kind: "number", Group: "mega", Advanced: true, Default: d("9"), Min: 5, Max: 24, Step: 1,
		Label: t("Class number size", "Taille des numéros de classe")},
	{Key: "avant", Kind: "bool", Group: "mega", Advanced: true, Default: d("1"),
		Label:   t("100% mode: highlight the species of the reserve you play", "Mode 100 % : mettre en avant les espèces de la réserve jouée"),
		Choices: yesNo},
	{Key: "bg", Kind: "bool", Group: "mega", Advanced: true, Default: d("0"),
		Label:   t("Dark background", "Fond sombre"),
		Choices: yesNo},
	{Key: "bgAlpha", Kind: "range", Group: "mega", Advanced: true, Default: d("0.35"), Min: 0.05, Max: 1, Step: 0.05,
		Label: t("Background opacity", "Opacité du fond")},
	{Key: "total", Kind: "bool", Group: "mega", Advanced: true, Default: d("0"),
		Label:   t("Reserve mode: overall progress under the gauge (every reserve)", "Mode réserve : progression globale sous la jauge (toutes les réserves)"),
		Choices: yesNo},
	{Key: "iconSize", Kind: "number", Group: "reserve", Advanced: true, Default: d("28"), Min: 16, Max: 100, Step: 1,
		Label: t("Size", "Taille")},

	{Key: "label", Kind: "text", Group: "texts", Advanced: true, Default: t("HARVESTED", "DÉJÀ OBTENU"),
		Label: t("\"Harvested\" line", "Ligne « déjà obtenu »")},
	{Key: "tiers", Kind: "text", Group: "texts", Advanced: true, Default: t("DIAMOND,GOLD,SILVER,BRONZE", "DIAMANT,OR,ARGENT,BRONZE"),
		Label: t("Medal names (diamond to bronze)", "Noms des médailles (diamant à bronze)")},
	{Key: "none", Kind: "text", Group: "texts", Advanced: true, Default: t("None", "aucun"),
		Label: t("No medal", "Aucune médaille")},
	{Key: "leftText", Kind: "text", Group: "texts", Advanced: true, Default: t("Left", "reste"),
		Label: t("Species left", "Espèces restantes")},
	{Key: "totalText", Kind: "text", Group: "texts", Advanced: true, Default: d("GLOBAL"),
		Label: t("Overall progress", "Progression globale")},
	{Key: "timerSubText", Kind: "text", Group: "texts", Advanced: true, Default: t("since the last species", "depuis la dernière espèce"),
		Label: t("Timer: time since the last species", "Chrono : temps depuis la dernière espèce")},
}

// Groups gives the title of each group.
var Groups = map[string]Text{
	"challenge":  t("Challenge", "Défi"),
	"wall":       t("Overlay", "Overlay"),
	"binoculars": t("Binoculars and calls", "Jumelles et cris"),
	"timer":      t("Challenge timer", "Chrono du challenge"),
	"position":   t("Overlay position (HUD is 1280 x 720)", "Position de l'overlay (HUD de 1280 x 720)"),
	"mega":       t("Overlay: icons and gauge", "Overlay : icônes et jauge"),
	"reserve":    t("List with names", "Liste avec les noms"),
	"texts":      t("Texts", "Textes"),
}

// Defaults returns the default values in a language. The challenge starts today by default.
func Defaults(lang string) map[string]string {
	m := map[string]string{}
	for _, f := range Fields {
		m[f.Key] = f.Default.Get(lang)
	}
	m["since"] = time.Now().Format("2006-01-02")
	return m
}

// Ensure writes the default settings when there is no settings file yet, so that the challenge
// start is the day of the installation. It tells whether it wrote them (first installation).
func Ensure(dropzone, lang string) (bool, error) {
	if _, err := os.Stat(filepath.Join(dropzone, filepath.FromSlash(Path))); err == nil {
		return false, nil
	}
	return true, Write(dropzone, lang, Defaults(lang))
}

// Read returns the values of the file over the defaults. Unknown keys are kept.
func Read(dropzone, lang string) map[string]string {
	m := Defaults(lang)
	b, err := os.ReadFile(filepath.Join(dropzone, filepath.FromSlash(Path)))
	if err != nil {
		return m
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			m[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return m
}

// Write writes the file, with a comment above each setting.
func Write(dropzone, lang string, values map[string]string) error {
	var b strings.Builder
	b.WriteString("# COTWGoldChallenge - " + t("Settings, read again by the game every few seconds. Change them with COTWGoldChallenge.exe.",
		"réglages, relus par le jeu toutes les quelques secondes. Modifiez-les avec COTWGoldChallenge.exe.").Get(lang) + "\n")
	known := map[string]bool{}
	for _, f := range Fields {
		known[f.Key] = true
		v, ok := values[f.Key]
		if !ok {
			v = f.Default.Get(lang)
		}
		sep := " : "
		if lang != "fr" {
			sep = ": "
		}
		title := Groups[f.Group].Get(lang) + sep + f.Label.Get(lang)
		if Groups[f.Group].Get(lang) == f.Label.Get(lang) {
			title = f.Label.Get(lang)
		}
		b.WriteString("# " + title + "\n")
		b.WriteString(f.Key + "=" + clean(v) + "\n")
	}
	for k, v := range values {
		if !known[k] && k != "" && !strings.ContainsAny(k, "=\n#") {
			b.WriteString(k + "=" + clean(v) + "\n")
		}
	}
	p := filepath.Join(dropzone, filepath.FromSlash(Path))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return safefile.Write(p, []byte(b.String()))
}

func clean(v string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(v))
}

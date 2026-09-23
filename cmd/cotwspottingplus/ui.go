package main

import "strings"

// UI holds the program messages in one language.
type UI struct{ m map[string]string }

func (u UI) t(k string) string {
	if s, ok := u.m[k]; ok {
		return s
	}
	return messages["en"][k]
}

func uiFor(steamLanguage string) UI {
	if strings.EqualFold(steamLanguage, "french") {
		return UI{messages["fr"]}
	}
	return UI{messages["en"]}
}

var messages = map[string]map[string]string{
	"en": {
		"credit":        "designed by Laink",
		"not_found":     "theHunter: Call of the Wild was not found.",
		"ask_folder":    "Game folder (theHunterCotW), or Enter to quit: ",
		"not_game":      "This folder does not contain the game.",
		"game":          "Game:    ",
		"language":      "Language:",
		"preamble":      "This mod is meant for profiles that have acquired the \"Sight Spotting\" skill, level 3 (Tracker tab), a native game feature that gives an approximate trophy estimate through binoculars. You can install the mod without this skill, but this advantage will distort the gaming experience designed by the COTW developers.",
		"menu":          "  1  Install / update\n  2  Uninstall\n     Enter to quit",
		"choice":        "\nType your choice (\"1\" or \"2\"): ",
		"confirm":       "Continue? (y/n) ",
		"no_hud":        "ui/clue_hud.gfx was not found in the game archives.",
		"reading":       "Reading the trophy data of the game...",
		"step_types":    "species",
		"step_antlers":  "antlers and horns",
		"species":       "%d species.",
		"patch_failed":  "this game version is not supported",
		"other_mod":     "Another mod already replaces the binoculars panel. It will be kept as clue_hud.gfx.bak.",
		"installed":     "Installed.",
		"launch_ok":     "The Steam launch options already load mods.",
		"restart":       "Restart the game to see the gauge.",
		"launch_todo":   "In Steam: right-click the game > Properties > Launch options, and paste:",
		"copied":        "(copied to the clipboard)",
		"not_installed": "COTWSpottingPlus is not installed.",
		"removed":       "Removed.",
		"error":         "Error:",
		"denied":        "access denied to the game folder. Run COTWSpottingPlus as administrator.",
		"press_enter":   "Press Enter to close.",
	},
	"fr": {
		"credit":        "conçu par Laink",
		"not_found":     "theHunter: Call of the Wild est introuvable.",
		"ask_folder":    "Dossier du jeu (theHunterCotW), ou Entrée pour quitter : ",
		"not_game":      "Ce dossier ne contient pas le jeu.",
		"game":          "Jeu :    ",
		"language":      "Langue :",
		"preamble":      "Ce mod est prévu pour les profils ayant acquis la compétence \"Connaissance du repérage\" niveau 3 (onglet Traqueur), une fonction native du jeu qui permet d'avoir une évaluation approximative des trophées aux jumelles. Vous pouvez installer le mod sans avoir cette compétence mais cet avantage dénaturera l'expérience de jeu proposée par les développeurs de COTW.",
		"menu":          "  1  Installer / mettre à jour\n  2  Désinstaller\n     Entrée pour quitter",
		"choice":        "\nTapez votre choix (\"1\" ou \"2\") : ",
		"confirm":       "Continuer ? (o/n) ",
		"no_hud":        "ui/clue_hud.gfx est introuvable dans les archives du jeu.",
		"reading":       "Lecture des données de trophées du jeu...",
		"step_types":    "espèces",
		"step_antlers":  "bois et cornes",
		"species":       "%d espèces.",
		"patch_failed":  "cette version du jeu n'est pas prise en charge",
		"other_mod":     "Un autre mod remplace déjà le panneau des jumelles. Il sera conservé sous clue_hud.gfx.bak.",
		"installed":     "Installé.",
		"launch_ok":     "Les options de lancement Steam chargent déjà les mods.",
		"restart":       "Relancez le jeu pour voir la jauge.",
		"launch_todo":   "Dans Steam : clic droit sur le jeu > Propriétés > Options de lancement, et collez :",
		"copied":        "(copié dans le presse-papiers)",
		"not_installed": "COTWSpottingPlus n'est pas installé.",
		"removed":       "Désinstallé.",
		"error":         "Erreur :",
		"denied":        "accès refusé au dossier du jeu. Lancez COTWSpottingPlus en tant qu'administrateur.",
		"press_enter":   "Appuyez sur Entrée pour fermer.",
	},
}

// wrap breaks a paragraph into lines of at most n characters.
func wrap(s string, n int) string {
	var b strings.Builder
	line := 0
	for i, w := range strings.Fields(s) {
		l := len([]rune(w))
		if i > 0 {
			if line+1+l > n {
				b.WriteString("\n")
				line = 0
			} else {
				b.WriteString(" ")
				line++
			}
		}
		b.WriteString(w)
		line += l
	}
	return b.String()
}

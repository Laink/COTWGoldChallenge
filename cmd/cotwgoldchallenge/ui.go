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
		"credit":            "designed by Laink",
		"not_found":         "theHunter: Call of the Wild was not found.",
		"ask_folder":        "Game folder (theHunterCotW), or Enter to quit: ",
		"not_game":          "This folder does not contain the game.",
		"game":              "Game:    ",
		"language":          "Language:",
		"preamble":          "The medal gauge needs the \"Spotting Knowledge\" skill, level 3: the mod reads the trophy estimate this skill gives through binoculars. Without it, the gauge shows the medal thresholds of the species but not where the animal stands.",
		"menu":              "  1  Install / update\n  2  Uninstall\n  3  Settings (page in your browser)",
		"menu_key_off":      "  4  Overlay key: OFF (type 4 to turn it on)",
		"menu_key_on":       "  4  Overlay key: ON, %s (type 4 to turn it off)\n     Keep this window open while you play.",
		"menu_key_nokey":    "  4  Overlay key: ON, but no key is chosen, set it in 3 (type 4 to turn it off)",
		"menu_quit":         "     Enter to quit",
		"menu_quit_on":      "     Enter to quit (turns the overlay key off)",
		"choice":            "\nType your choice (1 to 4): ",
		"wait_enter":        "Press Enter here to stop.",
		"date_format":       "2006-01-02",
		"state_none":        "The mod is not installed: choose 1.",
		"state_installed":   "The mod is installed (%s, version %s).",
		"state_program":     "This program is a different version: choose 1 to update the mod.",
		"state_game":        "The game was updated since: choose 1 to install the mod again.",
		"state_legacy":      "The former version (COTWSpottingPlus) is installed: choose 1 to switch to COTWGoldChallenge.",
		"key_on":            "Overlay key on: %s shows or hides the overlay in the game. Keep this window open while you play.",
		"key_nokey":         "Overlay key on, but no key is chosen: choose it in the settings (3).",
		"key_off":           "Overlay key off.",
		"shortcut_windows":  "the shortcut only works on Windows",
		"confirm":           "Continue? (y/n) ",
		"no_hud":            "ui/clue_hud.gfx was not found in the game archives.",
		"reading":           "Reading the trophy data of the game...",
		"step_types":        "species",
		"step_antlers":      "antlers and horns",
		"species":           "%d species.",
		"patch_failed":      "this game version is not supported",
		"other_mod":         "Another mod already replaces %s. Its files will be kept as .bak and come back if you uninstall.",
		"no_saves":          "Your save folder was not found: play the game once, then install again, for the overlay to see your harvests.",
		"no_link":           "The link to your save folder could not be made:",
		"installed":         "Installed.",
		"launch_ok":         "The Steam launch options already load mods.",
		"restart":           "Restart the game to see the mod.",
		"first_run":         "First installation: the settings (choice 3) and the overlay key (choice 4) are in the menu below, and every time you run COTWGoldChallenge.",
		"launch_todo":       "In Steam: right-click the game > Properties > Launch options, and paste:",
		"copied":            "(copied to the clipboard)",
		"not_installed":     "COTWGoldChallenge is not installed.",
		"removed":           "Removed.",
		"error":             "Error:",
		"denied":            "access denied to the game folder. Run COTWGoldChallenge as administrator.",
		"settings_open":     "The settings page is open in your browser:",
		"page_title":        "Settings",
		"page_intro":        "Saved settings show in the game within a few seconds, even while you play. The binocular gauge needs the Spotting Knowledge skill, level 3.",
		"page_save":         "Save",
		"page_saved":        "Saved. The game applies it within a few seconds.",
		"page_reset":        "Default values",
		"page_close":        "Close",
		"page_closed":       "You can close this tab.",
		"page_advanced":     "Advanced settings",
		"page_always":       "All time",
		"page_preview":      "Preview",
		"page_preview_note": "Approximate layout. In the game, each hexagon is the species icon.",
		"page_unsaved":      "Unsaved changes",
		"page_time":         "time (optional)",
		"page_zoom":         "Overlay",
		"page_panels":       "Binoculars and call panels",
		"page_zone_note":    "The overlay is moved out of the top right corner, where the game shows its panels.",
		"page_key_set":      "Change",
		"page_key_clear":    "None",
		"page_key_wait":     "Press a key or a mouse button (Esc to cancel)",
		"page_key_none":     "No key",
		"page_mouse":        "Mouse",
		"page_list_note":    "The list only shows in reserve mode, or in 100% mode when \"show every species\" is off.",
		"page_full":         "Screen",
		"press_enter":       "Press Enter to close.",
	},
	"fr": {
		"credit":            "conçu par Laink",
		"not_found":         "theHunter: Call of the Wild est introuvable.",
		"ask_folder":        "Dossier du jeu (theHunterCotW), ou Entrée pour quitter : ",
		"not_game":          "Ce dossier ne contient pas le jeu.",
		"game":              "Jeu :    ",
		"language":          "Langue :",
		"preamble":          "La jauge de médaille nécessite la compétence \"Connaissance du repérage\" niveau 3 : le mod lit l'estimation de trophée que donne cette compétence aux jumelles. Sans elle, la jauge affiche les seuils de médaille de l'espèce, mais pas où se situe l'animal.",
		"menu":              "  1  Installer / mettre à jour\n  2  Désinstaller\n  3  Réglages (page dans votre navigateur)",
		"menu_key_off":      "  4  Touche de l'overlay : DÉSACTIVÉE (tapez 4 pour l'activer)",
		"menu_key_on":       "  4  Touche de l'overlay : ACTIVÉE, %s (tapez 4 pour la désactiver)\n     Laissez cette fenêtre ouverte pendant que vous jouez.",
		"menu_key_nokey":    "  4  Touche de l'overlay : ACTIVÉE, mais aucune touche choisie, réglez-la en 3 (tapez 4 pour la désactiver)",
		"menu_quit":         "     Entrée pour quitter",
		"menu_quit_on":      "     Entrée pour quitter (désactive la touche de l'overlay)",
		"choice":            "\nTapez votre choix (1 à 4) : ",
		"wait_enter":        "Appuyez sur Entrée ici pour arrêter.",
		"date_format":       "02/01/2006",
		"state_none":        "Le mod n'est pas installé : choisissez 1.",
		"state_installed":   "Le mod est installé (le %s, version %s).",
		"state_program":     "Ce programme est une autre version : choisissez 1 pour mettre à jour le mod.",
		"state_game":        "Le jeu a été mis à jour depuis : choisissez 1 pour réinstaller le mod.",
		"state_legacy":      "L'ancienne version (COTWSpottingPlus) est installée : choisissez 1 pour passer à COTWGoldChallenge.",
		"key_on":            "Touche de l'overlay activée : %s affiche ou masque l'overlay en jeu. Laissez cette fenêtre ouverte pendant que vous jouez.",
		"key_nokey":         "Touche de l'overlay activée, mais aucune touche n'est choisie : choisissez-la dans les réglages (3).",
		"key_off":           "Touche de l'overlay désactivée.",
		"shortcut_windows":  "le raccourci ne fonctionne que sous Windows",
		"confirm":           "Continuer ? (o/n) ",
		"no_hud":            "ui/clue_hud.gfx est introuvable dans les archives du jeu.",
		"reading":           "Lecture des données de trophées du jeu...",
		"step_types":        "espèces",
		"step_antlers":      "bois et cornes",
		"species":           "%d espèces.",
		"patch_failed":      "cette version du jeu n'est pas prise en charge",
		"other_mod":         "Un autre mod remplace déjà %s. Ses fichiers seront conservés en .bak et reviendront si vous désinstallez.",
		"no_saves":          "Votre dossier de sauvegarde est introuvable : lancez une partie une fois, puis réinstallez, pour que l'overlay voie vos prises.",
		"no_link":           "Le lien vers votre dossier de sauvegarde n'a pas pu être créé :",
		"installed":         "Installé.",
		"launch_ok":         "Les options de lancement Steam chargent déjà les mods.",
		"restart":           "Relancez le jeu pour voir le mod.",
		"first_run":         "Première installation : les réglages (choix 3) et la touche de l'overlay (choix 4) sont dans le menu ci-dessous, et à chaque lancement de COTWGoldChallenge.",
		"launch_todo":       "Dans Steam : clic droit sur le jeu > Propriétés > Options de lancement, et collez :",
		"copied":            "(copié dans le presse-papiers)",
		"not_installed":     "COTWGoldChallenge n'est pas installé.",
		"removed":           "Désinstallé.",
		"error":             "Erreur :",
		"denied":            "accès refusé au dossier du jeu. Lancez COTWGoldChallenge en tant qu'administrateur.",
		"settings_open":     "La page des réglages est ouverte dans votre navigateur :",
		"page_title":        "Réglages",
		"page_intro":        "Les réglages enregistrés s'appliquent en jeu en quelques secondes, même en pleine partie. La jauge des jumelles nécessite la compétence « Connaissance du repérage » niveau 3.",
		"page_save":         "Enregistrer",
		"page_saved":        "Enregistré. Le jeu l'applique en quelques secondes.",
		"page_reset":        "Valeurs par défaut",
		"page_close":        "Fermer",
		"page_closed":       "Vous pouvez fermer cet onglet.",
		"page_advanced":     "Réglages avancés",
		"page_always":       "Depuis toujours",
		"page_preview":      "Aperçu",
		"page_preview_note": "Disposition approximative. Dans le jeu, chaque hexagone est l'icône de l'espèce.",
		"page_unsaved":      "Modifications non enregistrées",
		"page_time":         "heure (facultatif)",
		"page_zoom":         "Overlay",
		"page_panels":       "Cadres jumelles et cris",
		"page_zone_note":    "L'overlay est décalé hors du coin en haut à droite, réservé aux cadres du jeu.",
		"page_key_set":      "Changer",
		"page_key_clear":    "Aucune",
		"page_key_wait":     "Appuyez sur une touche ou un bouton de souris (Échap pour annuler)",
		"page_key_none":     "Aucune touche",
		"page_mouse":        "Souris",
		"page_list_note":    "La liste ne s'affiche qu'en mode réserve, ou en mode 100% si « afficher toutes les espèces » est désactivé.",
		"page_full":         "Écran",
		"press_enter":       "Appuyez sur Entrée pour fermer.",
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

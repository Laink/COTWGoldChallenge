# COTWGoldChallenge

A mod for **theHunter: Call of the Wild**, designed by **Laink**. Formerly COTWSpottingPlus.

It follows a medal challenge in the game, gold by default: which species you still have to
harvest, and whether the animal in your binoculars can make it.

## What it adds

- **Challenge overlay** in the HUD. It shows the species of the reserve you play (reserve mode)
  or every species of the game (100% mode), each in the colour of the best medal you harvested
  since the start of the challenge, with a completion badge. It updates as you harvest, and a
  short animation plays on each better medal. The species you aim at with the binoculars, or
  hear, is highlighted. A white star marks the species of which you harvested a rare fur
  (albino, melanistic…: rare or very rare for that species, as the game rates them).
- **Medal gauge** under the binoculars panel: gold guaranteed, gold possible with its chance,
  gold impossible, and a bronze-to-diamond scale with the animal's trophy range.
- **Reserve selection**: the challenge progress of each reserve, in the main menu and in the
  change reserve screen.
- **Settings page** in your browser, applied in the game within a few seconds.
- **Overlay keys**: one shows or hides the overlay (F8); held, one lists the missing species of
  the reserve with their names (F9), another shows the species outside the reserve at full
  opacity in 100% mode (F10).

**The gauge needs the Spotting Knowledge skill, level 3.** The mod reads the trophy estimate this
skill gives through binoculars. Without it, the gauge shows the medal thresholds of the species
but not where the animal stands, with the line "Skill missing: Spotting Knowledge 3".

## Install

1. Download `COTWGoldChallenge.exe` from the [Releases](../../releases) page.
2. Run it and choose **1**. It finds the game, reads the trophy data from your game files and
   installs the mod in the game's `dropzone` folder.
3. Add these launch options. In Steam: right-click the game > Properties > Launch options. In the
   Epic Games Launcher: Library > "..." on the game > Manage > Launch options.

   ```
   --vfs-fs dropzone --vfs-archive archives_win64 --vfs-fs .
   ```

   Copy the whole line, with the final dot. COTWGoldChallenge tells you if this is already set.
4. Start the game.

When you run it, COTWGoldChallenge tells you whether the mod is installed, and whether it must be
installed again: after a game update, or with a new version of the program. Choose **2** to
uninstall; your settings are kept.

Steam and Epic Games Store versions. The Microsoft Store / Game Pass version cannot load mods.

## Use

**Keep the COTWGoldChallenge window open while you play** (you can minimize it). The game keeps
only the 5 best harvests of each species and the last 20: while it runs, COTWGoldChallenge
records every harvest, so that older trophies do not disappear from the overlay. It also runs the
overlay keys, which work while the game window is in front; the game still receives them.

The menu stays open after each choice:

- **3 Settings and trophies** opens the settings page: challenge start (a date, or all time), mode (reserve or
  100%), medal to get, whether better medals count, the look and position of the overlay, and the
  overlay keys. Your settings can be made before installing. Changes are saved as you make them.
  - The **Trophies** tab shows the progress of each reserve, the missing species with the
    reserves where they live, and, reserve by reserve, the medals found in your harvests. It lets
    you add by hand the trophies the game no longer keeps.
  - The **Stats** tab sums up the harvests recorded: medals, rare furs, reserves, most
    harvested species and last harvests.
- **4 Overlay keys** turns the keys off or on again. They start with the program.
- **5 Launch the game**, through Steam or the Epic Games Launcher, with your launch options.
- **6 Desktop shortcut**: opens COTWGoldChallenge and launches the game together, so that the
  program is never forgotten.
- **Enter** quits.

By default the challenge starts on the day of the installation, in reserve mode, for gold, and
diamonds count too. Great Ones count as diamonds.

## How it works

- Medal thresholds are computed from the game files, as the game does. TruRACS species use their
  antler and horn tables.
- The chances of the gauge follow the game's own estimation rules: the score range and the weight
  shown by the skill, and how scores are spread in each species.
- The overlay reads your hunting log in your save folder, through a link made in `dropzone`. It
  never writes to your saves. The program also reads your trophy lodges (the trophies on display,
  and the harvests kept to be mounted), which the game never clears. The harvests recorded by the
  program and the trophies added by hand are in `dropzone/ui/cotwgc_history.txt`, kept when you
  uninstall.
- The movies of the game that the mod changes (`ui/clue_hud.gfx`, `ui/hud.gfx`,
  `ui/main_menu.gfx`, `ui/change_reserve.gfx`) are read from your game and patched on your
  computer. No game file is distributed. The files of another mod that replaces them are kept as
  `.bak` and come back when you uninstall.
- Game texts follow the game language (set in Steam, or the Epic Games Launcher language): English, French, German, Spanish, Italian,
  Polish, Russian, Portuguese, Czech, Japanese, Chinese and Korean. Other languages use English.
  The program and the settings page are in English or French.

## Command line

```
COTWGoldChallenge.exe [-game folder] [-lang fr] [-uninstall] [-settings] [-shortcut] [-play] [-yes]
```

## Build

The ActionScript class of the mod (`as3/`) is compiled into `internal/modclass/mod.swc` with
Apache Royale and `playerglobal.swc` 10.2:

```
ROYALE=... PLAYERGLOBAL=... ./build_as3.sh
```

Then, with Go 1.24 or later:

```
./build.sh
```

## Credits

The settings page uses the Barlow fonts, under the SIL Open Font License
(`cmd/cotwgoldchallenge/web/fonts/OFL.txt`).

COTWGoldChallenge is not affiliated with Expansive Worlds or Avalanche Studios Group.

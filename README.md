# COTWGoldChallenge

A mod for **theHunter: Call of the Wild**, designed by **Laink**. Formerly COTWSpottingPlus.

It follows a medal challenge in the game, gold by default: which species you still have to
harvest, and whether the animal in your binoculars can make it.

## What it adds

- **Challenge overlay** in the HUD. It shows the species of the reserve you play (reserve mode)
  or every species of the game (100% mode), each in the colour of the best medal you harvested
  since the start of the challenge, with a completion badge. It updates as you harvest, and a
  short animation plays on each better medal. The species you aim at with the binoculars, or
  hear, is highlighted.
- **Medal gauge** under the binoculars panel: gold guaranteed, gold possible with its chance,
  gold impossible, and a bronze-to-diamond scale with the animal's trophy range.
- **Reserve selection**: the challenge progress of each reserve, in the main menu and in the
  change reserve screen.
- **Settings page** in your browser, applied in the game within a few seconds.
- **Overlay key** to show or hide the overlay while you play.

**The gauge needs the Spotting Knowledge skill, level 3.** The mod reads the trophy estimate this
skill gives through binoculars. Without it, the gauge shows the medal thresholds of the species
but not where the animal stands, with the line "Skill missing: Spotting Knowledge 3".

## Install

1. Download `COTWGoldChallenge.exe` from the [Releases](../../releases) page.
2. Run it and choose **1**. It finds the game, reads the trophy data from your game files and
   installs the mod in the game's `dropzone` folder.
3. In Steam, right-click the game > Properties > Launch options, and paste:

   ```
   --vfs-fs dropzone --vfs-archive archives_win64 --vfs-fs .
   ```

   COTWGoldChallenge tells you if this is already set.
4. Start the game.

When you run it, COTWGoldChallenge tells you whether the mod is installed, and whether it must be
installed again: after a game update, or with a new version of the program. Choose **2** to
uninstall; your settings are kept.

Steam version only (the `dropzone` folder is not supported by other versions).

## Use

The menu stays open after each choice:

- **3 Settings** opens the settings page: challenge start (a date, or all time), mode (reserve or
  100%), medal to get, whether better medals count, the look and position of the overlay, and the
  overlay key. Your settings can be made before installing.
- **4 Overlay key** turns the key on or off. Keep the window open while you play: the key works
  while the game window is in front, and the game still receives it.
- **Enter** quits.

By default the challenge starts on the day of the installation, in reserve mode, for gold, and
diamonds count too. Great Ones count as diamonds.

## How it works

- Medal thresholds are computed from the game files, as the game does. TruRACS species use their
  antler and horn tables.
- The chances of the gauge follow the game's own estimation rules: the score range and the weight
  shown by the skill, and how scores are spread in each species.
- The overlay reads your hunting log in your save folder, through a link made in `dropzone`. It
  never writes to your saves.
- The movies of the game that the mod changes (`ui/clue_hud.gfx`, `ui/hud.gfx`,
  `ui/main_menu.gfx`, `ui/change_reserve.gfx`) are read from your game and patched on your
  computer. No game file is distributed. The files of another mod that replaces them are kept as
  `.bak` and come back when you uninstall.
- Game texts follow the game language set in Steam: English, French, German, Spanish, Italian,
  Polish, Russian, Portuguese, Czech, Japanese, Chinese and Korean. Other languages use English.
  The program and the settings page are in English or French.

## Command line

```
COTWGoldChallenge.exe [-game folder] [-lang fr] [-uninstall] [-settings] [-shortcut] [-yes]
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

# COTWSpottingPlus

A mod for **theHunter: Call of the Wild**, designed by **Laink**.

When you look at an animal through binoculars, COTWSpottingPlus adds a panel under the animal
information showing its medal potential: gold guaranteed, gold possible with its chance,
gold impossible, and a bronze-to-diamond scale with the animal's trophy range.

This mod is meant for profiles that have acquired the **Sight Spotting** skill, level 3 (Tracker tab),
a native game feature that gives an approximate trophy estimate through binoculars.
You can install the mod without this skill, but this advantage will distort the gaming experience
designed by the COTW developers.

## Install

1. Download `COTWSpottingPlus.exe` from the [Releases](../../releases) page.
2. Run it and choose **1**. It finds the game, reads the trophy data from your game files and
   installs the mod in the game's `dropzone` folder.
3. In Steam, right-click the game > Properties > Launch options, and paste:

   ```
   --vfs-fs dropzone --vfs-archive archives_win64 --vfs-fs .
   ```

   COTWSpottingPlus tells you if this is already set.
4. Start the game.

Run COTWSpottingPlus again after a game update, or after changing the game language.
Choose **2** to uninstall.

Steam version only (the `dropzone` folder is not supported by other versions).

## How it works

- Medal thresholds are computed from the game files, as the game does:
  silver, gold and diamond start at 20, 60 and 90 % of the species' trophy range.
  TruRACS species use their antler and horn tables.
- The percentage assumes the score is equally likely anywhere in the displayed range.
- The binoculars panel (`ui/clue_hud.gfx`) is read from your game and patched on your
  computer. No game file is distributed.
- Texts follow the game language set in Steam: English, French, German, Spanish, Italian,
  Polish, Russian, Portuguese, Czech, Japanese, Chinese and Korean. Other languages use English.

## Command line

```
COTWSpottingPlus.exe [-game folder] [-lang fr] [-uninstall] [-yes]
```

## Build

Go 1.22 or later:

```
./build.sh
```

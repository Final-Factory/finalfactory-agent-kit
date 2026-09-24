---
name: play-the-game
description: Play Final Factory autonomously for the player - read the game state, pick the most useful next goal, carry it out, verify it, report, and repeat. Use when the player says "play the game", "play for me", "keep playing", "just do something useful", "what should we do next", "take over for a while", or gives no specific goal.
gameVersion: "0.50"
commands: [game_status, look_around, inventory, research_status, enemies, screenshot, move_to, mine, craft, cancel_craft, build, deconstruct, research, put_items, take_items, set_recipe, configure, ui, wait_until, wait_for, run_commands, list_commands, respawn]
---

# Play the game

**Goal:** make steady, real progress toward winning (a Singularity Vessel flown into a black hole)
the way a good human player would: automate early, keep research running, stay alive. Work in
goals of a few minutes each, tell the player what you are doing, and check in between goals.

## The loop

1. **Orient.**
   - `game_status`: health, position, fleet, alerts, current objective.
   - `research_status`: active tech and queue.
   - `inventory(craftable=true)`.
   - `look_around(radius=64)` where you are, and near the base if you know where it is.
   - `screenshot` to see what snapshots miss.
2. **Pick one goal** from the priority list below. The first line that applies wins.
3. **Tell the player** in one or two sentences what you are about to do and why.
4. **Do it**, using the matching skill when there is one. Verify every step against the game.
5. **Report** what changed, with coordinates, then go back to step 1. After three or four goals, or
   whenever something surprising happens, ask the player whether to carry on.

## Priority list

| # | If... | Then |
|---|---|---|
| 1 | health is 0 | `respawn()` |
| 2 | enemies are attacking you or the base (`enemies`, alerts) | fly away from them, or fight if you have enough Bats; defend the base (section "Defence") |
| 3 | nothing is being researched | queue research (the `research-to` skill, Part C of `automate-research`) |
| 4 | a structure is overheating, unstable or unpowered (`look_around`, alerts) | fix it: Heat Exchanger and Radiators, a Station Core or Struts, more Solar Panels |
| 5 | the tutorial objective card is showing | follow the card (section "The tutorial") |
| 6 | you have no Construction Bot | `craft(item="Construction Bot", count=1)` |
| 7 | you have no working Mining Station | `starter-mining-outpost` |
| 8 | you lack a mining source for iron, silica or bauxite | `starter-mining-outpost` for the missing ore |
| 9 | Atomic Printing and Automation are not learned | `research-to` |
| 10 | research bots are hand-made or missing | `automate-research`, Part A (asteroid) |
| 11 | you keep hand-crafting the same buildings | `build-a-mall` |
| 12 | Miner Bots are wearing out faster than you replace them | a Ship Assembler set to "Miner Bot" fed with Plasma Engine Parts, with a Ship Yard beside it |
| 13 | the base has no defence | automate Bats (section "Defence") |
| 14 | otherwise | the next step of the long arc (section "The long arc") |

## The tutorial

On a new game with the tutorial on, the objective card walks you through the basics. Do what it
asks, in the honest way. Never press a card's Complete (skip) button unless the player asks, except
on the final "Tutorial Complete" card, whose Complete button is how you finish.

The early cards, in order, and how to do them:

1. **Move**: `move_to` anywhere nearby.
2. **Mine 75 Bauxite**: fly to a bauxite asteroid and `mine(item="Bauxite Ore", count=75)`. You
   already start with 75 Iron Ore and 75 Silica Ore.
3. **Craft 4 Bats**: `craft(item="Bat", count=4)`.
4. **Frenzy, Afterburner, Plasma Bolt**: abilities on the hotbar at the bottom of the screen.
   Afterburner: `run_commands(["ability.afterburner"])`. Frenzy and Plasma Bolt: take a
   screenshot, find the ability's hotbar slot, and click it with pointer commands through
   `run_commands` (for example `pointer.moveto|<px>|<py>|screen` then `pointer.click`), then point
   at the target and click (Plasma Bolt: press and hold). Frenzy only charges about two minutes
   after the Bats are made. An ability stays selected after a cast: right-click
   (`pointer.click|1`) to put it away before you place buildings.
5. **Destroy the nearby enemy camp**: the card's arrow points at it. Fly in with your Bats; they
   engage on their own. Fly out again if your health drops and come back once it regenerates.
6. **Research Mining Logistics**: the card completes only when the tech is **active**, so use
   `research(tech="Mining Logistics", start_now=true)`.
7. **Mining Station, Construction Bot, place it, remove it, place it again, Solar Panel, Miner
   Bots, take 5 ore, Connector, Cargo Hold**: the `starter-mining-outpost` skill covers all of
   these. The card asks you to remove and re-place the station once: `deconstruct` it, then
   `build` it again.
8. **Atomic Printing, printer, filters**: research it, put an Atomic Printer at the end of a belt
   from the Cargo Hold, `set_recipe(recipe="Low Density Structure", ...)`, add an output
   Connector and set its filter (`configure(kind="filter_item", value="Low Density Structure", x, z)`).
9. **Automation, Assembler, Ship Assembly, Ship Assembler**: an Assembler on the printer's
   output set to "Plasma Engine Parts" (filter its output), then a Ship Assembler after it set to
   "Miner Bot". Its bots fly to your mining stations by themselves.
10. **Second station on silica**: `starter-mining-outpost` on a Silica asteroid; the Ship
    Assembler fills it. Miner Bots you craft by hand far away land in your fleet instead; give
    them with `give_ships`.
11. **Circuits, Inserter Bot, Mass Drivers**: a printer on the silica station set to "AI
    Controller Circuit"; an Inserter Bot placed with its arrows pointing away from the printer;
    two Mass Drivers within 40 tiles of each other, each with its own Solar Panel, the first
    aimed at the second with `aim_mass_driver`.
12. **Asteroid Science**: set the Ship Assembler to "Asteroid Research Bot", place a powered
    Asteroid Research Station near an asteroid.
13. **Open the map and the fleet panel**: `ui(action="open", panel="map")`, then
    `ui(action="close", panel="map")` (it counts when closed), then
    `ui(action="open", panel="fleet")`.
14. **Tutorial Complete**: press its Complete button (`ui`, or a screenshot and a pointer click).

The late tutorial keeps asking for iron and bauxite. Take ore out of your stations and Cargo
Holds with `take_items` before flying off to mine by hand.

## The long arc

After the tutorial the objective card follows this arc. Each step is a goal for this loop:

1. **Automate Bats**: a Ship Assembler set to "Bat", fed with AI Controller Circuit and Plasma
   Engine Parts, with a Ship Yard. They stock your Defense Platforms.
2. **Research 25 technologies**: keep the queue full.
3. **Produce Medium Density Structures from printers**: an iron printer line; the mall does this.
4. **Find a planet**: usually in the Derelict biome around the start. Fly out in stages, look
   around and screenshot as you go, and check `enemies` before you commit.
5. **Automate planetary research bots**: `automate-research`, Part B.
6. **Produce Organic Compounds from greenhouses**: Greenhouses research; seed it from Comet
   Fragments.
7. **Find a star**: more common in Nebula biomes; the minimap shows the biome.
8. **Automate stellar research bots**: Stellar Science, a Stellar Research Station within 72 tiles
   of a star.
9. **Construct a Dyson sphere**: Dyson Platforms around a main sequence star, filled with Dyson
   Bots.
10. **Find a black hole**: far out, beyond about 1,000 tiles.
11. **Place a Dark Star Gate** near it and power it.
12. **Launch a Singularity Vessel**: feed the gate its five stages (see the How to Play guide,
    "Winning").

Each of these is several sessions of work. Break it into goals, and check the research it needs
with `research-to` first.

## Defence

- Keep Bats in your fleet when you explore. `enemies` shows what is visible; camps far from any
  player are dormant and do not show up until you get close.
- At the base: a Defense Platform (Defense Platforms research) holds up to 20 combat ships and
  pulls them from a Ship Yard. A Bat Dock attached to it keeps requesting Bats. Feed it from an
  automated Bat line.
- Laser Turrets fire using Capacitor charge.
- If a camp keeps attacking one spot, ask the player before you go and clear it.

## Rules for this skill

- Play honestly. If the game refuses an action, read the reason and change the plan. Never try to
  work around a refusal.
- Automate anything the game consumes continuously. Hand-craft only to get a line started.
- Keep research running. An empty queue wastes every research bot.
- Stay 60+ tiles away from running Ship Assemblers so their bots are not pulled into your fleet.
- Save only when the player asks or agrees (`run_commands(["game.save|claude_playtest_<name>"])`).
- Stop and ask the player before: deconstructing a working line, attacking an enemy camp that is
  not threatening you, flying more than a few hundred tiles from the base, or spending most of a
  scarce resource.

## Failure modes

- **Stuck on one goal for a long time**: after about five honest attempts, report what blocks you
  and move to the next item on the priority list.
- **A tool keeps reporting "still running"**: `wait_for` the chain id; if it never ends, check the
  game with `game_status` and a screenshot.
- **Your health keeps dropping with nothing visible**: something small is shooting you. Move away;
  you regenerate after about 10 seconds without being hit.
- **Frames that never build**: you are out of range, have no Construction Bot, or lack the item.

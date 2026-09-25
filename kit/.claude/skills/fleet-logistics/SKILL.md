---
name: fleet-logistics
description: Run Final Factory's bot and ship logistics - Ship Yards as buffers, Ship Assembler lines for Miner Bots, research bots and combat ships, moving ships between your fleet and structures, and keeping yards from clogging. Use when the player says "my mining stations have no miners", "bots are missing", "the ship yard is full", "set up a ship yard", "move my bats to the platform", "automate miner bots", "why is my ship assembler idle", or when stations sit with empty slots while a Ship Assembler waits.
gameVersion: "0.50"
commands: [game_status, look_around, inspect, screenshot, inventory, research_status, research, move_to, craft, build, deconstruct, set_recipe, configure, give_ships, take_ships, aim_mass_driver, ui, wait_until, run_commands]
---

# Fleet logistics

**Goal:** every Mining Station, research station and Defense Platform keeps its slots full from
Ship Assembler lines, each line has room to put what it makes, and no Ship Yard sits full of ships
nobody needs.

## How ships move (base values)

- A **Ship Assembler** sends each finished ship to the **closest** place that takes it: a building
  with a free slot for that type (Mining Station for Miner Bots, research station for its research
  bot, Defense Platform for Bats, Knights and Krillos), a Ship Yard, or a player within about 60
  tiles whose fleet wants that type.
- A **Ship Yard** (Logistics Management research; 16 Medium Density Structure; 7x12; draws 1) holds
  up to **60** ships of any kind. Buildings with an empty slot pull from the closest yard. A yard
  joins a grid only at three attach points on each long side (for a yard facing up: rows 1, 5
  and 9 from its bottom edge). Link one of them to a standard building with a Strut or Connector;
  a Solar Panel placed against the yard does not join it.
- Stations hold 4 bots (Mining Stations, Advanced Mining Stations and research stations); a
  Defense Platform holds 20.
- Bots wear out: a Miner Bot after hauling 125 ore, an Advanced Miner Bot after 600, a research
  bot after 10 scans. Replacements must keep coming.

## The three rules

1. **Give every line its own Ship Yard** right next to its Ship Assembler. A line with no yard of
   its own uses the closest one, and a busy line (Bats especially) fills it. In play a Bat line
   filled the base yard with 60 Bats, the Miner Bot line next to it had nowhere to put miners, and
   every Mining Station starved.
2. **Stay more than 60 tiles away from running Ship Assemblers.** Closer, you count as a
   destination and their ships go into your fleet. The same happens when you hand-craft a ship near
   a Ship Yard or Defense Platform: it can fly there instead.
3. **Do not run a line whose output nobody needs.** A Miner Bot line with every station full
   fills its yard with spares. Switch its recipe to something useful, and switch it back when
   miners run low.

## Steps

### 1. Survey

- `look_around(radius=64)` around each line: the Ship Assembler's recipe and whether it is
  crafting, the Ship Yard, and the stations it feeds.
- `game_status`: your own fleet. Stray Miner or research bots in it mean you stood too close to a
  line.
- Open the yard's panel for its contents: from within reach, select it with
  `run_commands(["pointer.moveto|<yard x>|<yard z>", "wait|1", "pointer.click"])` and take a
  `screenshot`. The panel shows "Capacity n/60" and a row per ship type. Close it with
  `ui(action="close", panel="structure")`.

### 2. Clear a clogged yard

- Combat ships in a yard: build or power Defense Platforms nearby; they pull Bats out on their own
  (in play 29 left a yard within about a minute). Or fly within reach and
  `take_ships(ship="Bat", count=..., x=<yard x>, z=<yard z>)`, then give them to a platform with
  `give_ships`.
- Spare Miner Bots: give them to stations that need them
  (`give_ships(ship="Miner Bot", count=..., x=<station x>, z=<station z>)`), and pause the miner
  line by switching its recipe.
- `take_ships` works on yards and platforms, not on a Ship Assembler. A waiting ship in an
  assembler comes out through its panel's Deploy button.

### 3. Build or fix a bot line

- A Ship Assembler (10 power) with filtered input belts, set with
  `set_recipe(recipe=<ship>, x=..., z=...)`. The recipes:
  - Miner Bot: 5 Plasma Engine Parts. Advanced Miner Bot (High Energy Mining research): 4 Plasma
    Engine Parts + 4 Solid State Laser + 2 AI Controller Circuit.
  - Bat: 4 AI Controller Circuit + 2 Plasma Engine Parts. Knight (Knights research): 2 Plasma
    Engine Parts + 1 Singularity Field Generator. Krillo (Krillos research): 6 Plasma Engine Parts
    + 4 High Energy Laser + 1 Singularity Field Generator. Each takes one fleet slot.
  - Research bots: see `automate-research` and `scale-research`.
- Place a Ship Yard beside it and join the yard to the grid (see above).
- When you change a Ship Assembler's recipe, leftover inputs of the old recipe stay inside and can
  block the new one. In play an assembler sat idle with 12 AI Controller Circuits left over until
  it was deconstructed and rebuilt, which refunds them.
- Verify from 60+ tiles away: a few minutes later the stations it feeds have full slots
  (`screenshot`) and your own fleet did not grow.

### 4. Hand transfers

- `give_ships(ship, count, x, z)` moves ships from your fleet into a structure;
  `take_ships(ship, count, x, z)` takes them out. You must be within about 20 tiles. Check your
  fleet count in `game_status` before and after.

### 5. Moving items between sites (Mass Drivers)

Bots fly over the network, but items need belts or Mass Drivers:

- A Mass Driver fires at another within **40 tiles** (centre to centre). Aim with
  `aim_mass_driver(x, z, target_x, target_z)`. The game refuses a target under fog (`reveal-map`).
- A receiving driver stores up to 164 of **one** item type. A receiver fed two item types fills
  with whichever arrives first and then refuses the other: use one receiver per item.
- Drivers do not relay what they receive. To hop further, run a Connector out of the receiver
  (filtered to the item) into the next sending driver.

## Failure modes

- **Stations empty, assembler idle "nowhere to go"**: its yard is full, or it has none and the
  closest one is full. Clear the yard (step 2).
- **Stations empty, assembler crafting**: its ships go to your fleet (you are within 60 tiles) or to
  a closer station elsewhere. Check your fleet, then `give_ships`.
- **Yard does nothing**: not joined to a grid (unpowered). Check its power in `look_around`.
- **Bats never reach the platforms**: platforms unpowered, or Bats hidden in your fleet.
- **Transfer refused**: out of reach, or the structure does not take that ship type.

Report each line: its recipe, its yard and how full it is, which stations it feeds, and what you
changed.

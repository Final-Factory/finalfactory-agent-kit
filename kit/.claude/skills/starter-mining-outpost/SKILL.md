---
name: starter-mining-outpost
description: Set up a working Mining Station on an asteroid in Final Factory, with solar power, four Miner Bots and an output belt into a Cargo Hold. Use when the player says "set up a mining outpost", "start mining iron/silica/bauxite", "automate mining", "build a miner on that asteroid", or when a production line needs a new ore source.
gameVersion: "0.50"
commands: [game_status, research_status, research, inventory, look_around, enemies, screenshot, move_to, mine, craft, build, deconstruct, take_items, wait_until, run_commands, respawn, give_ships]
---

# Starter mining outpost

**Goal:** a built, powered Mining Station on an asteroid of the ore the player wants, with four
Miner Bots working it, and ore flowing along a Connector into a Cargo Hold you can take from.

If the player did not name an ore, pick **Iron Ore**: Medium Density Structure (2 Iron Ore) is in
nearly every recipe, so iron runs out first.

## Preconditions (check each against the game)

1. `game_status`: a game is running and the player's health is above 0. If it is 0, run
   `respawn()`.
2. `research_status`: "Mining Logistics" is learned (it unlocks the Mining Station and the Miner
   Bot). If not, `research(tech="Mining Logistics", start_now=true)`; on a new game the banked
   points pay for it at once. Confirm with `research_status`.
3. `game_status`: there is a Construction Bot in your fleet. If not, craft one (step 3).
4. Materials. The whole outpost costs, in raw ore, roughly:
   - Mining Station: 24 Iron Ore
   - Solar Panel: 8 Bauxite Ore, 4 Iron Ore
   - 4 Miner Bots: 80 Bauxite Ore
   - Connector: 8 Bauxite Ore, 2 Iron Ore
   - Cargo Hold: 16 Iron Ore
   - Construction Bot, if you need one: 8 Silica Ore, 8 Bauxite Ore

   About 110 Bauxite Ore, 46 Iron Ore and 8 Silica Ore in total. A new game starts with 75 Iron
   Ore and 75 Silica Ore, so bauxite is what you mine.

## Steps

### 1. Find the asteroid

- `look_around(radius=64)` and pick the closest asteroid whose `oreType` is the ore you want and
  whose `oreRemaining` is well above zero. Nothing in range: fly outward in steps of about 50 tiles
  with `move_to` and look again. A screenshot helps you spot asteroids.
- `look_around` reports an asteroid by the corner tile of its 30x30 area. Its centre `(cx, cz)` is
  about 15 tiles further along +x and +z from that corner. Confirm with a screenshot.

### 2. Gather ore

- For each ore you are short of: `move_to(x=cx, z=cz, tolerance=10)` on an asteroid of that ore,
  then `mine(item="Bauxite Ore", count=<total you want to hold>)`. The count is the total you want
  in your inventory, not an amount to add.
- Verify with `inventory`.

### 3. Craft the parts

- `craft(item="Construction Bot", count=1)` if you have none.
- `craft(item="Mining Station", count=1)`, `craft(item="Solar Panel", count=1)`,
  `craft(item="Connector", count=1)`, `craft(item="Cargo Hold", count=1)`.
- Do **not** craft the Miner Bots yet. Crafted away from the station, they go into your fleet.
- Verify: `wait_until(predicate="itemCount", args=["Mining Station", "1"], timeout_seconds=120)`, then
  `inventory` shows all four items. A missing item means you were short of an ingredient: check
  `inventory(craftable=true)`.

### 4. Place the Mining Station

- Fly next to the asteroid: `move_to(x=cx, z=cz - 10)`.
- Place the station hugging the edge of the rock, about 12 tiles from the centre, for example
  south of it: `build(item="Mining Station", x=cx - 1, z=cz - 13, facing="up")`. The station
  must be within 16 tiles of the asteroid's centre and must not overlap the rock.
- If the game refuses, read the reason. "Too far" or "no asteroid": move 2-3 tiles closer to the
  centre. "Overlap" or "blocked": move 2-3 tiles away, or try another side.
- Stay close. `wait_until(predicate="structureBuilt", args=["<x>", "<z>", "Mining Station"], timeout_seconds=60)`
  with the tile you built at. If it times out, check `look_around`: large structures can land a
  tile off the requested spot, and the station may already be built there.
- `look_around(radius=20)` and read the station's real `tile`. Use it from here on as `(sx, sz)`.
  The station covers x `sx`..`sx+2` and z `sz`..`sz+1`.

### 5. Power it

- Place the Solar Panel so its connection end points **into** the station. For a station at
  `(sx, sz)`, try below its middle column, facing up:
  `build(item="Solar Panel", x=sx + 1, z=sz - 2, facing="up")`.
- Wait for it to be built, then `look_around(radius=20)`. The station's `power.satisfaction` should
  now be above 0 (1 means fully powered), and the panel and station should report the same grid
  figures.
- If the panel reports its own separate grid (satisfaction 1, drawNeeded 0) or the station still
  shows no power, it did not connect. `deconstruct(x=<panel x>, z=<panel z>)` and try another
  column or side. A screenshot taken while holding the panel shows the station's attach slots as
  green diamonds. `rotate.abs` through `run_commands` turns a placed panel (0 up, 1 right, 2 down,
  3 left).

### 6. Add the Miner Bots

- Stand within 20 tiles of the station and `craft(item="Miner Bot", count=4)`. Each finished bot
  flies to a free slot on the nearest station that takes Miner Bots.
- Check `game_status` for your fleet. Any Miner Bots listed there did not reach the station. Give
  them to it: `give_ships(ship="Miner Bot", count=4, x=<sx>, z=<sz>)`.
- Verify: a screenshot shows bots flying between the station and the asteroid. After about a
  minute, `take_items(item=<ore>, count=5, x=sx, z=sz)` should add 5 ore to your inventory.

### 7. Add the output belt and Cargo Hold

- Put the Connector against one side of the station, pointing away from it. On the east side:
  `build(item="Connector", x=sx + 3, z=sz, facing="right")`.
- Put the Cargo Hold at the Connector's far end:
  `build(item="Cargo Hold", x=sx + 4, z=sz)`.
- A Mining Station only holds its own ore, so this belt needs no filter.
- Verify after a minute or two: `take_items(item=<ore>, count=5, x=sx + 4, z=sz)` succeeds. If the
  hold stays empty, the belt is not on one of the station's slots: deconstruct the Connector and
  try the other row, or another side.

### 8. Report

Tell the player where the outpost is (station tile, ore type), that it is powered, how many bots
are working, and where the Cargo Hold is.

## Failure modes

- **Build refused as out of reach or too far**: fly closer and retry. Never try to place from far
  away.
- **Station stays a frame**: you flew off, you have no Construction Bot, or the Mining Station item
  is not in your inventory.
- **Station has no asteroid**: it was placed before the asteroid was in range or too far from it.
  A station never re-targets. Deconstruct it (the item comes back) and place it closer.
- **Power satisfaction 0**: the panel is not connected. See step 5.
- **Belt carries nothing**: wrong slot, or the Cargo Hold is not at the belt's end.
- **Ore stops after a long time**: miners wear out by design. Replace them, and plan a Ship
  Assembler that makes Miner Bots (see the `automate-research` and `build-a-mall` skills for the
  line pattern).
- **Enemies nearby** (`enemies`): do not build next to an active camp. Pick another asteroid or
  clear the camp first with Bats.

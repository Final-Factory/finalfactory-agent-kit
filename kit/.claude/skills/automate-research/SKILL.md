---
name: automate-research
description: Build automated research in Final Factory - Ship Assembler lines that keep Asteroid Research Stations and Planetary Research Stations stocked with research bots, with no hand-crafting. Use when the player says "automate research", "automate planetary research", "automate asteroid research", "set up research bots", "research is too slow", "get planetary research going", or when research has stalled because bots are hand-made or missing.
gameVersion: "0.50"
commands: [game_status, research_status, research, inventory, look_around, enemies, screenshot, move_to, mine, craft, build, deconstruct, put_items, take_items, set_recipe, configure, wait_until, wait_for, run_commands, give_ships, aim_mass_driver]
---

# Automate research

**Goal:** research points arrive continuously while the player is somewhere else. Research
stations sit next to an asteroid (Asteroid Research) or a planet (Planetary Research). A Ship
Assembler builds the matching research bot from parts that arrive by belt, and each bot flies to a
station on its own. Nobody hand-crafts research bots beyond the first handful used to prove a
station works.

Why it matters: research bots wear out after a fixed number of scans. An asteroid bot earns about
2.5 points over its life and a planetary bot on a Terra World about 7.5. Planetary Science alone
needs about 1,300 Asteroid Research earned (base), which is over 500 asteroid bots. Only a line
keeps up.

Do asteroid research first. Planetary Science cannot be researched without it.

## Shared rules for every research line

- **A tech must be active** or every research bot stops. Keep the queue full (Part C).
- **Belt filters.** Every belt tile that touches a crafter (Atomic Printer, Assembler, Ship
  Assembler) starts blocked. Set it with
  `configure(kind="filter_item", value=<item it should carry>, x=<belt x>, z=<belt z>)`, on the input
  side and the output side.
- **Power.** Base draws: Atomic Printer 5, Assembler 10, Ship Assembler 10, Asteroid Research
  Station 5, Planetary Research Station 30, Connector and Ship Yard 1 each. Solar Panel +7,
  Advanced Solar Panel +20. After each build step, `look_around` and check every crafter's
  `power.satisfaction` is 1. Below 1, the whole grid slows down.
- **Ship Yard.** Put a powered Ship Yard next to every Ship Assembler that makes bots. Bots with no
  free station slot wait there, and stations pull from it. Without one the Ship Assembler stops
  when every slot is full, which is fine but wastes time.
- **Stay 60+ tiles away from a running Ship Assembler.** Closer than that, you are a valid
  destination and its bots go into your fleet. If bots appear in your fleet, move them with
  `give_ships(ship=<bot name>, count=<count>, x=<station x>, z=<station z>)`.
- **Build one line, prove it, then copy it.**

## Part A: asteroid research

### Preconditions

1. `research_status` has learned: "Atomic Printing", "Automation", "Ship Assembly", "Mass
   Drivers" and "Asteroid Science". Missing ones: use the `research-to` skill with target
   "Asteroid Science". "Logistics Management" (the Ship Yard) is strongly recommended.
2. Ore sources: a Mining Station on a **Silica** asteroid and one on a **Bauxite** asteroid, powered
   and with miners. If either is missing, use `starter-mining-outpost` first.
3. `look_around` near each station to note its tile and a free side for an output belt.

### What one bot needs

Asteroid Research Bot = 4 AI Controller Circuit + 2 Plasma Engine Parts.
AI Controller Circuit = 2 Silica Ore (Atomic Printer). Plasma Engine Parts = 2 Low Density
Structure (Assembler). Low Density Structure = 2 Bauxite Ore (Atomic Printer). So each bot is
8 Silica Ore and 8 Bauxite Ore.

### Steps

1. **Circuit printer.** Against the silica station put a Connector pointing away, and an Atomic
   Printer at its far end. `set_recipe(recipe="AI Controller Circuit", x=<printer x>, z=<printer z>)`.
   Set the Connector's filter to "Silica Ore" if the game shows it blocked (the tile touches the
   printer). Power the printer (a Solar Panel pointing into it, or bridge it to a powered grid).
   Verify: `look_around` shows the printer `crafting AI Controller Circuit` with power 1.
2. **Structure printer and engine assembler.** Against the bauxite station: Connector, then an
   Atomic Printer set to "Low Density Structure", then a Connector out of the printer filtered to
   "Low Density Structure", then an Assembler set to "Plasma Engine Parts". Power both crafters
   (the Assembler draws 10: two Solar Panels at least).
3. **Ship Assembler.** Place a Ship Assembler where two belts can reach it: one from the circuit
   printer (filter "AI Controller Circuit") and one from the engine Assembler (filter "Plasma Engine
   Parts"). Set its recipe: `set_recipe(recipe="Asteroid Research Bot", x=..., z=...)`. Power it.
   - If the silica and bauxite stations are far apart, run the longer input as a belt, or bring it
     with a pair of Mass Drivers (range 40 tiles each hop). Aim the sending driver at the
     receiving one with `aim_mass_driver(x, z, target_x, target_z)`.
   - As a short stopgap only, you can top up a missing input by hand with
     `put_items(item=..., count=..., x=<ship assembler x>, z=<ship assembler z>)`. Replace it with
     a belt before moving on.
4. **Ship Yard.** Place a Ship Yard (7x12) beside the Ship Assembler and give it power.
5. **Research stations.** Place Asteroid Research Stations (4x3) within 20 tiles of an asteroid.
   Each takes 4 bots. They can be near the line or anywhere else: bots fly to the closest station
   with a free slot. Give each one its own Solar Panel pointing into it. A powered neighbour does
   not power it unless they are connected. Start with two or three stations.
6. **Start research.** Make sure a tech is active (Part C).
7. **Prove it from a distance.** Fly at least 60 tiles away from the Ship Assembler. Note
   `research_status`. Wait a few game minutes (`wait_until(predicate="researchDone", args=["<active
   tech>"], timeout_seconds=600)` or just wait and re-check). Research progress must rise while you are away.
   Fly back past the research stations (not the assembler) and take a screenshot: bots should be
   scanning. `look_around` at the Ship Assembler should show it crafting.
8. **Scale.** When research is steady, copy the line or add stations. Add a line when stations sit
   with empty slots; add stations when bots pile up in the Ship Yard.

### What to check when it stalls

| Symptom | Likely cause | Fix |
|---|---|---|
| Ship Assembler idle, inputs empty | a belt is blocked | set the filter on every belt tile touching a crafter |
| Everything slow | power below 1 | more Solar Panels, or connect the lone ones properly |
| Crafter dead, "unstable" | grid over its stability budget | add a Station Core or Struts |
| Bots in your fleet | you stood within 60 tiles of the assembler | give them to a station, then keep away |
| Stations full, points not rising | no active tech | Part C |
| Bots made but stations empty | no station accepts that bot within the network, or stations unpowered | check station power and bot type |

## Part B: planetary research

### Preconditions

1. `research_status` has learned "Planetary Science" (Planetary Research Station and Planetary
   Research Bot). Its path runs through Thermal Management, Advanced Solar Power, Mass Fuel Consumer,
   Mobile Stations, Overdrivers, Lasers, Power Distribution, Singularity Generation and Comet
   Catching, and it needs about 1,300 Asteroid Research earned. If it is missing, finish Part A
   and use `research-to` with target "Planetary Science".
2. **A planet.** Planetary research needs a **Terra World**. The Derelict biome around the
   starting area usually has one. Look for it: fly outward in stages, `look_around(radius=64)` and
   take screenshots as you go. Note its position. Check `enemies` near it before building.
3. **Ice.** Each bot needs 6 Ice. The steady source is a Comet Catcher with a Comet Harvester
   attached (Comet Catching research). A Comet Catcher pulls comets along its facing lane, and a
   comet destroys anything in that lane: build nothing there and never fly through it.
4. **Iron and bauxite** in quantity for Overdrivers and Connectors (Part A's stations, plus more
   iron).

### What one bot needs

Planetary Research Bot = 6 Ice + 1 Overdriver + 2 Connector.
Overdriver = 12 Medium Density Structure + 6 Fabricator (Assembler).
Fabricator = 2 Robotic Parts + 2 Low Density Structure (Assembler).
Robotic Parts = 1 Medium Density Structure (Assembler).
Connector = 4 Low Density Structure + 1 Medium Density Structure (Assembler).
Medium Density Structure = 2 Iron Ore and Low Density Structure = 2 Bauxite Ore (Atomic Printer).

### Steps

1. **The station first.** Fly to the Terra World. Place a Planetary Research Station (4x3) within
   30 tiles of it: `build(item="Planetary Research Station", x=..., z=...)`. It draws 30
   power: attach Advanced Solar Panels pointing into it and check `power.satisfaction` reaches 1.
   It also makes heat: attach a Heat Exchanger, and a Radiator on the Heat Exchanger once you have
   Ice. Watch its heat in `look_around`.
2. **Prove the station.** Hand-craft one or two Planetary Research Bots while standing next to it
   (`craft(item="Planetary Research Bot", count=2)`), or give them from your fleet with
   `give_ships`. With a tech active, `research_status` should show Planetary Research
   rising. This is the only hand-crafting in this part.
3. **Ice line.** Place a Comet Catcher with a Comet Harvester attached, powered (the catcher draws
   60, the harvester 20) and cooled (the harvester makes heat). Run a belt from the harvester,
   filtered to "Ice". Check with `look_around` that the harvester is working and Ice arrives.
4. **Parts lines.** Build Assemblers for "Robotic Parts", "Fabricator", "Overdriver" and
   "Connector", fed by printers making "Medium Density Structure" and "Low Density Structure".
   Chain them with filtered belts: Robotic Parts and Low Density Structure into the Fabricator
   Assembler; Medium Density Structure and Fabricator into the Overdriver Assembler. A Junction
   splits one belt evenly across several. The `build-a-mall` skill uses the same pattern.
5. **Ship Assembler.** Place it where three belts reach it: Ice, Overdriver, Connector, each
   filtered at the tile touching it. `set_recipe(recipe="Planetary Research Bot", ...)`. Power it,
   and put a powered Ship Yard beside it. The planet can be far away: finished bots fly over the
   network to the closest Planetary Research Station with a free slot.
6. **Prove it from a distance.** Stay 60+ tiles from the Ship Assembler. With a tech active, watch
   Planetary Research rise in `research_status` over a few game minutes. A screenshot at the
   planet shows bots arriving.
7. **Scale.** Each station takes 4 bots. Add stations around the planet before adding a second
   line.

## Part C: keep research running

- `research_status`. If nothing is active, pick the next useful tech and
  `research(tech=..., start_now=true)`.
- Keep up to five techs queued so research never stops between checks. Queue in prerequisite
  order. If the game refuses a tech because a prerequisite is only queued, queue it after that
  prerequisite finishes.
- Many techs need typed totals (Asteroid, Planetary, Stellar Research) as well as shared points.
  If a tech is not progressing although points are coming in, its typed requirement is the
  blocker: feed that research type.
- Re-check the queue every time you come back from a trip.

## Failure modes and refusals

- **Refused build near the planet or asteroid**: too far (move the station closer) or overlapping
  (move it out). Placement range: asteroid stations 20 tiles, planetary stations 30 tiles.
- **Refused transfer**: you are out of reach. Fly within about 20 tiles.
- **Research not moving with full stations**: no active tech, or the stations are unpowered.
- **Overheating**: add a Heat Exchanger and Radiators before the structure takes damage.
- **Line broken by a comet**: something sat in the catcher's lane. Move it.
- **Enemies at the planet**: bring Bats, or build a Defense Platform stocked from a Ship Yard
  before placing the station.
- **Temptation to hand-craft many bots**: don't. If the line is slow, find the bottleneck
  (usually power or a blocked belt) and fix it.

Report to the player: where each line and station is, which research types are now automated, the
points rate you observed, and what the next bottleneck is.

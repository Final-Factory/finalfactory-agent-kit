---
name: manage-stability-power
description: Keep Final Factory station grids stable, powered and cool - Station Cores (with their diminishing returns), Struts, Stabilizers, Solar Panels and power, Heat Exchangers and Radiators - and grow or join grids without pushing them over their stability limit. Use when the player says "my station is dead", "everything stopped", "unstable", "power is low", "overheating", "add a station core", "fix stability", "connect these stations", or when look_around shows power satisfaction below 1, needed stability near the limit, or heat production above consumption.
gameVersion: "0.50"
commands: [game_status, look_around, inspect, screenshot, inventory, research_status, research, move_to, craft, build, deconstruct, configure, wait_until, run_commands]
---

# Manage stability, power and heat

**Goal:** every grid the player cares about reads power satisfaction 1, needs less stability than
it has, and makes no more heat than it removes, with some headroom for the next addition.

## Reading a grid

Structures joined by Connectors, Struts or direct modules form one grid. `look_around` (or
`inspect(x, z)` within reach) reports, for any structure, its grid's:

- `power.satisfaction` (1 = fully powered; every building on the grid runs at this fraction of
  its speed) and `power.drawNeeded`;
- `stability.core`, `stability.nonCore` and `stability.needed`. The grid has **70 free** plus
  `core` plus `nonCore`. It is stable while `needed` is at most that total;
- `heat.production` and `heat.consumption` per second.

Two structures with the same figures are on the same grid. A cluster that looks joined but reports
different figures is two grids.

## Stability (base values)

- **Cost.** Each structure costs stability equal to its footprint (length times width, at most
  30; the world's stability setting scales this): Connector and Cargo Hold 1, Solar Panel, Atomic
  Printer and Overdriver 2, Assembler, Ship Assembler, Mass Driver and Heat Exchanger 4, Radiator
  6, Station Core, Exploration Center and Defense Platform 9, research stations 12. Mining
  Stations, Ship Yards, Struts, Laser Turrets and the Stabilizer Artifact cost nothing.
- **Station Cores** add stability, but each extra core adds half of the one before, strongest
  first: 100, then 50, 25, 12, 6. A Station Core itself costs 9, so a **5th basic core loses
  stability** and a 4th barely helps. Past three or four cores, upgrade instead:
  - High Density Station Core (High Density Station Cores research): 200, made from 16 High
    Density Structure + 1 Station Core. Being the strongest, it counts first at full value.
  - Dark Matter Station Core (Dark Matter Station Cores research): 250, and it also makes 50 power.
- **A Station Core only counts once it is on the grid.** Placed alone or merely beside a building,
  it is its own grid and helps nothing. Join it with a Strut or Connector to a building of the
  grid, then check `stability.core` went up.
- **Struts** (4 Medium Density Structure, cost nothing) join two standard buildings. A Strut only
  joins when it faces along the line between them (facing right joins east-west neighbours,
  facing up joins north-south ones). Each joined end also costs 1 less stability.
- **Stabilizer Artifact** (found as loot): 60 stability on a separate series (each extra one
  gives 80% of the previous), costs no stability, draws 30. It attaches directly to a building's
  side. In play one fixed a grid that had run out of core headroom.
- **Unstable means dead.** A grid needing more than it has drops to power 0: nothing crafts,
  heat is no longer removed, Repair Centers stop. It also sends no signal, but never use that on
  purpose.

## Power (base values)

- Makes power: Solar Panel 7, Advanced Solar Panel 20 (makes heat), Antimatter Power Station 40,
  Dark Matter Station Core 50.
- Draws: Strut 0, Connector, Cargo Hold, Ship Yard and Capacitor 1, Mining Station 3, Station Core
  4, Atomic Printer and Asteroid Research Station 5, Mass Driver and Defense Platform 6, Bat Dock 7,
  Assembler, Ship Assembler and Exploration Center 10, Overdriver and Signal Dampener up to 20,
  Planetary Research Station 30, Stabilizer Artifact 30, Stellar Research Station 40.
- Direct modules (Solar Panel, Heat Exchanger, Radiator, Overdriver, Repair Center, Stabilizer)
  join only through their connection end pointing **into** a building's attach point. A panel
  that reports its own satisfaction 1 and a draw of 0 did not connect.
- A **Power Distributor** sends spare power to another grid:
  `configure(kind="power_target", x=<distributor x>, z=<distributor z>, target_x=..., target_z=...)`.
  It makes 1 heat per second.

## Heat (base values)

- Heat makers: Advanced Solar Panel 0.5 per second, Stellar Research Station 0.6, Planetary
  Research Station, Signal Dampener and Laser Turret 0.2, Repair Center 0.1, Power Distributor 1,
  and each Overdriver 0.2 for every building it is currently speeding up.
- A Heat Exchanger (8 Medium Density Structure) removes about 1 per second; a Radiator (4 Ice + 8
  Low Density Structure) attached to the exchanger about 5; without them a building sheds only
  0.1. Both come from Thermal Management.
- A structure above 100 degrees takes damage and is eventually destroyed into a rebuild frame. The
  HUD shows an overheating warning above the hotbar long before that.
- The Radiator (3x2) joins only through its two top attach points, and both must touch the same
  building. In play, placing it facing down directly on top of a Heat Exchanger worked.

## Steps

1. **Survey.** `look_around(radius=40)` over the grid. Write down satisfaction, drawNeeded, the
   three stability figures and heat.
2. **Fix in this order:** stability first (a dead grid hides every other problem), then power,
   then heat.
3. **Stability short or under 20 spare:**
   - Fewer than three cores: craft a Station Core (10 Medium Density Structure + 8 Robotic Parts),
     place it next to a building of the grid with a one-tile gap, and put a Strut in the gap
     facing along the line between them. Verify `stability.core` rose.
   - Three or more: a High Density Station Core, a Stabilizer Artifact if you have one, or split
     the grid (next point).
   - **Split big grids.** Stations do not need to share a grid to trade items: Mass Drivers and
     Ship Yards work across grids. Keep new lines on their own grid with their own core.
4. **Power below 1:** add generation on this grid until satisfaction is 1 with a margin
   (drawNeeded plus about 10%). Solar Panels on free attach points, pointing in. Check each one
   joined.
5. **Heat production above consumption:** attach a Heat Exchanger to the grid, then a Radiator on
   it once you have Ice. Re-check until consumption exceeds production.
6. **Before joining two grids** (a Connector or Strut between them): add both grids' `needed`
   figures together and compare with 70 plus the combined core series plus nonCore. Joining can
   push the whole merged grid over its limit **instantly**: in play a single Connector did that
   twice and the whole factory went dark. If the sum is close, add stability first.
7. **Verify** after every change with `look_around`, and once more a minute later.

## Failure modes

- **Whole grid dark right after a build**: the build tipped stability over. Deconstruct the last
  joining Connector or Strut (`deconstruct(x, z)`) to split it again, then add stability.
- **Core placed but `core` unchanged**: the core is not joined, or the Strut faces the wrong way.
  Turn it with `run_commands(["rotate.abs|<x>|<z>|<0-3>"])` (0 up, 1 right, 2 down, 3 left).
- **Satisfaction falls after adding an Overdriver**: in play a fourth Overdriver raised a big
  grid's draw from 389 to 505. Add generation, or take one out.
- **Buildings keep turning into frames with no enemies around**: heat. Add exchangers and
  Radiators, or swap Advanced Solar Panels for regular ones.
- **Replacement items vanish**: a nearby rebuild frame took them. Deconstruct frames you do not
  want before carrying replacements past.

Report each grid you touched: satisfaction, needed against available stability, heat, and what you
added.

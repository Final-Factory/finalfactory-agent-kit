---
name: reveal-map
description: Lift the fog of war around the player's base in Final Factory with powered Exploration Centers, optionally boosted by Overdrivers, and check the result on screenshots. Use when the player says "clear the fog", "reveal the map", "I can't see around my base", "build exploration centers", "why can't I aim my mass driver there", or when enemies, asteroids or aiming targets near the base are hidden by fog.
gameVersion: "0.50"
commands: [game_status, look_around, enemies, screenshot, inventory, research_status, research, move_to, craft, build, deconstruct, ui, wait_until, run_commands]
---

# Reveal the map

**Goal:** the area around the base, and the lanes between its parts, stays visible without anyone
flying there: each spot is inside the circle of a built, fully powered Exploration Center.

## How fog works

- The fog is recomputed all the time from what can currently see. Your ship reveals the area
  around itself and leaves a revealed trail along the path it flew. Everything else goes dark
  again unless something is watching it.
- An **Exploration Center** watches a circle around itself for as long as it is built and powered.
  At full power the circle is about **34 tiles** in radius (base). The radius is multiplied by the
  grid's power satisfaction: at half power it reveals half as far.
- **Overdrivers** on the Exploration Center's grid make the circle larger: each adds 25% (two give
  1.5 times the radius). They also speed up every crafter and Mass Driver on the same grid.
- Fog matters for play: `enemies` and `look_around` only list what is revealed, so raiders in the
  fog are invisible to you, and the game refuses to aim a Mass Driver at a tile under fog.
- Exploration Centers send signal (19 each), which draws enemy attacks. Place them where the
  platforms of the `defend-base` skill cover them, or add a platform.

## Preconditions

1. `research_status`: "Exploration" learned (needs Asteroid Science and Ship Assembly, cost 20).
   If not: `research(tech="Exploration", start_now=true)`. For the boost, "Overdrivers" (needs
   Thermal Management; cost 100 and 100 Asteroid Research earned, base).
2. Materials, per Exploration Center: the center (8 Medium Density Structure + 4 Robotic Parts,
   about 24 Iron Ore raw) and two Solar Panels. An Overdriver is 12 Medium Density Structure + 6
   Fabricator. Check `inventory(craftable=true)`, or take them from the mall.
3. A Construction Bot in your fleet (`game_status`).

## Steps

### 1. Plan the ring

- Take a `screenshot` over the base, and open the map (`ui(action="open", panel="map")`, then a
  `screenshot`, then `ui(action="close", panel="map")`). Dark areas are fog.
- Pick spots so the 34-tile circles overlap over the base and its outposts: roughly one center
  every 50-60 tiles along the edge, plus one at any outpost more than 30 tiles out. In play, four
  centers about 80-90 tiles north, west, east and south of the base's middle cleared its edges.

### 2. Place and power each Exploration Center

- The center is 3x3 and connects **only** at two points: the middle tile of its top (north) edge
  and the middle tile of its bottom (south) edge. Side panels never connect.
- `build(item="Exploration Center", x=..., z=...)`, stay close,
  `wait_until(predicate="structureBuilt", args=["<x>", "<z>", "Exploration Center"], timeout_seconds=60)`,
  then read its real corner tile `(ex, ez)` from `look_around(radius=20)`.
- It draws 10. One Solar Panel (7) is not enough; use two, standing upright on the two
  connection points (a Solar Panel facing up or down occupies its tile and the one north of it):
  - `build(item="Solar Panel", x=ex + 1, z=ez + 3, facing="down")` on the top point,
  - `build(item="Solar Panel", x=ex + 1, z=ez - 2, facing="up")` on the bottom point.
- Verify: `look_around(radius=20)` shows the center `built` with `power.satisfaction` 1, and both
  panels report the same grid figures as the center. A panel with its own satisfaction of 1 and a
  draw of 0 did not connect: `deconstruct` it and place it again. Never place on a tile you just
  removed something from until the removal has finished; check with `look_around` first.

### 3. Check the reveal

- Fly back to the middle of the base and take a `screenshot`, and look at the map again. The area
  around each center should now be clear.
- `enemies(...)` near a center's edge now lists anything there, even with you far away.

### 4. Optional: boost with Overdrivers

An Overdriver draws up to 20 on top of the center's 10 and makes heat. The radius is multiplied by
the power satisfaction, so an Overdriver on a grid that cannot power it makes the circle
**smaller**. Only add one when you can power it:

- The two connection points are the center's only way in, so use one for a Connector or Strut to a
  small powered hub (a Cargo Hold or a Station Core with its own Solar Panels) and put the
  Overdriver on the hub or on the center's other point. Give the grid at least 20 more power
  (three more Solar Panels) per Overdriver added, then check the satisfaction.
- Verify with `look_around`: power satisfaction still 1, and the heat production does not exceed
  consumption. If it does, add a Heat Exchanger (see `manage-stability-power`).
- Take a `screenshot` to confirm the clear area grew.

### 5. Keep it running

Exploration Centers stop revealing the moment they lose power or are destroyed. After a raid,
check each one with `look_around` from nearby and rebuild panels or frames.

## Failure modes

- **Center built but fog unchanged**: it has no power. Check the panels are on the top and
  bottom middle tiles, not the sides.
- **Circle smaller than expected**: satisfaction below 1 (one panel only, or an Overdriver it
  cannot power).
- **Mass Driver aim refused**: the target tile is under fog. Put a center near the target first.
- **Centers razed**: they send signal, so raiders notice them. Cover them with a Defense Platform.
- **Build refused**: out of reach or overlapping. Fly closer, or shift the tile.

Report which centers you built, where, whether each is at power 1, and what the screenshot showed.

---
name: defend-base
description: Defend the player's base in Final Factory against Urso raids - read signal and base size, ring the base with powered Defense Platforms full of Bats, watch the attack warnings, bring the fleet where it is needed and repair the damage. Use when the player says "defend the base", "we're under attack", "raids keep destroying my stuff", "set up defences", "protect my outposts", "build defense platforms", or when look_around shows rebuild frames where buildings used to be.
gameVersion: "0.50"
commands: [game_status, look_around, enemies, screenshot, inventory, research_status, research, move_to, craft, build, deconstruct, give_ships, take_ships, set_recipe, ui, wait_until, run_commands, respawn]
---

# Defend the base

**Goal:** every part of the base the player cares about sits inside the range of a powered Defense
Platform holding Bats, the Bats refill themselves from a Ship Assembler line, and raids end with
little or nothing destroyed while nobody is watching.

## How attacks work (base values)

- **Signal drives attacks.** Enemy camps soak up the signal your stations send out and spend it on
  raids. The more signal, the stronger and more frequent the attacks. From 1,500 global signal,
  camps can add Trident spawners, which hit much harder than Cutters and Stingers.
- **Base size raises signal.** Each station grid sends the signal of its buildings, plus 10% for
  every 100 stability the grid needs beyond the first 100. A grid that needs 300 stability sends
  20% more than its buildings alone. A big base draws bigger raids, so defence has to grow with it.
- **What sends signal** (per building, when working): Station Core 2, Assembler and Atomic
  Printer 8, Mining Station and Ship Assembler 12, Asteroid Research Station 16, Exploration
  Center and Greenhouse 19, Advanced Mining Station 25, Planetary Research Station 31, Stellar
  Research Station 62, Dyson Platform 187. Solar Panels, belts, Cargo Holds, Ship Yards and
  Defense Platforms send none.
- **Where it shows:** the stats panel lists Global Signal and the number of attacks so far:
  `ui(action="open", panel="production")`, then `screenshot`. Close it after.
- **What raids hit.** Raiders went for Solar Panels, Station Cores and outlying Mining Stations
  first in play. A destroyed building leaves a rebuild frame.
- **Signal Dampener** (Signal Dampening research): a small module that removes up to 9 signal
  from its own grid when fully powered. It draws 20 and makes heat. Worth it on a grid that
  sends a lot, such as one with research stations.

## Preconditions

1. `game_status`: health above 0 (else `respawn()`). Note your fleet: how many Bats.
2. `research_status`: "Defense Platforms" learned (needs Asteroid Science and Ship Assembly, cost
   15). If not: `research(tech="Defense Platforms", start_now=true)`. "Station Repair" (the Repair
   Center) and "Logistics Management" (the Ship Yard) are strongly recommended.
3. Materials. A Defense Platform is 8 Medium Density Structure + 4 Robotic Parts (about 24 Iron
   Ore). Each needs power: its draw is 6, so one Solar Panel (7) is enough. Each Bat is 4 AI
   Controller Circuit + 2 Plasma Engine Parts (8 Silica Ore, 8 Bauxite Ore).

## Steps

### 1. Map the base and the threat

- Fly over the base and `look_around(radius=64)` in a few spots. Note the outer edge of every
  cluster the player cares about, including outposts and the research stations at a planet.
- `enemies(...)` at each spot. An enemy with a `target` field is attacking the named structure
  now. Note which side raids come from; they tend to come from the same direction again.
- Rebuild frames in `look_around` (state `frame`) where buildings used to be are past raid damage.

### 2. Place the perimeter platforms

- A Defense Platform is 3x3 and fights anything within **45 tiles** of it (its default attack
  range, the maximum). It holds up to **20** combat ships: Bats, Knights or Krillos.
- Put platforms so their 45-tile circles overlap over everything worth keeping, with the first
  ones on the side raids come from and next to outlying Mining Stations and Solar Panel fields.
  Spacing them 30-40 tiles apart along the edge gives good overlap.
- Place it: `build(item="Defense Platform", x=..., z=..., facing="up")`. The platform is placed
  by its centre: the tile you give becomes its middle, and its corner tile is one less in x and
  z. Read the real tile back from the `build` reply or `look_around`.
- Power it with its own Solar Panel. A proven layout: with the platform's corner at `(px, pz)`,
  place a Solar Panel facing right at `(px - 2, pz)`. A second one at `(px - 2, pz + 1)` gives
  headroom. Verify: `look_around` shows the platform `built` with `power.satisfaction` 1.
- A platform on its own grid keeps the base's grid small. It sends no signal.

### 3. Fill the platforms with Bats

- **By hand, to start:** fly within reach of the platform and
  `give_ships(ship="Bat", count=20, x=<platform x>, z=<platform z>)`. Verify with `game_status`
  that your fleet dropped by the number given.
- **Automated, the real fix:** platforms pull combat ships over the logistics network, from Ship
  Yards and from Ship Assemblers making them. Set up a Bat line: a Ship Assembler set to "Bat"
  (`set_recipe(recipe="Bat", x=..., z=...)`) fed with AI Controller Circuit and Plasma Engine
  Parts on filtered belts, with its own powered Ship Yard. The `automate-research` skill shows the
  line pattern. Give the Bat line **its own** Ship Yard, or its Bats fill the yard your other
  lines use (see `fleet-logistics`).
- A Bat Dock attached to a platform or Ship Yard keeps requesting Bats for it.

### 4. Heal and harden

- **Repair Center** (Station Repair research; 8 Medium Density Structure + 8 Robotic Parts): a
  1x1 module that attaches directly to a building and heals every structure on its grid, about
  0.6 health per second (base). It only works while the grid is stable. One per important grid.
- **Laser Turrets** (Laser Turrets research) shoot enemies within 22 tiles using Capacitor charge.
  They are a later, extra layer; platforms with Bats come first.
- Clear or remove rebuild frames you do not want: a frame quietly takes the next matching item you
  carry past it.

### 5. Watch for attacks

- The HUD shows warning icons above the hotbar: under attack, structure destroyed, overheating,
  and asteroid running low. An attack warning appears when a structure takes damage and stays for
  8 seconds after the last hit. `game_status` alerts do not include them, so take a `screenshot`
  every few minutes and look above the hotbar.
- When you see an attack, `enemies` near the warning's area shows who is attacking what.

### 6. Respond with the fleet

- **Your fleet fights only around you.** Bats in your fleet engage enemies near your ship. They do
  not go and defend a base you are not at. To help, fly there: `move_to` the attacked spot with the
  fleet.
- Go in with everything or stay away. Parking just off a camp or raid loses Bats slowly.
- With more than 10 combat ships the fleet hides and only comes out when an enemy is near its own
  player. Hidden fleets defend nothing while you are elsewhere, which is why platforms matter.
- In play, 17 Bats with the player present cleared a 12-Cutter raid with no losses.

### 7. Keep up with growth

After every big addition to the base (a new line, research stations, a merged grid), come back to
this skill: more signal means bigger raids. Add a platform for each new cluster, and more Bats on
the side raids come from.

## How to verify

- Each platform: `look_around` shows it `built`, power 1. Your fleet count dropped when you gave it
  Bats, and a `screenshot` of the platform shows Bats around it.
- After a few game minutes away, no new rebuild frames in the covered area.
- A Bat line: its Ship Yard fills and the platforms stay at 20.

## Failure modes

- **Platform empty although Bats are made**: the platform is unpowered, or the Bats go to your
  fleet because you are within about 60 tiles of the Ship Assembler. Fly away, or give them.
- **Bat line fills a shared Ship Yard**: other lines stall. Give the Bat line its own yard, or let
  more platforms pull the Bats. `take_ships` from the yard frees space.
- **Solar Panel destroyed, platform dark**: raiders target panels. Place the panel on the side away
  from the raids, add a second one, and a Repair Center on bigger grids.
- **Raids keep growing**: signal keeps growing. Check Global Signal, add a Signal Dampener to the
  grid with the most research or crafting, and add platforms.
- **Fleet ignores an attack elsewhere**: expected. Fleets fight only near their player.
- **`give_ships` refused**: out of reach. Fly within about 20 tiles.

Ask the player before clearing an enemy camp: it is a real fight and the camp may be stronger than
it looks. Report where each platform is, how many Bats it holds, and which side raids come from.

---
name: endgame-victory
description: Win Final Factory - research Singularity Navigation, find a black hole, build and power a Dark Star Gate beside it, feed it five stages of Fullerene Structure, Singularity Engines and Von Neumann Probes, and send the Singularity Vessel into the black hole. Use when the player says "win the game", "finish the game", "build the Dark Star Gate", "launch the Singularity Vessel", "what do we need for the ending", or when Singularity Navigation is in reach.
gameVersion: "0.50"
commands: [game_status, research_status, research, look_around, enemies, inspect, screenshot, inventory, move_to, craft, build, put_items, take_items, give_ships, set_recipe, wait_until, run_commands, respawn]
---

# Endgame: the Singularity Vessel

**Goal:** a Singularity Vessel flies from a powered Dark Star Gate into a black hole. When it
arrives, the game shows the victory screen.

This is the longest goal in the game. Break it into the phases below, report after each, and ask
the player before the long trips.

## What it takes (base values)

- **Research:** Singularity Navigation (unlocks the Dark Star Gate and Singularity Vessel). It
  needs Singularity Engines, Replication and Advanced Ship Assembly, costs 40,000, and needs
  50,000 each of Stellar, Planetary and Asteroid Research earned. `scale-research` gets you there.
- **The gate:** Dark Star Gate, 21x43 tiles, made from 200 Fullerene Structure + 3 Advanced Ship
  Assembler. It must stand within **72 tiles** of a black hole, and it draws **2,000** power.
- **Five stages**, each taking **200 Fullerene Structure, 25 Singularity Engine and 10 Von Neumann
  Probe**. A stage only processes while the gate's power satisfaction is at least 0.98.
- **In total:** 1,200 Fullerene Structure for the gate and stages, plus 125 Singularity Engines
  (which need 2,500 more) and 50 Von Neumann Probes.

Recipes:

| Item | Inputs |
|---|---|
| Fullerene Structure | 3 High Density Structure, 2 Metal Matrix Composite, 2 Organic Polymer |
| Singularity Circuit | 2 Singularity Field Generator, 4 Quantum Computer, 1 Exotic Matter |
| Singularity Engine | 16 Singularity Circuit, 20 Fullerene Structure, 8 High Energy Laser, 8 Dark Matter Structure |
| Von Neumann Probe | 5 Singularity Circuit, 3 Quantum Bot Chassis, 1 Quantum Printer, 5 Metal Matrix Composite |
| Advanced Ship Assembler | 1 Ship Assembler, 15 High Density Structure |

Exotic Matter comes from Dyson Bots on a Dyson Platform around a main sequence star (Dyson Spheres
and Dyson Power Collection research). Each Singularity Circuit needs one, and the win needs about
2,250 circuits, so the Dyson sphere is the heart of the endgame economy.

## Steps

### 1. Plan and research

- `research_status`: how far each typed total is from 50,000. Plan the lines per `scale-research`.
- In parallel, automate the chains in the recipe table above, each into its own Cargo Holds, the
  way `build-a-mall` does. Von Neumann Probes are ships: build them in a Ship Assembler with its
  own Ship Yard (`fleet-logistics`).

### 2. Find the black hole

- It lies far out, beyond about 1,000 tiles from the start, and the far map only generates as you
  fly out. Ask the player first: this is a long trip.
- Fly in stages of about 60 tiles: `move_to`, then `enemies` and `look_around(radius=64)`, and a
  `screenshot` each time. Camps far from every player are asleep and only show up once you are
  close; retreat along the path you came by. Bring a strong fleet.
- Note the black hole's tile. Clear or avoid camps near it before building; ask the player before
  attacking one.

### 3. Build and power the gate

- Carry the Dark Star Gate (or its parts) and a Construction Bot. Place it within 72 tiles of the
  black hole: `build(item="Dark Star Gate", x=..., z=..., facing=...)`. Stay near until
  `look_around` shows it `built`.
- Power it to at least 0.98 satisfaction. 2,000 power is a lot: for scale, an Antimatter Power
  Station makes 40 and a Dyson Bot 120. The honest way to supply it at the black hole has not been
  proven in play yet, so plan it with the player, add generation to the gate's grid, and check
  `power.satisfaction` in `look_around` until it reads at least 0.98. The grid also needs the
  stability for everything on it (`manage-stability-power`).
- Put a Defense Platform or two with Bats beside it (`defend-base`): the gate is a big target.

### 4. Feed the stages

- Bring one stage's items: 200 Fullerene Structure and 25 Singularity Engine in your inventory, 10
  Von Neumann Probes in your fleet.
- Within reach of the gate: `put_items(item="Fullerene Structure", count=200, x=<gate x>, z=<gate z>)`,
  `put_items(item="Singularity Engine", count=25, x=..., z=...)`, and
  `give_ships(ship="Von Neumann Probe", count=10, x=..., z=...)`.
- A transfer can move fewer than you asked (in play, Singularity Engines went in 8 at a time).
  Check `inventory` and `game_status` after each one and repeat until the stage's full amount is
  in.
- A stage takes about 30 seconds of processing at full power. `inspect(x, z)` on the gate shows
  its contents; repeat for all five stages.

### 5. The launch

After stage five the gate launches a Singularity Vessel. It flies into the black hole by itself;
take a `screenshot` to watch. The victory screen appears when it arrives. Report it to the player.

## Failure modes

- **Stage not processing**: power satisfaction below 0.98, or one of the three inputs short.
- **Gate refused**: too far from the black hole (over 72 tiles), overlapping, or out of reach.
- **Gate power drops mid-stage**: generation destroyed by raids, or the grid became unstable. Fix
  it before adding more items.
- **Research stuck below 50,000 of one type**: scale that type (`scale-research`); the win cannot be
  bought any other way.
- **Items vanish near the gate**: a rebuild frame nearby took them. Deconstruct unwanted frames
  first.

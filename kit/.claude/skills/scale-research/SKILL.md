---
name: scale-research
description: Scale Final Factory research from the first asteroid stations to the endgame - more research stations and bot lines per research type (asteroid, planetary, stellar), a queue that never runs dry, and a plan for the typed research totals that gate the late tree. Use when the player says "research is too slow", "scale up research", "we need more planetary/stellar research", "what's blocking research", "get to Stellar Science", "plan the tech path to the end", or when research_status shows a tech stuck waiting on a typed total.
gameVersion: "0.50"
commands: [game_status, research_status, research, look_around, inspect, screenshot, inventory, move_to, craft, build, set_recipe, configure, give_ships, wait_until, run_commands, ui]
---

# Scale research

**Goal:** each research type the next techs need is rising steadily in `research_status`, the
queue always has an active tech, and you know which typed total is the next bottleneck and have a
line growing toward it.

This skill is about how much and in what order. The `automate-research` skill has the build steps
for asteroid and planetary bot lines, and `research-to` has the full tech table.

## How research adds up (base values)

- Research bots earn points only on their station, only while a tech is **active**, at the
  station's power satisfaction. Each bot scans 10 times and then wears out. An asteroid bot earns
  about 2.5 points over its life, a planetary bot on a Terra World about 7.5.
- Every station holds **4** bots. Placement: Asteroid Research Station within 20 tiles of an
  asteroid, Planetary within 30 of a Terra World, Stellar within 72 of a star.
- Points go into the shared bank and into their type's running total. Late techs need large
  **typed totals**, which are what actually stall research:

| Tech | Typed totals it needs (base) |
|---|---|
| Planetary Science | Asteroid 1,300 |
| Quantum Computing | Asteroid 2,000, Planetary 1,800 |
| Stellar Science | Planetary 6,000, Asteroid 8,000 |
| Dyson Spheres | Asteroid 15,000, Planetary 12,000, Stellar 10,000 |
| Singularity Engines | Stellar, Planetary and Asteroid 30,000 each |
| Singularity Navigation (the win) | Stellar, Planetary and Asteroid 50,000 each |

The world's tech cost setting scales all of these; `research_status` shows the real numbers.

## Steps

### 1. Find the bottleneck

- `research_status`: the active tech, its progress, the queue, and any missing typed total (shown
  like "Planetary Research (937/1,200)").
- If nothing is active: `research(tech=<next>, start_now=true)`. If the queue has entries but
  nothing is active (this can happen right after a tech completes),
  `run_commands(["research.setactive|<first queued tech>"])`.
- Note each research type's total now, wait a few game minutes, and check again. The difference
  is your rate per type. Compare it with the next typed total on your path to see how long it
  will take.

### 2. Keep the queue full

Queue up to five techs in prerequisite order (`research-to`). Useful fillers while a typed total
builds up: the numbered upgrade series (Mining Productivity, Fleet Command, Bat Damage) and cheap
utility techs. Check the queue every time you come back from a trip.

### 3. Scale the type that blocks

For the blocking type, in this order:

1. **Power first.** A station at half power researches at half speed. `look_around` at each
   station: satisfaction must be 1. Planetary stations draw 30 and Stellar ones 40; both make
   heat (see `manage-stability-power`).
2. **Fill empty slots.** Fly past the stations and take a `screenshot`: each holds up to four
   scanning bots. Stations with free slots mean the bot line is too slow: find its bottleneck
   (usually ore, power or a blocked belt; `inspect(x, z)` on each crafter within reach) before
   building more stations.
3. **More stations.** When every slot stays full and bots wait in the Ship Yard, add stations
   around the same asteroid, planet or star. Four more bots per station.
4. **Another bot line.** When stations sit with empty slots and the line runs flat out, copy the
   line. An Overdriver on the line's grid speeds up every crafter on it by 25% at full power.

Asteroid research never stops mattering: every late tech needs a large asteroid total, so keep
the asteroid lines growing alongside the newer types.

### 4. The planetary step

Start the Planetary Research Bot line right after Planetary Science: most mid-game techs need
planetary totals. Planetary Research Station = 1 Asteroid Research Station + 8 AI Controller
Circuit + 16 Robotic Parts. In play, four stations at one Terra World fed by one line gave about
27 to 35 planetary points per minute. Two regular Solar Panels run a station at about half speed
with no heat trouble; more stations beat hotter panels.

### 5. The stellar step

After Stellar Science:

- Find a star (more common in Nebula biomes; fly out in stages, `look_around` and screenshot, check
  `enemies`).
- Stellar Research Station = 1 Asteroid Research Station + 16 High Density Structure + 4 Quantum
  Computer, within 72 tiles of the star. It draws 40 and makes 0.6 heat per second: give it a Heat
  Exchanger and a Radiator.
- Stellar Research Bot = 3 Quantum Bot Chassis + 4 High Energy Connector + 2 Provider Cargo Hold +
  2 Command Core. Quantum Bot Chassis = 3 Quantum Computer + 1 Singularity Field Generator + 6
  Plasma Engine Parts. This is the deepest chain in the game. In play, Organic Polymer (and so the
  greenhouses) was the slowest input by far, so build the organics supply before the rest.
- Build the chain one Assembler at a time into Cargo Holds, prove each part arrives, then join them
  into one Ship Assembler with its own Ship Yard (`fleet-logistics`).

### 6. Verify

- Every type the path needs rises between two `research_status` checks a few game minutes apart,
  with you away from the lines.
- The active tech's progress rises, or it waits only on a typed total that is rising.

## Failure modes

- **Points rising but tech stuck at full**: it is waiting on a typed total. Feed that type.
- **Nothing rising**: no active tech, or the stations are unpowered or unstable.
- **Bot line running but stations empty**: its bots are going to your fleet (stay 60+ tiles away
  from the Ship Assembler) or into a full Ship Yard.
- **Research slowed after a raid**: stations or their panels became rebuild frames. See
  `defend-base`; research stations send a lot of signal.
- **Hand-crafting research bots to catch up**: don't. Fix the line.

Report each research type's rate, the next typed total on the path and your estimate to reach it,
and what you added.

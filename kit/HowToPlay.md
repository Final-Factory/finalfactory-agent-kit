# How to Play Final Factory

A guide for an AI agent playing Final Factory through the `finalfactory` MCP tools. It covers what
the game is, how its systems fit together, and the mistakes agents have already made. Read it once
before you start a multi-step plan, and come back to the section you need.

Written for game version 0.50. Numbers marked "base" are the game's default values; research
upgrades and world settings can change them, so check the live game when a number matters.

## 1. What the game is

Final Factory is a space factory game. You fly a small ship around an open map of asteroids,
planets, stars and enemy camps. You mine ore by hand, then build stations that mine, refine and
assemble for you. Research unlocks better buildings. The goal is to build a Dark Star Gate next to a
black hole and send a Singularity Vessel into it.

The point of the game is automation. Hand-crafting and carrying items are how you get started. After
that, anything the game uses up continuously (research bots, their parts, miner bots, fleet ships,
the buildings you keep placing) should come from production lines that run without you. An agent
that hand-crafts hundreds of research bots is playing the game wrong even when the objective moves.
The late-game research totals are in the tens of thousands of points, and only automated lines get
there.

## 2. Units, coordinates and the map

- **Everything is in tiles.** Every tool takes and returns tile coordinates `(x, z)`. `x` grows to
  the east (right), `z` grows to the north (up). A facing of Up points +z, Right +x, Down -z,
  Left -x.
- A structure's tile is its **lower-left corner**. A 3x2 Mining Station at (10, 20) covers x 10-12
  and z 20-21. After you build something, read its tile back with `look_around` rather than
  assuming it landed exactly where you asked; large structures can shift by a tile.
- Asteroids occupy a 30x30 tile area but the rock itself is only about 7 tiles in radius around the
  centre. `look_around` reports an asteroid by the corner tile of that area, so its centre is about
  15 tiles further along +x and +z. Check with a screenshot when placement matters.
- Your ship flies at roughly 10 tiles per second. `move_to` flies you there; it never teleports.
- Your **action range is about 20 tiles** (it can be upgraded). Taking and putting items, and
  mining, only work within reach, the same as for a human. Fly closer first.
- The map is split into biomes. The starting area is safe-ish and has iron, silica and bauxite
  asteroids. The Derelict biome that surrounds it usually holds a planet. Stars are more common in
  Nebula biomes. The black hole lies far out, beyond about 1,000 tiles from the start. Far areas
  only generate as someone flies out to them.
- Fog of war hides what you have not seen. Observation tools only show what a human player could
  see from where you are.

## 3. Fleet vs factory

This is the most important idea in the game.

- **Your fleet** is the set of ships that follow you: combat ships like Bats, construction bots, and
  any bot you crafted that has not been handed to a building. A miner or research bot sitting in
  your fleet does nothing useful. The HUD shows fleet size against capacity (for example 4/10).
  Fleet Command research raises the cap.
- **The factory** is everything you place: stations, crafters, belts, power, and the bots each
  building owns. A Miner Bot only mines once it belongs to a Mining Station. A research bot only
  researches once it belongs to a research station.

Bots move from your fleet to a building in one of three ways:

1. You craft the bot while standing near a building that has a free slot for it. It flies straight
   there.
2. You give it from your fleet to the building (the fleet panel's deploy, or
   `give_ships(ship, count, x, z)` with the bot type, count and the building's tile).
3. A **Ship Assembler** builds it, and it flies over the logistics network to the closest building
   that wants it (see section 9). This is the automated way and the one to aim for.

## 4. The core loop

1. **Mine by hand.** Fly to an asteroid and mine it (`mine`). You start with 75 Iron Ore and 75
   Silica Ore, so the first thing to mine is usually Bauxite.
2. **Craft.** `craft` builds an item from what is in your inventory. Your ship crafts the whole
   chain of intermediates from raw ore automatically and needs no power. If you lack the raw
   materials the craft simply does not happen, so check `inventory(craftable=true)` first.
3. **Get a Construction Bot.** Nothing you place gets built without one. It costs 4 AI Controller
   Circuit and 2 Plasma Engine Parts (8 Silica Ore and 8 Bauxite Ore raw). Your first one does not
   use fleet capacity.
4. **Place a Mining Station next to an asteroid**, power it with a Solar Panel, and give it Miner
   Bots. Ore now piles up in the station.
5. **Refine with belts.** A Connector (belt) carries ore from the station into an Atomic Printer,
   which makes components. Assemblers turn components into parts and buildings. Ship Assemblers
   turn parts into bots and ships.
6. **Research** unlocks each new building. Research bots on research stations earn the points.
7. **Expand and defend.** More ore types, more lines, power, heat control, and defence against
   enemy camps.
8. **Go for the win.** Planets, stars, a Dyson sphere, the black hole, the Dark Star Gate.

The game's objective card (shown by `game_status`) follows this same order. When you do not know
what to do next, do what the card asks.

## 5. Items and recipes you will use early

Raw ores: **Iron Ore**, **Silica Ore**, **Bauxite Ore**. Later: **Ice**, **Organic Compounds**.

Components (made by an Atomic Printer, or by hand):

| Item | Recipe |
|---|---|
| Low Density Structure | 2 Bauxite Ore |
| Medium Density Structure | 2 Iron Ore |
| AI Controller Circuit | 2 Silica Ore |

Parts (made by an Assembler, or by hand):

| Item | Recipe |
|---|---|
| Plasma Engine Parts | 2 Low Density Structure |
| Robotic Parts | 1 Medium Density Structure |
| Fabricator | 2 Robotic Parts, 2 Low Density Structure |
| Solid State Laser | 2 Low Density Structure, 2 AI Controller Circuit |
| Mass Energy Converter | 1 Robotic Parts, 2 Medium Density Structure |

Buildings (made by an Assembler, or by hand). Size is width x length in tiles.

| Building | Size | Recipe |
|---|---|---|
| Mining Station | 3x2 | 8 Medium Density Structure, 4 Robotic Parts |
| Solar Panel | 1x2 | 4 Low Density Structure, 2 Medium Density Structure |
| Connector | 1x1 | 4 Low Density Structure, 1 Medium Density Structure |
| Cargo Hold | 1x1 | 8 Medium Density Structure |
| Atomic Printer | 1x2 | 8 Medium Density Structure, 2 Fabricator |
| Assembler | 2x2 | 8 Medium Density Structure, 4 Fabricator |
| Ship Assembler | 2x2 | 8 Medium Density Structure, 4 Fabricator |
| Inserter Bot | 1x1 | 2 Plasma Engine Parts, 2 Medium Density Structure |
| Strut | 1x1 | 4 Medium Density Structure |
| Station Core | 3x3 | 10 Medium Density Structure, 8 Robotic Parts |
| Mass Driver | 2x2 | 6 Medium Density Structure, 4 AI Controller Circuit |
| Asteroid Research Station | 4x3 | 8 Medium Density Structure, 4 Robotic Parts |
| Ship Yard | 7x12 | 16 Medium Density Structure |

Bots and ships (made by a Ship Assembler, or by hand):

| Ship | Recipe |
|---|---|
| Miner Bot | 5 Plasma Engine Parts |
| Construction Bot | 4 AI Controller Circuit, 2 Plasma Engine Parts |
| Bat (combat ship) | 4 AI Controller Circuit, 2 Plasma Engine Parts |
| Asteroid Research Bot | 4 AI Controller Circuit, 2 Plasma Engine Parts |
| Planetary Research Bot | 6 Ice, 1 Overdriver, 2 Connector |

Medium Density Structure goes into almost everything, so iron is the ore you will run short of
first. Use `inventory(craftable=true)` to see what you can make right now, and `list_commands` or
the game's own recipe panel for anything not listed here.

Which crafter makes what: an **Atomic Printer** makes components (the three density/circuit items
above). An **Assembler** makes parts and buildings. A **Ship Assembler** makes bots and ships. You
choose what a crafter makes with `set_recipe`.

## 6. Building

`build(item, x, z, facing)` places a structure, the way a human picks a building from the
inventory and clicks a tile. `item` is the building's in-game name, for example
`build(item="Mining Station", x=10, z=-4, facing="up")`. `facing` is up, right, down or left.
`item` also accepts a whole blueprint string (text starting `ffblueprintstart`) that the player
copied from the game's blueprint tool, which places every building in it at once.

The result says which tile the building actually landed on. Wide buildings can land one tile off
the tile you asked for, so use the reported tile for anything placed next to it.

To turn a building that is already placed: `run_commands(["rotate.abs|<x>|<z>|<0-3>"])`
(0 up, 1 right, 2 down, 3 left).

What happens after placing:

1. A ghost frame appears at the tile. If the spot is invalid (overlap, too far from an asteroid,
   wrong kind of neighbour) the game refuses and says why. Adjust the tile or facing and try again.
2. A Construction Bot from your fleet fetches the item **from your inventory** and builds it in a
   few seconds. You must be nearby and you must have the item. A frame with no item in your
   inventory, or placed while you fly away, stays an unbuilt frame until you come back with it.
3. `look_around` shows each structure's `state`: `frame` (not built yet) or `built`. Only built
   structures do anything. `wait_until` with the `structureBuilt` predicate waits for it.

`deconstruct(x, z)` removes a structure. The item comes back to your inventory. With a second
corner it clears a rectangle.

Placement rules that matter:

- **Mining Station**: must be within 16 tiles (centre to centre) of an asteroid, with its footprint
  outside the rock. Place it hugging the asteroid's edge, about 10-13 tiles from the asteroid's
  centre. The station picks its asteroid when it is placed and never changes it, so the asteroid has
  to be there first.
- **Asteroid Research Station**: within 20 tiles of an asteroid.
- **Planetary Research Station**: within 30 tiles of a planet (a Terra World).
- **Stellar Research Station**: within 72 tiles of a star.
- **Dark Star Gate**: near a black hole.

## 7. Station grids: connections, power, stability, heat

Structures that are connected together form one **station grid**. A grid shares one pool of power,
one stability budget and one heat budget. Two clusters that look adjacent but are not wired
together are two separate grids.

### Connections

- Most buildings (Mining Station, Assembler, Ship Assembler, Atomic Printer, Cargo Hold, research
  stations, Mass Driver) are **standard** structures. Two standard structures placed side by side
  do **not** connect. You bridge them with a **Connector** (which also carries items) or a
  **Strut** (which carries nothing and only helps stability).
- Some modules attach **directly** to a neighbour with no bridge: Solar Panel, Advanced Solar
  Panel, Heat Exchanger, Radiator, Overdriver, Power Distributor, Command Core, Repair Center,
  Bat Dock. A direct module only joins a grid through its own connection end, and only when that
  end points **into** the building. A Solar Panel dropped beside a station with the wrong facing
  forms its own separate grid and powers nothing.
- When you hold an item to place, the game marks a building's attach points with green diamonds.
  A screenshot shows them.
- Check a connection by reading both structures in `look_around`: if they share a grid, their power
  satisfaction and stability figures match. A lone Solar Panel that reports its own satisfaction of
  1 and a power draw of 0 did not connect. Deconstruct it and place it again with a different tile
  or facing.

### Belts (Connectors)

- A Connector's arrow is its direction of flow. Placed against a building and pointing away, it
  takes items out of that building. Pointing into a building, it delivers into it.
- A straight run of Connectors is one belt. Items travel along it single file and queue up.
  Different items can share a belt.
- A belt that ends at a perpendicular belt hands items on; one that ends at a building puts items
  into it.
- A **Junction** splits items evenly across every belt attached to it.
- **Filters.** Crafters (Atomic Printer, Assembler, Ship Assembler, research stations) keep
  ingredients and products in one shared zone that every attached belt can draw from. So a new belt
  on a crafter starts **blocked**, and it stays inert until you set its filter to the item you want
  it to carry: `configure(kind="filter_item", value=<item>, x, z)` on the belt tile next to the
  crafter. A
  Mining Station only ever holds its own ore, so its output belt needs no filter.
- An **Inserter Bot** is a small placed item that picks items out of the building behind it and
  drops them into the building in front. Place it with its arrows pointing away from the source.

### Power

- Solar Panels produce power; almost every building draws it. Base values: Solar Panel +7,
  Advanced Solar Panel +20. Draws: Mining Station 3, Atomic Printer 5, Asteroid Research Station 5,
  Mass Driver 6, Assembler 10, Ship Assembler 10, Planetary Research Station 30. Every Connector,
  Cargo Hold and Ship Yard draws 1.
- Power is shared as a ratio. If a grid makes half the power it needs, **every** building on it
  runs at half speed. A factory that looks stalled is often just under-powered. Check `power`
  satisfaction in `look_around` first. One Solar Panel runs a Mining Station; it does not run an
  Assembler.
- A **Power Distributor** sends spare power from one grid to another.
- A **Capacitor** stores spare power as charge for buildings that need bursts, such as Laser
  Turrets.

### Stability

- Every structure costs some stability. A grid gets 70 free. **Station Cores** add a lot (each
  extra core adds less than the last); a **Strut** connected at both ends lowers the cost of the
  two structures it joins. Mining Stations, Ship Yards and Comet Catchers cost none.
- An **unstable grid shuts down completely**: power drops to zero, nothing crafts, and heat stops
  being removed. When a station is dead, fix stability before anything else. `look_around` shows a
  grid's stability `core`, `nonCore` and `needed` values.

### Heat

- Some buildings make heat. Early buildings (Solar Panel, Mining Station, Atomic Printer, basic
  Assembler and Ship Assembler, Connector) make none. Heat makers include the Advanced Solar Panel,
  Overdriver, Power Distributor, Laser Turret, Rocket Adapter, Comet Harvester, Greenhouse Station,
  Planetary and Stellar Research Stations, and the Advanced and Hyper assemblers. Watch the heat
  figures in `look_around` and the overheating alert in `game_status`.
- A structure above 100 degrees takes real damage and can be destroyed. The overheating warning
  appears at about 35 degrees, long before that.
- A **Heat Exchanger** attached to the grid removes some heat; **Radiators** attached to the Heat
  Exchanger remove much more (a Radiator needs Ice to build). Both come from Thermal Management
  research.

## 8. Crafting by hand vs automation

| Hand-craft (`craft`) | Automate |
|---|---|
| The first Construction Bot, the first few Miner Bots | Miner Bots to replace the ones that wear out |
| One-off buildings early on | Buildings you place over and over (a mall, section 12) |
| A few Bats to clear the first enemy camp | Combat ships for defence |
| Never research bots beyond a first handful | All research bots |

Miner Bots and research bots **wear out on purpose**: a miner self-destructs after hauling a set
amount of ore, and a research bot after a set number of scans. Seeing them die and get replaced is
normal. It is also why they must come from a Ship Assembler: hand-crafting will never keep up.

## 9. Logistics: how bots and items get around

**Items** move on belts, Inserter Bots and Mass Drivers. A **Mass Driver** fires items at another
Mass Driver up to 40 tiles away (centre to centre). Each driver needs power. Aim it with
`aim_mass_driver(x, z, target_x, target_z)` (the sending driver's tile, then the receiving driver's
tile), or `aim_mass_driver(x, z, clear=true)` to stop it. A
driver holds fire while its target is full, so a stalled link usually means a full receiver. Cargo
Barges, Logistics Bays, Haulers and mobile stations come later.

**Bots and ships** move over the logistics network, which links every logistics building you own:

- A Ship Assembler pushes each finished bot to the **closest** place that will take it: a building
  with a free slot that accepts that bot type (a Mining Station for Miner Bots, a research station
  for its research bots, a Defense Platform for combat ships), a **Ship Yard**, or a player.
- A **Ship Yard** (Logistics Management research) is a buffer. Bots with nowhere to go wait there,
  and stations with an empty slot pull from the closest Ship Yard. Give every bot line a powered
  Ship Yard next to its Ship Assembler.
- With every slot full and no Ship Yard, the Ship Assembler stops and waits. That is normal, not a
  fault.
- **Stay away from running lines.** A player within about 60 tiles of a Ship Assembler counts as a
  place that will take its bots, and the bots go into your fleet instead of to the stations. When
  bots are "missing", check your own fleet first. Build the line, then leave it and check it from a
  distance.
- If crafted bots end up in your fleet anyway, give them to the right building with
  `give_ships`.

## 10. Research

- `research_status` shows the active tech and its progress, the queue (at most five) and what is
  learned. `research(tech, start_now)` queues a tech or starts it now.
- Tech names are exact in-game names with spaces: "Mining Logistics", "Ship Assembly",
  "Asteroid Science".
- A tech needs its prerequisites learned. Queue them in order. If the game refuses to queue a tech
  whose prerequisite is only queued, wait until the prerequisite finishes and queue it then.
- Each tech costs points from the shared bank. Many also require a **typed total**: a number of
  Asteroid, Planetary, Stellar or Black Hole Research points earned over the whole game. Costs scale
  with the world's tech cost setting, so read the real numbers from `research_status`.
- Points come from:
  - **objective rewards** (the early techs finish the moment you start them, from banked points),
  - a trickle of Asteroid Research while you mine by hand,
  - **research bots**. An Asteroid Research Bot on an Asteroid Research Station scans a nearby
    asteroid (about 2.5 points over its life). A Planetary Research Bot on a Planetary Research
    Station scans a planet (about 7.5 on a Terra World). Stellar Research Bots scan stars. A bot's
    points go into the shared bank and into its type's running total.
- **Research bots only work while a tech is active.** An empty queue wastes every bot you own. Keep
  the queue full.
- Research runs at the station's power satisfaction: a half-powered station researches at half
  speed.

The early spine, in order: Mining Logistics, Atomic Printing, Automation, Ship Assembly, Mass
Drivers, Asteroid Science. After that the tree fans out. The path to planetary research is Thermal
Management, then Advanced Solar Power, Mass Fuel Consumer, Mobile Stations, Overdrivers, Lasers,
Power Distribution, Singularity Generation, Comet Catching, and finally Planetary Science, which
also needs about 1,300 Asteroid Research earned (base). Logistics Management (the Ship Yard) is
cheap and worth getting early.

## 11. The later game

- **Ice and comets.** Comet Catching unlocks the Comet Catcher, which pulls passing comets in, and
  the Comet Harvester, which attaches to it and extracts Ice. A comet travels along the catcher's
  facing lane and **destroys anything in its path**. Never build, park or fly in that lane. Comet
  Fragments found in debris fields can be mined by hand for a first batch of Ice and Organic
  Compounds.
- **Organics.** A Greenhouse Station makes Organic Compounds from Ice and Biological Substrate,
  and substrate is made from Organic Compounds, so the chain needs a starting stock. Seed it from
  Comet Fragments. High Density Structure (3 Iron Ore, 2 Organic Compounds) and Organic Polymer
  unlock most mid-game buildings.
- **Planetary research.** Find a Terra World (the Derelict biome around the start usually has one),
  place a Planetary Research Station within 30 tiles of it, power it (it draws 30, so Advanced Solar
  Panels), cool it, and feed it Planetary Research Bots from a Ship Assembler.
- **Stars.** Find one (Nebula biomes). Stellar Science gives the Stellar Research Station and bot.
  Mega Structures and Dyson Spheres let you build a Dyson Platform around a main sequence star;
  Dyson Bots on it make power and Exotic Matter.
- **Mobile stations.** After Mobile Stations research, any station can lift off and fly slowly;
  Rocket Adapters make it faster and burn any item as fuel. A Command Core gives a station a stop
  schedule between named Landing Zones. `configure` covers stops and turning the schedule on.
- **Artifacts.** Oracles, obelisks and portals are activated by mining them. An Oracle sells
  upgrades (speed, health, reach and more) for Lumin Orbs, which the whole crew shares.

## 12. What a "mall" is here

A mall is a set of Assemblers that each make one building you keep placing (Connectors, Solar
Panels, Mining Stations, Assemblers, Cargo Holds, Struts) and push it on a filtered belt into a
Cargo Hold. You fly past, take what you need with `take_items`, and build. It replaces
hand-crafting buildings once you place them in bulk. The inputs come by belt from printers fed by
mining stations. The `build-a-mall` skill walks through one.

## 13. Combat and defence

- **Enemies** live in camps and spawners (Urso bases, Stingers, Cutters, Razors, Tridents).
  `enemies` lists what you can see. Camps far from every player go dormant and do not show up in a
  distant scan, so an empty scan far away proves nothing.
- **Your fleet fights for you.** Bats in your fleet engage enemies when you fly close. Eight Bats
  and the player's Plasma Bolt cleared the first tutorial camp. Build up Bats before flying into a camp.
- **Abilities**: Afterburner (a dash; `ability.afterburner` through `run_commands`), Plasma Bolt,
  and Frenzy, which needs four ready Bats per charge. Freshly crafted Bats take about two minutes
  before they count toward Frenzy.
- **Damage and death.** You regenerate after about 10 seconds without being hit; a constant trickle
  of fire keeps resetting that timer. If `game_status` shows health 0, you are dead: respawn with
  `respawn()` before anything else.
- **Defending the base.** A Defense Platform holds up to 20 combat ships and pulls them from a
  Ship Yard; a Bat Dock attached to a Defense Platform or Ship Yard keeps requesting Bats. Laser
  Turrets fire on nearby enemies using Capacitor charge. Automate Bat production in a Ship
  Assembler so defences refill themselves.
- A Repair Center attached to a station slowly heals every connected structure.

## 14. Winning

1. Research up to **Singularity Navigation**, which unlocks the Dark Star Gate and the Singularity
   Vessel. It needs Singularity Engines, Replication and Advanced Ship Assembly, and very large
   typed research totals.
2. Find a **black hole** (far out, beyond about 1,000 tiles). Travel there in stages and clear
   camps as they wake up.
3. Build a **Dark Star Gate** near it (21x43 tiles; 200 Fullerene Structure and 3 Advanced Ship
   Assembler).
4. Power the gate. It needs about 2,000 power to process a stage. How best to supply that is not
   documented here yet; check its power satisfaction in `look_around` and add generation to its
   grid until it reads 1.
5. Feed it five stages. Each stage takes 200 Fullerene Structure, 25 Singularity Engine and 10 Von
   Neumann Probe.
6. After stage five a Singularity Vessel launches and flies into the black hole. When it arrives,
   you win.

## 15. Using the tools well

- **Observe, act, verify.** Look before you act (`game_status`, `look_around`, `inventory`). After
  each action, confirm the result: a built structure, an item count, a power ratio. A tool
  returning "accepted" only means the game took the request.
- **Screenshots** show what snapshots cannot: unbuilt frames, attach diamonds, comets, enemies,
  warning icons, open panels. Take one every so often and before and after big builds, trips and
  fights. Check it agrees with the snapshots.
- **Waiting.** Long actions report "still running" with a chain id; use `wait_for`. To wait for a
  game condition, use `wait_until` with a predicate: `itemCount`, `structureBuilt`, `structureAt`,
  `powerSatisfaction`, `researchDone`, `objectiveComplete`, `taskQueueIdle`, `notification`. The
  timeout counts game time.
- **`mine(item, count)`** mines the nearest asteroid of that ore until you **hold** `count` in total.
  Fly next to the asteroid first.
- **`run_commands`** runs game commands that have no dedicated tool. `list_commands` shows every
  command you are allowed to use, with its arguments. Ones you will need:
  `ability.afterburner`, `rotate.abs` (turn a placed structure), `transfer.trash`, `game.save`
  (save names must start with `claude_playtest_`).
- **Refusals.** The game refuses anything a human player could not do: creating items from
  nothing, teleporting, skipping a cost or build time, acting out of reach, seeing more than the
  player can see. A refusal names the reason. Change your plan (fly closer, mine the ore, craft the
  item); never look for a way around it.

## 16. Common mistakes

1. **Placing a Solar Panel next to a station without connecting it.** The panel must point into
   the station through the station's attach slot. Verify the station's power satisfaction rose.
2. **Bridging two standard buildings by adjacency.** They need a Connector or Strut between them.
3. **Forgetting belt filters.** A new belt on a crafter is blocked until you set its filter.
4. **Under-powering.** Every building on a grid slows to the grid's power ratio. Check power first
   when a line looks stuck.
5. **Letting a grid go unstable.** Everything on it stops. Add a Station Core or Struts.
6. **Hand-crafting research bots at scale.** Build a Ship Assembler line instead.
7. **Standing next to your own Ship Assembler.** Its bots go into your fleet. Stay 60+ tiles away.
8. **Leaving the research queue empty.** Research bots stop working.
9. **Crafting Miner Bots far from the station.** They land in your fleet. Give them to the station.
10. **Placing a frame and flying off.** It stays a frame until you return with the item and a
    Construction Bot.
11. **Building in a Comet Catcher's lane.** Comets destroy what they hit.
12. **Ignoring heat.** Overheated structures take damage and can be destroyed.
13. **Trusting a distant enemy scan.** Dormant camps wake up when you arrive.
14. **Running out of iron.** Medium Density Structure is in nearly every recipe. Automate iron
    early and keep a spare Mining Station on an iron asteroid.

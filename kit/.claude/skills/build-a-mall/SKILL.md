---
name: build-a-mall
description: Build a "mall" in Final Factory - Assemblers that automatically make the buildings the player keeps placing (Connectors, Solar Panels, Mining Stations, Cargo Holds, Assemblers and similar) and store them in Cargo Holds to take from. Use when the player says "build a mall", "automate building production", "stop hand-crafting buildings", "make connectors/solar panels automatically", or when you are about to hand-craft the same building many times.
gameVersion: "0.50"
commands: [game_status, research_status, research, inventory, look_around, screenshot, move_to, craft, build, deconstruct, take_items, put_items, set_recipe, configure, wait_until, run_commands]
---

# Build a mall

**What a mall is in Final Factory:** a block of Assemblers, each set to one building you place
often, each pushing its product on a filtered belt into its own Cargo Hold. Iron and bauxite come
in by belt from Mining Stations through Atomic Printers. When you need buildings, you fly to the
mall and `take_items` them from the Cargo Holds instead of hand-crafting them.

Bots and ships (Construction Bots, Miner Bots, Bats) come from Ship Assemblers, not Assemblers, and
fly off to whoever takes them rather than into a Cargo Hold. They are not part of this mall; see
`automate-research` for the bot line pattern.

## Preconditions

1. `research_status`: "Atomic Printing" and "Automation" learned (Atomic Printer and Assembler).
   "Junctions" helps (the Junction splits belts). Otherwise use `research-to`.
2. A Mining Station on an **Iron** asteroid and one on a **Bauxite** asteroid, powered, with
   miners. If not, run `starter-mining-outpost` first. More iron is better: Medium Density
   Structure (2 Iron Ore) is in every building.
3. Materials to hand-craft the mall itself: each Atomic Printer is 24 Iron Ore and 8 Bauxite Ore
   raw; each Assembler 32 Iron Ore and 16 Bauxite Ore. Check `inventory(craftable=true)`.
4. Ask the player which buildings they want. If they don't say, start with **Connector** and
   **Solar Panel**, then **Mining Station**.

## Recipes this mall uses

| Product | Made by | Inputs |
|---|---|---|
| Medium Density Structure | Atomic Printer | 2 Iron Ore |
| Low Density Structure | Atomic Printer | 2 Bauxite Ore |
| Robotic Parts | Assembler | 1 Medium Density Structure |
| Fabricator | Assembler | 2 Robotic Parts, 2 Low Density Structure |
| Connector | Assembler | 4 Low Density Structure, 1 Medium Density Structure |
| Solar Panel | Assembler | 4 Low Density Structure, 2 Medium Density Structure |
| Cargo Hold | Assembler | 8 Medium Density Structure |
| Strut | Assembler | 4 Medium Density Structure |
| Mining Station | Assembler | 8 Medium Density Structure, 4 Robotic Parts |
| Assembler / Ship Assembler | Assembler | 8 Medium Density Structure, 4 Fabricator |
| Atomic Printer | Assembler | 8 Medium Density Structure, 2 Fabricator |

Confirm any other building's recipe in the game before you plan around it.

## Layout

Two input belts run past the mall, one carrying Medium Density Structure and one carrying Low
Density Structure:

1. Iron Mining Station, belt, Atomic Printer set to "Medium Density Structure", output belt
   filtered to "Medium Density Structure".
2. Bauxite Mining Station, belt, Atomic Printer set to "Low Density Structure", output belt
   filtered to "Low Density Structure".

Each product Assembler takes a branch from one or both input belts (a Junction splits a belt
evenly), and has one output belt filtered to its product, ending in a Cargo Hold. Leave at least
one clear tile between Assemblers so belts and Solar Panels fit, and so you can add more later.

## Steps

1. **Input printers.** Build both printer chains above next to their Mining Stations.
   `set_recipe(recipe="Medium Density Structure", x=..., z=...)` and
   `set_recipe(recipe="Low Density Structure", x=..., z=...)`. Set the filters on the belt tiles
   touching each printer with `configure(kind="filter_item", value=..., x=..., z=...)`. Power each printer
   (5 each; one Solar Panel pointing into it covers it). Verify with `look_around`: each printer
   shows `crafting ...` and power satisfaction 1.
2. **First product: Connector.** Place an Assembler where both input belts can reach it, bring a
   branch of each in, and `set_recipe(recipe="Connector", ...)`. Set filters on both input tiles
   ("Medium Density Structure", "Low Density Structure"). Add an output belt pointing away,
   filtered to "Connector", ending in a Cargo Hold. Power: the Assembler draws 10, so two Solar
   Panels, plus 1 per belt tile and Cargo Hold.
3. **Verify.** Wait a couple of game minutes, then `take_items(item="Connector", count=1, x=<cargo
   hold x>, z=<cargo hold z>)`. It should succeed. If not, walk the chain with `look_around`:
   printers crafting? Assembler crafting? Belt filters set? Power 1?
4. **More products.** Repeat step 2 for each building the player wants. Products that need an
   intermediate get their own Assembler for it:
   - **Mining Station** needs Robotic Parts: add an Assembler set to "Robotic Parts" fed by the
     Medium Density Structure belt, and belt its output (filtered "Robotic Parts") into the Mining
     Station Assembler alongside Medium Density Structure.
   - **Assembler, Ship Assembler, Atomic Printer** need Fabricators: an Assembler set to
     "Fabricator" fed with Robotic Parts and Low Density Structure.
5. **Power and stability.** After each addition, `look_around` over the whole mall. Every crafter
   should read power satisfaction 1 and the grid should be stable (stability `needed` below what
   the grid has). A growing mall needs a Station Core joined to it by a Strut or Connector, and
   more Solar Panels.
6. **Prove it runs alone.** Fly away for a few game minutes, come back, and check each Cargo Hold
   has more than before.
7. **Use it.** From now on, before hand-crafting a building, check the mall:
   `take_items(item=..., count=..., x=..., z=...)` from the matching Cargo Hold.

## Failure modes

- **Assembler shows no recipe progress**: an input belt tile is still blocked, or one ingredient is
  not arriving. Every belt tile touching a crafter needs a filter.
- **Wrong item in a Cargo Hold**: the output belt's filter is missing or wrong. A crafter's belts
  draw from a shared zone that holds its ingredients too.
- **Cargo Hold full**: the line stops. That is fine; take items out or add a second hold further
  along the belt.
- **Iron starved**: one iron station cannot feed many Assemblers. Add Mining Stations on iron
  asteroids feeding the same Medium Density Structure belt.
- **Everything slow**: power below 1. Add panels, and make sure each one actually joined the grid.
- **Unstable grid**: add a Station Core or Struts.
- **`take_items` refused**: you are out of reach. Fly within about 20 tiles.

Report to the player: where the mall is, which Cargo Hold holds which building, and what rate you
saw.

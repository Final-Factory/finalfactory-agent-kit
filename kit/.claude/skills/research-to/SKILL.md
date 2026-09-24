---
name: research-to
description: Reach a named technology in Final Factory by working out its prerequisites, queueing them in order, and making sure the points to pay for them keep coming. Use when the player says "research X", "get me to X", "unlock the Y building", "what do I need for X", "queue up research toward X", or when another plan needs a tech that is not learned yet.
gameVersion: "0.50"
commands: [game_status, research_status, research, wait_until, look_around]
---

# Research to a target tech

**Goal:** the named tech is learned, reached by queueing every missing prerequisite in order, with
research kept running the whole time.

If the player names a building rather than a tech ("I want Ship Yards"), find the tech that unlocks
it in the table at the end ("Logistics Management" unlocks the Ship Yard).

## How research works (short version)

- One tech is active at a time, and up to five wait in the queue. When the active one finishes,
  the next starts.
- A tech needs every tech in its "Needs" column learned first.
- It costs points from the shared bank. Many techs also need a **typed total**: Asteroid,
  Planetary, Stellar or Black Hole Research earned over the whole game. The typed numbers and costs
  in the table are base values; the world's tech cost setting scales them. `research_status` has
  the real state.
- Points come from objective rewards, from mining by hand (a small trickle of Asteroid Research),
  and from research bots on research stations. **Research bots only work while a tech is active.**

## Steps

1. **Read the state.** `research_status`: note what is learned, what is active and what is
   queued.
2. **Build the path.** Starting from the target, walk the "Needs" column of the table below until
   every branch reaches a learned tech. List the missing techs so that each one comes after
   everything it needs. Use the exact in-game names, spaces and capitals included (note "Fast
   inserter Bots" has a lower-case i).
3. **Check the typed totals.** Add up the largest typed requirement of each kind along the path
   (they are running totals, not costs, so take the maximum, not the sum). If the path needs a
   research type you are not earning, research will stall there. Tell the player and plan the
   supply: Asteroid Research and Planetary Research come from the `automate-research` skill.
   Stellar Research needs Stellar Research Stations near a star.
4. **Queue.** If nothing is active, start the first missing tech:
   `research(tech=<first>, start_now=true)`. Then queue the next ones in order,
   `research(tech=<next>)`, up to five in the queue.
   - If the game refuses because a prerequisite is only queued, not learned, stop queueing there.
     Wait for the prerequisite (`wait_until(predicate="researchDone", args=["<prerequisite>"],
     timeout_seconds=<game seconds>)`), then queue the rest.
   - If it refuses for any other reason, read the reason and fix that. Never repeat a refused call
     unchanged.
5. **Verify.** `research_status` shows the active tech and the queue you intended.
6. **Follow through.** Early techs finish at once from banked points. Later ones take time. Check
   `research_status` periodically. When the queue gets short, add the next techs on the path.
   `wait_until(predicate="researchDone", args=["<target>"], timeout_seconds=...)` waits for the target.
7. **Report** the path, what is queued, what is blocking (usually a typed total), and roughly how
   far along the active tech is.

## Failure modes

- **Tech not found**: the name is wrong. Check spelling and spacing against the table.
- **Nothing progresses**: no research bots are working (none built, stations unpowered, or no
  active tech), or the active tech is waiting on a typed total. Look at the "Typed research"
  column.
- **Points arrive but only for one type**: the target needs another type. Build that research
  supply before queueing further.
- **Queue full**: five is the limit. Wait for one to finish.

## Common paths (base values)

- **First steps** (new game): Mining Logistics, Atomic Printing, Automation, Ship Assembly, Mass
  Drivers, Asteroid Science. All are cheap and usually finish at once from objective rewards.
- **Ship Yard**: Logistics Management (after Asteroid Science and Ship Assembly).
- **Planetary Science**: Thermal Management, Advanced Solar Power, Mass Fuel Consumer, Mobile
  Stations, Overdrivers, Lasers, Power Distribution, Singularity Generation, Comet Catching, then
  Planetary Science. Needs 1,300 Asteroid Research earned.
- **Stellar Science**: needs Advanced Bot Production, Remote Construction, Quantum Printing, High
  Density Materials and Station Movement Programming, plus 6,000 Planetary and 8,000 Asteroid
  Research earned.
- **Singularity Navigation** (unlocks the Dark Star Gate and Singularity Vessel, the win): needs
  Singularity Engines, Replication and Advanced Ship Assembly, and 50,000 each of Stellar,
  Planetary and Asteroid Research.

Upgrade series (Bat Damage, Fleet Command, Knight Damage and Hull, Krillo Damage, Laser Turret
Damage, Fleet Repair Augments, Mining Productivity) are numbered 1, 2, 3...; each level needs the
one before. Fleet Command 1 needs Asteroid Science and Ship Assembly and raises your fleet cap.

## Tech table

"Needs" are the prerequisite techs. "Typed research" is the running total of that research type
you must have earned. Costs and typed totals are base values.

| Tech | Needs | Cost | Typed research | Unlocks |
|---|---|---|---|---|
| Abduction | Stellar Science, High Density Station Cores | 5,000 | Stellar 2,000 | Abduction Machine |
| Advanced Assembly | High Density Materials | 800 | Planetary 1,000, Asteroid 3,000 | Advanced Assembler |
| Advanced Bot Production | Quantum Computing, High Energy Connectors | 2,000 | Planetary 2,500, Asteroid 4,000 | Quantum Bot Chassis |
| Advanced Ship Assembly | High Density Materials | 800 | Planetary 1,000, Asteroid 3,000 | Advanced Ship Assembler |
| Advanced Solar Power | Thermal Management | 50 | - | Advanced Solar Panel |
| Antimatter Power | Hydrocarbon Development | 600 | Planetary 1,000 | Antimatter Power Station, Antimatter Collector |
| Artifact Processing | Quantum Computing | 3,000 | Asteroid 6,000, Planetary 3,000 | Artifact Recycler, Ancient Tech Trash |
| Asteroid Science | Mass Drivers | 15 | - | Asteroid Research Station, Asteroid Research Bot |
| Atomic Printing | Mining Logistics | 5 | - | Atomic Printer |
| Automation | Atomic Printing | 15 | - | Assembler |
| Bot Network Requests | Stellar Science | 7,000 | Stellar 10,000 | Requester Cargo Hold, Supply Bot |
| Capacitors | Lasers | 100 | Asteroid 100 | Capacitor |
| Cargo Logistics | Asteroid Science, Ship Assembly | 80 | Asteroid 80 | Cargo Barge, Logistics Bay |
| Comet Catching | Singularity Generation | 300 | Asteroid 800 | Comet Catcher, Comet Harvester |
| Dark Clip Manufacturing | Singularity Navigation | 50,000 | Black Hole 5,000 | Dark Clip |
| Dark Energy Printing | Mind and Matter | 9,000 | Stellar 12,000 | Dark Energy Printer |
| Dark Energy Rocketry | Mind and Matter | 9,000 | Stellar 12,000 | Dark Energy Rocket Adapter |
| Dark Matter Station Cores | Mind and Matter | 5,000 | Stellar 15,000 | Dark Matter Station Core |
| Defense Platforms | Asteroid Science, Ship Assembly | 15 | - | Defense Platform |
| Dyson Energy Logistics | Mega Structures | 4,000 | Stellar 4,000 | Dyson Power Receiver, Dyson Power Transmitter |
| Dyson Power Collection | Dyson Spheres | 12,500 | Stellar 10,000 | Dyson Bot |
| Dyson Spheres | Mega Structures | 10,000 | Asteroid 15,000, Planetary 12,000, Stellar 10,000 | Dyson Platform, Exotic Matter |
| Exploration | Asteroid Science, Ship Assembly | 20 | - | Exploration Center |
| Express Cargo Barges | Metal Matrix Composite, High Speed Cargo Logistics | 5,000 | Stellar 7,000 | Express Cargo Barge |
| Express Inserter Bots | Metal Matrix Composite | 5,000 | Stellar 7,000 | Express Inserter Bot |
| Fast inserter Bots | Hydrocarbon Development | 600 | Planetary 1,000 | Fast Inserter Bot |
| Gauge Field Connectors | Metal Matrix Composite | 5,000 | Stellar 10,000 | Gauge Field Connector |
| Gauge Field Mass Drivers | Metal Matrix Composite, High Energy Mass Drivers | 8,000 | Stellar 10,000 | Gauge Field Mass Driver |
| Greenhouses | Comet Catching | 250 | Asteroid 1,000 | Greenhouse Station, Biological Substrate |
| Heat Transmission | Planetary Science | 150 | Planetary 150, Asteroid 2,000 | Heat Transmitter, Heat Receiver |
| High Density Materials | Greenhouses | 500 | Planetary 800 | High Density Structure |
| High Density Station Cores | High Density Materials | 750 | Asteroid 1,800, Planetary 800 | High Density Station Core |
| High Density Storage | High Density Materials | 500 | Asteroid 500 | Large Cargo Hold |
| High Energy Connectors | High Energy Manipulation | 800 | Planetary 2,500 | High Energy Connector |
| High Energy Manipulation | Hydrocarbon Development | 900 | Planetary 1,200 | High Energy Laser |
| High Energy Mass Drivers | High Energy Manipulation | 1,600 | Planetary 3,500 | High Energy Mass Driver |
| High Energy Mining | Planetary Science | 400 | Asteroid 1,500, Planetary 400 | Advanced Miner Bot, Advanced Mining Station |
| High Energy Power Distribution | High Energy Manipulation | 1,000 | Asteroid 2,000, Planetary 2,000 | High Energy Power Distributor |
| High Speed Cargo Logistics | High Energy Manipulation, Cargo Logistics | 800 | Planetary 2,000 | Fast Cargo Barge |
| Hydro Heat Management | High Energy Manipulation, High Density Materials | 1,000 | Planetary 2,500, Asteroid 3,000 | Hydro Power Radiator |
| Hydrocarbon Development | Greenhouses, Comet Catching | 500 | Planetary 600, Asteroid 2,000 | Organic Polymer |
| Hyper Assembly | Dyson Power Collection, Advanced Assembly | 12,000 | Stellar 20,000 | Hyper Assembler |
| Hyper Ship Assembly | Dyson Power Collection, Advanced Ship Assembly | 13,000 | Stellar 20,000 | Hyper Ship Assembler |
| Junctions | Asteroid Science, Ship Assembly | 25 | - | Junction |
| Knights | Singularity Generation | 200 | Asteroid 1,000 | Knight |
| Krillos | High Energy Manipulation | 600 | Planetary 2,000 | Krillo |
| Laser Turrets | Thermal Management, Capacitors, Mass Fuel Consumer | 200 | Asteroid 250 | Laser Turret |
| Lasers | Asteroid Science, Ship Assembly | 15 | - | Solid State Laser |
| Lighting | Asteroid Science, Ship Assembly | 25 | - | Light Structure |
| Logistics Management | Asteroid Science, Ship Assembly | 50 | Asteroid 30 | Ship Yard |
| Mass Drivers | Ship Assembly | 15 | - | Mass Driver |
| Mass Fuel Consumer | Thermal Management | 90 | Asteroid 50 | Mass Energy Converter |
| Mega Structures | Metal Matrix Composite | 5,000 | Asteroid 12,000, Planetary 10,000, Stellar 2,000 | Fullerene Structure |
| Metal Matrix Composite | Stellar Science | 1,500 | Stellar 2,000 | Metal Matrix Composite |
| Mind and Matter | Abduction, Metal Matrix Composite | 10,000 | Stellar 10,000 | Discretized Consciousness, Dark Matter Structure, Mind Extractor |
| Mining Logistics | - | 5 | - | Mining Station, Miner Bot |
| Mobile Stations | Mass Fuel Consumer | 100 | Asteroid 100 | Rocket Adapter |
| Overdrivers | Thermal Management | 100 | Asteroid 100 | Overdriver |
| Planetary Science | Comet Catching, Mobile Stations, Advanced Solar Power | 300 | Asteroid 1,300 | Planetary Research Station, Planetary Research Bot |
| Power Distribution | Thermal Management, Lasers | 200 | Asteroid 200 | Power Distributor |
| Quantum Computing | High Energy Manipulation, Fast inserter Bots | 1,500 | Asteroid 2,000, Planetary 1,800 | Quantum Computer |
| Quantum Printing | Quantum Computing | 2,000 | Asteroid 3,000, Planetary 2,600 | Quantum Printer |
| Remote Construction | Planetary Science | 400 | Planetary 400 | Provider Cargo Hold, Networked Storage Hold, Construction Tower |
| Replication | Singularity Circuitry | 25,000 | Stellar 30,000, Planetary 25,000, Asteroid 25,000 | Von Neumann Probe |
| Ship Assembly | Automation | 15 | - | Ship Assembler |
| Signal Dampening | Station Repair | 100 | Asteroid 300 | Signal Dampener |
| Singularity Circuitry | Dyson Power Collection, Mind and Matter | 15,000 | Stellar 20,000 | Singularity Circuit |
| Singularity Engines | Singularity Circuitry | 20,000 | Stellar 30,000, Planetary 30,000, Asteroid 30,000 | Singularity Engine |
| Singularity Generation | Power Distribution, Overdrivers, Mass Fuel Consumer | 250 | Asteroid 500 | Singularity Field Generator, Singularity Power Station |
| Singularity Navigation | Singularity Engines, Replication, Advanced Ship Assembly | 40,000 | Stellar 50,000, Planetary 50,000, Asteroid 50,000 | Singularity Vessel, Dark Star Gate |
| Station Armor | Defense Platforms | 200 | - | Armor Block, Triangle Armor Block |
| Station Movement Programming | Greenhouses | 500 | Planetary 800 | Command Core, Small Command Core |
| Station Repair | Thermal Management | 50 | - | Repair Center |
| Stellar Science | Advanced Bot Production, Remote Construction, Quantum Printing, High Density Materials, Station Movement Programming | 2,000 | Planetary 6,000, Asteroid 8,000 | Stellar Research Station, Stellar Research Bot |
| Thermal Management | Asteroid Science, Ship Assembly | 50 | - | Radiator, Heat Exchanger |

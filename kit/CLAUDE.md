# Final Factory agent kit

You are playing Final Factory for the person who opened you in this folder. The game is running
right now on their machine with **Agent control** turned on, and you act as their player through
the `finalfactory` MCP tools. You are not editing code. There is no code here to edit.

## Before you start

1. Call `game_status`. If it says agent control is off or no game is running, tell the player to
   start or load a game and turn on Agent control from the pause menu, then stop.
2. If the player's health is 0, you are dead: `respawn()` first.
3. Read `HowToPlay.md` in this folder before any plan longer than a few steps. It explains the
   game's systems and the mistakes earlier agents made. The same guide is served by the MCP server
   as the resource `finalfactory://guide/how-to-play`.

## How to act: observe, act, verify

- **Observe** with `game_status`, `look_around`, `inventory`, `research_status`, `enemies` and
  `screenshot`. Take a screenshot every so often and before and after big builds, trips and fights;
  it shows things the snapshots do not.
- **Act** with the action tools: `move_to`, `mine`, `craft`, `build`, `deconstruct`, `put_items`,
  `take_items`, `set_recipe`, `configure`, `research`, `ui`. For anything else, `list_commands`
  shows every game command you may use and `run_commands` runs them.
- **Verify** every action against the game, not against the tool's reply. "Accepted" means the game
  took the request. Check the structure is `built`, the item count went up, the power ratio is
  where you expect. Use `wait_until` for game conditions and `wait_for` for a long action that
  reported "still running".

## Coordinates

Every tool uses **tiles**, as `(x, z)`. `x` grows east, `z` grows north. A structure's tile is its
lower-left corner. Your reach is about 20 tiles, so fly close before you take, put or mine.
`build` takes a building's in-game name (or a copied blueprint string) and reports the tile the
building actually landed on. Use `inspect` to see what a building holds or is doing.

## Play honestly

- The game refuses anything a human player could not do: creating items from nothing, teleporting,
  skipping a cost or build time, acting out of reach, or seeing more of the map than the player
  has. Travel is always flown.
- A refusal says why. Treat it as information about the game and change your plan (fly closer,
  mine the ore, craft the part). Never try to get around a refusal, and never retry the same
  refused action unchanged.
- Act only inside the game. Do not read, write or run anything outside the game, do not touch
  save files, settings or other programs, and do not go to the internet on the player's behalf.
- The player can take the controls back at any time. If they start moving or building
  themselves, stop and ask what they want.
- Only save the game when the player asks or agrees. Save names must start with
  `claude_playtest_`.

## Skills

The skills in `.claude/skills/` are plans for common goals. Use the one that matches what the
player asked for:

- `play-the-game`: open-ended play; picks the next goal from the game state.
- `starter-mining-outpost`: a powered Mining Station with miners and an output belt.
- `automate-research`: Ship Assembler lines that keep research stations full, for asteroid and
  planetary research.
- `build-a-mall`: Assemblers that make the buildings you keep placing, into Cargo Holds.
- `research-to`: reach a named technology by queueing its prerequisites.
- `scale-research`: grow asteroid, planetary and stellar research toward the late typed totals.
- `defend-base`, `reveal-map`, `manage-stability-power`, `fleet-logistics`: defence against
  raids, lifting the fog, stable powered grids, and ship lines and yards.
- `endgame-victory`: the Dark Star Gate and the Singularity Vessel.

A skill's steps are a plan, not a script. Check each precondition it lists against the live game,
and adapt when the game disagrees.

## Talking to the player

- Report progress in plain language: what you did, what you saw, what you will do next. Keep it
  short. Give coordinates when you build something they may want to find.
- Say when you are blocked and why, and what you tried.
- Ask before anything large or hard to undo: deconstructing a working line, spending most of a
  scarce resource, flying far from the base, attacking an enemy camp.
- Automate rather than hand-craft. If the player asks for something that only makes sense as a
  one-off, hand-crafting is fine; for anything ongoing, build a line.

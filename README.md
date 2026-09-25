# Final Factory Agent Kit

Let Claude Code, or any agent that speaks MCP, play [Final Factory](https://store.steampowered.com/app/1383150/Final_Factory/) as you.

The kit ships inside the game. Turn on **Agent Control** in the pause menu, open your agent in
the folder the game shows you, and tell it what you want: "play the game", "set up a mining
outpost", "automate planetary research", "build a mall".

This repository is where the kit is developed in the open: the `ff-agent` command-line tool and
MCP server, the player guide the agent reads, and the skills it uses. Read it to see exactly
what an agent can and cannot do in your game.

## Getting started

1. Load a game, open the pause menu and turn on **Agent Control** (bottom left). Settings >
   Interface > Agent Control shows the kit folder (with Copy path and Open folder buttons) and
   the command that starts the MCP server.
2. Open [Claude Code](https://claude.com/claude-code) in that folder. The folder's `.mcp.json`
   registers the `finalfactory` MCP server, so there is nothing else to install.
3. Say what you want, for example "play the game" or "set up a mining outpost".

Any other agent that speaks MCP over stdio works too: point it at `ff-agent mcp` in the kit
folder (`ff-agent.exe mcp` on Windows). Keep the game window un-minimised while an agent plays;
on Windows a minimised game renders no frames, so screenshots are refused.

Turn the toggle off at any time and the agent loses its connection. While it is on, a chip in
the game's HUD shows that the channel is open and whether an agent is attached.

## What an agent can and cannot do

The agent plays through the same rules as you. The game checks every request before it runs:

- It can fly, mine, craft, build, deconstruct, move items and ships in and out of buildings, set
  recipes, queue research, use the game's panels, and take screenshots.
- It cannot create items, teleport, skip costs or build times, act beyond your ship's reach, or
  see map areas you have not revealed. The game refuses those requests and says why.
- It cannot use the game's development and testing commands. `kit/commands.md` lists every
  command the game accepts from an agent; the game refuses the rest.
- It can save only under names that start with `claude_playtest_`, so it cannot overwrite your
  saves.
- The `ff-agent` binary talks only to the game on `127.0.0.1` and writes only a screenshot file
  you ask it for. It has no file, shell or network access beyond that.

## What is in the kit

| File | Purpose |
|---|---|
| `ff-agent` / `ff-agent.exe` | the command-line tool and the MCP server (`ff-agent mcp`) |
| `.mcp.json` | registers `ff-agent mcp` with Claude Code |
| `CLAUDE.md` | the agent's entry point: how to observe, act and verify, and the honest-play rules |
| `HowToPlay.md` | the game guide agents read before they plan |
| `commands.md` | every game command an agent may send, generated from the game |
| `.claude/skills/` | plans for common goals: `play-the-game`, `starter-mining-outpost`, `automate-research`, `build-a-mall`, `research-to`, `scale-research`, `defend-base`, `reveal-map`, `manage-stability-power`, `fleet-logistics`, `endgame-victory` |

The MCP server's tools use tile coordinates and in-game names: `game_status`, `look_around`,
`inventory`, `research_status`, `enemies`, `inspect`, `screenshot` (returned as an image),
`move_to`, `mine`, `build`, `deconstruct`, `craft`, `research`, `put_items`, `take_items`,
`set_recipe`, `configure`, `wait_until`, and `run_commands` for anything else in `commands.md`.

## The command line

`ff-agent` also works on its own, for scripts or for checking the connection:

```sh
ff-agent hello                 # game version, state, and whether the game's commands match the kit
ff-agent snapshot player       # your ship (--json for machine-readable output)
ff-agent screenshot -o shot.png
ff-agent cmd "ffauto:mining.startnearest" --wait
```

Exit codes: 0 ok, 2 Agent control is off or no game is running, 3 unauthorised, 4 the game
refused the command, 5 timeout, 6 the game's commands differ from the kit's (`hello --strict`),
1 anything else.

## Contributing

Skills and guide fixes are the most useful contributions. See [CONTRIBUTING.md](CONTRIBUTING.md)
for the skill format and the review checklist every skill passes before it ships to players.

## For developers

The `ff-agent` binary is Go (module at the repo root, so `kit/` can be embedded). It needs Go
1.25+ and nothing else; the only dependency is the official MCP Go SDK.

```sh
go build ./... && go vet ./... && go test ./...   # what CI runs (plus -race and cross-builds)
go run ./cmd/ff-agent hello                        # against a running game with Agent control on
scripts/build-kit.sh                               # dist/AgentKit-<ver>-{win,mac}.zip, no tag needed (macOS)
```

Releases: bump the first line of `kit/VERSION`, then push the tag `v<that version>`;
`.github/workflows/release.yml` builds and publishes both zips with the same script.

| Path | What it is |
|---|---|
| `kit/` | the folder players get: `.mcp.json`, `CLAUDE.md`, `HowToPlay.md`, `commands.md`, `.claude/skills/`, `VERSION` — embedded into the binary as MCP resources and prompts |
| `cmd/ff-agent` | entry point |
| `internal/discovery` | finds the game's `AgentControl/session-{pid}.json` and checks the pid is alive |
| `internal/channel` | the loopback `/v1` HTTP client, with re-discovery when the game restarts |
| `internal/cli` | the command line (`hello`, `cmd`, `chain`, `snapshot`, `screenshot`, …) |
| `internal/mcpserver` | `ff-agent mcp`: the tools, resources, prompts and server instructions |
| `internal/kitdocs` | reads the embedded kit documents |
| `internal/fakegame` | an in-process fake of the game's channel, for tests only |

The binary talks only to `127.0.0.1` and writes only the PNG you ask `ff-agent screenshot -o` for.

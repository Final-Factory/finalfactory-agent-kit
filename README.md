# Final Factory Agent Kit

Let Claude Code, or any agent that speaks MCP, play [Final Factory](https://store.steampowered.com/app/1383150/Final_Factory/) as you.

The kit ships inside the game. Turn on **Agent control** from the pause menu, open your agent in
the folder the game shows you, and tell it what you want: "play the game", "set up a mining
outpost", "automate planetary research", "build a mall".

This repository is where the kit is developed in the open: the `ff-agent` command-line tool and
MCP server, the player guide the agent reads, and the skills it uses. Read it to see exactly
what an agent can and cannot do in your game.

Work in progress. Full README to follow with the first release.

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

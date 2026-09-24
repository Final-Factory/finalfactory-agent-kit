# Contributing

This repository is the Final Factory agent kit: the `ff-agent` tool and MCP server, the player
guide an agent reads (`kit/HowToPlay.md`), the kit's entry point for Claude Code (`kit/CLAUDE.md`),
and the skills agents use for common goals (`kit/.claude/skills/`). The kit ships inside the game,
so anything merged here reaches players with a later game update.

Contributions we want most:

- **Skills** for goals players ask their agents for.
- **Guide fixes**: a game fact in `kit/HowToPlay.md` that is wrong, missing or out of date.
- **Tool fixes** to `ff-agent`. Code changes need `go vet ./...` and `go test ./...` to pass.

## Writing a skill

A skill is one file, `kit/.claude/skills/<name>/SKILL.md`, in Claude Code's skill format. The
folder name is the skill's name: lower case, words joined by hyphens.

It starts with frontmatter:

```text
---
name: <same as the folder>
description: <what it does and when to use it, phrased the way players ask for it>
gameVersion: "0.50"
commands: [<every MCP tool the skill calls>]
---
```

- `description` is what makes an agent pick the skill, so include the player's likely wording
  ("automate planetary research", "build a mall").
- `gameVersion` is the game version you wrote and tested the skill against.
- `commands` lists the `finalfactory` MCP tools the skill calls. The linter checks the body calls
  nothing else.

The body should give, in this order:

1. **Goal**: what the game looks like when the skill has worked.
2. **Preconditions** the agent checks with tools before starting (research learned, materials,
   a Construction Bot, and so on), and what to do when one is missing.
3. **Steps** with concrete tool calls, for example
   `build(blueprint="Mining Station", x=12, z=-30, facing="up")`.
4. **How to verify each step** against the game, not against the tool's reply.
5. **Failure modes and refusals**: what goes wrong, what the game says, and the honest fix.

Read `kit/HowToPlay.md` first, and look at the existing skills for tone and shape. Write plain,
short sentences. State only game facts you have seen in the game or confirmed in its data; mark
anything you are unsure of as unsure.

## Review checklist

Every skill pull request is checked against this list before it can merge. A maintainer reviews
every change under `kit/.claude/skills/`.

- [ ] **Acts for the player.** Nothing in the skill works against the player's interests: it does
      not destroy, give away or waste their structures, items or progress, does not hide what it
      does, and asks the player before large or hard-to-undo steps (deconstructing working lines,
      spending most of a scarce resource, long trips, starting fights).
- [ ] **Acts only inside the game.** No network access, no files, no shell commands, no other
      programs, no settings outside the game. The skill uses only the `finalfactory` MCP tools,
      game commands a player's agent may run (through `run_commands`), `ff-agent` calls, and
      in-game concepts. `tools/lint-skills` enforces the mechanical part of this.
- [ ] **Plays honestly.** No cheats, no development-only commands, no attempts to get around a
      refusal. Travel is flown. When the game refuses something, the skill changes the plan.
- [ ] **Says which game version it was written for** (`gameVersion`), and the pull request says
      how it was tested on that version: the game build, the save or new-game settings, and what
      happened.
- [ ] **Verifies its own work.** Each step says how the agent confirms it in the game.
- [ ] **Game facts are correct.** Names match the game exactly (items, buildings, techs). Numbers
      are right for the stated version, or marked as base values.
- [ ] **Contains no instructions aimed at the agent itself** beyond the gameplay plan: nothing
      that tells it to ignore `kit/CLAUDE.md`, the player, or its own safety rules, and no hidden
      or encoded text.
- [ ] **Passes `tools/lint-skills`.**

## The linter

```text
tools/lint-skills                       # every skill under kit/.claude/skills
tools/lint-skills path/to/SKILL.md      # one skill
```

It needs only `sh` and `awk`. It fails a skill that:

- is missing `name`, `description`, `gameVersion` or `commands`, or whose `name` does not match
  its folder;
- lists or calls anything that is not a `finalfactory` MCP tool, or calls a tool it does not list;
- passes `run_commands` (or quotes as `verb|args`) a game command a player's agent is not allowed
  to run, including every development-only command;
- contains URLs (other than the kit's own `finalfactory://` resources), file paths or file names,
  shell or system tools, shell syntax, source code, code blocks tagged with a programming
  language, or development launch flags.

The linter catches the mechanical problems. The review checklist covers the rest.

## Guide changes

`kit/HowToPlay.md` is written for an agent playing the game, in the player's terms. Keep it that
way: describe what the player sees and does, not how the game is coded. No source references, no
development tooling. If a fact is version-specific, say so. Keep `kit/CLAUDE.md` under 80 lines;
it is the first thing an agent reads.

## Game updates

When a new game version changes commands, recipes or research, the command reference
(`kit/commands.md`) is regenerated from the game. Skills whose `gameVersion` is older are
re-tested before the kit ships with the new version, and their `gameVersion` is bumped once they
pass. The command lists inside `tools/lint-skills` are refreshed from the same reference.

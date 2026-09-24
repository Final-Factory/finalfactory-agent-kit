# ffauto command reference

> Game version: 0.50.0.24
>
> **Generated file — do not hand-edit.** Edit the XML docs in the runner, then regenerate.
>
> - Generator: `python scripts/generate-ffauto-reference.py`
> - Source: `Assets/Scripts/Behaviours/Multiplayer/LocalMultiplayerAutomationCommandRunner.cs`
> - Commands: 66 total — 51 documented, 15 undocumented.
> - Tier: `player` — only the commands `Assets/Scripts/Behaviours/Multiplayer/CommandTiers.cs` classifies `PlayerSafe`. Development fixtures, cheats and desync injectors are omitted here and refused by the game's agent channel.

Commands are typed into the automation entry points as `ffauto:<family>.<name>|<arg>|<arg>`; `ffauto:help` (or `ffauto:help|<family>`) prints the same catalog from inside a running game.

## Families

- [ability](#ability) — 1 command(s)
- [assembler](#assembler) — 1 command(s)
- [blueprint](#blueprint) — 2 command(s)
- [construction](#construction) — 6 command(s)
- [fleettransfer](#fleettransfer) — 2 command(s)
- [game](#game) — 1 command(s)
- [hauler](#hauler) — 2 command(s)
- [help](#help) — 1 command(s)
- [mining](#mining) — 1 command(s)
- [mobilestation](#mobilestation) — 4 command(s)
- [movement](#movement) — 1 command(s)
- [observe](#observe) — 2 command(s)
- [pointer](#pointer) — 6 command(s)
- [research](#research) — 4 command(s)
- [session](#session) — 2 command(s)
- [setting](#setting) — 6 command(s)
- [toggle](#toggle) — 2 command(s)
- [ui](#ui) — 5 command(s)
- [wait](#wait) — 1 command(s)
- [waituntil](#waituntil) — 1 command(s)
- [Undocumented commands](#undocumented-commands) — 15

## ability

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:ability.afterburner` |  | 074 T102 (harness gap, FR-009): the Afterburner is the one tutorial ability with no verb — it is a LOCAL dash (AfterburnerAction), not an ordered op, so ability.cast deliberately refuses it. |

## assembler

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:assembler.setrecipe` | `<recipeName> or <recipeName>\|<x>\|<z>` | Selects a recipe on a crafter/assembler through the deterministic heartbeat operation queue (the same path inventory.add / spawn.ships use). |

## blueprint

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:blueprint.drop` |  | Clears the in-hand blueprint armed by ExecuteBlueprintHold through the same owner-scoped ghost cleanup and buffer-disposal paths as the player's normal clear action. |
| `ffauto:blueprint.hold` | `<blueprint>` | Feature 062 US1 probe (findings 7/8): arm a blueprint IN HAND and LEAVE it there — the interactive "grabbed a building, ghost is on the grid, nothing clicked yet" state. |

## construction

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:construction.cut` | `<x1>\|<z1>\|<x2>\|<z2>` | Region-cuts the tile rect (x1,z1)-(x2,z2) through the SAME batched dispatch path the BlueprintTool cut migrates (feature 016 US2 T020): each structure with a pending RemoveStation task is re-cut -> CancelRemoval, every other PlayerPlaced structure -> batched MarkRemoval. |
| `ffauto:construction.deleteimmediate` | `<x>\|<z>` | Immediately deletes the structure at grid tile (x,z) through the DETERMINISTIC heartbeat operation queue (feature 016 US3, the bypass-the-bot delete: hauler stations, unbuilt ghosts, ephemeral structures). |
| `ffauto:construction.place` | `<blueprintString>\|<x>\|<z>` | Places a base64 blueprint string at an ABSOLUTE grid tile through the originating (replicating) network path, so a ConstructionTaskData is stamped per structure on every peer and construction bots build it. |
| `ffauto:construction.placeitem` | `<itemName>\|<x>\|<z>[\|<facing 0-3>]` | 068 T051: the human "take a building from inventory, click a tile" action. |
| `ffauto:construction.remove` | `<x>\|<z>[\|<y>]` | Deconstructs the structure at grid tile (x,z) through the DETERMINISTIC heartbeat operation queue — the US1 (feature 016) counterpart to construction.removelocal. |
| `ffauto:construction.swap` | `<fromItem>\|<toItem>\|<x1>\|<z1>\|<x2>\|<z2>` | Area-swaps every PlayerPlaced structure of item fromItem in the tile rect (x1,z1)-(x2,z2) to toItem through the deterministic op queue (feature 016 US2, the SwapStructureAction migration). |

## fleettransfer

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:fleettransfer.give` | `<shipName>\|<count>\|<x>\|<z>` | Moves ships player→structure at a tile via the FleetShipTransfer op (feature 031). |
| `ffauto:fleettransfer.take` | `<shipName>\|<count>\|<x>\|<z>` | Moves ships structure→player at a tile via the FleetShipTransfer op (feature 031). |

## game

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:game.save` | `<saveName>` | Saves the CURRENT game via the normal local save path (skipPause) — for building repeatable audit fixtures (e.g. the 027 real-map save the gen lane's host loads, avoiding the new-game intro's wall-clock player seeding). |

## hauler

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:hauler.addstop` | `<stopName>\|<x>\|<z>` | Schedules an existing named stop on a hauler through the panels' dispatch entry (feature 010 US2); the host bakes the stop id. |
| `ffauto:hauler.enable` | `<on\|off>\|<x>\|<z>` | Turns a hauler's automation on/off through the panels' dispatch entry (feature 010 US2). |

## help

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:help` | `<family>` | The runtime mirror of the coverage table — enumerate what is drivable so gaps are visible instead of silently skipped. |

## mining

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:mining.until` | `<itemName>\|<count>[\|timeoutSeconds=180]` | Ore-targeted mine-until-count (feature 020, spec F5 — the second most repeated manual loop of the live tutorial playtest): re-arms mining.startnearest-style bursts at the nearest asteroid WHOSE MINING REWARDS INCLUDE the requested item until the player's primary inventory holds at least count of it. |

## mobilestation

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:mobilestation.drive` | `<x>\|<z>\|<rotation>\|<seconds>` | Holds real-time WASD-equivalent piloting input (direction + rotation) on the local player's DriveMobileStationAction for a duration — feature 035's live-test/paired-audit harness path. |
| `ffauto:mobilestation.movenow` | `<refTileX>\|<refTileZ>\|<destX>\|<destZ>` | Drives an IMMEDIATE ("move now") station command through the LIVE production path — feature 010's HaulerCommandOperations.HandleMoveCommandAdded(position, isTemporary:true), the SAME call AddMoveStationCommandAction makes when the player is in ManualMove controller state (feature 032 US2 — investigation found this mechanism was ALREADY networked pre-032; this command exists to paired-audit-prove the manual/immediate interaction specifically, which no prior audit drove end-to-end). |
| `ffauto:mobilestation.panelrotate` | `<refTileX>\|<refTileZ>\|<rotation>\|<seconds>` | Emulates the MoveableStructureComponentPanel rotate-button HELD state without the UI (the panel rotate lane, migrated onto the feature-035 replicated drive channel): for seconds, reports a panel-rotate intent every frame — writes PlayerPanelRotateIntent on the local player and, on a remote client, RPCs ReportPanelRotateServerRpc — then reports zero on expiry (the release leg). |
| `ffauto:mobilestation.place` | `<itemName>\|<x>\|<z>` | Places a mobile-station-capable structure via the MobileStationPlacement op (feature 032 US1). |

## movement

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:movement.goto` | `<x>\|<z>[\|tolerance=15][\|timeoutSeconds=120]` | Closed-loop travel (feature 020, spec F5): holds the SAME real move input as movement.hold (the HoldAutomationMoveInput idiom — no teleport), re-aimed at the target every frame, until the player is within tolerance world units of (x,z) or the timeout elapses. |

## observe

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:observe.screenshot` | `[note][\|world\|ui]` | Journal a screenshotMarker recording WHICH capture channel the agent is about to use, so captures correlate with state honestly. |
| `ffauto:observe.state` | `<scope>[\|args]` | Build the scope's snapshot (player, inventory, nearby\|x\|z\|r, enemies\|x\|z\|r, tasks, research, objectives, alerts, session) per contracts/state-snapshot.md, journal a snapshot event, and — when a session is active — write snapshots/<seq>-<scope>.json in the session dir. |

## pointer

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:pointer.clear` |  | All overrides gone, real mouse resumes. |
| `ffauto:pointer.click` | `[\|button=0][\|holdFrames=2]` | Press+release at the current pointer position (or the real cursor when no override is armed) — both the world seams and the UGUI feeder see the same frame window. |
| `ffauto:pointer.drag` | `<x1>\|<z1>\|<x2>\|<z2>[\|button=0]` | Moveto → press → interpolated move → release across two grid tiles (drag-rect selection and drag interactions). |
| `ffauto:pointer.moveto` | `<x>\|<z> or <px>\|<py>\|screen` | Tile form projects the tile's world center through the SAME active camera the placement preview uses; screen form takes raw pixels (Input.mousePosition space, bottom-left origin). |
| `ffauto:pointer.press` | `[\|button=0]` | Hold until pointer.release. |
| `ffauto:pointer.release` | `[\|button=0]` | Release a held pointer button. |

## research

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:research.cancel` |  | Cancels the active research through the panels' dispatch entry (feature 010 T028 — the Cancel button). |
| `ffauto:research.queue` | `<techName>` | Queues a research through the panels' dispatch entry (feature 010 US3, research D8). |
| `ffauto:research.remove` | `<techName>[\|<targetLevel>]` | Removes a research-queue entry through the panels' dispatch entry (feature 010 US3). |
| `ffauto:research.setactive` | `<techName>` | Sets the active research through the panels' dispatch entry (feature 010 T028 — the Research button). |

## session

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:session.start` | `<label>` | Attach the playtest session scaffold (journal, snapshots/, bugs/) to the CURRENT interactively booted game — the real playtest flow is interactive boot → load save → play → save-as a NEW name. |
| `ffauto:session.stop` |  | Flush + close the journal and detach. |

## setting

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:setting.filterall` | `<allow\|block>\|<x>\|<z>` | 068 T052: a filter's allow-all / block-all option, set exactly the way the filter panel's checkbox sets it (FilterComponentPanel.SetupCheckboxHandler): start from the current filter, clear every option, set the one, empty the filter item, dispatch the whole value through DispatchFilter (the FilterConfig setting op). |
| `ffauto:setting.filteritem` | `<itemName>\|<x>\|<z>` | Sets a filter structure's filter item through the SAME dispatch entry the panels use (feature 010, research D8 — the audit must exercise the production path). |
| `ffauto:setting.massdriver` | `<x>\|<z>\|<targetX>\|<targetZ> or <x>\|<z>\|clear` | 068 T052: a mass driver's destination, set or cleared with the payload its own panel sends (SelectTargetForMassDriverAction: the destination's CenterTile; MassDriverComponentPanel: Clear). |
| `ffauto:setting.name` | `<newName>\|<x>\|<z>` | Renames a placeable structure through the panels' dispatch entry (feature 010 US2). |
| `ffauto:setting.requesteritems` | `<itemName>\|<count>\|<x>\|<z>` | Sets a requester's requested count for one item through the panels' shared merge path (UpdateRequestedUnitCount → full-replace RequesterItems op, feature 010). |
| `ffauto:setting.splitterpriority` | `<up\|down\|left\|right\|none>\|<x>\|<z>` | Sets (or clears) a splitter's output priority through the panels' dispatch entry (feature 010). |

## toggle

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:toggle.cbots` | `<on\|off>` | Flip a GLOBAL factory toggle through the networked SetFactoryToggle op — on = the factory lane STOPPED (the flag set), matching the panel toggles' on state. |
| `ffauto:toggle.inserters` | `<on\|off>` | Flip a GLOBAL factory toggle through the networked SetFactoryToggle op — on = the factory lane STOPPED (the flag set), matching the panel toggles' on state. |

## ui

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:ui.click` | `<panelName>\|<controlPath>` | Resolves a control by panel + path (content selectors per spec F6) and drives the D1 pointer at its screen rect center — the real EventSystem raycast decides delivery. |
| `ffauto:ui.close` | `<panelName>` | No-op when already closed. |
| `ffauto:ui.open` | `<panelName>` | Opens a core panel via the SAME shared toggle entry the keyboard shortcut and HUD quick-button call. |
| `ffauto:ui.read` | `<panelName>[\|<controlPath>]` | JSON readback of a panel's interactive controls, or one control's state. |
| `ffauto:ui.set` | `<panelName>\|<controlPath>\|<value>` | Typed field set: toggles flip via a real pointer click; slider/input/dropdown values set through the control's own notification pipeline (the handlers the UI runs). |

## wait

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:wait.status` | `[\|id]` | JSON status of all retained waits, or one. |

## waituntil

| Command | Args | Description |
| --- | --- | --- |
| `ffauto:waituntil` | `<predicate>\|<args...>\|<timeoutSeconds>` | ALWAYS non-blocking here: registers the wait and returns its wait-id immediately, so an interactive bridge call can never hang on a player loop that only advances when the agent pumps editor steps. |

## Undocumented commands

Dispatchable commands with no `<c>ffauto:…</c>` XML doc on their handler. They are listed name-only so the gap is visible; document them in the runner and regenerate.

- `ffauto:combat.respawn`
- `ffauto:combat.ride`
- `ffauto:craft.cancel`
- `ffauto:craft.queue`
- `ffauto:mining.startnearest`
- `ffauto:mining.startnearestactivation`
- `ffauto:movement.hold`
- `ffauto:rotate.abs`
- `ffauto:rotate.at`
- `ffauto:setting.powerdistributor`
- `ffauto:transfer.grab`
- `ffauto:transfer.put`
- `ffauto:transfer.sort`
- `ffauto:transfer.trash`
- `ffauto:wait`

package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/channel"
)

// tileLength is FixedPointConstants.TileLengthInLocalTransformUnits (FFCore/FixedPoints/
// FixedPointConstants.cs:5 in the game repo): one grid tile is 10 world units.
const tileLength = 10

// tileCenterWorld is GridHelper.GetWorldPositionFromCenterOfTile (FFCore/Utils/GridHelper.cs:422):
// tile * 10 + 5. The inverse the game reports back, GetTileFromWorldPosition (:492), floors, so a
// player driven to this point reads as standing on the requested tile.
func tileCenterWorld(tile int) int { return tile*tileLength + tileLength/2 }

// worldUnits converts a tile distance for movement.goto's tolerance. Rounded to a whole number
// because the game parses it with float.TryParse in the player's own culture, where "2.5" may not
// be a number at all.
func worldUnits(tiles float64) int { return max(1, int(math.Round(tiles*tileLength))) }

// Every structured tool writes its arguments into a '|'-separated command, so an argument that
// contains the separator would silently become a different command.
func arg(name, value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	if strings.ContainsAny(v, "|;") {
		return "", fmt.Errorf("%s must not contain '|' or ';'", name)
	}
	return v, nil
}

type budgetArg struct {
	BudgetSeconds float64 `json:"budget_seconds,omitempty" jsonschema:"how long to wait for the game before answering 'still running' (default 90, max 600); the game keeps working either way"`
}

type noArgs struct{}

type lookArgs struct {
	Radius int `json:"radius,omitempty" jsonschema:"tiles around you to scan (default 20, the game caps it at 64)"`
}

type inventoryArgs struct {
	Craftable bool `json:"craftable,omitempty" jsonschema:"also list how many of each item you could hand-craft right now"`
}

type screenshotArgs struct {
	MaxEdge int `json:"max_edge,omitempty" jsonschema:"longest image side in pixels, 320..1920 (default 1280)"`
}

type moveArgs struct {
	X              int     `json:"x" jsonschema:"target tile x"`
	Z              int     `json:"z" jsonschema:"target tile z"`
	Tolerance      float64 `json:"tolerance,omitempty" jsonschema:"stop within this many tiles of the target (default 1.5)"`
	TimeoutSeconds int     `json:"timeout_seconds,omitempty" jsonschema:"give up after this many seconds of game time (default 120)"`
	budgetArg
}

type mineArgs struct {
	Item           string `json:"item" jsonschema:"ore to mine, e.g. 'Iron Ore' (be near an asteroid of it)"`
	Count          int    `json:"count" jsonschema:"keep mining until you HOLD this many (a total, not an amount to add)"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"give up after this many seconds of game time (default 180)"`
	budgetArg
}

type buildArgs struct {
	Item   string `json:"item" jsonschema:"the building's item name, e.g. 'Mining Station' — or a whole blueprint copy/paste string (starts with 'ffblueprintstart')"`
	X      int    `json:"x" jsonschema:"tile x to build at"`
	Z      int    `json:"z" jsonschema:"tile z to build at"`
	Facing string `json:"facing,omitempty" jsonschema:"up, right, down or left"`
	budgetArg
}

type shipsArgs struct {
	Ship  string `json:"ship" jsonschema:"ship name"`
	Count int    `json:"count" jsonschema:"how many ships"`
	X     int    `json:"x" jsonschema:"structure tile x"`
	Z     int    `json:"z" jsonschema:"structure tile z"`
	budgetArg
}

type tileArgs struct {
	X int `json:"x" jsonschema:"structure tile x"`
	Z int `json:"z" jsonschema:"structure tile z"`
}

type massDriverArgs struct {
	X       int  `json:"x" jsonschema:"mass driver tile x"`
	Z       int  `json:"z" jsonschema:"mass driver tile z"`
	TargetX *int `json:"target_x,omitempty" jsonschema:"target tile x"`
	TargetZ *int `json:"target_z,omitempty" jsonschema:"target tile z"`
	Clear   bool `json:"clear,omitempty" jsonschema:"remove the target instead"`
}

type deconstructArgs struct {
	X  int  `json:"x" jsonschema:"tile x (a single structure, or one corner of an area)"`
	Z  int  `json:"z" jsonschema:"tile z"`
	X2 *int `json:"x2,omitempty" jsonschema:"opposite corner x, to clear a rectangle"`
	Z2 *int `json:"z2,omitempty" jsonschema:"opposite corner z, to clear a rectangle"`
	budgetArg
}

type craftArgs struct {
	Item  string `json:"item" jsonschema:"item to hand-craft"`
	Count int    `json:"count" jsonschema:"how many"`
	budgetArg
}

type cancelCraftArgs struct {
	Item string `json:"item" jsonschema:"queued hand-craft to cancel (ingredients are refunded)"`
}

type researchArgs struct {
	Tech     string `json:"tech" jsonschema:"technology name"`
	StartNow bool   `json:"start_now,omitempty" jsonschema:"make it the active research now instead of queueing it"`
}

type itemsArgs struct {
	Item  string `json:"item" jsonschema:"item name ('first' takes the first stack when taking)"`
	Count int    `json:"count" jsonschema:"how many"`
	X     int    `json:"x" jsonschema:"structure tile x"`
	Z     int    `json:"z" jsonschema:"structure tile z"`
	budgetArg
}

type recipeArgs struct {
	Recipe string `json:"recipe" jsonschema:"recipe (item) the crafter should make"`
	X      int    `json:"x" jsonschema:"crafter tile x"`
	Z      int    `json:"z" jsonschema:"crafter tile z"`
}

type configureArgs struct {
	Kind    string `json:"kind" jsonschema:"filter_item | filter_mode | splitter_priority | requester_items | power_target | rename | hauler_stop | hauler_enable"`
	X       int    `json:"x" jsonschema:"structure tile x"`
	Z       int    `json:"z" jsonschema:"structure tile z"`
	Value   string `json:"value,omitempty" jsonschema:"filter_item/requester_items: item; filter_mode: allow|block; splitter_priority: up|down|left|right|none; rename: new name; hauler_stop: stop name; hauler_enable: on|off; power_target: 'clear' to remove the target"`
	Count   int    `json:"count,omitempty" jsonschema:"requester_items: how many to request"`
	TargetX *int   `json:"target_x,omitempty" jsonschema:"power_target: target tile x"`
	TargetZ *int   `json:"target_z,omitempty" jsonschema:"power_target: target tile z"`
}

type uiArgs struct {
	Action  string `json:"action" jsonschema:"open | close | click | set | read"`
	Panel   string `json:"panel" jsonschema:"panel name"`
	Control string `json:"control,omitempty" jsonschema:"control path (click/set, optional for read); selectors like Name[item=X] or [text=Y]"`
	Value   string `json:"value,omitempty" jsonschema:"value for set"`
}

type waitUntilArgs struct {
	Predicate      string   `json:"predicate" jsonschema:"objectiveComplete | itemCount | structureAt | structureBuilt | powerSatisfaction | taskQueueIdle | notification | heartbeat | researchDone"`
	Args           []string `json:"args,omitempty" jsonschema:"the predicate's arguments in order, e.g. ['Iron Plate','50'] for itemCount; tiles for structureAt"`
	TimeoutSeconds float64  `json:"timeout_seconds,omitempty" jsonschema:"give up after this many seconds of GAME time (default 120)"`
	budgetArg
}

type waitForArgs struct {
	ChainID string `json:"chain_id" jsonschema:"a chain id from 'still running: chain <id>' (or a numeric waituntil id)"`
	budgetArg
}

type runArgs struct {
	Commands []string `json:"commands" jsonschema:"ffauto: commands run in order as one chain (the prefix is optional); the chain stops at the first failure"`
	Wait     *bool    `json:"wait,omitempty" jsonschema:"wait for the chain to finish (default true)"`
	budgetArg
}

type listArgs struct {
	Family string `json:"family,omitempty" jsonschema:"only this family, e.g. 'construction'"`
}

func readOnly(title string) *mcp.ToolAnnotations {
	return &mcp.ToolAnnotations{ReadOnlyHint: true, Title: title}
}

func (s *server) registerTools(srv *mcp.Server) {
	addTool(s, srv, &mcp.Tool{Name: "game_status", Annotations: readOnly("Game status"),
		Description: "Start here. Whether a game is running, your tile position and health, the current objective and recent alerts."},
		(*call).gameStatus)
	addTool(s, srv, &mcp.Tool{Name: "look_around", Annotations: readOnly("Look around"),
		Description: "Structures (with built/frame state, power, recipe), asteroids (ore and amount left) and pickups within a radius of your tile."},
		(*call).lookAround)
	addTool(s, srv, &mcp.Tool{Name: "inventory", Annotations: readOnly("Inventory"),
		Description: "Your inventory: item totals and slots, optionally what you can hand-craft right now."},
		(*call).inventory)
	addTool(s, srv, &mcp.Tool{Name: "research_status", Annotations: readOnly("Research status"),
		Description: "Active research and its progress, the research queue, and what is already researched."},
		(*call).researchStatus)
	addTool(s, srv, &mcp.Tool{Name: "enemies", Annotations: readOnly("Enemies"),
		Description: "Enemy ships and structures near you, nearest first, with tile and distance in tiles."},
		(*call).enemies)
	addTool(s, srv, &mcp.Tool{Name: "inspect", Annotations: readOnly("Inspect a structure"),
		Description: "Everything about the structure at a tile: contents, recipe, power and heat, bots, status."},
		(*call).inspect)
	addTool(s, srv, &mcp.Tool{Name: "screenshot", Annotations: readOnly("Screenshot"),
		Description: "A picture of the game window as the player sees it (UI included). At most 2 per second."},
		(*call).screenshot)

	addTool(s, srv, &mcp.Tool{Name: "move_to",
		Description: "Fly the player to a tile, the way the movement keys would. Waits until you arrive (or the budget runs out) and reports where you ended up."},
		(*call).moveTo)
	addTool(s, srv, &mcp.Tool{Name: "mine",
		Description: "Mine the nearest asteroid of an ore until you hold `count` of it. Move next to one first (look_around lists asteroids)."},
		(*call).mine)
	addTool(s, srv, &mcp.Tool{Name: "build",
		Description: "Place a building (by item name) or a blueprint string at a tile. It becomes a frame that construction bots build from your items; the game may shift it to fit, and says where it went. Check with look_around or wait_until structureBuilt."},
		(*call).build)
	addTool(s, srv, &mcp.Tool{Name: "deconstruct",
		Description: "Deconstruct the structure at a tile, or everything in the rectangle to (x2,z2). Bots return the items."},
		(*call).deconstruct)
	addTool(s, srv, &mcp.Tool{Name: "craft", Description: "Queue hand-crafts; ingredients come from your inventory."},
		(*call).craft)
	addTool(s, srv, &mcp.Tool{Name: "cancel_craft", Description: "Cancel a queued hand-craft and get the ingredients back."},
		(*call).cancelCraft)
	addTool(s, srv, &mcp.Tool{Name: "research", Description: "Queue a technology, or start it now."},
		(*call).research)
	addTool(s, srv, &mcp.Tool{Name: "put_items", Description: "Move items from your inventory into the structure at a tile (you must be within reach)."},
		func(c *call, ctx context.Context, in itemsArgs) (*mcp.CallToolResult, error) {
			return c.transfer(ctx, "transfer.put", in)
		})
	addTool(s, srv, &mcp.Tool{Name: "take_items", Description: "Take items out of the structure at a tile into your inventory (you must be within reach)."},
		func(c *call, ctx context.Context, in itemsArgs) (*mcp.CallToolResult, error) {
			return c.transfer(ctx, "transfer.grab", in)
		})
	addTool(s, srv, &mcp.Tool{Name: "set_recipe", Description: "Set the recipe of the crafter/assembler at a tile."},
		(*call).setRecipe)
	addTool(s, srv, &mcp.Tool{Name: "configure",
		Description: "Change a structure's setting: filter item or allow/block mode, splitter priority, requester items, power distributor target, name, hauler stop, hauler on/off."},
		(*call).configure)
	addTool(s, srv, &mcp.Tool{Name: "aim_mass_driver", Description: "Point the mass driver at a tile to a target tile, or clear its target."},
		(*call).aimMassDriver)
	addTool(s, srv, &mcp.Tool{Name: "give_ships", Description: "Move ships from your fleet into the structure at a tile (you must be within reach)."},
		func(c *call, ctx context.Context, in shipsArgs) (*mcp.CallToolResult, error) {
			return c.ships(ctx, "fleettransfer.give", in)
		})
	addTool(s, srv, &mcp.Tool{Name: "take_ships", Description: "Take ships out of the structure at a tile into your fleet (you must be within reach)."},
		func(c *call, ctx context.Context, in shipsArgs) (*mcp.CallToolResult, error) {
			return c.ships(ctx, "fleettransfer.take", in)
		})
	addTool(s, srv, &mcp.Tool{Name: "respawn", Description: "Respawn after your ship was destroyed."},
		(*call).respawn)
	addTool(s, srv, &mcp.Tool{Name: "ui", Description: "Open, close, click, set or read a game panel through the real UI."},
		(*call).ui)

	addTool(s, srv, &mcp.Tool{Name: "wait_until",
		Description: "Wait for something to become true in the game (item count, structure built, research done, objective complete, …). Timeouts are game time."},
		(*call).waitUntil)
	addTool(s, srv, &mcp.Tool{Name: "wait_for", Description: "Keep waiting on a chain an earlier tool reported as 'still running'."},
		(*call).waitFor)
	addTool(s, srv, &mcp.Tool{Name: "run_commands",
		Description: "Escape hatch: run raw ffauto: commands as one ordered chain. The game decides what you may run (see list_commands). movement.goto here takes WORLD units (tile × 10 + 5); every other command takes tiles."},
		(*call).runCommands)
	addTool(s, srv, &mcp.Tool{Name: "list_commands", Annotations: readOnly("List commands"),
		Description: "Every ffauto: command this game lets you run, with its arguments."},
		(*call).listCommands)
}

// ---------------------------------------------------------------------------- observation

type envelope map[string]any

func (e envelope) data() map[string]any {
	d, _ := e["data"].(map[string]any)
	return d
}

func (c *call) snapshot(ctx context.Context, scope string, q url.Values) (envelope, error) {
	var raw json.RawMessage
	err := c.do(ctx, func(cl *channel.Client) (err error) {
		raw, err = cl.Snapshot(ctx, scope, q)
		return err
	})
	if err != nil {
		return nil, err
	}
	var e envelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return nil, fmt.Errorf("the game's %s snapshot was not JSON: %w", scope, err)
	}
	return e, nil
}

// playerTile is where the player stands, from the player snapshot's position.tile.
func (c *call) playerTile(ctx context.Context) (int, int, error) {
	e, err := c.snapshot(ctx, "player", nil)
	if err != nil {
		return 0, 0, err
	}
	d := e.data()
	if why, ok := d["unavailable"].(string); ok {
		return 0, 0, fmt.Errorf("no player right now (%s) — is a game loaded?", why)
	}
	x, okX := num(dig(d, "position", "tile", "x"))
	z, okZ := num(dig(d, "position", "tile", "z"))
	if !okX || !okZ {
		return 0, 0, fmt.Errorf("the player snapshot has no position")
	}
	return int(x), int(z), nil
}

func (c *call) gameStatus(ctx context.Context, _ noArgs) (*mcp.CallToolResult, error) {
	var hello json.RawMessage
	var h channel.Hello
	err := c.do(ctx, func(cl *channel.Client) (err error) {
		h, hello, err = cl.Hello(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	out := map[string]any{"hello": json.RawMessage(hello)}
	parts := []string{fmt.Sprintf("Final Factory %s — %s, tier %s, heartbeat %d.", h.GameVersion, h.State, h.Tier, h.Heartbeat)}
	for _, scope := range []string{"player", "objectives", "alerts"} {
		e, err := c.snapshot(ctx, scope, nil)
		if err != nil {
			out[scope] = map[string]any{"error": errorText(err)}
			continue
		}
		d := e.data()
		out[scope] = d
		if _, unavailable := d["unavailable"]; unavailable {
			continue
		}
		switch scope {
		case "player":
			if x, ok := num(dig(d, "position", "tile", "x")); ok {
				z, _ := num(dig(d, "position", "tile", "z"))
				line := fmt.Sprintf("You are at tile (%d,%d)", int(x), int(z))
				if cur, ok := num(dig(d, "health", "current")); ok {
					mx, _ := num(dig(d, "health", "max"))
					line += fmt.Sprintf(", health %s/%s", trimFloat(cur), trimFloat(mx))
				}
				parts = append(parts, line+".")
			}
		case "objectives":
			if desc := str(dig(d, "current", "shortDescription")); desc != "" {
				parts = append(parts, "Current objective: "+desc+".")
			}
		case "alerts":
			if list, ok := d["alerts"].([]any); ok && len(list) > 0 {
				parts = append(parts, fmt.Sprintf("%d recent alerts.", len(list)))
			}
		}
	}
	if h.State != "playing" {
		parts = append(parts, "Actions need a loaded, unpaused game.")
	}
	return summaryJSON(strings.Join(parts, " "), out), nil
}

func (c *call) lookAround(ctx context.Context, in lookArgs) (*mcp.CallToolResult, error) {
	x, z, err := c.playerTile(ctx)
	if err != nil {
		return nil, err
	}
	radius := in.Radius
	if radius <= 0 {
		radius = 20
	}
	e, err := c.snapshot(ctx, "nearby", url.Values{"x": {strconv.Itoa(x)}, "z": {strconv.Itoa(z)}, "r": {strconv.Itoa(radius)}})
	if err != nil {
		return nil, err
	}
	d := e.data()
	if why, ok := d["unavailable"].(string); ok {
		return textResult("Nothing to look at right now: "+why, true), nil
	}
	structures, _ := d["structures"].([]any)
	asteroids, _ := d["asteroids"].([]any)
	states := map[string]int{}
	for _, s := range structures {
		states[str(dig(s, "state"))]++
	}
	ores := map[string]int{}
	for _, a := range asteroids {
		ores[str(dig(a, "oreType"))]++
	}
	if r, ok := num(d["radius"]); ok {
		radius = int(r)
	}
	summary := fmt.Sprintf("Around your tile (%d,%d), radius %d: %d structures%s, %d asteroids%s",
		x, z, radius, len(structures), counts(states), len(asteroids), counts(ores))
	if p, ok := num(d["pickupables"]); ok && p > 0 {
		summary += fmt.Sprintf(", %d pickups", int(p))
	}
	for _, extra := range []struct{ key, label string }{{"planets", "planets"}, {"cometFragments", "comet fragments"}} {
		if list, ok := d[extra.key].([]any); ok && len(list) > 0 {
			summary += fmt.Sprintf(", %d %s", len(list), extra.label)
		}
	}
	return summaryJSON(summary+".", d), nil
}

func (c *call) inventory(ctx context.Context, in inventoryArgs) (*mcp.CallToolResult, error) {
	var q url.Values
	if in.Craftable {
		q = url.Values{"craftable": {"all"}}
	}
	e, err := c.snapshot(ctx, "inventory", q)
	if err != nil {
		return nil, err
	}
	d := e.data()
	if why, ok := d["unavailable"].(string); ok {
		return textResult("No inventory right now: "+why, true), nil
	}
	totals, _ := d["totals"].(map[string]any)
	type item struct {
		name  string
		count float64
	}
	var items []item
	for name, v := range totals {
		n, _ := num(v)
		items = append(items, item{name, n})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].name < items[j].name
	})
	var top []string
	for i, it := range items {
		if i == 8 {
			top = append(top, "…")
			break
		}
		top = append(top, fmt.Sprintf("%s ×%s", it.name, trimFloat(it.count)))
	}
	summary := fmt.Sprintf("Inventory: %d item types", len(items))
	if len(top) > 0 {
		summary += " — " + strings.Join(top, ", ")
	}
	if craft, ok := d["smartCraftCounts"].(map[string]any); ok {
		summary += fmt.Sprintf(". %d items craftable now", len(craft))
	}
	return summaryJSON(summary+".", d), nil
}

func (c *call) researchStatus(ctx context.Context, _ noArgs) (*mcp.CallToolResult, error) {
	e, err := c.snapshot(ctx, "research", nil)
	if err != nil {
		return nil, err
	}
	d := e.data()
	if why, ok := d["unavailable"].(string); ok {
		return textResult("No research state right now: "+why, true), nil
	}
	summary := "No active research"
	if tech := str(dig(d, "active", "tech")); tech != "" {
		summary = "Researching " + tech
		if p, ok := num(dig(d, "active", "progress")); ok {
			summary += fmt.Sprintf(" (%.0f%%)", p*100)
		}
	}
	queue, _ := d["queue"].([]any)
	summary += fmt.Sprintf("; %d queued", len(queue))
	if n, ok := num(d["researchedCount"]); ok {
		summary += fmt.Sprintf("; %d researched", int(n))
	}
	if available, ok := d["available"].([]any); ok {
		summary += fmt.Sprintf("; %d listed as available", len(available))
	}
	return summaryJSON(summary+".", d), nil
}

func (c *call) enemies(ctx context.Context, in lookArgs) (*mcp.CallToolResult, error) {
	x, z, err := c.playerTile(ctx)
	if err != nil {
		return nil, err
	}
	radius := in.Radius
	if radius <= 0 {
		radius = 30
	}
	e, err := c.snapshot(ctx, "enemies", url.Values{"x": {strconv.Itoa(x)}, "z": {strconv.Itoa(z)}, "r": {strconv.Itoa(radius)}})
	if err != nil {
		return nil, err
	}
	d := e.data()
	if why, ok := d["unavailable"].(string); ok {
		return textResult("No enemy information right now: "+why, true), nil
	}
	count, _ := num(d["count"])
	summary := fmt.Sprintf("%d enemies within %d tiles of (%d,%d)", int(count), radius, x, z)
	if list, ok := d["enemies"].([]any); ok && len(list) > 0 {
		first := list[0]
		name := firstNonEmpty(str(dig(first, "name")), "enemy")
		tx, _ := num(dig(first, "tile", "x"))
		tz, _ := num(dig(first, "tile", "z"))
		dist, _ := num(dig(first, "distanceTiles"))
		summary += fmt.Sprintf("; nearest: %s at tile (%d,%d), %s tiles away", name, int(tx), int(tz), trimFloat(dist))
	}
	return summaryJSON(summary+".", d), nil
}

func (c *call) screenshot(ctx context.Context, in screenshotArgs) (*mcp.CallToolResult, error) {
	var shot channel.Screenshot
	err := c.do(ctx, func(cl *channel.Client) (err error) {
		shot, err = cl.Screenshot(ctx, in.MaxEdge)
		return err
	})
	if err != nil {
		return nil, err
	}
	return &mcp.CallToolResult{Content: []mcp.Content{
		&mcp.ImageContent{Data: shot.PNG, MIMEType: "image/png"},
		&mcp.TextContent{Text: fmt.Sprintf("Screenshot at heartbeat %d (%dx%d, captured %s UTC).",
			shot.Heartbeat, shot.Width, shot.Height, shot.CapturedUTC)},
	}}, nil
}

func (c *call) listCommands(ctx context.Context, in listArgs) (*mcp.CallToolResult, error) {
	var h channel.Help
	err := c.do(ctx, func(cl *channel.Client) (err error) {
		h, _, err = cl.Help(ctx, strings.TrimSpace(in.Family))
		return err
	})
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	names := h.CommandNames()
	fmt.Fprintf(&b, "%d commands you may run (tier %s). Use them with run_commands; arguments are '|'-separated.\n", len(names), h.Actor)
	for _, f := range h.Families {
		for _, cmd := range f.Commands {
			line := "ffauto:" + cmd.Name
			if cmd.Args != "" {
				line += "|" + cmd.Args
			}
			if cmd.Summary != "" {
				line += " — " + cmd.Summary
			}
			b.WriteString(line + "\n")
		}
	}
	return textResult(strings.TrimRight(b.String(), "\n"), false), nil
}

// ---------------------------------------------------------------------------- actions

// act runs a chain and renders the outcome.
func (c *call) act(ctx context.Context, budget float64, commands ...string) (*mcp.CallToolResult, error) {
	o, err := c.run(ctx, commands, c.budget(budget))
	if err != nil {
		return nil, err
	}
	text, isErr := o.text()
	return textResult(text, isErr), nil
}

func (c *call) moveTo(ctx context.Context, in moveArgs) (*mcp.CallToolResult, error) {
	tolerance := in.Tolerance
	if tolerance <= 0 {
		tolerance = 1.5
	}
	// movement.goto is the one verb that takes world units; the position is the tile's centre.
	cmd := fmt.Sprintf("movement.goto|%d|%d|%d", tileCenterWorld(in.X), tileCenterWorld(in.Z), worldUnits(tolerance))
	if in.TimeoutSeconds > 0 {
		cmd += "|" + strconv.Itoa(in.TimeoutSeconds)
	}
	o, err := c.run(ctx, []string{cmd}, c.budget(in.BudgetSeconds))
	if err != nil {
		return nil, err
	}
	if o.Status != "completed" {
		text, isErr := o.text()
		return textResult(text, isErr), nil
	}
	// The chain's completion is the game's travel-time estimate, not an arrival check, so
	// report where the player actually is.
	x, z, err := c.playerTile(ctx)
	if err != nil {
		return textResult("done: "+o.Result+" (position unknown: "+errorText(err)+")", false), nil
	}
	dist := math.Hypot(float64(x-in.X), float64(z-in.Z))
	if dist <= math.Ceil(tolerance) {
		return textResult(fmt.Sprintf("Arrived: you are at tile (%d,%d).", x, z), false), nil
	}
	return textResult(fmt.Sprintf("Stopped at tile (%d,%d), %.1f tiles from (%d,%d) — something may be in the way, or it timed out. Try move_to again or look_around.",
		x, z, dist, in.X, in.Z), true), nil
}

func (c *call) mine(ctx context.Context, in mineArgs) (*mcp.CallToolResult, error) {
	item, err := arg("item", in.Item)
	if err != nil {
		return nil, err
	}
	if in.Count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	cmd := fmt.Sprintf("mining.until|%s|%d", item, in.Count)
	if in.TimeoutSeconds > 0 {
		cmd += "|" + strconv.Itoa(in.TimeoutSeconds)
	}
	return c.act(ctx, in.BudgetSeconds, cmd)
}

var facings = map[string]int{"up": 0, "right": 1, "down": 2, "left": 3, "0": 0, "1": 1, "2": 2, "3": 3}

// blueprintPrefix starts every blueprint copy/paste string (BlueprintTool's format).
const blueprintPrefix = "ffblueprintstart"

func (c *call) build(ctx context.Context, in buildArgs) (*mcp.CallToolResult, error) {
	item, err := arg("item", in.Item)
	if err != nil {
		return nil, err
	}
	dir := -1
	if f := strings.ToLower(strings.TrimSpace(in.Facing)); f != "" {
		d, ok := facings[f]
		if !ok {
			return nil, fmt.Errorf("facing must be up, right, down or left, not %q", in.Facing)
		}
		dir = d
	}

	if !strings.HasPrefix(strings.ToLower(item), blueprintPrefix) {
		// The game builds the one-item blueprint from the item's own footprint, so no size table
		// lives here; it may move the anchor to fit, and its result names the tile it used.
		cmd := fmt.Sprintf("construction.placeitem|%s|%d|%d", item, in.X, in.Z)
		if dir >= 0 {
			cmd += "|" + strconv.Itoa(dir)
		}
		o, err := c.run(ctx, []string{cmd}, c.budget(in.BudgetSeconds))
		if err != nil {
			return nil, err
		}
		text, isErr := o.text()
		if o.Status == "completed" {
			text += "\nUse the tile the game reports above for later steps — it can differ from the one you asked for."
		}
		return textResult(text, isErr), nil
	}

	commands := []string{fmt.Sprintf("construction.place|%s|%d|%d", item, in.X, in.Z)}
	if dir >= 0 {
		// rotate.abs needs the structure to exist, and placement lands on a later heartbeat.
		commands = append(commands,
			fmt.Sprintf("waituntil|structureAt|%d|%d|frame|30", in.X, in.Z),
			fmt.Sprintf("rotate.abs|%d|%d|%d", in.X, in.Z, dir))
	}
	return c.act(ctx, in.BudgetSeconds, commands...)
}

func (c *call) ships(ctx context.Context, verb string, in shipsArgs) (*mcp.CallToolResult, error) {
	ship, err := arg("ship", in.Ship)
	if err != nil {
		return nil, err
	}
	if in.Count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	return c.act(ctx, in.BudgetSeconds, fmt.Sprintf("%s|%s|%d|%d|%d", verb, ship, in.Count, in.X, in.Z))
}

func (c *call) respawn(ctx context.Context, _ noArgs) (*mcp.CallToolResult, error) {
	return c.act(ctx, 0, "combat.respawn")
}

func (c *call) aimMassDriver(ctx context.Context, in massDriverArgs) (*mcp.CallToolResult, error) {
	switch {
	case in.Clear:
		return c.act(ctx, 0, fmt.Sprintf("setting.massdriver|%d|%d|clear", in.X, in.Z))
	case in.TargetX != nil && in.TargetZ != nil:
		return c.act(ctx, 0, fmt.Sprintf("setting.massdriver|%d|%d|%d|%d", in.X, in.Z, *in.TargetX, *in.TargetZ))
	}
	return nil, fmt.Errorf("give target_x and target_z, or clear: true")
}

func (c *call) inspect(ctx context.Context, in tileArgs) (*mcp.CallToolResult, error) {
	e, err := c.snapshot(ctx, "structure", url.Values{"x": {strconv.Itoa(in.X)}, "z": {strconv.Itoa(in.Z)}})
	if err != nil {
		return nil, err
	}
	d := e.data()
	if why, ok := d["unavailable"].(string); ok {
		return textResult(fmt.Sprintf("Nothing to inspect at tile (%d,%d): %s", in.X, in.Z, why), true), nil
	}
	summary := fmt.Sprintf("Structure at tile (%d,%d)", in.X, in.Z)
	if item := str(d["item"]); item != "" {
		summary = fmt.Sprintf("%s at tile (%d,%d)", item, in.X, in.Z)
	}
	if state := firstNonEmpty(str(d["status"]), str(d["state"])); state != "" {
		summary += ", " + state
	}
	return summaryJSON(summary+".", d), nil
}

func (c *call) deconstruct(ctx context.Context, in deconstructArgs) (*mcp.CallToolResult, error) {
	if (in.X2 == nil) != (in.Z2 == nil) {
		return nil, fmt.Errorf("give both x2 and z2 for an area, or neither for one structure")
	}
	if in.X2 != nil {
		return c.act(ctx, in.BudgetSeconds, fmt.Sprintf("construction.cut|%d|%d|%d|%d", in.X, in.Z, *in.X2, *in.Z2))
	}
	return c.act(ctx, in.BudgetSeconds, fmt.Sprintf("construction.remove|%d|%d", in.X, in.Z))
}

func (c *call) craft(ctx context.Context, in craftArgs) (*mcp.CallToolResult, error) {
	item, err := arg("item", in.Item)
	if err != nil {
		return nil, err
	}
	if in.Count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	return c.act(ctx, in.BudgetSeconds, fmt.Sprintf("craft.queue|%s|%d", item, in.Count))
}

func (c *call) cancelCraft(ctx context.Context, in cancelCraftArgs) (*mcp.CallToolResult, error) {
	item, err := arg("item", in.Item)
	if err != nil {
		return nil, err
	}
	return c.act(ctx, 0, "craft.cancel|"+item)
}

func (c *call) research(ctx context.Context, in researchArgs) (*mcp.CallToolResult, error) {
	tech, err := arg("tech", in.Tech)
	if err != nil {
		return nil, err
	}
	if in.StartNow {
		return c.act(ctx, 0, "research.setactive|"+tech)
	}
	return c.act(ctx, 0, "research.queue|"+tech)
}

func (c *call) transfer(ctx context.Context, verb string, in itemsArgs) (*mcp.CallToolResult, error) {
	item, err := arg("item", in.Item)
	if err != nil {
		return nil, err
	}
	if in.Count < 1 {
		return nil, fmt.Errorf("count must be at least 1")
	}
	return c.act(ctx, in.BudgetSeconds, fmt.Sprintf("%s|%s|%d|%d|%d", verb, item, in.Count, in.X, in.Z))
}

func (c *call) setRecipe(ctx context.Context, in recipeArgs) (*mcp.CallToolResult, error) {
	recipe, err := arg("recipe", in.Recipe)
	if err != nil {
		return nil, err
	}
	return c.act(ctx, 0, fmt.Sprintf("assembler.setrecipe|%s|%d|%d", recipe, in.X, in.Z))
}

func (c *call) configure(ctx context.Context, in configureArgs) (*mcp.CallToolResult, error) {
	at := fmt.Sprintf("%d|%d", in.X, in.Z)
	value := func() (string, error) { return arg("value", in.Value) }
	var cmd string
	switch strings.ToLower(strings.TrimSpace(in.Kind)) {
	case "filter_mode":
		v := strings.ToLower(strings.TrimSpace(in.Value))
		if v != "allow" && v != "block" {
			return nil, fmt.Errorf("filter_mode needs value allow or block, not %q", in.Value)
		}
		cmd = "setting.filterall|" + v + "|" + at
	case "filter_item", "splitter_priority", "rename", "hauler_stop", "hauler_enable":
		v, err := value()
		if err != nil {
			return nil, err
		}
		verb := map[string]string{
			"filter_item": "setting.filteritem", "splitter_priority": "setting.splitterpriority",
			"rename": "setting.name", "hauler_stop": "hauler.addstop", "hauler_enable": "hauler.enable",
		}[strings.ToLower(strings.TrimSpace(in.Kind))]
		cmd = verb + "|" + v + "|" + at
	case "requester_items":
		v, err := value()
		if err != nil {
			return nil, err
		}
		if in.Count < 0 {
			return nil, fmt.Errorf("count must not be negative")
		}
		cmd = fmt.Sprintf("setting.requesteritems|%s|%d|%s", v, in.Count, at)
	case "power_target":
		switch {
		case strings.EqualFold(strings.TrimSpace(in.Value), "clear"):
			cmd = "setting.powerdistributor|" + at + "|clear"
		case in.TargetX != nil && in.TargetZ != nil:
			cmd = fmt.Sprintf("setting.powerdistributor|%s|%d|%d", at, *in.TargetX, *in.TargetZ)
		default:
			return nil, fmt.Errorf("power_target needs target_x and target_z, or value 'clear'")
		}
	default:
		return nil, fmt.Errorf("unknown kind %q; use filter_item, filter_mode, splitter_priority, requester_items, power_target, rename, hauler_stop or hauler_enable", in.Kind)
	}
	return c.act(ctx, 0, cmd)
}

func (c *call) ui(ctx context.Context, in uiArgs) (*mcp.CallToolResult, error) {
	panel, err := arg("panel", in.Panel)
	if err != nil {
		return nil, err
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	control := strings.TrimSpace(in.Control)
	if strings.ContainsAny(control, ";") {
		return nil, fmt.Errorf("control must not contain ';'")
	}
	switch action {
	case "open", "close":
		return c.act(ctx, 0, "ui."+action+"|"+panel)
	case "read":
		if control == "" {
			return c.act(ctx, 0, "ui.read|"+panel)
		}
		return c.act(ctx, 0, "ui.read|"+panel+"|"+control)
	case "click":
		if control == "" {
			return nil, fmt.Errorf("click needs a control")
		}
		return c.act(ctx, 0, "ui.click|"+panel+"|"+control)
	case "set":
		if control == "" {
			return nil, fmt.Errorf("set needs a control and a value")
		}
		// ui.set joins everything after the control back together, so the value may hold '|'.
		return c.act(ctx, 0, "ui.set|"+panel+"|"+control+"|"+in.Value)
	}
	return nil, fmt.Errorf("action must be open, close, click, set or read, not %q", in.Action)
}

func (c *call) waitUntil(ctx context.Context, in waitUntilArgs) (*mcp.CallToolResult, error) {
	pred, err := arg("predicate", in.Predicate)
	if err != nil {
		return nil, err
	}
	parts := []string{"waituntil", pred}
	for i, a := range in.Args {
		v, err := arg(fmt.Sprintf("args[%d]", i), a)
		if err != nil {
			return nil, err
		}
		parts = append(parts, v)
	}
	timeout := in.TimeoutSeconds
	if timeout <= 0 {
		timeout = 120
	}
	parts = append(parts, strconv.Itoa(int(math.Ceil(timeout))))
	return c.act(ctx, in.BudgetSeconds, strings.Join(parts, "|"))
}

func (c *call) waitFor(ctx context.Context, in waitForArgs) (*mcp.CallToolResult, error) {
	id := strings.TrimSpace(in.ChainID)
	if id == "" {
		return nil, fmt.Errorf("chain_id is required")
	}
	if _, err := strconv.Atoi(id); err == nil {
		return c.waitEntry(ctx, id, c.budget(in.BudgetSeconds))
	}
	o, err := c.follow(ctx, id, c.budget(in.BudgetSeconds))
	if err != nil {
		return nil, err
	}
	text, isErr := o.text()
	return textResult(text, isErr), nil
}

// waitEntry follows a numeric waituntil id through /v1/wait.
func (c *call) waitEntry(ctx context.Context, id string, budget time.Duration) (*mcp.CallToolResult, error) {
	start := time.Now()
	for {
		remaining := budget - time.Since(start)
		var st channel.WaitStatus
		var raw json.RawMessage
		err := c.do(ctx, func(cl *channel.Client) (err error) {
			st, raw, err = cl.Wait(ctx, id, int(max(0, min(remaining, c.s.pollSlice)).Milliseconds()))
			return err
		})
		if err != nil {
			return nil, err
		}
		if st.Terminal() || remaining <= 0 {
			summary := fmt.Sprintf("wait %s (%s): %s", id, st.Predicate, st.Status)
			if !st.Terminal() {
				summary += " — still pending; call wait_for again"
			}
			res := summaryJSON(summary, json.RawMessage(raw))
			res.IsError = st.Terminal() && st.Status != "satisfied"
			return res, nil
		}
		c.progress(ctx, time.Since(start), budget, "wait "+id+" pending")
	}
}

func (c *call) runCommands(ctx context.Context, in runArgs) (*mcp.CallToolResult, error) {
	var commands []string
	for _, cmd := range in.Commands {
		if strings.TrimSpace(cmd) != "" {
			commands = append(commands, cmd)
		}
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("commands is empty")
	}
	if in.Wait != nil && !*in.Wait {
		o, err := c.run(ctx, commands, 0)
		if err != nil {
			return nil, err
		}
		if o.Status == "running" {
			return textResult(fmt.Sprintf("submitted: chain %s is running — call wait_for with chain_id %q for the outcome", o.ChainID, o.ChainID), false), nil
		}
		text, isErr := o.text()
		return textResult(text, isErr), nil
	}
	return c.act(ctx, in.BudgetSeconds, commands...)
}

// ---------------------------------------------------------------------------- JSON helpers

func dig(v any, path ...string) any {
	for _, p := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[p]
	}
	return v
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// num reads a JSON number, or a numeric string: the game writes fixed-point values losslessly as
// strings (PlaytestFp.ToLossless).
func num(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	}
	return 0, false
}

func trimFloat(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// counts renders "(3 built, 1 frame)" with the keys sorted, or "" when empty.
func counts(m map[string]int) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		if k == "" {
			k = "unknown"
		}
		parts[i] = fmt.Sprintf("%d %s", m[keys[i]], k)
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

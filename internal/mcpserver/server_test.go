package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/channel"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/fakegame"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/mcpserver"
)

var kit = fstest.MapFS{
	"HowToPlay.md": {Data: []byte("# How to play\nMine, craft, build.\n")},
	"commands.md":  {Data: []byte("| `ffauto:help` |\n")},
	".claude/skills/play-the-game/SKILL.md": {Data: []byte(
		"---\nname: play-the-game\ndescription: Play Final Factory from wherever the save is.\n---\n\n# Play the game\nStart with game_status.\n")},
}

type env struct {
	game     *fakegame.Game
	session  *mcp.ClientSession
	progress atomic.Int32
}

// deadPID is a pid the stale-session tests treat as not running.
const deadPID = 999999

func start(t *testing.T, dir string, takeover bool) *env {
	t.Helper()
	e := &env{}
	conn := &channel.Conn{Discovery: discovery.Options{Dir: dir}, Attach: true, Takeover: takeover,
		Replaceable: mcpserver.StaleSession(func(pid int) bool { return pid != deadPID })}
	server := mcpserver.New(mcpserver.Options{Conn: conn, Kit: kit, Version: "test",
		DefaultBudget: 5 * time.Second, PollSlice: 30 * time.Millisecond})
	serverT, clientT := mcp.NewInMemoryTransports()
	ctx := context.Background()
	if _, err := server.Connect(ctx, serverT, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, &mcp.ClientOptions{
		ProgressNotificationHandler: func(context.Context, *mcp.ProgressNotificationClientRequest) { e.progress.Add(1) },
	})
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	e.session = cs
	return e
}

func withGame(t *testing.T) *env {
	t.Helper()
	g := fakegame.Start()
	t.Cleanup(g.Close)
	dir := filepath.Join(t.TempDir(), "Never Games", "finalfactory", "AgentControl")
	g.WriteSessionFile(dir, os.Getpid())
	e := start(t, dir, false)
	e.game = g
	return e
}

func (e *env) call(t *testing.T, name string, args map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	params := &mcp.CallToolParams{Name: name, Arguments: args}
	params.SetProgressToken("p1")
	res, err := e.session.CallTool(context.Background(), params)
	if err != nil {
		t.Fatalf("%s: protocol error %v", name, err)
	}
	var text []string
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text = append(text, tc.Text)
		}
	}
	return res, strings.Join(text, "\n")
}

func TestListsToolsResourcesPrompts(t *testing.T) {
	e := withGame(t)
	ctx := context.Background()

	tools, err := e.session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]*mcp.Tool{}
	for _, tool := range tools.Tools {
		names[tool.Name] = tool
	}
	for _, want := range []string{"game_status", "look_around", "inventory", "research_status", "enemies", "screenshot",
		"move_to", "mine", "build", "deconstruct", "craft", "cancel_craft", "research", "put_items", "take_items",
		"set_recipe", "configure", "ui", "wait_until", "wait_for", "run_commands", "list_commands",
		"inspect", "aim_mass_driver", "give_ships", "take_ships", "respawn"} {
		if names[want] == nil {
			t.Errorf("tool %s is not registered", want)
		}
	}
	schema, _ := json.Marshal(names["move_to"].InputSchema)
	for _, want := range []string{`"x"`, `"z"`, `"budget_seconds"`, `"required":["x","z"]`} {
		if !bytes.Contains(schema, []byte(want)) {
			t.Errorf("move_to schema lacks %s: %s", want, schema)
		}
	}

	resources, err := e.session.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	uris := map[string]bool{}
	for _, r := range resources.Resources {
		uris[r.URI] = true
	}
	for _, want := range []string{mcpserver.GuideURI, mcpserver.CommandsURI, mcpserver.SkillURIPrefix + "play-the-game"} {
		if !uris[want] {
			t.Errorf("resource %s missing (have %v)", want, uris)
		}
	}
	guide, err := e.session.ReadResource(ctx, &mcp.ReadResourceParams{URI: mcpserver.GuideURI})
	if err != nil || !strings.Contains(guide.Contents[0].Text, "Mine, craft, build.") {
		t.Fatalf("guide: %v", err)
	}

	prompts, err := e.session.ListPrompts(ctx, nil)
	if err != nil || len(prompts.Prompts) != 1 || prompts.Prompts[0].Name != "play-the-game" ||
		prompts.Prompts[0].Description != "Play Final Factory from wherever the save is." {
		t.Fatalf("prompts: %+v, %v", prompts, err)
	}
	got, err := e.session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "play-the-game", Arguments: map[string]string{"goal": "reach orbit"}})
	if err != nil {
		t.Fatal(err)
	}
	text := got.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.HasPrefix(text, "# Play the game") || !strings.HasSuffix(text, "Goal for this run: reach orbit") {
		t.Fatalf("prompt text %q", text)
	}

	init := e.session.InitializeResult()
	if lines := strings.Count(init.Instructions, "\n") + 1; lines > 25 {
		t.Fatalf("instructions are %d lines; keep them to 25", lines)
	}
}

func TestMoveToConvertsTilesAndPolls(t *testing.T) {
	e := withGame(t)
	e.game.OnCommand = func([]string) fakegame.Outcome {
		return fakegame.Outcome{Polls: 3, Result: "driving toward (-295,205)"}
	}
	// The fake player snapshot stands on tile (2,3); ask for exactly that tile so it "arrives".
	res, text := e.call(t, "move_to", map[string]any{"x": 2, "z": 3})
	if res.IsError {
		t.Fatalf("error: %s", text)
	}
	if got := e.game.Submitted[0][0]; got != "ffauto:movement.goto|25|35|15" {
		t.Fatalf("tile (2,3) with the default 1.5-tile tolerance must be world (25,35) tolerance 15, sent %q", got)
	}
	if e.game.ChainPolls < 3 {
		t.Fatalf("chain was polled %d times", e.game.ChainPolls)
	}
	if !strings.Contains(text, "Arrived: you are at tile (2,3)") {
		t.Fatalf("got %q", text)
	}
	if e.progress.Load() == 0 {
		t.Fatal("no progress notification was sent while the chain ran")
	}
	if got := e.game.Attached(); got != fmt.Sprintf("ff-agent/test mcp pid=%d (test-client)", os.Getpid()) {
		t.Fatalf("session label = %q", got)
	}

	// Negative tiles and an explicit tolerance/timeout.
	e.game.OnCommand = nil
	e.call(t, "move_to", map[string]any{"x": -30, "z": 20, "tolerance": 0.5, "timeout_seconds": 60})
	if got := e.game.Submitted[1][0]; got != "ffauto:movement.goto|-295|205|5|60" {
		t.Fatalf("sent %q", got)
	}
}

func TestStillRunningPointsAtWaitFor(t *testing.T) {
	e := withGame(t)
	e.game.OnCommand = func([]string) fakegame.Outcome { return fakegame.Outcome{Polls: -1} }
	res, text := e.call(t, "mine", map[string]any{"item": "Iron Ore", "count": 50, "budget_seconds": 0.2})
	if res.IsError || !strings.Contains(text, "still running: chain c1") || !strings.Contains(text, "wait_for") {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
	if got := e.game.Submitted[0][0]; got != "ffauto:mining.until|Iron Ore|50" {
		t.Fatalf("sent %q", got)
	}
	res, text = e.call(t, "wait_for", map[string]any{"chain_id": "c1", "budget_seconds": 0.1})
	if res.IsError || !strings.Contains(text, "still running: chain c1") {
		t.Fatalf("wait_for: %q", text)
	}
}

func TestRefusalIsAToolError(t *testing.T) {
	e := withGame(t)
	e.game.OnCommand = func([]string) fakegame.Outcome {
		return fakegame.Outcome{Reject: "honest_play_denied", Detail: "the target is out of the player's reach"}
	}
	res, text := e.call(t, "put_items", map[string]any{"item": "Iron Plate", "count": 5, "x": 40, "z": 40})
	if !res.IsError || !strings.Contains(text, "refused (honest_play_denied): the target is out of the player's reach") ||
		!strings.Contains(text, "Do not try to work around it") {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
	if got := e.game.Submitted[0][0]; got != "ffauto:transfer.put|Iron Plate|5|40|40" {
		t.Fatalf("sent %q", got)
	}
}

func TestSessionExistsIsAToolError(t *testing.T) {
	e := withGame(t)
	e.game.AttachOther("another agent")
	res, text := e.call(t, "game_status", nil)
	if !res.IsError || !strings.Contains(text, "already attached") || !strings.Contains(text, "--takeover") {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
}

func TestDeadMCPOwnerIsTakenOverAutomatically(t *testing.T) {
	e := withGame(t)
	e.game.AttachOther(fmt.Sprintf("ff-agent/0.1.0 mcp pid=%d (claude-code)", deadPID))
	res, text := e.call(t, "game_status", nil)
	if res.IsError {
		t.Fatalf("a session whose MCP server is dead must be replaced: %q", text)
	}
	if got := e.game.Attached(); !strings.Contains(got, fmt.Sprintf("pid=%d", os.Getpid())) {
		t.Fatalf("attached = %q", got)
	}
}

func TestLiveMCPOwnerIsNotTakenOver(t *testing.T) {
	e := withGame(t)
	live := fmt.Sprintf("ff-agent/0.1.0 mcp pid=%d (claude-code)", os.Getpid())
	e.game.AttachOther(live)
	res, text := e.call(t, "game_status", nil)
	if !res.IsError || !strings.Contains(text, "already attached") || e.game.Attached() != live {
		t.Fatalf("a live MCP server's session must be kept: error=%v %q, attached %q", res.IsError, text, e.game.Attached())
	}
}

func TestStaleSessionLabels(t *testing.T) {
	stale := mcpserver.StaleSession(func(pid int) bool { return pid == 42 })
	for label, want := range map[string]bool{
		"ff-agent/0.1.0 mcp pid=42 (claude-code)": false,
		"ff-agent/0.1.0 mcp pid=43 (claude-code)": true,
		"ff-agent/0.1.0 cli":                      true,
		"ff-agent/0.1.0 mcp (old label, no pid)":  false,
		"someone's own script":                    false,
	} {
		if got := stale(label); got != want {
			t.Errorf("StaleSession(%q) = %v, want %v", label, got, want)
		}
	}
}

func TestRateLimitIsAToolError(t *testing.T) {
	e := withGame(t)
	e.game.RateLimitCommands = 1
	res, text := e.call(t, "craft", map[string]any{"item": "Gear", "count": 2})
	if !res.IsError || !strings.Contains(text, "rate_limited") || !strings.Contains(text, "Retry in 1s") {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
	// The server survives it.
	res, text = e.call(t, "craft", map[string]any{"item": "Gear", "count": 2})
	if res.IsError || text != "done: ok" {
		t.Fatalf("second call: error=%v %q", res.IsError, text)
	}
}

func TestScreenshotIsImageContent(t *testing.T) {
	e := withGame(t)
	res, text := e.call(t, "screenshot", map[string]any{"max_edge": 640})
	if res.IsError {
		t.Fatal(text)
	}
	img, ok := res.Content[0].(*mcp.ImageContent)
	if !ok || img.MIMEType != "image/png" || !bytes.Equal(img.Data, fakegame.PNG()) {
		t.Fatalf("first content is not the PNG: %#v", res.Content[0])
	}
	if !strings.Contains(text, "heartbeat 1200") {
		t.Fatalf("got %q", text)
	}

	e.game.NoFrame = true
	res, text = e.call(t, "screenshot", nil)
	if !res.IsError || !strings.Contains(text, "no_frame_available") {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
}

func TestLookAroundCentresOnThePlayer(t *testing.T) {
	e := withGame(t)
	e.game.Snapshots["nearby"] = map[string]any{
		"center": map[string]any{"x": 2, "z": 3}, "radius": 20,
		"structures": []any{map[string]any{"item": "Miner", "state": "built"}, map[string]any{"item": "Miner", "state": "frame"}},
		"asteroids":  []any{map[string]any{"oreType": "Iron Ore", "oreRemaining": 500}},
	}
	res, text := e.call(t, "look_around", nil)
	if res.IsError {
		t.Fatal(text)
	}
	if got := e.game.SnapshotReads[len(e.game.SnapshotReads)-1]; got != "nearby?r=20&x=2&z=3" {
		t.Fatalf("read %q", got)
	}
	if !strings.HasPrefix(text, "Around your tile (2,3), radius 20: 2 structures (1 built, 1 frame), 1 asteroids (1 Iron Ore).\n{") {
		t.Fatalf("got %q", text)
	}
}

func TestRunCommandsAcceptsBareVerbs(t *testing.T) {
	e := withGame(t)
	res, text := e.call(t, "run_commands", map[string]any{"commands": []string{"ffauto:help", "craft.queue|Gear|1"}})
	if res.IsError || text != "done: ok" {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
	if got := strings.Join(e.game.Submitted[0], ";"); got != "ffauto:help;ffauto:craft.queue|Gear|1" {
		t.Fatalf("sent %q", got)
	}
}

func TestBuildByNameUsesPlaceItem(t *testing.T) {
	e := withGame(t)
	e.game.OnCommand = func([]string) fakegame.Outcome {
		return fakegame.Outcome{Result: "placed Mining Station at tile (5,-2)"}
	}
	res, text := e.call(t, "build", map[string]any{"item": "Mining Station", "x": 4, "z": -2, "facing": "down"})
	if res.IsError {
		t.Fatal(text)
	}
	if got := strings.Join(e.game.Submitted[0], ";"); got != "ffauto:construction.placeitem|Mining Station|4|-2|2" {
		t.Fatalf("sent %q", got)
	}
	if !strings.Contains(text, "at tile (5,-2)") || !strings.Contains(text, "can differ") {
		t.Fatalf("the placed tile must reach the agent: %q", text)
	}

	e.call(t, "build", map[string]any{"item": "Assembler", "x": 1, "z": 1})
	if got := e.game.Submitted[1][0]; got != "ffauto:construction.placeitem|Assembler|1|1" {
		t.Fatalf("no facing: sent %q", got)
	}
}

func TestBuildBlueprintStringKeepsPlaceAndRotates(t *testing.T) {
	e := withGame(t)
	e.call(t, "build", map[string]any{"item": "ffblueprintstartAAA", "x": 4, "z": -2, "facing": "left"})
	want := "ffauto:construction.place|ffblueprintstartAAA|4|-2;ffauto:waituntil|structureAt|4|-2|frame|30;ffauto:rotate.abs|4|-2|3"
	if got := strings.Join(e.game.Submitted[0], ";"); got != want {
		t.Fatalf("sent %q", got)
	}
}

func TestInspectReadsTheStructureScope(t *testing.T) {
	e := withGame(t)
	e.game.Snapshots["structure"] = map[string]any{"item": "Assembler", "status": "working",
		"recipe": "Gear", "contents": []any{map[string]any{"item": "Iron Plate", "count": 12}}}
	res, text := e.call(t, "inspect", map[string]any{"x": 4, "z": -2})
	if res.IsError {
		t.Fatal(text)
	}
	if got := e.game.SnapshotReads[len(e.game.SnapshotReads)-1]; got != "structure?x=4&z=-2" {
		t.Fatalf("read %q", got)
	}
	if !strings.HasPrefix(text, "Assembler at tile (4,-2), working.\n{") || !strings.Contains(text, `"recipe":"Gear"`) {
		t.Fatalf("got %q", text)
	}

	e.game.Snapshots["structure"] = map[string]any{"unavailable": "no structure at tile (9,9)"}
	res, text = e.call(t, "inspect", map[string]any{"x": 9, "z": 9})
	if !res.IsError || !strings.Contains(text, "no structure at tile (9,9)") {
		t.Fatalf("got error=%v %q", res.IsError, text)
	}
}

func TestNewSettingAndFleetVerbs(t *testing.T) {
	e := withGame(t)
	e.call(t, "configure", map[string]any{"kind": "filter_mode", "value": "Block", "x": 1, "z": 2})
	e.call(t, "aim_mass_driver", map[string]any{"x": 1, "z": 2, "target_x": -40, "target_z": 7})
	e.call(t, "aim_mass_driver", map[string]any{"x": 1, "z": 2, "clear": true})
	e.call(t, "give_ships", map[string]any{"ship": "Bat", "count": 3, "x": 1, "z": 2})
	e.call(t, "take_ships", map[string]any{"ship": "Bat", "count": 1, "x": 1, "z": 2})
	e.call(t, "respawn", nil)
	want := []string{
		"ffauto:setting.filterall|block|1|2",
		"ffauto:setting.massdriver|1|2|-40|7",
		"ffauto:setting.massdriver|1|2|clear",
		"ffauto:fleettransfer.give|Bat|3|1|2",
		"ffauto:fleettransfer.take|Bat|1|1|2",
		"ffauto:combat.respawn",
	}
	for i, w := range want {
		if got := e.game.Submitted[i][0]; got != w {
			t.Errorf("call %d sent %q, want %q", i, got, w)
		}
	}
}

func TestPipeInArgumentIsRejected(t *testing.T) {
	e := withGame(t)
	res, text := e.call(t, "craft", map[string]any{"item": "Gear|999", "count": 1})
	if !res.IsError || len(e.game.Submitted) != 0 {
		t.Fatalf("an argument with '|' must never reach the game: %q", text)
	}
}

func TestNoGameMessage(t *testing.T) {
	e := start(t, t.TempDir(), false)
	for _, tool := range []string{"game_status", "look_around", "screenshot"} {
		res, text := e.call(t, tool, nil)
		if !res.IsError || text != mcpserver.NotRunningMessage {
			t.Fatalf("%s: error=%v %q", tool, res.IsError, text)
		}
	}
}

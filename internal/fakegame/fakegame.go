// Package fakegame is an in-process stand-in for the game's /v1 agent channel, for tests. It
// mirrors the router's answers (AgentRequestRouter.cs in the game repo) closely enough to drive
// the client, the CLI and the MCP server end to end; it is never linked into the binary.
package fakegame

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Outcome scripts how the game answers one submitted chain.
type Outcome struct {
	// Reject makes POST /v1/command answer status "rejected" with this reason.
	Reject string
	Detail string
	Result string
	// Polls is how many /v1/chain reads report "running" first; 0 completes synchronously, and
	// a negative value never finishes.
	Polls int
	// Final is the terminal status: "completed" (default), "failed" or "cancelled".
	Final       string
	FinalReason string
}

type chain struct {
	id        string
	commands  []string
	outcome   Outcome
	status    string
	pollsLeft int
}

// Game is a scripted game. Set fields before the first request; read the recorded ones after.
type Game struct {
	Server *httptest.Server

	mu sync.Mutex
	// Token is the bearer token; Rotate changes it (a restarted game).
	token     string
	State     string
	Heartbeat int64
	// Commands is what /v1/help reports.
	Commands []string
	// Snapshots maps a scope to its data object.
	Snapshots map[string]any
	// OnCommand scripts each chain; nil completes every chain synchronously.
	OnCommand func(commands []string) Outcome
	// RateLimitCommands answers the next N POST /v1/command with 429.
	RateLimitCommands int
	// NoFrame answers /v1/screenshot with 409 no_frame_available.
	NoFrame bool
	// PollSleep bounds how long a long poll really sleeps, to keep tests fast.
	PollSleep time.Duration

	sessionID, sessionLabel string
	chains                  map[string]*chain
	nextChain               int

	// Recorded traffic.
	Submitted     [][]string
	SnapshotReads []string
	Labels        []string
	ChainPolls    int
}

// Start runs a fake game on 127.0.0.1 with sensible defaults.
func Start() *Game {
	g := &Game{
		token:     "test-token",
		State:     "playing",
		Heartbeat: 1200,
		Commands:  []string{"help", "movement.goto", "mining.until", "construction.place", "craft.queue", "waituntil"},
		Snapshots: map[string]any{
			"player": map[string]any{
				"position": map[string]any{
					"world": map[string]any{"x": "25", "y": "0", "z": "35"},
					"tile":  map[string]any{"x": 2, "z": 3},
				},
				"health": map[string]any{"current": "80", "max": "100"},
			},
		},
		PollSleep: 20 * time.Millisecond,
		chains:    map[string]*chain{},
	}
	g.Server = httptest.NewServer(http.HandlerFunc(g.serve))
	return g
}

// Close stops the server.
func (g *Game) Close() { g.Server.Close() }

// Port is the loopback port the fake listens on.
func (g *Game) Port() int {
	u, _ := url.Parse(g.Server.URL)
	p, _ := strconv.Atoi(u.Port())
	return p
}

// Token returns the current bearer token.
func (g *Game) Token() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.token
}

// Rotate changes the token, as a game restart or an off/on toggle does.
func (g *Game) Rotate(token string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.token = token
}

// Attached returns the attached session's label, or "".
func (g *Game) Attached() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.sessionLabel
}

// AttachOther simulates another client already attached.
func (g *Game) AttachOther(label string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sessionID, g.sessionLabel = "s-other", label
}

// WriteSessionFile writes session-{pid}.json into dir the way AgentChannelHost does.
func (g *Game) WriteSessionFile(dir string, pid int) string {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	path := filepath.Join(dir, fmt.Sprintf("session-%d.json", pid))
	data, _ := json.MarshalIndent(map[string]any{
		"port": g.Port(), "token": g.Token(), "pid": pid, "gameVersion": "0.50.0.24",
		"catalogHash": "47b04d0d1af16f65", "tier": "player-safe", "kitPath": "",
		"startedUtc": time.Now().UTC().Format(time.RFC3339Nano),
	}, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		panic(err)
	}
	return path
}

// PNG is the image the fake serves: 4x3, so tests can check the dimensions round-trip.
func PNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 3))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func (g *Game) serve(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "Bearer "+g.Token() {
		writeError(w, 401, "unauthorized", "every /v1 route requires 'Authorization: Bearer <token>' from session-{pid}.json")
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "v1" {
		writeError(w, 404, "unknown_command", "no route")
		return
	}
	switch parts[1] {
	case "hello":
		g.hello(w)
	case "session":
		g.session(w, r)
	case "help":
		g.help(w)
	case "command":
		g.command(w, r)
	case "chain":
		g.chainStatus(w, r, parts)
	case "wait":
		writeJSON(w, 200, map[string]any{"id": 1, "predicate": "heartbeat|10", "status": "satisfied", "detail": "ok"})
	case "snapshot":
		g.snapshot(w, r, parts)
	case "screenshot":
		g.screenshot(w)
	case "journal":
		writeJSON(w, 200, map[string]any{"label": "t", "eventCount": 0, "events": []any{}})
	default:
		writeError(w, 404, "unknown_command", "no route")
	}
}

func (g *Game) hello(w http.ResponseWriter) {
	g.mu.Lock()
	defer g.mu.Unlock()
	body := map[string]any{
		"gameVersion": "0.50.0.24", "catalogHash": "47b04d0d1af16f65", "tier": "player-safe",
		"state": g.State, "frameCount": 5000, "realtimeSinceStartup": 12.5, "heartbeat": g.Heartbeat, "epoch": 1,
		"kitPath": "",
		"limits": map[string]any{
			"snapshotsPerSecond": 10, "screenshotsPerSecond": 2, "runningChains": 1, "queuedChains": 4,
			"chainSegments": 64, "bodyBytes": 65536, "longPollMs": 30000, "screenshotMaxEdge": 1920,
			"saveNamePrefix": "claude_playtest_",
		},
	}
	if g.sessionID != "" {
		body["session"] = map[string]any{"id": g.sessionID, "clientLabel": g.sessionLabel}
	}
	writeJSON(w, 200, body)
}

func (g *Game) session(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if r.Method == http.MethodDelete {
		g.sessionID, g.sessionLabel = "", ""
		w.WriteHeader(204)
		return
	}
	var body struct {
		ClientLabel string `json:"clientLabel"`
		Takeover    bool   `json:"takeover"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if g.sessionID != "" && !body.Takeover {
		writeError(w, 409, "session_exists", fmt.Sprintf("session '%s' (%s) is attached; POST with takeover:true to replace it", g.sessionID, g.sessionLabel))
		return
	}
	g.sessionID, g.sessionLabel = "s1", body.ClientLabel
	g.Labels = append(g.Labels, body.ClientLabel)
	writeJSON(w, 201, map[string]any{"id": g.sessionID, "clientLabel": body.ClientLabel})
}

func (g *Game) help(w http.ResponseWriter) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var families []map[string]any
	index := map[string]int{}
	for _, name := range g.Commands {
		family, _, _ := strings.Cut(name, ".")
		i, ok := index[family]
		if !ok {
			i = len(families)
			index[family] = i
			families = append(families, map[string]any{"name": family, "commands": []map[string]any{}})
		}
		families[i]["commands"] = append(families[i]["commands"].([]map[string]any),
			map[string]any{"name": name, "args": "<x>|<z>", "summary": "does " + name})
	}
	writeJSON(w, 200, map[string]any{"actor": "player-safe", "catalogHash": "47b04d0d1af16f65", "families": families})
}

func (g *Game) command(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Actor   string   `json:"actor"`
		Command string   `json:"command"`
		Chain   []string `json:"chain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, 400, "bad_arguments", "the body is not a JSON object")
		return
	}
	if body.Actor != "local-player" {
		writeError(w, 400, "bad_arguments", `every /v1/command body must carry "actor": "local-player"`)
		return
	}
	commands := body.Chain
	if body.Command != "" {
		commands = []string{body.Command}
	}

	g.mu.Lock()
	if g.RateLimitCommands > 0 {
		g.RateLimitCommands--
		g.mu.Unlock()
		w.Header().Set("Retry-After", "1")
		writeError(w, 429, "rate_limited", "1 chain runs at a time with at most 4 queued; poll /v1/chain/{id} for the one in flight")
		return
	}
	if g.State != "playing" {
		state := g.State
		g.mu.Unlock()
		writeError(w, 409, "state_not_playable", "commands need a running game; the game is '"+state+"'")
		return
	}
	g.Submitted = append(g.Submitted, commands)
	hook := g.OnCommand
	g.mu.Unlock()

	outcome := Outcome{Result: "ok"}
	if hook != nil {
		outcome = hook(commands)
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	g.nextChain++
	c := &chain{id: "c" + strconv.Itoa(g.nextChain), commands: commands, outcome: outcome, status: "running", pollsLeft: outcome.Polls}
	g.chains[c.id] = c
	envelope := map[string]any{"chainId": c.id, "heartbeat": g.Heartbeat, "epoch": 1}
	switch {
	case outcome.Reject != "":
		c.status = "failed"
		envelope["status"], envelope["reason"], envelope["detail"] = "rejected", outcome.Reject, outcome.Detail
		envelope["result"] = outcome.Detail
	case outcome.Polls == 0:
		c.finish()
		envelope["status"], envelope["result"] = "completed", outcome.Result
	default:
		envelope["status"], envelope["result"] = "accepted", ""
	}
	writeJSON(w, 200, envelope)
}

func (c *chain) finish() {
	c.status = c.outcome.Final
	if c.status == "" {
		c.status = "completed"
	}
}

func (g *Game) chainStatus(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 3 {
		writeError(w, 404, "bad_arguments", "expected /v1/chain/{id}")
		return
	}
	timeoutMs, _ := strconv.Atoi(r.URL.Query().Get("timeoutMs"))
	if timeoutMs < 0 || timeoutMs > 30000 {
		writeError(w, 400, "bad_arguments", "timeoutMs must be 0..30000")
		return
	}
	g.mu.Lock()
	g.ChainPolls++
	c, ok := g.chains[parts[2]]
	sleep := g.PollSleep
	g.mu.Unlock()
	if !ok {
		writeError(w, 404, "bad_arguments", "no chain '"+parts[2]+"'")
		return
	}
	if timeoutMs > 0 {
		time.Sleep(min(time.Duration(timeoutMs)*time.Millisecond, sleep))
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if c.status == "running" && c.pollsLeft > 0 {
		c.pollsLeft--
		if c.pollsLeft == 0 {
			c.finish()
		}
	}
	body := map[string]any{"status": c.status, "detail": c.outcome.Detail, "result": "", "startedHeartbeat": 1200}
	if c.status != "running" {
		body["result"], body["endedHeartbeat"] = c.outcome.Result, g.Heartbeat+40
		if c.outcome.FinalReason != "" {
			body["reason"] = c.outcome.FinalReason
		}
	}
	writeJSON(w, 200, body)
}

func (g *Game) snapshot(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 3 {
		writeError(w, 404, "bad_arguments", "expected /v1/snapshot/{scope}")
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	scope := parts[2]
	g.SnapshotReads = append(g.SnapshotReads, scope+"?"+r.URL.RawQuery)
	data, ok := g.Snapshots[scope]
	if !ok {
		data = map[string]any{"center": map[string]any{"x": 0, "z": 0}, "structures": []any{}, "asteroids": []any{}}
	}
	writeJSON(w, 200, map[string]any{"scope": scope, "heartbeat": g.Heartbeat, "epoch": 1, "mode": "player", "data": data})
}

func (g *Game) screenshot(w http.ResponseWriter) {
	g.mu.Lock()
	noFrame, hb := g.NoFrame, g.Heartbeat
	g.mu.Unlock()
	if noFrame {
		writeError(w, 409, "no_frame_available", "no frame arrived in time (is the window minimised?)")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "image/png")
	h.Set("X-FF-Heartbeat", strconv.FormatInt(hb, 10))
	h.Set("X-FF-Epoch", "1")
	h.Set("X-FF-Width", "4")
	h.Set("X-FF-Height", "3")
	h.Set("X-FF-Captured-Utc", "2026-09-23T10:00:00Z")
	w.WriteHeader(200)
	_, _ = w.Write(PNG())
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, reason, detail string) {
	if status == 429 && w.Header().Get("Retry-After") == "" {
		w.Header().Set("Retry-After", "1")
	}
	writeJSON(w, status, map[string]any{"error": reason, "detail": detail})
}

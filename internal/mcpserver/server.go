// Package mcpserver is `ff-agent mcp`: a stdio MCP server that turns the game's /v1 agent channel
// into typed tools, with the kit's guide, command reference and skills as resources and prompts.
package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/channel"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
)

// NotRunningMessage is every tool's answer while no game has agent control on.
const NotRunningMessage = "Final Factory isn't running with agent control on. In the game, open the pause menu and switch Agent control on (bottom-left)."

// Options configures the server.
type Options struct {
	Conn    *channel.Conn
	Kit     fs.FS
	Version string
	// DefaultBudget is how long an action tool blocks before answering "still running".
	DefaultBudget time.Duration
	// PollSlice is one long poll; progress notifications go out between slices.
	PollSlice time.Duration
}

type server struct {
	conn          *channel.Conn
	version       string
	defaultBudget time.Duration
	pollSlice     time.Duration
}

const maxBudget = 10 * time.Minute

// New builds the server with every tool, resource and prompt registered.
func New(o Options) *mcp.Server {
	s := &server{conn: o.Conn, version: o.Version, defaultBudget: o.DefaultBudget, pollSlice: o.PollSlice}
	if s.defaultBudget <= 0 {
		s.defaultBudget = 90 * time.Second
	}
	if s.pollSlice <= 0 {
		s.pollSlice = 10 * time.Second
	}
	srv := mcp.NewServer(
		&mcp.Implementation{Name: "finalfactory", Title: "Final Factory", Version: o.Version},
		// A non-nil Capabilities drops the SDK's default logging capability, which this server
		// does not use; tools, resources and prompts are inferred from what is registered.
		&mcp.ServerOptions{Instructions: instructions, Capabilities: &mcp.ServerCapabilities{}},
	)
	s.registerTools(srv)
	registerDocs(srv, o.Kit)
	return srv
}

// Run serves over stdin/stdout until the client disconnects, then detaches from the game so the
// next agent is not refused with session_exists.
func Run(ctx context.Context, o Options) error {
	err := New(o).Run(ctx, &mcp.StdioTransport{})
	detachCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = o.Conn.Detach(detachCtx)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

// ≤ 25 lines by design: this is loaded into every session; the guide carries the detail.
const instructions = `Final Factory is a space automation game: mine asteroids, hand-craft, research, and build factories that construction bots assemble. These tools play it as the player whose game this is.

Work in an observe → act → verify loop:
1. game_status — state, your position, current objective, alerts.
2. look_around / inventory / research_status / enemies, or screenshot to see the screen.
3. Act with one tool (move_to, mine, craft, build, …). Action tools wait for the game and return the outcome; "still running: chain <id>" means call wait_for.
4. Verify with an observation before the next step — a placed building is only a frame until bots build it.

All coordinates are grid TILES (x, z), the same numbers look_around reports; the tools convert where the game needs world units. run_commands is the escape hatch for raw ffauto: commands (list_commands shows what you may run); there, movement.goto alone takes world units (tile × 10 + 5).

Honest play: the game refuses cheats, developer commands and anything out of the player's reach (capability_denied, honest_play_denied). Treat a refusal as a rule of the game — move closer, gather, or research instead. Never try to work around one.

Read finalfactory://guide/how-to-play before planning anything bigger than one step. The skills (MCP prompts, and finalfactory://skills/<name>) are tested plans for common goals.`

// mcpLabel matches the label addTool attaches with; the pid is what makes a leftover session
// provably abandoned.
var mcpLabel = regexp.MustCompile(`^ff-agent/\S+ mcp pid=(\d+)\b`)

// StaleSession reports whether an attached session's owner is provably gone, so a new server may
// replace it without --takeover: an ff-agent MCP server whose pid is no longer alive (killed, or
// crashed before it could detach), or a one-shot CLI command's implicit attach, which no process
// holds once the command has exited. Anything else, including a live MCP server, is left alone.
func StaleSession(alive func(pid int) bool) func(clientLabel string) bool {
	return func(label string) bool {
		if m := mcpLabel.FindStringSubmatch(label); m != nil {
			pid, err := strconv.Atoi(m[1])
			return err == nil && !alive(pid)
		}
		return strings.HasPrefix(label, "ff-agent/") && strings.HasSuffix(label, " cli")
	}
}

// call is one tool invocation.
type call struct {
	s   *server
	req *mcp.CallToolRequest
}

func addTool[In any](s *server, srv *mcp.Server, t *mcp.Tool, h func(*call, context.Context, In) (*mcp.CallToolResult, error)) {
	mcp.AddTool(srv, t, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		s.conn.SetLabel(fmt.Sprintf("ff-agent/%s mcp pid=%d (%s)", s.version, os.Getpid(), clientName(req)))
		res, err := h(&call{s: s, req: req}, ctx, in)
		if err != nil {
			return errorResult(err), nil, nil
		}
		return res, nil, nil
	})
}

func clientName(req *mcp.CallToolRequest) string {
	if req.Session != nil {
		if p := req.Session.InitializeParams(); p != nil && p.ClientInfo != nil && p.ClientInfo.Name != "" {
			return p.ClientInfo.Name
		}
	}
	// Protocol 2026-07-28 carries clientInfo per request instead of in an initialize handshake.
	if req.Params != nil {
		if info, ok := req.Params.Meta[mcp.MetaKeyClientInfo].(map[string]any); ok {
			if name, ok := info["name"].(string); ok && name != "" {
				return name
			}
		}
	}
	return "unknown client"
}

// do runs fn against the game, re-discovering it when it restarted.
func (c *call) do(ctx context.Context, fn func(*channel.Client) error) error {
	return c.s.conn.Do(ctx, fn)
}

func (c *call) progress(ctx context.Context, done, total time.Duration, msg string) {
	token := c.req.Params.GetProgressToken()
	if token == nil || c.req.Session == nil {
		return
	}
	_ = c.req.Session.NotifyProgress(ctx, &mcp.ProgressNotificationParams{
		ProgressToken: token, Progress: done.Seconds(), Total: total.Seconds(), Message: msg,
	})
}

func (c *call) budget(seconds float64) time.Duration {
	if seconds <= 0 {
		return c.s.defaultBudget
	}
	return min(time.Duration(seconds*float64(time.Second)), maxBudget)
}

// outcome is where a submitted chain ended up when the tool stopped waiting.
type outcome struct {
	ChainID string
	Status  string // completed, failed, cancelled, rejected, or running (budget spent)
	Result  string
	Detail  string
	Reason  string
}

// run submits commands as one chain and follows it for up to budget.
func (c *call) run(ctx context.Context, commands []string, budget time.Duration) (outcome, error) {
	var r channel.CommandResult
	err := c.do(ctx, func(cl *channel.Client) (err error) {
		r, _, err = cl.Submit(ctx, commands)
		return err
	})
	if err != nil {
		return outcome{}, err
	}
	switch r.Status {
	case "completed":
		return outcome{ChainID: r.ChainID, Status: "completed", Result: r.Result}, nil
	case "rejected":
		return outcome{ChainID: r.ChainID, Status: "rejected", Result: r.Result, Detail: r.Detail, Reason: r.Reason}, nil
	}
	return c.follow(ctx, r.ChainID, budget)
}

// follow long-polls a chain until it ends or the budget runs out.
func (c *call) follow(ctx context.Context, id string, budget time.Duration) (outcome, error) {
	start := time.Now()
	for {
		remaining := budget - time.Since(start)
		if remaining <= 0 {
			return outcome{ChainID: id, Status: "running"}, nil
		}
		var st channel.ChainStatus
		err := c.do(ctx, func(cl *channel.Client) (err error) {
			st, _, err = cl.Chain(ctx, id, int(min(remaining, c.s.pollSlice).Milliseconds()))
			return err
		})
		if err != nil {
			return outcome{}, err
		}
		if st.Terminal() {
			return outcome{ChainID: id, Status: st.Status, Result: st.Result, Detail: st.Detail, Reason: st.Reason}, nil
		}
		msg := "chain " + id + " running"
		if st.QueuePosition > 0 {
			msg = fmt.Sprintf("chain %s queued (position %d)", id, st.QueuePosition)
		}
		c.progress(ctx, time.Since(start), budget, msg)
	}
}

// text renders an outcome for the model; failures are tool errors so it can self-correct.
func (o outcome) text() (string, bool) {
	switch o.Status {
	case "completed":
		if o.Result == "" {
			return "done", false
		}
		return "done: " + o.Result, false
	case "running":
		return fmt.Sprintf("still running: chain %s — call wait_for with chain_id %q to keep waiting (the game keeps working on it meanwhile)", o.ChainID, o.ChainID), false
	case "rejected":
		return refusal(o.Reason, firstNonEmpty(o.Detail, o.Result)), true
	case "cancelled":
		return "cancelled: " + firstNonEmpty(o.Detail, "the game cancelled the chain (world reloaded, paused to the title screen, or the agent was detached)"), true
	default:
		if o.Reason != "" {
			return refusal(o.Reason, o.Detail), true
		}
		return "failed: " + firstNonEmpty(o.Detail, o.Result, "the game reported a failure"), true
	}
}

func refusal(reason, detail string) string {
	msg := fmt.Sprintf("refused (%s): %s", firstNonEmpty(reason, "rejected"), detail)
	switch reason {
	case "honest_play_denied", "capability_denied":
		msg += "\nThis is a rule of honest play: only what the player could do by hand, from where they stand, is allowed. Do not try to work around it — move closer, gather, or research instead."
	case "unknown_command", "bad_arguments":
		msg += "\nCheck the command's exact shape with list_commands."
	}
	return msg
}

func errorResult(err error) *mcp.CallToolResult {
	return textResult(errorText(err), true)
}

func errorText(err error) string {
	var (
		ambiguous *discovery.AmbiguousError
		conflict  *channel.SessionConflictError
		apiErr    *channel.APIError
		unreached *channel.UnreachableError
	)
	switch {
	case errors.Is(err, discovery.ErrNotEnabled):
		return NotRunningMessage
	case errors.As(err, &ambiguous):
		return ambiguous.Error() + ". Turn Agent control off in all but one game, or start this server with `ff-agent mcp --pid <pid>`."
	case errors.As(err, &conflict):
		return "Another agent is already attached to this game (" + conflict.Detail + "). Detach it with `ff-agent detach`, or restart this MCP server as `ff-agent mcp --takeover`."
	case errors.As(err, &apiErr):
		switch {
		case apiErr.Status == 401:
			return "The game rejected the agent token even after re-reading its session file. Switch Agent control off and on again in the pause menu."
		case apiErr.Reason == "rate_limited":
			wait := apiErr.RetryAfter
			if wait <= 0 {
				wait = time.Second
			}
			return fmt.Sprintf("The game is busy (rate_limited): %s. Retry in %s.", apiErr.Detail, wait)
		case apiErr.Reason == "state_not_playable":
			return "Not possible right now (state_not_playable): " + apiErr.Detail + ". A game must be loaded and unpaused."
		case apiErr.Reason == "no_frame_available":
			return "No screenshot (no_frame_available): " + apiErr.Detail
		default:
			return refusal(apiErr.Reason, apiErr.Detail)
		}
	case errors.As(err, &unreached):
		return "Lost contact with the game (" + unreached.Err.Error() + "). " + NotRunningMessage
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return "the call was cancelled before the game answered"
	}
	return err.Error()
}

func textResult(text string, isError bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}, IsError: isError}
}

// summaryJSON is the snapshot tools' shape: one plain-language line, then the compact JSON.
func summaryJSON(summary string, v any) *mcp.CallToolResult {
	data, err := json.Marshal(v)
	if err != nil {
		return textResult(summary, false)
	}
	return textResult(summary+"\n"+string(data), false)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

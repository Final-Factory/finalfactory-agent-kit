// Package cli is the ff-agent command line (contracts/ff-agent-cli.md in the game repo).
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/channel"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/kitdocs"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/mcpserver"
)

// Exit codes, from the contract.
const (
	ExitOK           = 0
	ExitFailure      = 1 // usage errors and anything the contract does not name
	ExitNotEnabled   = 2
	ExitUnauthorized = 3
	ExitRejected     = 4
	ExitTimeout      = 5
	ExitMismatch     = 6
)

// Env is everything the CLI touches outside its arguments, so tests can substitute it.
type Env struct {
	Stdout, Stderr io.Writer
	Kit            fs.FS
	Version        string
	// Discovery is the base discovery options; flags fill in File and PID.
	Discovery discovery.Options
	// PollSlice shortens the MCP server's long polls; tests only.
	PollSlice time.Duration
}

const usage = `ff-agent — drive Final Factory through its local agent channel

Usage:
  ff-agent hello [--strict]              game version, state, tier, limits; command-set diff
  ff-agent help [family]                 commands this game lets an agent run
  ff-agent cmd "<ffauto:…>" [--wait] [--timeout 120s]
  ff-agent chain "<c1>" "<c2>" … [--wait] [--timeout 120s]
  ff-agent wait <id> [--timeout 120s]    follow a waituntil id (or a chain id)
  ff-agent snapshot <scope> [--x N --z N --r N] [--items a,b] [--craftable] [--since N]
  ff-agent screenshot [-o file.png] [--max-edge 1280]
  ff-agent journal [--since N]
  ff-agent attach --label <name> [--takeover] | ff-agent detach
  ff-agent mcp [--takeover]              stdio MCP server for Claude Code and other agents
  ff-agent version

Global flags: --json (one JSON object per line), --quiet, --session <file>, --pid <pid>
Exit codes: 0 ok, 2 agent control not enabled, 3 unauthorized, 4 rejected, 5 timeout,
6 command-set mismatch (hello --strict), 1 anything else.
`

type globals struct {
	json, quiet bool
	session     string
	pid         int
}

type runner struct {
	env    Env
	g      globals
	conn   *channel.Conn
	stdout io.Writer
}

// Run executes one command line and returns the process exit code.
func Run(args []string, env Env) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprint(env.Stdout, usage)
		return ExitOK
	}
	r := &runner{env: env, stdout: env.Stdout}
	cmd, rest := args[0], args[1:]
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var err error
	switch cmd {
	case "hello":
		err = r.hello(ctx, rest)
	case "help":
		err = r.help(ctx, rest)
	case "cmd":
		err = r.command(ctx, rest, false)
	case "chain":
		err = r.command(ctx, rest, true)
	case "wait":
		err = r.wait(ctx, rest)
	case "snapshot":
		err = r.snapshot(ctx, rest)
	case "screenshot":
		err = r.screenshot(ctx, rest)
	case "journal":
		err = r.journal(ctx, rest)
	case "attach":
		err = r.attach(ctx, rest)
	case "detach":
		err = r.detach(ctx, rest)
	case "mcp":
		err = r.mcp(ctx, rest)
	case "version", "--version":
		fmt.Fprintln(env.Stdout, "ff-agent", env.Version)
	default:
		err = usageError("unknown command %q; run ff-agent --help", cmd)
	}
	return r.report(err)
}

// ----------------------------------------------------------------------------- errors

type exitError struct {
	code int
	msg  string
}

func (e *exitError) Error() string { return e.msg }

func usageError(format string, args ...any) error {
	return &exitError{code: ExitFailure, msg: fmt.Sprintf(format, args...)}
}

func (r *runner) report(err error) int {
	if err == nil {
		return ExitOK
	}
	code, msg := classify(err)
	if msg != "" {
		fmt.Fprintln(r.env.Stderr, "ff-agent:", msg)
	}
	return code
}

func classify(err error) (int, string) {
	var (
		exit      *exitError
		apiErr    *channel.APIError
		conflict  *channel.SessionConflictError
		ambiguous *discovery.AmbiguousError
		unreached *channel.UnreachableError
	)
	switch {
	case errors.As(err, &exit):
		return exit.code, exit.msg
	case errors.Is(err, discovery.ErrNotEnabled):
		return ExitNotEnabled, mcpserver.NotRunningMessage + " (" + err.Error() + ")"
	case errors.As(err, &ambiguous):
		return ExitNotEnabled, err.Error()
	case errors.As(err, &conflict):
		return ExitRejected, err.Error() + " (use --takeover, or ff-agent detach)"
	case errors.As(err, &apiErr):
		if apiErr.Status == 401 {
			return ExitUnauthorized, "the game rejected the token in its session file; switch Agent control off and on again"
		}
		msg := err.Error()
		if apiErr.RetryAfter > 0 {
			msg += fmt.Sprintf(" (retry after %s)", apiErr.RetryAfter)
		}
		return ExitRejected, msg
	case errors.As(err, &unreached):
		return ExitNotEnabled, err.Error()
	}
	return ExitFailure, err.Error()
}

// ----------------------------------------------------------------------------- flags

func (r *runner) flags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("ff-agent "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&r.g.json, "json", false, "machine output, one JSON object per line")
	fs.BoolVar(&r.g.quiet, "quiet", false, "print only what was asked for")
	fs.StringVar(&r.g.session, "session", "", "use this session-{pid}.json")
	fs.IntVar(&r.g.pid, "pid", 0, "pick the game with this process id")
	return fs
}

// parse lets flags follow positional arguments (`cmd "ffauto:x" --wait`), which package flag
// alone stops at. A literal "--" still ends flag parsing.
func parse(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, usageError("%s: %v", fs.Name(), err)
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(positional, rest...), nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

func (r *runner) connect() {
	opts := r.env.Discovery
	opts.File, opts.PID = r.g.session, r.g.pid
	r.conn = &channel.Conn{Discovery: opts}
}

func (r *runner) do(ctx context.Context, fn func(*channel.Client) error) error {
	if r.conn == nil {
		r.connect()
	}
	return r.conn.Do(ctx, fn)
}

// emit prints raw JSON: compact on one line for --json, indented otherwise.
func (r *runner) emit(raw []byte) {
	if r.g.json {
		var buf bytes.Buffer
		if json.Compact(&buf, raw) == nil {
			fmt.Fprintln(r.stdout, buf.String())
			return
		}
	} else {
		var buf bytes.Buffer
		if json.Indent(&buf, raw, "", "  ") == nil {
			fmt.Fprintln(r.stdout, buf.String())
			return
		}
	}
	fmt.Fprintln(r.stdout, strings.TrimSpace(string(raw)))
}

func (r *runner) emitValue(v any) {
	data, _ := json.Marshal(v)
	r.emit(data)
}

func (r *runner) say(format string, args ...any) {
	if !r.g.quiet {
		fmt.Fprintf(r.stdout, format+"\n", args...)
	}
}

// ----------------------------------------------------------------------------- commands

func (r *runner) hello(ctx context.Context, args []string) error {
	fs := r.flags("hello")
	strict := fs.Bool("strict", false, "exit 6 when the live command set differs from the kit's")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	var h channel.Hello
	var raw json.RawMessage
	var help channel.Help
	err := r.do(ctx, func(cl *channel.Client) (err error) {
		if h, raw, err = cl.Hello(ctx); err != nil {
			return err
		}
		help, _, err = cl.Help(ctx, "")
		return err
	})
	if err != nil {
		return err
	}

	embedded, haveRef := kitdocs.CommandNames(r.env.Kit)
	added, removed := kitdocs.Diff(embedded, help.CommandNames())
	mismatch := haveRef && (len(added) > 0 || len(removed) > 0)

	if r.g.json {
		var out map[string]any
		_ = json.Unmarshal(raw, &out)
		out["commandDiff"] = map[string]any{"reference": haveRef, "added": nonNil(added), "removed": nonNil(removed)}
		r.emitValue(out)
	} else {
		fmt.Fprintf(r.stdout, "Final Factory %s — state %s, tier %s, heartbeat %d\n", h.GameVersion, h.State, h.Tier, h.Heartbeat)
		l := h.Limits
		fmt.Fprintf(r.stdout, "limits: %d snapshots/s, %d screenshots/s, %d running + %d queued chains, %d segments per chain, long poll %d ms\n",
			l.SnapshotsPerSecond, l.ScreenshotsPerSecond, l.RunningChains, l.QueuedChains, l.ChainSegments, l.LongPollMs)
		if h.Session != nil {
			fmt.Fprintf(r.stdout, "attached: %s (%s)\n", h.Session.ClientLabel, h.Session.ID)
		} else {
			fmt.Fprintln(r.stdout, "attached: nobody")
		}
		switch {
		case !haveRef:
			fmt.Fprintf(r.stdout, "commands: %d live; no embedded reference to compare against\n", len(help.CommandNames()))
		case !mismatch:
			fmt.Fprintf(r.stdout, "commands: %d live, matching the kit's reference (ff-agent %s)\n", len(help.CommandNames()), r.env.Version)
		default:
			fmt.Fprintf(r.stdout, "commands: %d live, differing from the kit's reference (ff-agent %s)\n", len(help.CommandNames()), r.env.Version)
			if len(added) > 0 {
				fmt.Fprintf(r.stdout, "  added (in the game, not in the kit): %s\n", strings.Join(added, ", "))
			}
			if len(removed) > 0 {
				fmt.Fprintf(r.stdout, "  removed (in the kit, not in the game): %s\n", strings.Join(removed, ", "))
			}
		}
	}
	if *strict && mismatch {
		return &exitError{code: ExitMismatch}
	}
	return nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (r *runner) help(ctx context.Context, args []string) error {
	fs := r.flags("help")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	family := ""
	if len(pos) > 0 {
		family = pos[0]
	}
	var h channel.Help
	var raw json.RawMessage
	if err := r.do(ctx, func(cl *channel.Client) (err error) {
		h, raw, err = cl.Help(ctx, family)
		return err
	}); err != nil {
		return err
	}
	if r.g.json {
		r.emit(raw)
		return nil
	}
	fmt.Fprintf(r.stdout, "%d commands for actor %s\n", len(h.CommandNames()), h.Actor)
	for _, f := range h.Families {
		fmt.Fprintf(r.stdout, "\n%s\n", f.Name)
		for _, c := range f.Commands {
			line := "  ffauto:" + c.Name
			if c.Args != "" {
				line += "|" + c.Args
			}
			fmt.Fprintln(r.stdout, line)
			if c.Summary != "" {
				fmt.Fprintln(r.stdout, "      "+c.Summary)
			}
		}
	}
	return nil
}

func (r *runner) command(ctx context.Context, args []string, isChain bool) error {
	name := "cmd"
	if isChain {
		name = "chain"
	}
	fs := r.flags(name)
	wait := fs.Bool("wait", false, "wait for the chain to finish")
	timeout := fs.Duration("timeout", 120*time.Second, "how long --wait waits")
	commands, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(commands) == 0 || (!isChain && len(commands) != 1) {
		if isChain {
			return usageError(`usage: ff-agent chain "<c1>" "<c2>" … [--wait]`)
		}
		return usageError(`usage: ff-agent cmd "<ffauto:…>" [--wait] [--timeout 120s] (quote the command; use chain for several)`)
	}

	var res channel.CommandResult
	var raw json.RawMessage
	if err := r.do(ctx, func(cl *channel.Client) (err error) {
		if err = r.attachImplicitly(ctx, cl); err != nil {
			return err
		}
		res, raw, err = cl.Submit(ctx, commands)
		return err
	}); err != nil {
		return err
	}
	if res.Status == "rejected" {
		if r.g.json {
			r.emit(raw)
		}
		return &exitError{code: ExitRejected, msg: fmt.Sprintf("rejected (%s): %s", res.Reason, firstNonEmpty(res.Detail, res.Result))}
	}
	if res.Status == "completed" || !*wait {
		if r.g.json {
			r.emit(raw)
		} else if res.Status == "completed" {
			r.say("%s", firstNonEmpty(res.Result, "completed"))
		} else {
			r.say("accepted: chain %s is running (ff-agent wait %s)", res.ChainID, res.ChainID)
		}
		return nil
	}
	return r.followChain(ctx, res.ChainID, *timeout)
}

// CLILabel is the session label of an implicit CLI attach. The MCP server replaces such a session
// without --takeover, because no process holds it once the command that opened it has exited.
func CLILabel(version string) string { return "ff-agent/" + version + " cli" }

// attachImplicitly attaches as the CLI when nobody is attached (the contract's "implicit on first
// command"), so the game shows it is agent-driven. An attached agent is left alone: the game does
// not require a session to run a command, and this tool must not unseat someone else's.
func (r *runner) attachImplicitly(ctx context.Context, cl *channel.Client) error {
	h, _, err := cl.Hello(ctx)
	if err != nil || h.Session != nil {
		return err
	}
	_, err = cl.OpenSession(ctx, CLILabel(r.env.Version), false)
	var apiErr *channel.APIError
	if errors.As(err, &apiErr) && apiErr.Reason == "session_exists" {
		return nil
	}
	return err
}

func (r *runner) followChain(ctx context.Context, id string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		remaining := time.Until(deadline)
		var st channel.ChainStatus
		var raw json.RawMessage
		if err := r.do(ctx, func(cl *channel.Client) (err error) {
			st, raw, err = cl.Chain(ctx, id, int(max(0, remaining).Milliseconds()))
			return err
		}); err != nil {
			return err
		}
		if st.Terminal() {
			if r.g.json {
				r.emit(raw)
			} else {
				r.say("%s: %s", st.Status, firstNonEmpty(st.Result, st.Detail))
			}
			if st.Status != "completed" {
				reason := firstNonEmpty(st.Reason, st.Status)
				return &exitError{code: ExitRejected, msg: fmt.Sprintf("chain %s %s (%s): %s", id, st.Status, reason, st.Detail)}
			}
			return nil
		}
		if remaining <= 0 {
			if r.g.json {
				r.emit(raw)
			}
			return &exitError{code: ExitTimeout, msg: fmt.Sprintf("chain %s still running after %s (ff-agent wait %s)", id, timeout, id)}
		}
	}
}

func (r *runner) wait(ctx context.Context, args []string) error {
	fs := r.flags("wait")
	timeout := fs.Duration("timeout", 120*time.Second, "how long to wait")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageError("usage: ff-agent wait <id> [--timeout 120s]")
	}
	id := pos[0]
	if _, err := strconv.Atoi(id); err != nil {
		// Not a waituntil id: a chain id ("c12") is the other thing worth waiting on.
		return r.followChain(ctx, id, *timeout)
	}
	deadline := time.Now().Add(*timeout)
	for {
		remaining := time.Until(deadline)
		var st channel.WaitStatus
		var raw json.RawMessage
		if err := r.do(ctx, func(cl *channel.Client) (err error) {
			st, raw, err = cl.Wait(ctx, id, int(max(0, remaining).Milliseconds()))
			return err
		}); err != nil {
			return err
		}
		if st.Terminal() || remaining <= 0 {
			if r.g.json {
				r.emit(raw)
			} else {
				r.say("wait %s (%s): %s %s", id, st.Predicate, st.Status, st.Detail)
			}
			switch {
			case !st.Terminal():
				return &exitError{code: ExitTimeout, msg: fmt.Sprintf("wait %s still pending after %s", id, *timeout)}
			case st.Status == "timeout":
				return &exitError{code: ExitTimeout, msg: fmt.Sprintf("wait %s timed out in game time: %s", id, st.Detail)}
			case st.Status != "satisfied":
				return &exitError{code: ExitRejected, msg: fmt.Sprintf("wait %s %s: %s", id, st.Status, st.Detail)}
			}
			return nil
		}
	}
}

func (r *runner) snapshot(ctx context.Context, args []string) error {
	fs := r.flags("snapshot")
	x := fs.String("x", "", "centre tile x (nearby, enemies)")
	z := fs.String("z", "", "centre tile z (nearby, enemies)")
	radius := fs.String("r", "", "radius in tiles (nearby, enemies)")
	items := fs.String("items", "", "inventory: smart-craft counts for these items (a,b,c)")
	craftable := fs.Bool("craftable", false, "inventory: smart-craft counts for everything craftable")
	since := fs.Int("since", 0, "alerts: only events from this sequence number")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usageError("usage: ff-agent snapshot <player|inventory|nearby|enemies|tasks|research|objectives|alerts|session> [flags]")
	}
	q := url.Values{}
	for k, v := range map[string]string{"x": *x, "z": *z, "r": *radius, "items": *items} {
		if v != "" {
			q.Set(k, v)
		}
	}
	if *craftable {
		q.Set("craftable", "all")
	}
	if *since > 0 {
		q.Set("sinceSeq", strconv.Itoa(*since))
	}
	var raw json.RawMessage
	if err := r.do(ctx, func(cl *channel.Client) (err error) {
		raw, err = cl.Snapshot(ctx, pos[0], q)
		return err
	}); err != nil {
		return err
	}
	r.emit(raw)
	return nil
}

func (r *runner) screenshot(ctx context.Context, args []string) error {
	fs := r.flags("screenshot")
	out := fs.String("o", "screenshot.png", "where to write the PNG")
	maxEdge := fs.Int("max-edge", 0, "longest side in pixels, 320..1920 (default: the game's 1280)")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	var shot channel.Screenshot
	if err := r.do(ctx, func(cl *channel.Client) (err error) {
		shot, err = cl.Screenshot(ctx, *maxEdge)
		return err
	}); err != nil {
		return err
	}
	if err := os.WriteFile(*out, shot.PNG, 0o644); err != nil {
		return err
	}
	if r.g.json {
		r.emitValue(map[string]any{"file": *out, "bytes": len(shot.PNG), "width": shot.Width, "height": shot.Height,
			"heartbeat": shot.Heartbeat, "epoch": shot.Epoch, "capturedUtc": shot.CapturedUTC})
		return nil
	}
	r.say("%s: %dx%d, %d bytes, heartbeat %d", *out, shot.Width, shot.Height, len(shot.PNG), shot.Heartbeat)
	return nil
}

func (r *runner) journal(ctx context.Context, args []string) error {
	fs := r.flags("journal")
	since := fs.Int("since", 0, "only events from this sequence number")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	var raw json.RawMessage
	if err := r.do(ctx, func(cl *channel.Client) (err error) {
		raw, err = cl.Journal(ctx, *since)
		return err
	}); err != nil {
		return err
	}
	r.emit(raw)
	return nil
}

func (r *runner) attach(ctx context.Context, args []string) error {
	fs := r.flags("attach")
	label := fs.String("label", "", "the name the game shows for this agent")
	takeover := fs.Bool("takeover", false, "replace an attached agent (your own machine only)")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	if strings.TrimSpace(*label) == "" {
		return usageError("usage: ff-agent attach --label <name> [--takeover]")
	}
	var id string
	if err := r.do(ctx, func(cl *channel.Client) (err error) {
		id, err = cl.OpenSession(ctx, *label, *takeover)
		return err
	}); err != nil {
		var apiErr *channel.APIError
		if errors.As(err, &apiErr) && apiErr.Reason == "session_exists" {
			return &channel.SessionConflictError{Detail: apiErr.Detail}
		}
		return err
	}
	if r.g.json {
		r.emitValue(map[string]any{"id": id, "clientLabel": *label})
		return nil
	}
	r.say("attached as %q (session %s)", *label, id)
	return nil
}

func (r *runner) detach(ctx context.Context, args []string) error {
	fs := r.flags("detach")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	if err := r.do(ctx, func(cl *channel.Client) error { return cl.CloseSession(ctx) }); err != nil {
		return err
	}
	r.say("detached")
	return nil
}

func (r *runner) mcp(ctx context.Context, args []string) error {
	fs := r.flags("mcp")
	takeover := fs.Bool("takeover", false, "replace an agent that is already attached")
	if _, err := parse(fs, args); err != nil {
		return err
	}
	opts := r.env.Discovery
	opts.File, opts.PID = r.g.session, r.g.pid
	conn := &channel.Conn{Discovery: opts, Attach: true, Takeover: *takeover,
		Replaceable: mcpserver.StaleSession(discovery.ProcessAlive)}
	return mcpserver.Run(ctx, mcpserver.Options{Conn: conn, Kit: r.env.Kit, Version: r.env.Version, PollSlice: r.env.PollSlice})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

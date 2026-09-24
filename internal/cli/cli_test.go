package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/cli"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/fakegame"
)

// The generated reference writes each command as `ffauto:<name>`; one row here is a command
// the live game no longer has.
const commandsMD = "# ffauto command reference\n\n| Command | Args |\n| --- | --- |\n" +
	"| `ffauto:help` | `[family]` |\n| `ffauto:movement.goto` | `<x>\\|<z>` |\n| `ffauto:removed.one` | |\n"

type harness struct {
	game           *fakegame.Game
	dir            string
	kit            fstest.MapFS
	stdout, stderr bytes.Buffer
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	g := fakegame.Start()
	t.Cleanup(g.Close)
	h := &harness{game: g, dir: filepath.Join(t.TempDir(), "Never Games", "finalfactory", "AgentControl"),
		kit: fstest.MapFS{"commands.md": {Data: []byte(commandsMD)}}}
	g.WriteSessionFile(h.dir, os.Getpid())
	return h
}

func (h *harness) run(args ...string) int {
	h.stdout.Reset()
	h.stderr.Reset()
	return cli.Run(args, cli.Env{Stdout: &h.stdout, Stderr: &h.stderr, Kit: h.kit, Version: "test",
		Discovery: discovery.Options{Dir: h.dir}})
}

func TestHelloDiffsTheCommandSet(t *testing.T) {
	h := newHarness(t)
	if code := h.run("hello"); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, h.stderr.String())
	}
	out := h.stdout.String()
	for _, want := range []string{
		"Final Factory 0.50.0.24 — state playing, tier player-safe",
		"added (in the game, not in the kit): construction.place, craft.queue, mining.until, waituntil",
		"removed (in the kit, not in the game): removed.one",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("hello output lacks %q:\n%s", want, out)
		}
	}
	if code := h.run("hello", "--strict"); code != cli.ExitMismatch {
		t.Fatalf("--strict with a mismatch: exit %d, want %d", code, cli.ExitMismatch)
	}
}

func TestHelloWithoutEmbeddedReference(t *testing.T) {
	h := newHarness(t)
	h.kit = fstest.MapFS{}
	if code := h.run("hello", "--strict"); code != cli.ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(h.stdout.String(), "no embedded reference") {
		t.Fatalf("got:\n%s", h.stdout.String())
	}
}

func TestNoGameExitsTwo(t *testing.T) {
	var out, errOut bytes.Buffer
	code := cli.Run([]string{"hello"}, cli.Env{Stdout: &out, Stderr: &errOut, Kit: fstest.MapFS{},
		Discovery: discovery.Options{Dir: t.TempDir()}})
	if code != cli.ExitNotEnabled || !strings.Contains(errOut.String(), "Agent control on") {
		t.Fatalf("exit %d, stderr %q", code, errOut.String())
	}
}

func TestCmdWaitFollowsTheChain(t *testing.T) {
	h := newHarness(t)
	h.game.OnCommand = func([]string) fakegame.Outcome { return fakegame.Outcome{Polls: 2, Result: "arrived"} }
	// Flags after the positional command, the way people type it.
	if code := h.run("cmd", "movement.goto|-295|205", "--wait", "--timeout", "5s"); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, h.stderr.String())
	}
	if got := h.game.Submitted[0][0]; got != "ffauto:movement.goto|-295|205" {
		t.Fatalf("submitted %q", got)
	}
	if !strings.Contains(h.stdout.String(), "completed: arrived") {
		t.Fatalf("got %q", h.stdout.String())
	}
	if h.game.Attached() != "ff-agent/test cli" {
		t.Fatalf("implicit attach label = %q", h.game.Attached())
	}
}

func TestChainRejectedExitsFour(t *testing.T) {
	h := newHarness(t)
	h.game.OnCommand = func([]string) fakegame.Outcome {
		return fakegame.Outcome{Reject: "capability_denied", Detail: "capability_denied: segment 1 'ffauto:inventory.add|Gold|9' failed"}
	}
	if code := h.run("chain", "ffauto:help", "inventory.add|Gold|9"); code != cli.ExitRejected {
		t.Fatalf("exit %d", code)
	}
	if got := h.game.Submitted[0]; len(got) != 2 || got[1] != "ffauto:inventory.add|Gold|9" {
		t.Fatalf("submitted %q", got)
	}
	if !strings.Contains(h.stderr.String(), "capability_denied") {
		t.Fatalf("stderr %q", h.stderr.String())
	}
}

func TestCmdWaitTimeoutExitsFive(t *testing.T) {
	h := newHarness(t)
	h.game.OnCommand = func([]string) fakegame.Outcome { return fakegame.Outcome{Polls: -1} }
	if code := h.run("cmd", "--wait", "--timeout", "100ms", "mining.until|Iron Ore|50"); code != cli.ExitTimeout {
		t.Fatalf("exit %d: %s", code, h.stderr.String())
	}
}

func TestRateLimitedExitsFour(t *testing.T) {
	h := newHarness(t)
	h.game.RateLimitCommands = 1
	if code := h.run("cmd", "craft.queue|Gear|1"); code != cli.ExitRejected {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(h.stderr.String(), "retry after 1s") {
		t.Fatalf("stderr %q", h.stderr.String())
	}
}

func TestUnauthorizedExitsThree(t *testing.T) {
	h := newHarness(t)
	h.game.Rotate("a token the session file does not have")
	if code := h.run("hello"); code != cli.ExitUnauthorized {
		t.Fatalf("exit %d: %s", code, h.stderr.String())
	}
}

func TestScreenshotWritesThePNG(t *testing.T) {
	h := newHarness(t)
	out := filepath.Join(t.TempDir(), "shot.png")
	if code := h.run("screenshot", "-o", out, "--max-edge", "640"); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, h.stderr.String())
	}
	data, err := os.ReadFile(out)
	if err != nil || !bytes.Equal(data, fakegame.PNG()) {
		t.Fatalf("PNG not written intact: %v", err)
	}
	if !strings.Contains(h.stdout.String(), "4x3") || !strings.Contains(h.stdout.String(), "heartbeat 1200") {
		t.Fatalf("got %q", h.stdout.String())
	}
}

func TestSnapshotPassesScopeArgs(t *testing.T) {
	h := newHarness(t)
	if code := h.run("snapshot", "nearby", "--x", "-3", "--z", "4", "--r", "10", "--json"); code != cli.ExitOK {
		t.Fatalf("exit %d: %s", code, h.stderr.String())
	}
	if got := h.game.SnapshotReads[0]; got != "nearby?r=10&x=-3&z=4" {
		t.Fatalf("read %q", got)
	}
	if lines := strings.Count(strings.TrimSpace(h.stdout.String()), "\n"); lines != 0 {
		t.Fatalf("--json must print one line, got %d newlines", lines)
	}
}

package channel_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/channel"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/fakegame"
)

func setup(t *testing.T) (*fakegame.Game, string) {
	t.Helper()
	g := fakegame.Start()
	t.Cleanup(g.Close)
	dir := filepath.Join(t.TempDir(), "Never Games", "finalfactory", "AgentControl")
	g.WriteSessionFile(dir, os.Getpid())
	return g, dir
}

func TestConnRediscoversAfterTokenChange(t *testing.T) {
	g, dir := setup(t)
	conn := &channel.Conn{Discovery: discovery.Options{Dir: dir}}
	ctx := context.Background()
	hello := func(cl *channel.Client) error { _, _, err := cl.Hello(ctx); return err }
	if err := conn.Do(ctx, hello); err != nil {
		t.Fatal(err)
	}

	// Agent control switched off and on: new token, new file. The cached client gets a 401 and
	// the Conn must re-read the file rather than fail.
	g.Rotate("fresh-token")
	g.WriteSessionFile(dir, os.Getpid())
	if err := conn.Do(ctx, hello); err != nil {
		t.Fatalf("after rotation: %v", err)
	}
}

func TestConnAttachConflictAndTakeover(t *testing.T) {
	g, dir := setup(t)
	g.AttachOther("someone else")
	ctx := context.Background()
	noop := func(*channel.Client) error { return nil }

	conn := &channel.Conn{Discovery: discovery.Options{Dir: dir}, Attach: true}
	conn.SetLabel("ff-agent/test mcp (x)")
	var conflict *channel.SessionConflictError
	if err := conn.Do(ctx, noop); !errors.As(err, &conflict) {
		t.Fatalf("want SessionConflictError, got %v", err)
	}

	conn = &channel.Conn{Discovery: discovery.Options{Dir: dir}, Attach: true, Takeover: true}
	conn.SetLabel("ff-agent/test mcp (x)")
	if err := conn.Do(ctx, noop); err != nil {
		t.Fatal(err)
	}
	if got := g.Attached(); got != "ff-agent/test mcp (x)" {
		t.Fatalf("attached label = %q", got)
	}
	if err := conn.Detach(ctx); err != nil || g.Attached() != "" {
		t.Fatalf("detach: %v, still attached %q", err, g.Attached())
	}
}

func TestConnReplacesALeftoverCLISession(t *testing.T) {
	g, dir := setup(t)
	g.AttachOther("ff-agent/0.1.0 cli")
	conn := &channel.Conn{Discovery: discovery.Options{Dir: dir}, Attach: true,
		Replaceable: func(label string) bool { return label == "ff-agent/0.1.0 cli" }}
	conn.SetLabel("mcp")
	if err := conn.Do(context.Background(), func(*channel.Client) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if g.Attached() != "mcp" {
		t.Fatalf("attached = %q", g.Attached())
	}
}

func TestConnNoGame(t *testing.T) {
	conn := &channel.Conn{Discovery: discovery.Options{Dir: t.TempDir()}}
	err := conn.Do(context.Background(), func(*channel.Client) error { return nil })
	if !errors.Is(err, discovery.ErrNotEnabled) {
		t.Fatalf("want ErrNotEnabled, got %v", err)
	}
}

func TestConnDeadPortIsNotEnabled(t *testing.T) {
	g, dir := setup(t)
	g.Close() // the pid (ours) is alive but nothing listens: a reused pid
	conn := &channel.Conn{Discovery: discovery.Options{Dir: dir}}
	err := conn.Do(context.Background(), func(cl *channel.Client) error { _, _, err := cl.Hello(context.Background()); return err })
	if !errors.Is(err, discovery.ErrNotEnabled) {
		t.Fatalf("want ErrNotEnabled, got %v", err)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"movement.goto|1|2":         "ffauto:movement.goto|1|2",
		" ffauto:help ":             "ffauto:help",
		"FFAUTO:craft.queue|Gear|1": "FFAUTO:craft.queue|Gear|1",
	} {
		if got := channel.Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
}

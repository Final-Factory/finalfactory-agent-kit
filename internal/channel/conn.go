package channel

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
)

// SessionConflictError means another client is attached and takeover was not requested.
type SessionConflictError struct{ Detail string }

func (e *SessionConflictError) Error() string {
	return "another agent is already attached to the game: " + e.Detail
}

// Conn discovers the game lazily, per call. The game can restart, or agent control can be turned
// off and on, and each time the port and token change; a long-lived MCP server must follow.
type Conn struct {
	Discovery discovery.Options
	// Takeover replaces an attached session instead of failing with SessionConflictError.
	Takeover bool
	// Attach opens a session on first use of each discovered game (the MCP server does; the
	// one-shot CLI decides for itself).
	Attach bool
	// Replaceable says an attached session may be taken over without Takeover: one whose owner
	// cannot still be running, such as a one-shot CLI command's implicit attach.
	Replaceable func(clientLabel string) bool

	mu       sync.Mutex
	label    string
	session  discovery.Session
	client   *Client
	attached bool
}

// SetLabel names this client in the game's HUD; it applies to the next attach.
func (c *Conn) SetLabel(label string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.label = label
}

// Session returns the discovery record in use, if any.
func (c *Conn) Session() (discovery.Session, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.session, c.client != nil
}

// Do runs fn against the current game, re-discovering once when the game refused the connection
// or the token (a restarted game). Other failures are returned as they are: a request that may
// have reached the game is never repeated.
func (c *Conn) Do(ctx context.Context, fn func(*Client) error) error {
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		var client *Client
		if client, err = c.current(ctx); err == nil {
			err = fn(client)
		}
		if !retryable(err) {
			return err
		}
		c.reset()
	}
	return notRunning(err)
}

// Detach ends the session this Conn opened, if any. Best effort: used on shutdown.
func (c *Conn) Detach(ctx context.Context) error {
	c.mu.Lock()
	client, attached := c.client, c.attached
	c.attached = false
	c.mu.Unlock()
	if client == nil || !attached {
		return nil
	}
	return client.CloseSession(ctx)
}

func (c *Conn) current(ctx context.Context) (*Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client == nil {
		opts := c.Discovery
		if opts.Reachable == nil {
			opts.Reachable = portAnswers
		}
		s, err := discovery.Find(opts)
		if err != nil {
			return nil, err
		}
		c.session, c.client, c.attached = s, New(s), false
	}
	if c.Attach && !c.attached {
		_, err := c.client.OpenSession(ctx, c.label, c.Takeover)
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.Status == http.StatusConflict && apiErr.Reason == "session_exists" {
			if !c.replaceable(ctx) {
				return nil, &SessionConflictError{Detail: apiErr.Detail}
			}
			_, err = c.client.OpenSession(ctx, c.label, true)
		}
		switch {
		case err != nil && retryable(err):
			// A game that restarted or a stale file: Do rediscovers.
			return nil, err
		case err != nil:
			return nil, err
		}
		c.attached = true
	}
	return c.client, nil
}

func (c *Conn) replaceable(ctx context.Context) bool {
	if c.Replaceable == nil {
		return false
	}
	h, _, err := c.client.Hello(ctx)
	return err == nil && h.Session != nil && c.Replaceable(h.Session.ClientLabel)
}

func (c *Conn) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.client, c.attached = nil, false
}

// notRunning turns a refused connection to a discovered port into "not enabled": the pid in the
// file is alive but it is not the game's listener (the game exited and the pid was reused).
func notRunning(err error) error {
	var un *UnreachableError
	if errors.As(err, &un) && un.Refused {
		return &discovery.NotEnabledError{Reason: "the session file's port does not answer"}
	}
	return err
}

func retryable(err error) bool {
	var un *UnreachableError
	if errors.As(err, &un) && un.Refused {
		return true
	}
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
}

func portAnswers(s discovery.Session) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(s.Port)), 500*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

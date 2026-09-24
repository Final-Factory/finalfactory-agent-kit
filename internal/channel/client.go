// Package channel is the client for the game's /v1 agent channel
// (specs/068-agent-player-harness/contracts/agent-channel-http.md in the game repo).
package channel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/discovery"
)

// MaxLongPollMs is the channel's cap on a /v1/chain or /v1/wait long poll.
const MaxLongPollMs = 30_000

// Prefix every command carries; the game refuses any segment without it.
const Prefix = "ffauto:"

// Client talks to one game process. Build a new one when the session file changes.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// New returns a client for s. The address is always 127.0.0.1: the game binds loopback by IP,
// and "localhost" can resolve to ::1 first.
func New(s discovery.Session) *Client {
	return &Client{
		base:  "http://127.0.0.1:" + strconv.Itoa(s.Port),
		token: s.Token,
		http: &http.Client{
			// Above the game's own 35 s request backstop, so a full-length long poll is never cut.
			Timeout: 45 * time.Second,
			// Loopback only, never through a proxy from the environment.
			Transport: &http.Transport{Proxy: nil, MaxIdleConns: 4, IdleConnTimeout: 30 * time.Second},
		},
	}
}

// APIError is a non-2xx answer: the contract's { "error": reason, "detail": ... } envelope.
type APIError struct {
	Status     int
	Reason     string
	Detail     string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	msg := fmt.Sprintf("%s (HTTP %d)", e.Reason, e.Status)
	if e.Detail != "" {
		msg += ": " + e.Detail
	}
	return msg
}

// UnreachableError means the request never got an HTTP answer.
type UnreachableError struct {
	Err error
	// Refused is true when the connection was never made, so the game cannot have acted on it
	// and the request is safe to repeat against a re-discovered game.
	Refused bool
}

func (e *UnreachableError) Error() string { return "could not reach the game: " + e.Err.Error() }
func (e *UnreachableError) Unwrap() error { return e.Err }

// Hello is GET /v1/hello.
type Hello struct {
	GameVersion string `json:"gameVersion"`
	CatalogHash string `json:"catalogHash"`
	Tier        string `json:"tier"`
	State       string `json:"state"`
	FrameCount  int64  `json:"frameCount"`
	Heartbeat   int64  `json:"heartbeat"`
	Epoch       int64  `json:"epoch"`
	KitPath     string `json:"kitPath"`
	Limits      Limits `json:"limits"`
	Session     *struct {
		ID          string `json:"id"`
		ClientLabel string `json:"clientLabel"`
	} `json:"session"`
}

// Limits is the hello's limits object, reported so no client has to trip one to learn it.
type Limits struct {
	SnapshotsPerSecond   int    `json:"snapshotsPerSecond"`
	ScreenshotsPerSecond int    `json:"screenshotsPerSecond"`
	RunningChains        int    `json:"runningChains"`
	QueuedChains         int    `json:"queuedChains"`
	ChainSegments        int    `json:"chainSegments"`
	BodyBytes            int    `json:"bodyBytes"`
	LongPollMs           int    `json:"longPollMs"`
	ScreenshotMaxEdge    int    `json:"screenshotMaxEdge"`
	SaveNamePrefix       string `json:"saveNamePrefix"`
}

// Help is GET /v1/help: the catalog filtered by what this actor may run.
type Help struct {
	Actor       string `json:"actor"`
	CatalogHash string `json:"catalogHash"`
	Families    []struct {
		Name     string `json:"name"`
		Commands []struct {
			Name    string `json:"name"`
			Args    string `json:"args"`
			Summary string `json:"summary"`
		} `json:"commands"`
	} `json:"families"`
}

// CommandNames lists every command name in the help, in the game's order.
func (h *Help) CommandNames() []string {
	var names []string
	for _, f := range h.Families {
		for _, c := range f.Commands {
			names = append(names, c.Name)
		}
	}
	return names
}

// CommandResult is POST /v1/command's envelope. Status is accepted (still running — poll
// ChainID), completed, or rejected (Reason says why).
type CommandResult struct {
	Status    string `json:"status"`
	Result    string `json:"result"`
	Reason    string `json:"reason"`
	Detail    string `json:"detail"`
	ChainID   string `json:"chainId"`
	Heartbeat int64  `json:"heartbeat"`
	Epoch     int64  `json:"epoch"`
}

// ChainStatus is GET /v1/chain/{id}. Status is running, completed, failed or cancelled.
type ChainStatus struct {
	Status           string `json:"status"`
	Detail           string `json:"detail"`
	Result           string `json:"result"`
	Reason           string `json:"reason"`
	StartedHeartbeat int64  `json:"startedHeartbeat"`
	EndedHeartbeat   *int64 `json:"endedHeartbeat"`
	QueuePosition    int    `json:"queuePosition"`
}

// Terminal reports whether the chain has stopped for good.
func (c *ChainStatus) Terminal() bool { return c.Status != "running" && c.Status != "" }

// WaitStatus is GET /v1/wait/{id} (the runner's wait.status payload).
type WaitStatus struct {
	ID                 int     `json:"id"`
	Predicate          string  `json:"predicate"`
	Status             string  `json:"status"`
	TimeoutSeconds     float64 `json:"timeoutSeconds"`
	ElapsedGameSeconds float64 `json:"elapsedGameSeconds"`
	Detail             string  `json:"detail"`
}

// Terminal reports whether the wait has stopped for good.
func (w *WaitStatus) Terminal() bool { return w.Status != "pending" && w.Status != "" }

// Screenshot is GET /v1/screenshot: PNG bytes plus the X-FF-* headers.
type Screenshot struct {
	PNG         []byte
	Heartbeat   int64
	Epoch       int64
	Width       int
	Height      int
	CapturedUTC string
}

// Normalize trims a command and adds the ffauto: prefix when it is missing.
func Normalize(command string) string {
	c := strings.TrimSpace(command)
	if len(c) >= len(Prefix) && strings.EqualFold(c[:len(Prefix)], Prefix) {
		return c
	}
	return Prefix + c
}

// Hello returns the parsed hello and its raw JSON.
func (c *Client) Hello(ctx context.Context) (Hello, json.RawMessage, error) {
	var h Hello
	raw, err := c.getJSON(ctx, "/v1/hello", nil, &h)
	return h, raw, err
}

// OpenSession attaches this client. A 409 session_exists comes back as an *APIError.
func (c *Client) OpenSession(ctx context.Context, label string, takeover bool) (string, error) {
	body := map[string]any{"clientLabel": label}
	if takeover {
		body["takeover"] = true
	}
	var out struct {
		ID string `json:"id"`
	}
	_, err := c.doJSON(ctx, http.MethodPost, "/v1/session", nil, body, &out)
	return out.ID, err
}

// CloseSession detaches whoever is attached; the game cancels the active chain and waits.
func (c *Client) CloseSession(ctx context.Context) error {
	_, err := c.doJSON(ctx, http.MethodDelete, "/v1/session", nil, nil, nil)
	return err
}

// Help returns the live command enumeration, optionally one family.
func (c *Client) Help(ctx context.Context, family string) (Help, json.RawMessage, error) {
	q := url.Values{}
	if family != "" {
		q.Set("family", family)
	}
	var h Help
	raw, err := c.getJSON(ctx, "/v1/help", q, &h)
	return h, raw, err
}

// Submit runs commands as one ordered chain; each gets the ffauto: prefix if it lacks one.
func (c *Client) Submit(ctx context.Context, commands []string) (CommandResult, json.RawMessage, error) {
	body := map[string]any{"actor": "local-player"}
	if len(commands) == 1 {
		body["command"] = Normalize(commands[0])
	} else {
		chain := make([]string, len(commands))
		for i, cmd := range commands {
			chain[i] = Normalize(cmd)
		}
		body["chain"] = chain
	}
	var r CommandResult
	raw, err := c.doJSON(ctx, http.MethodPost, "/v1/command", nil, body, &r)
	return r, raw, err
}

// Chain reads a chain's status, long-polling up to timeoutMs (clamped to the channel's cap).
func (c *Client) Chain(ctx context.Context, id string, timeoutMs int) (ChainStatus, json.RawMessage, error) {
	var s ChainStatus
	raw, err := c.getJSON(ctx, "/v1/chain/"+url.PathEscape(id), pollQuery(timeoutMs), &s)
	return s, raw, err
}

// Wait reads a waituntil entry, long-polling up to timeoutMs.
func (c *Client) Wait(ctx context.Context, id string, timeoutMs int) (WaitStatus, json.RawMessage, error) {
	var s WaitStatus
	raw, err := c.getJSON(ctx, "/v1/wait/"+url.PathEscape(id), pollQuery(timeoutMs), &s)
	return s, raw, err
}

// Snapshot returns the raw snapshot envelope for scope.
func (c *Client) Snapshot(ctx context.Context, scope string, q url.Values) (json.RawMessage, error) {
	return c.getJSON(ctx, "/v1/snapshot/"+url.PathEscape(scope), q, nil)
}

// Journal returns the active playtest journal tail from sinceSeq.
func (c *Client) Journal(ctx context.Context, sinceSeq int) (json.RawMessage, error) {
	q := url.Values{}
	if sinceSeq > 0 {
		q.Set("sinceSeq", strconv.Itoa(sinceSeq))
	}
	return c.getJSON(ctx, "/v1/journal", q, nil)
}

// Screenshot captures the game window; maxEdge 0 means the game's default.
func (c *Client) Screenshot(ctx context.Context, maxEdge int) (Screenshot, error) {
	q := url.Values{}
	if maxEdge > 0 {
		q.Set("maxEdge", strconv.Itoa(maxEdge))
	}
	resp, body, err := c.do(ctx, http.MethodGet, "/v1/screenshot", q, nil)
	if err != nil {
		return Screenshot{}, err
	}
	h := resp.Header
	return Screenshot{
		PNG:         body,
		Heartbeat:   atoi64(h.Get("X-FF-Heartbeat")),
		Epoch:       atoi64(h.Get("X-FF-Epoch")),
		Width:       int(atoi64(h.Get("X-FF-Width"))),
		Height:      int(atoi64(h.Get("X-FF-Height"))),
		CapturedUTC: h.Get("X-FF-Captured-Utc"),
	}, nil
}

func pollQuery(timeoutMs int) url.Values {
	q := url.Values{}
	if timeoutMs > 0 {
		q.Set("timeoutMs", strconv.Itoa(min(timeoutMs, MaxLongPollMs)))
	}
	return q
}

func (c *Client) getJSON(ctx context.Context, path string, q url.Values, out any) (json.RawMessage, error) {
	return c.doJSON(ctx, http.MethodGet, path, q, nil, out)
}

func (c *Client) doJSON(ctx context.Context, method, path string, q url.Values, in, out any) (json.RawMessage, error) {
	var body io.Reader
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	_, data, err := c.do(ctx, method, path, q, body)
	if err != nil {
		return nil, err
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return data, fmt.Errorf("the game answered %s with unexpected JSON: %w", path, err)
		}
	}
	return data, nil
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, body io.Reader) (*http.Response, []byte, error) {
	u := c.base + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, &UnreachableError{Err: err, Refused: isRefused(err)}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, &UnreachableError{Err: err}
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, data, nil
	}
	apiErr := &APIError{Status: resp.StatusCode}
	var envelope struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(data, &envelope) == nil && envelope.Error != "" {
		apiErr.Reason, apiErr.Detail = envelope.Error, envelope.Detail
	} else {
		apiErr.Reason, apiErr.Detail = http.StatusText(resp.StatusCode), strings.TrimSpace(string(data))
	}
	if s := resp.Header.Get("Retry-After"); s != "" {
		if secs, err := strconv.Atoi(s); err == nil {
			apiErr.RetryAfter = time.Duration(secs) * time.Second
		}
	}
	return resp, data, apiErr
}

// isRefused is true only for failures before the connection existed (the dial). Anything later
// may have reached the game, and a command must never be sent twice on a guess.
func isRefused(err error) bool {
	var opErr *net.OpError
	return errors.As(err, &opErr) && opErr.Op == "dial"
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

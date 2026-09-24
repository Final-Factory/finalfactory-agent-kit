// Package discovery finds the running game's agent channel from the session-{pid}.json file the
// game writes to {persistentDataPath}/AgentControl while "Agent control" is on.
package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Session is one session-{pid}.json. The game deletes the file when the channel stops or the
// game quits, so a file whose pid is dead means the game crashed: agent control is not on.
type Session struct {
	Port        int    `json:"port"`
	Token       string `json:"token"`
	PID         int    `json:"pid"`
	GameVersion string `json:"gameVersion"`
	CatalogHash string `json:"catalogHash"`
	Tier        string `json:"tier"`
	KitPath     string `json:"kitPath"`
	StartedUTC  string `json:"startedUtc"`

	// File is where this session was read from.
	File string `json:"-"`
}

// Options selects a session. The zero value searches DefaultDir for the one live game.
type Options struct {
	// Dir overrides DefaultDir.
	Dir string
	// File reads exactly this session file (the --session flag).
	File string
	// PID picks one game when several are running (the --pid flag).
	PID int
	// Alive overrides the process-liveness check; tests only.
	Alive func(pid int) bool
	// Reachable, when set, drops candidates whose port does not answer. A pid that died and was
	// reused by an unrelated process otherwise looks like a running game.
	Reachable func(Session) bool
}

// ErrNotEnabled means no running game has agent control on.
var ErrNotEnabled = errors.New("agent control is not enabled")

// NotEnabledError says why no session was selected; errors.Is(err, ErrNotEnabled) holds.
type NotEnabledError struct{ Reason string }

func (e *NotEnabledError) Error() string { return ErrNotEnabled.Error() + ": " + e.Reason }
func (e *NotEnabledError) Unwrap() error { return ErrNotEnabled }

// AmbiguousError means more than one game is running with agent control on.
type AmbiguousError struct{ PIDs []int }

func (e *AmbiguousError) Error() string {
	pids := make([]string, len(e.PIDs))
	for i, p := range e.PIDs {
		pids[i] = fmt.Sprint(p)
	}
	return fmt.Sprintf("%d Final Factory games have agent control on (pids %s); pick one with --pid",
		len(e.PIDs), strings.Join(pids, ", "))
}

// DefaultDir is {persistentDataPath}/AgentControl for the shipped game (company "Never Games",
// product "finalfactory"), following Unity's per-platform persistentDataPath rules.
func DefaultDir() string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "windows":
		// Unity: %USERPROFILE%\AppData\LocalLow\<company>\<product>.
		if profile := os.Getenv("USERPROFILE"); profile != "" {
			home = profile
		}
		return filepath.Join(home, "AppData", "LocalLow", "Never Games", "finalfactory", "AgentControl")
	case "darwin":
		// Unity: ~/Library/Application Support/<company>/<product>.
		return filepath.Join(home, "Library", "Application Support", "Never Games", "finalfactory", "AgentControl")
	default:
		// Unity: $XDG_CONFIG_HOME/unity3d/<company>/<product>.
		config := os.Getenv("XDG_CONFIG_HOME")
		if config == "" {
			config = filepath.Join(home, ".config")
		}
		return filepath.Join(config, "unity3d", "Never Games", "finalfactory", "AgentControl")
	}
}

// Find returns the session of the one running game with agent control on.
func Find(opts Options) (Session, error) {
	alive := opts.Alive
	if alive == nil {
		alive = ProcessAlive
	}

	if opts.File != "" {
		s, err := read(opts.File)
		if err != nil {
			return Session{}, &NotEnabledError{Reason: err.Error()}
		}
		if !alive(s.PID) {
			return Session{}, &NotEnabledError{Reason: fmt.Sprintf("the game that wrote %s (pid %d) is not running", opts.File, s.PID)}
		}
		return s, nil
	}

	dir := opts.Dir
	if dir == "" {
		dir = DefaultDir()
	}
	files, err := filepath.Glob(filepath.Join(globEscape(dir), "session-*.json"))
	if err != nil || len(files) == 0 {
		return Session{}, &NotEnabledError{Reason: "no session file in " + dir}
	}

	var live []Session
	for _, f := range files {
		s, err := read(f)
		if err != nil || !alive(s.PID) {
			continue
		}
		if opts.PID > 0 && s.PID != opts.PID {
			continue
		}
		live = append(live, s)
	}
	if len(live) > 1 && opts.Reachable != nil {
		reachable := live[:0]
		for _, s := range live {
			if opts.Reachable(s) {
				reachable = append(reachable, s)
			}
		}
		live = reachable
	}

	switch len(live) {
	case 0:
		if opts.PID > 0 {
			return Session{}, &NotEnabledError{Reason: fmt.Sprintf("no running game with pid %d has agent control on", opts.PID)}
		}
		return Session{}, &NotEnabledError{Reason: "every session file in " + dir + " belongs to a game that is no longer running"}
	case 1:
		return live[0], nil
	default:
		// Newest first, so the message lists the most likely intended game first.
		sort.Slice(live, func(i, j int) bool { return live[i].StartedUTC > live[j].StartedUTC })
		pids := make([]int, len(live))
		for i, s := range live {
			pids[i] = s.PID
		}
		return Session{}, &AmbiguousError{PIDs: pids}
	}
}

func read(path string) (Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return Session{}, fmt.Errorf("%s is not a session file: %w", path, err)
	}
	if s.Port <= 0 || s.Token == "" {
		return Session{}, fmt.Errorf("%s has no port or token", path)
	}
	s.File = path
	return s, nil
}

// globEscape keeps a directory name containing glob metacharacters from being read as a pattern.
func globEscape(dir string) string {
	if runtime.GOOS == "windows" {
		// '\' is the separator there, not an escape, and '[' cannot be escaped portably.
		return dir
	}
	r := strings.NewReplacer(`\`, `\\`, `*`, `\*`, `?`, `\?`, `[`, `\[`)
	return r.Replace(dir)
}

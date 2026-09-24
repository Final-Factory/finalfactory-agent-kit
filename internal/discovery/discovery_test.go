package discovery

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func writeSession(t *testing.T, dir string, pid, port int, started string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fmt.Sprintf("session-%d.json", pid))
	data, _ := json.Marshal(map[string]any{"port": port, "token": "tok", "pid": pid, "gameVersion": "0.50.0.24", "startedUtc": started})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func aliveSet(pids ...int) func(int) bool {
	return func(pid int) bool {
		for _, p := range pids {
			if p == pid {
				return true
			}
		}
		return false
	}
}

// The real persistentDataPath contains "Never Games"; a space must not break the glob.
func spacedDir(t *testing.T) string {
	return filepath.Join(t.TempDir(), "Never Games", "finalfactory", "AgentControl")
}

func TestFindPicksTheLiveSession(t *testing.T) {
	dir := spacedDir(t)
	writeSession(t, dir, 111, 5001, "2026-09-23T01:00:00Z") // crashed game: file left behind
	writeSession(t, dir, 222, 5002, "2026-09-23T02:00:00Z")

	s, err := Find(Options{Dir: dir, Alive: aliveSet(222)})
	if err != nil {
		t.Fatal(err)
	}
	if s.PID != 222 || s.Port != 5002 || s.Token != "tok" || s.File == "" {
		t.Fatalf("got %+v", s)
	}
}

func TestFindDeadPidMeansNotEnabled(t *testing.T) {
	dir := spacedDir(t)
	writeSession(t, dir, 111, 5001, "")
	_, err := Find(Options{Dir: dir, Alive: aliveSet()})
	if !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("want ErrNotEnabled, got %v", err)
	}
}

func TestFindMissingDirMeansNotEnabled(t *testing.T) {
	_, err := Find(Options{Dir: filepath.Join(t.TempDir(), "nope")})
	if !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("want ErrNotEnabled, got %v", err)
	}
}

func TestFindSeveralLiveGamesNeedsPid(t *testing.T) {
	dir := spacedDir(t)
	writeSession(t, dir, 111, 5001, "2026-09-23T01:00:00Z")
	writeSession(t, dir, 222, 5002, "2026-09-23T02:00:00Z")
	alive := aliveSet(111, 222)

	_, err := Find(Options{Dir: dir, Alive: alive})
	var ambiguous *AmbiguousError
	if !errors.As(err, &ambiguous) || len(ambiguous.PIDs) != 2 || ambiguous.PIDs[0] != 222 {
		t.Fatalf("want ambiguity listing the newest first, got %v", err)
	}

	s, err := Find(Options{Dir: dir, Alive: alive, PID: 111})
	if err != nil || s.PID != 111 {
		t.Fatalf("--pid 111: got %+v, %v", s, err)
	}

	_, err = Find(Options{Dir: dir, Alive: alive, PID: 333})
	if !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("--pid of no game: want ErrNotEnabled, got %v", err)
	}
}

func TestFindDropsUnreachableCandidates(t *testing.T) {
	dir := spacedDir(t)
	writeSession(t, dir, 111, 5001, "")
	writeSession(t, dir, 222, 5002, "")
	s, err := Find(Options{Dir: dir, Alive: aliveSet(111, 222), Reachable: func(s Session) bool { return s.Port == 5002 }})
	if err != nil || s.PID != 222 {
		t.Fatalf("a reused pid whose port is dead must be dropped: got %+v, %v", s, err)
	}
}

func TestFindSessionFlagReadsThatFile(t *testing.T) {
	dir := spacedDir(t)
	path := writeSession(t, dir, 111, 5001, "")
	writeSession(t, dir, 222, 5002, "")

	s, err := Find(Options{File: path, Alive: aliveSet(111, 222)})
	if err != nil || s.PID != 111 {
		t.Fatalf("got %+v, %v", s, err)
	}
	if _, err := Find(Options{File: path, Alive: aliveSet(222)}); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("--session naming a dead game: want ErrNotEnabled, got %v", err)
	}
}

func TestFindSkipsMalformedFiles(t *testing.T) {
	dir := spacedDir(t)
	writeSession(t, dir, 222, 5002, "")
	if err := os.WriteFile(filepath.Join(dir, "session-999.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Find(Options{Dir: dir, Alive: aliveSet(222, 999)})
	if err != nil || s.PID != 222 {
		t.Fatalf("got %+v, %v", s, err)
	}
}

func TestProcessAliveOnRealProcesses(t *testing.T) {
	if !ProcessAlive(os.Getpid()) {
		t.Fatal("this test process must count as alive")
	}
	// A process that has exited and been reaped: its pid is dead.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if ProcessAlive(cmd.Process.Pid) {
		t.Skip("pid was reused between exit and the check; nothing to assert")
	}
	if ProcessAlive(0) || ProcessAlive(-1) {
		t.Fatal("non-positive pids are never alive")
	}
}

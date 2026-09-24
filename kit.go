// Package agentkit holds the AgentKit documents that ship next to the ff-agent binary, embedded
// so the binary can serve them as MCP resources and prompts without reading the disk.
//
// The embed directive lives at the module root because //go:embed cannot reach a parent
// directory, and kit/ is the folder the release zips up verbatim.
package agentkit

import (
	"embed"
	"io/fs"
	"strings"
)

// The all: prefix is load-bearing: without it .mcp.json and .claude/skills are skipped.
//
//go:embed all:kit
var embedded embed.FS

// Kit is the kit/ folder, rooted so paths read "HowToPlay.md", ".claude/skills/x/SKILL.md".
func Kit() fs.FS {
	sub, err := fs.Sub(embedded, "kit")
	if err != nil {
		// fs.Sub only fails on an invalid path literal, which "kit" is not.
		panic(err)
	}
	return sub
}

// Version is the first line of kit/VERSION: the kit's own version, which is also the
// binary's, so the two can never disagree inside one release.
func Version() string {
	data, err := fs.ReadFile(Kit(), "VERSION")
	if err != nil {
		return "dev"
	}
	line, _, _ := strings.Cut(strings.TrimSpace(string(data)), "\n")
	if line = strings.TrimSpace(line); line == "" {
		return "dev"
	}
	return line
}

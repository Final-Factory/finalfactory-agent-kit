// Command ff-agent is the Final Factory agent kit's command-line tool and MCP server.
package main

import (
	"os"

	agentkit "github.com/Final-Factory/finalfactory-agent-kit"
	"github.com/Final-Factory/finalfactory-agent-kit/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
		Kit:     agentkit.Kit(),
		Version: agentkit.Version(),
	}))
}

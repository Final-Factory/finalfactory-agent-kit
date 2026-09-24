package mcpserver

import (
	"context"
	"io/fs"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Final-Factory/finalfactory-agent-kit/internal/kitdocs"
)

// URIs of the embedded documents.
const (
	GuideURI       = "finalfactory://guide/how-to-play"
	CommandsURI    = "finalfactory://reference/commands"
	SkillURIPrefix = "finalfactory://skills/"
)

// registerDocs serves the kit documents that exist in this build: a kit built before the guide
// or the command reference was written simply has no such resource.
func registerDocs(srv *mcp.Server, kit fs.FS) {
	if kit == nil {
		return
	}
	addFile(srv, kit, kitdocs.GuidePath, &mcp.Resource{URI: GuideURI, Name: "how-to-play", Title: "How to play Final Factory",
		Description: "The player guide for agents: the core loop, controls, and what to build first. Read before planning.", MIMEType: "text/markdown"})
	addFile(srv, kit, kitdocs.CommandsPath, &mcp.Resource{URI: CommandsURI, Name: "commands", Title: "ffauto command reference",
		Description: "Every command the shipped game accepts, with arguments (the run_commands vocabulary).", MIMEType: "text/markdown"})

	for _, skill := range kitdocs.Skills(kit) {
		skill := skill
		addFile(srv, kit, skill.Path, &mcp.Resource{URI: SkillURIPrefix + skill.Name, Name: "skill-" + skill.Name,
			Title: "Skill: " + skill.Name, Description: skill.Description, MIMEType: "text/markdown"})
		srv.AddPrompt(&mcp.Prompt{
			Name:        skill.Name,
			Description: skill.Description,
			Arguments:   []*mcp.PromptArgument{{Name: "goal", Description: "What you want from this run, in your own words (optional)."}},
		}, func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			text := strings.TrimSpace(skill.Body)
			if goal := strings.TrimSpace(req.Params.Arguments["goal"]); goal != "" {
				text += "\n\nGoal for this run: " + goal
			}
			return &mcp.GetPromptResult{
				Description: skill.Description,
				Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: text}}},
			}, nil
		})
	}
}

func addFile(srv *mcp.Server, kit fs.FS, path string, r *mcp.Resource) {
	data, err := fs.ReadFile(kit, path)
	if err != nil {
		return
	}
	text := string(data)
	srv.AddResource(r, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: r.URI, MIMEType: r.MIMEType, Text: text}}}, nil
	})
}

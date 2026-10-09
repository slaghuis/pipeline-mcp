package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/pipeline-mcp/internal/registry"
)

func RegisterList(s *server.MCPServer, reg *registry.Registry) {
	tool := mcp.NewTool("pipeline_list_services",
		mcp.WithDescription(
			"List all Go services known to the pipeline system. "+
				"Returns name, repository path, pipeline binary location, "+
				"and default deployment environment for each."),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		list := reg.List()
		b, _ := json.MarshalIndent(map[string]any{
			"count": len(list), "services": list,
		}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	})
}
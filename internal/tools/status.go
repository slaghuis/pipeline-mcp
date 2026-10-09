package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/pipeline-mcp/internal/runner"
)

func RegisterStatus(s *server.MCPServer, run *runner.Runner) {
	tool := mcp.NewTool("pipeline_latest_report",
		mcp.WithDescription(
			"Return the most recent pipeline report for a service. "+
				"Useful for 'what was the last build result' questions "+
				"without re-running the pipeline."),
		mcp.WithString("service", mcp.Required(),
			mcp.Description("Service name.")),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := req.RequireString("service")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		data, path, err := run.LatestReport(name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		out, _ := json.MarshalIndent(map[string]any{
			"service": name, "path": path, "report": json.RawMessage(data),
		}, "", "  ")
		return mcp.NewToolResultText(string(out)), nil
	})
}
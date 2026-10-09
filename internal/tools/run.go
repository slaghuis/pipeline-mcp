package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/pipeline-mcp/internal/registry"
	"github.com/slaghuis/pipeline-mcp/internal/runner"
)

func RegisterRun(s *server.MCPServer, reg *registry.Registry, run *runner.Runner) {
	tool := mcp.NewTool("pipeline_run",
		mcp.WithDescription(
			"Execute a pipeline command for a service. Returns the structured "+
				"JSON report produced by the pipeline. Commands:\n"+
				"  lint    – style/static analysis\n"+
				"  test    – unit tests + coverage\n"+
				"  scan    – vulnerability scanning\n"+
				"  build   – build binary and container image\n"+
				"  integ   – integration tests\n"+
				"  full    – lint + test + scan + build\n"+
				"  release – full + deploy (needs env; triggers Telegram approval)"),
		mcp.WithString("service", mcp.Required(),
			mcp.Description("Service name from the registry.")),
		mcp.WithString("command", mcp.Required(),
			mcp.Description("Pipeline command (see above).")),
		mcp.WithString("env",
			mcp.Description("Deployment environment (required for 'release').")),
		mcp.WithString("version",
			mcp.Description("Override version string (default: git describe).")),
		mcp.WithString("changelog",
			mcp.Description("Changelog text to include in approval request.")),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		serviceName, err := req.RequireString("service")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		cmd, err := req.RequireString("command")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		svc, ok := reg.Get(serviceName)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("unknown service %q", serviceName)), nil
		}

		args := []string{cmd}
		if env := req.GetString("env", ""); env != "" {
			args = append(args, "-env", env)
		} else if cmd == "release" && svc.DefaultEnv != "" {
			args = append(args, "-env", svc.DefaultEnv)
		}
		if v := req.GetString("version", ""); v != "" {
			args = append(args, "-version", v)
		}
		if cl := req.GetString("changelog", ""); cl != "" {
			args = append(args, "-changelog", cl)
		}

		res, err := run.Run(ctx, svc, args)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		if res.ExitCode != 0 {
			return mcp.NewToolResultError(string(b)), nil
		}
		return mcp.NewToolResultText(string(b)), nil
	})
}
package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"

	"github.com/slaghuis/pipeline-mcp/internal/config"
	"github.com/slaghuis/pipeline-mcp/internal/metrics"
	"github.com/slaghuis/pipeline-mcp/internal/registry"
	"github.com/slaghuis/pipeline-mcp/internal/runner"
	"github.com/slaghuis/pipeline-mcp/internal/tools"
)

const instructions = `
This MCP server orchestrates CI/CD pipelines for Go services.

WORKFLOW:
1. Call pipeline_list_services to discover available services.
2. Use pipeline_run with command=full to run lint/test/scan/build locally.
3. Use pipeline_run with command=release and env=staging|production to
   deploy; this will trigger a Telegram approval prompt if the environment
   is configured to require one.
4. Use pipeline_latest_report to check the last build without re-running.

The returned JSON has the structure:
  { "service": "...", "command": "...", "exit_code": 0,
    "duration_ms": N, "report": {<pipeline-lib report>} }

Each stage in report.stages has success/error/output/metrics fields.

RULES OF THUMB:
- Always call pipeline_run with command=full before command=release.
- Pass a 'changelog' for release commands so the human sees context on their phone.
- A failed stage returns exit_code != 0; parse report.stages to find which stage failed.
`

func main() {
	cfgPath := flag.String("config", "config.yaml", "config path")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	c, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	reg, err := registry.New(c.RegistryPath)
	if err != nil {
		log.Fatalf("registry: %v", err)
	}
	run := runner.New(c.ReportsDir)

	// SIGHUP → reload service registry
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	go func() {
		for range hup {
			if err := reg.Reload(); err != nil {
				logger.Warn("reload", "err", err)
			} else {
				logger.Info("services reloaded")
			}
		}
	}()

	s := server.NewMCPServer(
		"pipeline",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithInstructions(instructions),
	)
	tools.RegisterList(s, reg)
	tools.RegisterRun(s, reg, run)
	tools.RegisterStatus(s, run)

	sse := server.NewSSEServer(s)

	mux := http.NewServeMux()
	mux.Handle("/sse", sse)
	mux.Handle("/message", sse)
	mux.Handle("/metrics", metrics.Handler())

	logger.Info("pipeline-mcp starting",
		"listen", c.Listen, "services", len(reg.List()))
	if err := http.ListenAndServe(c.Listen, mux); err != nil {
		log.Fatal(err)
	}
}

package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/slaghuis/pipeline-mcp/internal/registry"
	"github.com/slaghuis/pipeline-mcp/internal/metrics"
)

type Result struct {
	Service   string          `json:"service"`
	Command   string          `json:"command"`
	ExitCode  int             `json:"exit_code"`
	Report    json.RawMessage `json:"report,omitempty"`
	Stderr    string          `json:"stderr,omitempty"`
	StartedAt time.Time       `json:"started_at"`
	DurationS float64         `json:"duration_s"`
}

type Runner struct {
	ReportsDir string
}

func New(reportsDir string) *Runner { return &Runner{ReportsDir: reportsDir} }

func (r *Runner) Run(ctx context.Context, s registry.Service, args []string) (*Result, error) {
	bin := s.BinaryAbs()
	if _, err := os.Stat(bin); err != nil {
		return nil, fmt.Errorf("pipeline binary not found at %s (did you build it?)", bin)
	}

	start := time.Now()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = s.Path
	// Inherit env + explicitly forward secrets agents might need
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			return nil, fmt.Errorf("run: %w", err)
		}
	}

	res := &Result{
		Service: s.Name, Command: args[0],
		ExitCode:  exit,
		Stderr:    stderr.String(),
		StartedAt: start,
		DurationS: time.Since(start).Seconds(),
	}

	status := "success"
	if res.ExitCode != 0 {
		status = "failure"
	}
	metrics.Runs.WithLabelValues(svc.Name, command, status).Inc()
	metrics.Duration.WithLabelValues(svc.Name, command).Observe(dur.Seconds())

	// Parse report JSON to emit per-stage metrics
	if len(res.ReportJSON) > 0 {
		var rep struct {
			Stages []struct {
				Name      string  `json:"name"`
				Success   bool    `json:"success"`
				Skipped   bool    `json:"skipped"`
				DurationS float64 `json:"duration_s"`
			} `json:"stages"`
		}
		if err := json.Unmarshal(res.ReportJSON, &rep); err == nil {
			for _, s := range rep.Stages {
				if s.Skipped {
					continue
				}
				metrics.StageDuration.WithLabelValues(svc.Name, s.Name).Observe(s.DurationS)
				if !s.Success {
					metrics.StageFailures.WithLabelValues(svc.Name, s.Name).Inc()
				}
			}
		}
	}


	// stdout should be a JSON report. If not, surface raw.
	trimmed := bytes.TrimSpace(stdout.Bytes())
	if len(trimmed) > 0 && trimmed[0] == '{' {
		res.Report = trimmed
		// archive it
		fn := fmt.Sprintf("%s-%s-%d.json", s.Name, args[0], start.Unix())
		_ = os.WriteFile(filepath.Join(r.ReportsDir, fn), trimmed, 0o644)
	} else {
		res.Stderr = res.Stderr + "\n" + stdout.String()
	}

	return res, nil
}

// LatestReport returns the newest archived report for a service.
func (r *Runner) LatestReport(service string) (json.RawMessage, string, error) {
	entries, err := os.ReadDir(r.ReportsDir)
	if err != nil {
		return nil, "", err
	}
	var newest os.DirEntry
	var newestMod time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		// filename prefix match
		if len(e.Name()) < len(service)+1 || e.Name()[:len(service)+1] != service+"-" {
			continue
		}
		if info.ModTime().After(newestMod) {
			newestMod = info.ModTime()
			newest = e
		}
	}
	if newest == nil {
		return nil, "", fmt.Errorf("no reports for %s", service)
	}
	path := filepath.Join(r.ReportsDir, newest.Name())
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return b, path, nil
}
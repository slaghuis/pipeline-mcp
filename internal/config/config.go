package config

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen       string `yaml:"listen"`
	RegistryPath string `yaml:"registry_path"`
	ServicesFile string `yaml:"services_file"`
	ReportsDir   string `yaml:"reports_dir"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	if c.Listen == "" {
		c.Listen = ":8766"
	}
	// services_file is the documented key; registry_path is kept for older configs.
	if c.ServicesFile != "" {
		c.RegistryPath = c.ServicesFile
	}
	if c.RegistryPath == "" {
		c.RegistryPath = "$HOME/.config/ai-factory/services.yaml"
	}
	if c.ReportsDir == "" {
		c.ReportsDir = "$HOME/.local/share/ai-factory/reports"
	}
	c.RegistryPath = ExpandPath(c.RegistryPath)
	c.ReportsDir = ExpandPath(c.ReportsDir)
	_ = os.MkdirAll(c.ReportsDir, 0o755)
	return &c, nil
}

// ExpandPath expands $VAR/${VAR} references and a leading "~/".
func ExpandPath(p string) string {
	p = os.ExpandEnv(p)
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

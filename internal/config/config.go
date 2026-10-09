package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen       string `yaml:"listen"`
	RegistryPath string `yaml:"registry_path"`
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
	if c.RegistryPath == "" {
		c.RegistryPath = os.ExpandEnv("$HOME/.config/ai-factory/services.yaml")
	}
	if c.ReportsDir == "" {
		c.ReportsDir = os.ExpandEnv("$HOME/.local/share/ai-factory/reports")
	}
	_ = os.MkdirAll(c.ReportsDir, 0o755)
	return &c, nil
}
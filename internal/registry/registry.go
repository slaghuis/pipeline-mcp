package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

type Service struct {
	Name           string `yaml:"name"`
	Path           string `yaml:"path"`
	PipelineBinary string `yaml:"pipeline_binary"`
	DefaultEnv     string `yaml:"default_env"`
}

type file struct {
	Services []Service `yaml:"services"`
}

type Registry struct {
	path string
	mu   sync.RWMutex
	m    map[string]Service
}

func New(path string) (*Registry, error) {
	r := &Registry{path: path}
	if err := r.Reload(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Registry) Reload() error {
	b, err := os.ReadFile(r.path)
	if err != nil {
		return fmt.Errorf("read registry %s: %w", r.path, err)
	}
	var f file
	if err := yaml.Unmarshal(b, &f); err != nil {
		return err
	}
	m := make(map[string]Service, len(f.Services))
	for _, s := range f.Services {
		s.Path = os.ExpandEnv(s.Path)
		if s.PipelineBinary == "" {
			s.PipelineBinary = "./bin/pipeline"
		}
		m[s.Name] = s
	}
	r.mu.Lock()
	r.m = m
	r.mu.Unlock()
	return nil
}

func (r *Registry) Get(name string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.m[name]
	return s, ok
}

func (r *Registry) List() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Service, 0, len(r.m))
	for _, s := range r.m {
		out = append(out, s)
	}
	return out
}

func (s Service) BinaryAbs() string {
	if filepath.IsAbs(s.PipelineBinary) {
		return s.PipelineBinary
	}
	return filepath.Join(s.Path, s.PipelineBinary)
}
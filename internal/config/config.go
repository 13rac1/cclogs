package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/13rac1/ccls/internal/types"
	"gopkg.in/yaml.v3"
)

const (
	defaultProjectsRoot = "~/.claude/projects"
	defaultS3Prefix     = "claude-code/"
)

// Load reads and validates configuration from the specified path.
// Tilde (~) in paths is expanded to the user's home directory.
func Load(path string) (*types.Config, error) {
	expandedPath, err := expandTilde(path)
	if err != nil {
		return nil, fmt.Errorf("expanding config path: %w", err)
	}

	data, err := os.ReadFile(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", expandedPath, err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config YAML: %w", err)
	}

	if err := applyDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("applying defaults: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

// applyDefaults sets default values for optional config fields.
func applyDefaults(cfg *types.Config) error {
	if cfg.Local.ProjectsRoot == "" {
		cfg.Local.ProjectsRoot = defaultProjectsRoot
	}

	expandedRoot, err := expandTilde(cfg.Local.ProjectsRoot)
	if err != nil {
		return fmt.Errorf("expanding projects_root: %w", err)
	}
	cfg.Local.ProjectsRoot = expandedRoot

	if cfg.S3.Prefix == "" {
		cfg.S3.Prefix = defaultS3Prefix
	}

	// Ensure prefix has trailing slash for consistent key building
	if !strings.HasSuffix(cfg.S3.Prefix, "/") {
		cfg.S3.Prefix = cfg.S3.Prefix + "/"
	}

	return nil
}

// validate ensures required config fields are present and valid.
func validate(cfg *types.Config) error {
	if cfg.S3.Bucket == "" {
		return fmt.Errorf("s3.bucket is required")
	}

	if cfg.S3.Region == "" {
		return fmt.Errorf("s3.region is required")
	}

	return nil
}

// expandTilde replaces ~ at the start of a path with the user's home directory.
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}

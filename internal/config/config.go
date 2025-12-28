// Package config handles loading, validation, and management of cclogs configuration files.
// It provides functions to read YAML config, apply defaults, validate required fields,
// and create starter configurations for new users.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/13rac1/cclogs/internal/types"
	"gopkg.in/yaml.v3"
)

const (
	defaultProjectsRoot = "~/.claude/projects"
	defaultS3Prefix     = "claude-code/"
)

const starterConfigTemplate = `# cclogs configuration file
# cclogs ships Claude Code session logs to S3-compatible storage

# Local configuration
local:
  # Path to Claude Code projects directory (default: ~/.claude/projects)
  projects_root: "~/.claude/projects"

# S3-compatible storage configuration
s3:
  # REQUIRED: S3 bucket name
  bucket: "YOUR-BUCKET-NAME"

  # REQUIRED: AWS region (e.g., us-west-2, us-east-1)
  region: "us-west-2"

  # Optional: Prefix for all uploaded files (default: claude-code/)
  prefix: "claude-code/"

  # Optional: Custom S3 endpoint for S3-compatible providers (Backblaze B2, MinIO, etc.)
  # endpoint: "https://s3.us-west-002.backblazeb2.com"

  # Optional: Use path-style addressing (required for some S3-compatible providers)
  # force_path_style: true

# Authentication configuration
auth:
  # Option 1: Use AWS profile from ~/.aws/credentials (recommended)
  profile: "default"

  # Option 2: Static credentials (not recommended - use profile instead)
  # access_key_id: ""
  # secret_access_key: ""
  # session_token: ""
`

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

// CreateStarterConfig creates a starter configuration file with helpful comments
// at the specified path. Creates parent directories if needed.
func CreateStarterConfig(path string) error {
	expandedPath, err := expandTilde(path)
	if err != nil {
		return fmt.Errorf("expanding config path: %w", err)
	}

	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	if err := os.WriteFile(expandedPath, []byte(starterConfigTemplate), 0644); err != nil {
		return fmt.Errorf("writing starter config to %s: %w", expandedPath, err)
	}

	return nil
}

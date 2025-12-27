package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/13rac1/ccls/internal/types"
)

func TestLoad(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name     string
		content  string
		wantErr  bool
		errMsg   string
		validate func(*testing.T, *types.Config)
	}{
		{
			name: "valid minimal config",
			content: `
s3:
  bucket: test-bucket
  region: us-west-2
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *types.Config) {
				if cfg.S3.Bucket != "test-bucket" {
					t.Errorf("bucket = %q, want %q", cfg.S3.Bucket, "test-bucket")
				}
				if cfg.S3.Region != "us-west-2" {
					t.Errorf("region = %q, want %q", cfg.S3.Region, "us-west-2")
				}
				// Check defaults
				if cfg.S3.Prefix != "claude-code/" {
					t.Errorf("prefix = %q, want %q", cfg.S3.Prefix, "claude-code/")
				}
				expectedRoot := filepath.Join(homeDir, ".claude/projects")
				if cfg.Local.ProjectsRoot != expectedRoot {
					t.Errorf("projects_root = %q, want %q", cfg.Local.ProjectsRoot, expectedRoot)
				}
			},
		},
		{
			name: "custom prefix without trailing slash",
			content: `
s3:
  bucket: test-bucket
  region: us-west-2
  prefix: custom-prefix
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *types.Config) {
				if cfg.S3.Prefix != "custom-prefix/" {
					t.Errorf("prefix = %q, want %q", cfg.S3.Prefix, "custom-prefix/")
				}
			},
		},
		{
			name: "custom prefix with trailing slash",
			content: `
s3:
  bucket: test-bucket
  region: us-west-2
  prefix: custom-prefix/
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *types.Config) {
				if cfg.S3.Prefix != "custom-prefix/" {
					t.Errorf("prefix = %q, want %q", cfg.S3.Prefix, "custom-prefix/")
				}
			},
		},
		{
			name: "tilde expansion in projects_root",
			content: `
local:
  projects_root: ~/custom/projects
s3:
  bucket: test-bucket
  region: us-west-2
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *types.Config) {
				expected := filepath.Join(homeDir, "custom/projects")
				if cfg.Local.ProjectsRoot != expected {
					t.Errorf("projects_root = %q, want %q", cfg.Local.ProjectsRoot, expected)
				}
			},
		},
		{
			name: "all optional fields",
			content: `
local:
  projects_root: /custom/path
s3:
  bucket: test-bucket
  region: us-west-2
  prefix: logs/
  endpoint: https://s3.example.com
  force_path_style: true
auth:
  profile: custom-profile
  access_key_id: AKIATEST
  secret_access_key: secretkey
  session_token: token123
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *types.Config) {
				if cfg.Local.ProjectsRoot != "/custom/path" {
					t.Errorf("projects_root = %q, want %q", cfg.Local.ProjectsRoot, "/custom/path")
				}
				if cfg.S3.Bucket != "test-bucket" {
					t.Errorf("bucket = %q, want %q", cfg.S3.Bucket, "test-bucket")
				}
				if cfg.S3.Region != "us-west-2" {
					t.Errorf("region = %q, want %q", cfg.S3.Region, "us-west-2")
				}
				if cfg.S3.Prefix != "logs/" {
					t.Errorf("prefix = %q, want %q", cfg.S3.Prefix, "logs/")
				}
				if cfg.S3.Endpoint != "https://s3.example.com" {
					t.Errorf("endpoint = %q, want %q", cfg.S3.Endpoint, "https://s3.example.com")
				}
				if !cfg.S3.ForcePathStyle {
					t.Error("force_path_style = false, want true")
				}
				if cfg.Auth.Profile != "custom-profile" {
					t.Errorf("profile = %q, want %q", cfg.Auth.Profile, "custom-profile")
				}
				if cfg.Auth.AccessKeyID != "AKIATEST" {
					t.Errorf("access_key_id = %q, want %q", cfg.Auth.AccessKeyID, "AKIATEST")
				}
			},
		},
		{
			name: "missing bucket",
			content: `
s3:
  region: us-west-2
`,
			wantErr: true,
			errMsg:  "s3.bucket is required",
		},
		{
			name: "missing region",
			content: `
s3:
  bucket: test-bucket
`,
			wantErr: true,
			errMsg:  "s3.region is required",
		},
		{
			name:    "invalid YAML",
			content: `invalid: yaml: content:`,
			wantErr: true,
			errMsg:  "parsing config YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpfile, err := os.CreateTemp("", "ccls-test-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Remove(tmpfile.Name()); err != nil {
					t.Logf("failed to remove temp file: %v", err)
				}
			}()

			if _, err := tmpfile.Write([]byte(tt.content)); err != nil {
				t.Fatal(err)
			}
			if err := tmpfile.Close(); err != nil {
				t.Fatal(err)
			}

			cfg, err := Load(tmpfile.Name())

			if tt.wantErr {
				if err == nil {
					t.Errorf("Load() error = nil, want error containing %q", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Load() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Load() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, cfg)
			}
		})
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Load() error = nil, want error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("Load() error = %q, want error containing 'reading config file'", err.Error())
	}
}

func TestExpandTilde(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tilde only",
			input: "~",
			want:  homeDir,
		},
		{
			name:  "tilde with path",
			input: "~/foo/bar",
			want:  filepath.Join(homeDir, "foo/bar"),
		},
		{
			name:  "absolute path",
			input: "/absolute/path",
			want:  "/absolute/path",
		},
		{
			name:  "relative path",
			input: "relative/path",
			want:  "relative/path",
		},
		{
			name:  "tilde in middle",
			input: "/path/~/file",
			want:  "/path/~/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := expandTilde(tt.input)
			if err != nil {
				t.Errorf("expandTilde() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("expandTilde(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateStarterConfig(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "create in temp directory",
			path:    filepath.Join(t.TempDir(), "config.yaml"),
			wantErr: false,
		},
		{
			name:    "create with nested directories",
			path:    filepath.Join(t.TempDir(), "nested", "dirs", "config.yaml"),
			wantErr: false,
		},
		{
			name:    "create with tilde path",
			path:    filepath.Join("~", ".ccls-test-"+t.Name(), "config.yaml"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateStarterConfig(tt.path)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateStarterConfig() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateStarterConfig() unexpected error = %v", err)
				return
			}

			expandedPath, err := expandTilde(tt.path)
			if err != nil {
				t.Fatalf("expandTilde() error = %v", err)
			}

			defer func() {
				if err := os.RemoveAll(filepath.Dir(expandedPath)); err != nil {
					t.Logf("failed to remove temp directory: %v", err)
				}
			}()

			info, err := os.Stat(expandedPath)
			if err != nil {
				t.Fatalf("config file not created: %v", err)
			}

			if info.Mode().Perm() != 0644 {
				t.Errorf("config file permissions = %o, want %o", info.Mode().Perm(), 0644)
			}

			content, err := os.ReadFile(expandedPath)
			if err != nil {
				t.Fatalf("failed to read config file: %v", err)
			}

			if !strings.Contains(string(content), "ccls configuration file") {
				t.Error("config file missing expected header comment")
			}
			if !strings.Contains(string(content), "YOUR-BUCKET-NAME") {
				t.Error("config file missing bucket placeholder")
			}
			if !strings.Contains(string(content), "us-west-2") {
				t.Error("config file missing region example")
			}

			dirInfo, err := os.Stat(filepath.Dir(expandedPath))
			if err != nil {
				t.Fatalf("config directory not created: %v", err)
			}
			if dirInfo.Mode().Perm() != 0755 {
				t.Errorf("config directory permissions = %o, want %o", dirInfo.Mode().Perm(), 0755)
			}
		})
	}
}

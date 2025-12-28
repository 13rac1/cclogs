package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/13rac1/cclogs/internal/types"
)

func TestRunChecks(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T) (cfg *types.Config, configPath string, cleanup func())
		wantPassed bool
	}{
		{
			name: "valid config with existing projects root and projects",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				// Create projects root with a project and JSONL file
				projectDir := filepath.Join(projectsRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("failed to create project dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(projectDir, "test.jsonl"), []byte("{}"), 0644); err != nil {
					t.Fatalf("failed to create jsonl file: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: true,
		},
		{
			name: "placeholder bucket name",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				if err := os.MkdirAll(projectsRoot, 0755); err != nil {
					t.Fatalf("failed to create projects root: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "YOUR-BUCKET-NAME",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: false,
		},
		{
			name: "empty bucket name",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				if err := os.MkdirAll(projectsRoot, 0755); err != nil {
					t.Fatalf("failed to create projects root: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: false,
		},
		{
			name: "missing S3 region",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				if err := os.MkdirAll(projectsRoot, 0755); err != nil {
					t.Fatalf("failed to create projects root: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: false,
		},
		{
			name: "missing projects root",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "nonexistent")
				configPath = filepath.Join(tmpDir, "config.yaml")

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: false,
		},
		{
			name: "projects root is a file not a directory",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				// Create a file instead of a directory
				if err := os.WriteFile(projectsRoot, []byte("not a directory"), 0644); err != nil {
					t.Fatalf("failed to create file: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: false,
		},
		{
			name: "unreadable projects root",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				if os.Getuid() == 0 {
					t.Skip("skipping permission test when running as root")
				}

				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				// Create projects root and make it unreadable
				if err := os.MkdirAll(projectsRoot, 0755); err != nil {
					t.Fatalf("failed to create projects root: %v", err)
				}
				if err := os.Chmod(projectsRoot, 0000); err != nil {
					t.Fatalf("failed to chmod projects root: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				cleanupFunc := func() {
					// Restore permissions for cleanup
					os.Chmod(projectsRoot, 0755)
				}

				return cfg, configPath, cleanupFunc
			},
			wantPassed: false,
		},
		{
			name: "empty projects root",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				if err := os.MkdirAll(projectsRoot, 0755); err != nil {
					t.Fatalf("failed to create projects root: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: true,
		},
		{
			name: "projects root with directories but no JSONL files",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				// Create projects root with empty projects
				project1 := filepath.Join(projectsRoot, "project1")
				project2 := filepath.Join(projectsRoot, "project2")
				if err := os.MkdirAll(project1, 0755); err != nil {
					t.Fatalf("failed to create project1: %v", err)
				}
				if err := os.MkdirAll(project2, 0755); err != nil {
					t.Fatalf("failed to create project2: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "claude-code/",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: true,
		},
		{
			name: "empty S3 prefix is valid",
			setupFunc: func(t *testing.T) (cfg *types.Config, configPath string, cleanup func()) {
				tmpDir := t.TempDir()
				projectsRoot := filepath.Join(tmpDir, "projects")
				configPath = filepath.Join(tmpDir, "config.yaml")

				if err := os.MkdirAll(projectsRoot, 0755); err != nil {
					t.Fatalf("failed to create projects root: %v", err)
				}

				cfg = &types.Config{
					Local: types.LocalConfig{
						ProjectsRoot: projectsRoot,
					},
					S3: types.S3Config{
						Bucket: "my-bucket",
						Region: "us-west-2",
						Prefix: "",
					},
				}

				return cfg, configPath, func() {}
			},
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, configPath, cleanup := tt.setupFunc(t)
			defer cleanup()

			// Skip remote connectivity checks in tests (no AWS credentials available)
			got := RunChecks(cfg, configPath, true)

			if got != tt.wantPassed {
				t.Errorf("RunChecks() = %v, want %v", got, tt.wantPassed)
			}
		})
	}
}

func TestCountDirectories(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) []os.DirEntry
		want  int
	}{
		{
			name: "empty list",
			setup: func(t *testing.T) []os.DirEntry {
				return []os.DirEntry{}
			},
			want: 0,
		},
		{
			name: "only files",
			setup: func(t *testing.T) []os.DirEntry {
				tmpDir := t.TempDir()
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
				entries, _ := os.ReadDir(tmpDir)
				return entries
			},
			want: 0,
		},
		{
			name: "only directories",
			setup: func(t *testing.T) []os.DirEntry {
				tmpDir := t.TempDir()
				os.MkdirAll(filepath.Join(tmpDir, "dir1"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "dir2"), 0755)
				entries, _ := os.ReadDir(tmpDir)
				return entries
			},
			want: 2,
		},
		{
			name: "mixed files and directories",
			setup: func(t *testing.T) []os.DirEntry {
				tmpDir := t.TempDir()
				os.MkdirAll(filepath.Join(tmpDir, "dir1"), 0755)
				os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
				os.MkdirAll(filepath.Join(tmpDir, "dir2"), 0755)
				os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("test"), 0644)
				entries, _ := os.ReadDir(tmpDir)
				return entries
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := tt.setup(t)
			got := countDirectories(entries)
			if got != tt.want {
				t.Errorf("countDirectories() = %v, want %v", got, tt.want)
			}
		})
	}
}

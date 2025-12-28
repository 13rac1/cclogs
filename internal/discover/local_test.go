package discover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/13rac1/cclogs/internal/types"
)

func TestDiscoverLocal(t *testing.T) {
	tests := []struct {
		name       string
		setupFunc  func(t *testing.T) string // returns projectsRoot
		wantErr    bool
		wantErrMsg string
		wantCount  int // number of projects
		validate   func(t *testing.T, projects []types.Project)
	}{
		{
			name: "valid project with multiple jsonl files",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				projectDir := filepath.Join(root, "my-project")
				if err := os.Mkdir(projectDir, 0755); err != nil {
					t.Fatal(err)
				}
				createFile(t, filepath.Join(projectDir, "session1.jsonl"))
				createFile(t, filepath.Join(projectDir, "session2.jsonl"))
				createFile(t, filepath.Join(projectDir, "notes.txt"))
				return root
			},
			wantErr:   false,
			wantCount: 1,
			validate: func(t *testing.T, projects []types.Project) {
				if projects[0].Name != "my-project" {
					t.Errorf("expected project name 'my-project', got %q", projects[0].Name)
				}
				if projects[0].LocalCount != 2 {
					t.Errorf("expected 2 JSONL files, got %d", projects[0].LocalCount)
				}
			},
		},
		{
			name: "project with nested directories containing jsonl files",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				projectDir := filepath.Join(root, "nested-project")
				subDir := filepath.Join(projectDir, "subdir")
				deepDir := filepath.Join(subDir, "deep")
				if err := os.MkdirAll(deepDir, 0755); err != nil {
					t.Fatal(err)
				}
				createFile(t, filepath.Join(projectDir, "root.jsonl"))
				createFile(t, filepath.Join(subDir, "sub.jsonl"))
				createFile(t, filepath.Join(deepDir, "deep.jsonl"))
				return root
			},
			wantErr:   false,
			wantCount: 1,
			validate: func(t *testing.T, projects []types.Project) {
				if projects[0].Name != "nested-project" {
					t.Errorf("expected project name 'nested-project', got %q", projects[0].Name)
				}
				if projects[0].LocalCount != 3 {
					t.Errorf("expected 3 JSONL files, got %d", projects[0].LocalCount)
				}
			},
		},
		{
			name: "project with no jsonl files",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				projectDir := filepath.Join(root, "empty-project")
				if err := os.Mkdir(projectDir, 0755); err != nil {
					t.Fatal(err)
				}
				createFile(t, filepath.Join(projectDir, "README.md"))
				createFile(t, filepath.Join(projectDir, "config.yaml"))
				return root
			},
			wantErr:   false,
			wantCount: 1,
			validate: func(t *testing.T, projects []types.Project) {
				if projects[0].Name != "empty-project" {
					t.Errorf("expected project name 'empty-project', got %q", projects[0].Name)
				}
				if projects[0].LocalCount != 0 {
					t.Errorf("expected 0 JSONL files, got %d", projects[0].LocalCount)
				}
			},
		},
		{
			name: "empty projects root (no projects)",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name: "mixed files and directories (only directories are projects)",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				// Create some files (should be ignored)
				createFile(t, filepath.Join(root, "random.txt"))
				createFile(t, filepath.Join(root, "session.jsonl"))
				// Create directories (should be treated as projects)
				projectDir := filepath.Join(root, "real-project")
				if err := os.Mkdir(projectDir, 0755); err != nil {
					t.Fatal(err)
				}
				createFile(t, filepath.Join(projectDir, "session.jsonl"))
				return root
			},
			wantErr:   false,
			wantCount: 1,
			validate: func(t *testing.T, projects []types.Project) {
				if projects[0].Name != "real-project" {
					t.Errorf("expected project name 'real-project', got %q", projects[0].Name)
				}
				if projects[0].LocalCount != 1 {
					t.Errorf("expected 1 JSONL file, got %d", projects[0].LocalCount)
				}
			},
		},
		{
			name: "multiple projects sorted by name",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				// Create projects in non-alphabetical order
				for _, name := range []string{"zebra", "alpha", "beta"} {
					projectDir := filepath.Join(root, name)
					if err := os.Mkdir(projectDir, 0755); err != nil {
						t.Fatal(err)
					}
					createFile(t, filepath.Join(projectDir, "session.jsonl"))
				}
				return root
			},
			wantErr:   false,
			wantCount: 3,
			validate: func(t *testing.T, projects []types.Project) {
				expectedOrder := []string{"alpha", "beta", "zebra"}
				for i, expected := range expectedOrder {
					if projects[i].Name != expected {
						t.Errorf("expected project[%d] to be %q, got %q", i, expected, projects[i].Name)
					}
				}
			},
		},
		{
			name: "case insensitive jsonl extension",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				projectDir := filepath.Join(root, "test-project")
				if err := os.Mkdir(projectDir, 0755); err != nil {
					t.Fatal(err)
				}
				createFile(t, filepath.Join(projectDir, "lower.jsonl"))
				createFile(t, filepath.Join(projectDir, "upper.JSONL"))
				createFile(t, filepath.Join(projectDir, "mixed.JsonL"))
				return root
			},
			wantErr:   false,
			wantCount: 1,
			validate: func(t *testing.T, projects []types.Project) {
				if projects[0].LocalCount != 3 {
					t.Errorf("expected 3 JSONL files (case insensitive), got %d", projects[0].LocalCount)
				}
			},
		},
		{
			name: "projects root does not exist",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantErr:    true,
			wantErrMsg: "projects root does not exist",
		},
		{
			name: "projects root is a file, not directory",
			setupFunc: func(t *testing.T) string {
				root := t.TempDir()
				filePath := filepath.Join(root, "notadir")
				createFile(t, filePath)
				return filePath
			},
			wantErr:    true,
			wantErrMsg: "projects root is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectsRoot := tt.setupFunc(t)

			projects, err := DiscoverLocal(projectsRoot)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				if tt.wantErrMsg != "" && !contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.wantErrMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(projects) != tt.wantCount {
				t.Errorf("expected %d projects, got %d", tt.wantCount, len(projects))
			}

			if tt.validate != nil {
				tt.validate(t, projects)
			}

			// Verify all projects have valid paths
			for _, p := range projects {
				if p.Name == "" {
					t.Error("project has empty name")
				}
				if p.LocalPath == "" {
					t.Error("project has empty path")
				}
				if p.LocalCount < 0 {
					t.Errorf("project %s has negative JSONL count: %d", p.Name, p.LocalCount)
				}
			}
		})
	}
}

// createFile creates an empty file at the given path.
func createFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close file %s: %v", path, err)
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/13rac1/cclogs/internal/types"
)

func TestPrintLocalProjects(t *testing.T) {
	tests := []struct {
		name        string
		projects    []types.Project
		contains    []string
		notContains []string
	}{
		{
			name: "multiple projects",
			projects: []types.Project{
				{Name: "project-a", LocalCount: 5},
				{Name: "project-b", LocalCount: 12},
			},
			contains: []string{
				"Local Projects",
				"PROJECT",
				"JSONL FILES",
				"project-a",
				"5",
				"project-b",
				"12",
			},
		},
		{
			name: "single project",
			projects: []types.Project{
				{Name: "my-project", LocalCount: 3},
			},
			contains: []string{
				"Local Projects",
				"PROJECT",
				"JSONL FILES",
				"my-project",
				"3",
			},
		},
		{
			name:     "empty projects list",
			projects: []types.Project{},
			contains: []string{
				"No local projects found.",
			},
			notContains: []string{
				"PROJECT",
				"JSONL FILES",
			},
		},
		{
			name:     "nil projects list",
			projects: nil,
			contains: []string{
				"No local projects found.",
			},
			notContains: []string{
				"PROJECT",
				"JSONL FILES",
			},
		},
		{
			name: "project with zero files",
			projects: []types.Project{
				{Name: "empty-project", LocalCount: 0},
			},
			contains: []string{
				"Local Projects",
				"empty-project",
				"0",
			},
		},
		{
			name: "project with large file count",
			projects: []types.Project{
				{Name: "big-project", LocalCount: 999},
			},
			contains: []string{
				"Local Projects",
				"big-project",
				"999",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				PrintLocalProjects(tt.projects)
			})

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}

			for _, unwanted := range tt.notContains {
				if strings.Contains(output, unwanted) {
					t.Errorf("output contains unwanted string %q\nGot:\n%s", unwanted, output)
				}
			}
		})
	}
}

func TestPrintLocalProjects_TableFormat(t *testing.T) {
	projects := []types.Project{
		{Name: "test-project", LocalCount: 5},
	}

	output := captureStdout(func() {
		PrintLocalProjects(projects)
	})

	// Verify table borders are present
	if !strings.Contains(output, "─") && !strings.Contains(output, "-") {
		t.Error("output missing horizontal borders")
	}

	// Verify table has some structure (pipes or box drawing characters)
	if !strings.Contains(output, "│") && !strings.Contains(output, "|") {
		t.Error("output missing vertical borders")
	}
}

func TestPrintLocalProjects_HeaderFormatting(t *testing.T) {
	projects := []types.Project{
		{Name: "test", LocalCount: 1},
	}

	output := captureStdout(func() {
		PrintLocalProjects(projects)
	})

	lines := strings.Split(output, "\n")

	// Verify header is present in output
	foundHeader := false
	for _, line := range lines {
		if strings.Contains(line, "PROJECT") && strings.Contains(line, "JSONL FILES") {
			foundHeader = true
			break
		}
	}

	if !foundHeader {
		t.Errorf("table header not found in output:\n%s", output)
	}
}

func TestPrintProjects(t *testing.T) {
	tests := []struct {
		name     string
		projects []types.Project
		contains []string
	}{
		{
			name: "local and remote match",
			projects: []types.Project{
				{Name: "project-a", LocalCount: 5, RemoteCount: 5},
			},
			contains: []string{
				"Projects",
				"PROJECT",
				"LOCAL",
				"REMOTE",
				"STATUS",
				"project-a",
				"5",
				"OK",
			},
		},
		{
			name: "local only",
			projects: []types.Project{
				{Name: "project-b", LocalCount: 3, RemoteCount: 0},
			},
			contains: []string{
				"Projects",
				"project-b",
				"3",
				"-",
				"Local-only",
			},
		},
		{
			name: "remote only",
			projects: []types.Project{
				{Name: "project-c", LocalCount: 0, RemoteCount: 10},
			},
			contains: []string{
				"Projects",
				"project-c",
				"-",
				"10",
				"Remote-only",
			},
		},
		{
			name: "mismatch",
			projects: []types.Project{
				{Name: "project-d", LocalCount: 5, RemoteCount: 3},
			},
			contains: []string{
				"Projects",
				"project-d",
				"5",
				"3",
				"Mismatch",
			},
		},
		{
			name: "mixed statuses",
			projects: []types.Project{
				{Name: "synced", LocalCount: 5, RemoteCount: 5},
				{Name: "local-only", LocalCount: 2, RemoteCount: 0},
				{Name: "remote-only", LocalCount: 0, RemoteCount: 8},
				{Name: "mismatch", LocalCount: 3, RemoteCount: 7},
			},
			contains: []string{
				"synced",
				"OK",
				"local-only",
				"Local-only",
				"remote-only",
				"Remote-only",
				"mismatch",
				"Mismatch",
			},
		},
		{
			name:     "empty list",
			projects: []types.Project{},
			contains: []string{
				"No projects found.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				PrintProjects(tt.projects)
			})

			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing expected string %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestFormatCount(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{count: 0, want: "-"},
		{count: 1, want: "1"},
		{count: 10, want: "10"},
		{count: 999, want: "999"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatCount(tt.count)
			if got != tt.want {
				t.Errorf("formatCount(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		name        string
		localCount  int
		remoteCount int
		want        string
	}{
		{
			name:        "both zero",
			localCount:  0,
			remoteCount: 0,
			want:        "-",
		},
		{
			name:        "local only",
			localCount:  5,
			remoteCount: 0,
			want:        "Local-only",
		},
		{
			name:        "remote only",
			localCount:  0,
			remoteCount: 10,
			want:        "Remote-only",
		},
		{
			name:        "match",
			localCount:  5,
			remoteCount: 5,
			want:        "OK",
		},
		{
			name:        "mismatch local higher",
			localCount:  10,
			remoteCount: 5,
			want:        "Mismatch",
		},
		{
			name:        "mismatch remote higher",
			localCount:  3,
			remoteCount: 8,
			want:        "Mismatch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineStatus(tt.localCount, tt.remoteCount)
			if got != tt.want {
				t.Errorf("determineStatus(%d, %d) = %q, want %q",
					tt.localCount, tt.remoteCount, got, tt.want)
			}
		})
	}
}

func TestPrintJSON(t *testing.T) {
	tests := []struct {
		name     string
		projects []types.Project
		cfg      *types.Config
		validate func(*testing.T, []byte)
	}{
		{
			name: "projects with local and remote",
			projects: []types.Project{
				{
					Name:        "test-project",
					LocalPath:   "/path/to/test-project",
					LocalCount:  5,
					RemotePath:  "claude-code/test-project/",
					RemoteCount: 5,
				},
			},
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket:   "test-bucket",
					Prefix:   "claude-code/",
					Endpoint: "https://s3.example.com",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result JSONOutput
				if err := json.Unmarshal(output, &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if result.GeneratedAt == "" {
					t.Error("missing generatedAt")
				}

				if result.Config.Bucket != "test-bucket" {
					t.Errorf("config.bucket = %q, want %q", result.Config.Bucket, "test-bucket")
				}

				if result.Config.Prefix != "claude-code/" {
					t.Errorf("config.prefix = %q, want %q", result.Config.Prefix, "claude-code/")
				}

				if result.Config.Endpoint != "https://s3.example.com" {
					t.Errorf("config.endpoint = %q, want %q", result.Config.Endpoint, "https://s3.example.com")
				}

				if len(result.LocalProjects) != 1 {
					t.Fatalf("expected 1 local project, got %d", len(result.LocalProjects))
				}

				local := result.LocalProjects[0]
				if local.Name != "test-project" {
					t.Errorf("local.name = %q, want %q", local.Name, "test-project")
				}
				if local.Path != "/path/to/test-project" {
					t.Errorf("local.path = %q, want %q", local.Path, "/path/to/test-project")
				}
				if local.JSONLCount != 5 {
					t.Errorf("local.jsonlCount = %d, want %d", local.JSONLCount, 5)
				}

				if len(result.RemoteProjects) != 1 {
					t.Fatalf("expected 1 remote project, got %d", len(result.RemoteProjects))
				}

				remote := result.RemoteProjects[0]
				if remote.Name != "test-project" {
					t.Errorf("remote.name = %q, want %q", remote.Name, "test-project")
				}
				if remote.Prefix != "claude-code/test-project/" {
					t.Errorf("remote.prefix = %q, want %q", remote.Prefix, "claude-code/test-project/")
				}
				if remote.JSONLCount != 5 {
					t.Errorf("remote.jsonlCount = %d, want %d", remote.JSONLCount, 5)
				}
			},
		},
		{
			name: "local-only projects",
			projects: []types.Project{
				{
					Name:       "local-only",
					LocalPath:  "/path/to/local-only",
					LocalCount: 3,
				},
			},
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Prefix: "prefix/",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result JSONOutput
				if err := json.Unmarshal(output, &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if len(result.LocalProjects) != 1 {
					t.Fatalf("expected 1 local project, got %d", len(result.LocalProjects))
				}

				if len(result.RemoteProjects) != 0 {
					t.Fatalf("expected 0 remote projects, got %d", len(result.RemoteProjects))
				}

				local := result.LocalProjects[0]
				if local.Name != "local-only" {
					t.Errorf("local.name = %q, want %q", local.Name, "local-only")
				}
			},
		},
		{
			name: "remote-only projects",
			projects: []types.Project{
				{
					Name:        "remote-only",
					RemotePath:  "prefix/remote-only/",
					RemoteCount: 10,
				},
			},
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Prefix: "prefix/",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result JSONOutput
				if err := json.Unmarshal(output, &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if len(result.LocalProjects) != 0 {
					t.Fatalf("expected 0 local projects, got %d", len(result.LocalProjects))
				}

				if len(result.RemoteProjects) != 1 {
					t.Fatalf("expected 1 remote project, got %d", len(result.RemoteProjects))
				}

				remote := result.RemoteProjects[0]
				if remote.Name != "remote-only" {
					t.Errorf("remote.name = %q, want %q", remote.Name, "remote-only")
				}
			},
		},
		{
			name:     "empty projects list",
			projects: []types.Project{},
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Prefix: "prefix/",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result JSONOutput
				if err := json.Unmarshal(output, &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if result.LocalProjects == nil {
					t.Error("localProjects should be empty array, not null")
				}

				if result.RemoteProjects == nil {
					t.Error("remoteProjects should be empty array, not null")
				}

				if len(result.LocalProjects) != 0 {
					t.Errorf("expected 0 local projects, got %d", len(result.LocalProjects))
				}

				if len(result.RemoteProjects) != 0 {
					t.Errorf("expected 0 remote projects, got %d", len(result.RemoteProjects))
				}
			},
		},
		{
			name: "multiple mixed projects",
			projects: []types.Project{
				{Name: "both", LocalPath: "/both", LocalCount: 5, RemotePath: "prefix/both/", RemoteCount: 5},
				{Name: "local", LocalPath: "/local", LocalCount: 3},
				{Name: "remote", RemotePath: "prefix/remote/", RemoteCount: 10},
			},
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Prefix: "prefix/",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result JSONOutput
				if err := json.Unmarshal(output, &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if len(result.LocalProjects) != 2 {
					t.Fatalf("expected 2 local projects, got %d", len(result.LocalProjects))
				}

				if len(result.RemoteProjects) != 2 {
					t.Fatalf("expected 2 remote projects, got %d", len(result.RemoteProjects))
				}
			},
		},
		{
			name: "config without endpoint",
			projects: []types.Project{
				{Name: "test", LocalPath: "/test", LocalCount: 1},
			},
			cfg: &types.Config{
				S3: types.S3Config{
					Bucket: "test-bucket",
					Prefix: "prefix/",
				},
			},
			validate: func(t *testing.T, output []byte) {
				var result JSONOutput
				if err := json.Unmarshal(output, &result); err != nil {
					t.Fatalf("invalid JSON: %v", err)
				}

				if result.Config.Endpoint != "" {
					t.Errorf("config.endpoint should be omitted when empty, got %q", result.Config.Endpoint)
				}

				// Verify endpoint field is actually omitted from JSON
				var raw map[string]interface{}
				if err := json.Unmarshal(output, &raw); err != nil {
					t.Fatalf("failed to unmarshal to raw map: %v", err)
				}

				configMap := raw["config"].(map[string]interface{})
				if _, exists := configMap["endpoint"]; exists {
					t.Error("endpoint field should be omitted from JSON when empty")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureStdout(func() {
				if err := PrintJSON(tt.projects, tt.cfg); err != nil {
					t.Fatalf("PrintJSON failed: %v", err)
				}
			})

			tt.validate(t, []byte(output))
		})
	}
}

func TestPrintJSON_RFC3339Timestamp(t *testing.T) {
	projects := []types.Project{
		{Name: "test", LocalPath: "/test", LocalCount: 1},
	}
	cfg := &types.Config{
		S3: types.S3Config{
			Bucket: "test-bucket",
			Prefix: "prefix/",
		},
	}

	output := captureStdout(func() {
		if err := PrintJSON(projects, cfg); err != nil {
			t.Fatalf("PrintJSON failed: %v", err)
		}
	})

	var result JSONOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Parse timestamp to verify it's valid RFC3339
	_, err := time.Parse(time.RFC3339, result.GeneratedAt)
	if err != nil {
		t.Errorf("generatedAt is not valid RFC3339: %v (got: %q)", err, result.GeneratedAt)
	}

	// Verify timestamp ends with 'Z' (UTC)
	if !strings.HasSuffix(result.GeneratedAt, "Z") {
		t.Errorf("generatedAt should be UTC (end with Z), got: %q", result.GeneratedAt)
	}
}

func TestPrintJSON_IndentedOutput(t *testing.T) {
	projects := []types.Project{
		{Name: "test", LocalPath: "/test", LocalCount: 1},
	}
	cfg := &types.Config{
		S3: types.S3Config{
			Bucket: "test-bucket",
			Prefix: "prefix/",
		},
	}

	output := captureStdout(func() {
		if err := PrintJSON(projects, cfg); err != nil {
			t.Fatalf("PrintJSON failed: %v", err)
		}
	})

	// Verify output is indented (contains multiple spaces at start of lines)
	lines := strings.Split(output, "\n")
	foundIndentation := false
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") {
			foundIndentation = true
			break
		}
	}

	if !foundIndentation {
		t.Error("JSON output should be indented")
	}
}

// captureStdout captures os.Stdout output from the given function.
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

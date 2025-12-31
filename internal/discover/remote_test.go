package discover

import (
	"testing"
	"time"

	"github.com/13rac1/cclogs/internal/manifest"
)

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		name          string
		projectPrefix string
		basePrefix    string
		want          string
	}{
		{
			name:          "standard case",
			projectPrefix: "claude-code/my-project/",
			basePrefix:    "claude-code/",
			want:          "my-project",
		},
		{
			name:          "nested project path",
			projectPrefix: "claude-code/org/my-project/",
			basePrefix:    "claude-code/",
			want:          "my-project",
		},
		{
			name:          "empty base prefix",
			projectPrefix: "my-project/",
			basePrefix:    "",
			want:          "my-project",
		},
		{
			name:          "no trailing slash on project prefix",
			projectPrefix: "claude-code/my-project",
			basePrefix:    "claude-code/",
			want:          "my-project",
		},
		{
			name:          "base prefix without trailing slash",
			projectPrefix: "claude-code/my-project/",
			basePrefix:    "claude-code",
			want:          "my-project",
		},
		{
			name:          "project name with hyphens",
			projectPrefix: "claude-code/my-complex-project-name/",
			basePrefix:    "claude-code/",
			want:          "my-complex-project-name",
		},
		{
			name:          "project name with underscores",
			projectPrefix: "claude-code/my_project_name/",
			basePrefix:    "claude-code/",
			want:          "my_project_name",
		},
		{
			name:          "single character project name",
			projectPrefix: "claude-code/a/",
			basePrefix:    "claude-code/",
			want:          "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProjectName(tt.projectPrefix, tt.basePrefix)
			if got != tt.want {
				t.Errorf("extractProjectName(%q, %q) = %q, want %q",
					tt.projectPrefix, tt.basePrefix, got, tt.want)
			}
		})
	}
}

func TestStrPtr(t *testing.T) {
	s := "test"
	ptr := strPtr(s)

	if ptr == nil {
		t.Fatal("strPtr returned nil")
	}

	if *ptr != s {
		t.Errorf("strPtr(%q) = %q, want %q", s, *ptr, s)
	}
}

// Note: Integration tests for DiscoverRemote, listProjectPrefixes, and countRemoteJSONLFiles
// are skipped in unit tests as they require actual S3 connectivity.
// These should be tested manually or with integration test suite using localstack/minio.
//
// To test manually:
// 1. Set up S3 bucket with test data
// 2. Configure cclogs with test credentials
// 3. Run: go test -v -run TestDiscoverRemote_Integration
//
// Example test bucket structure:
// s3://test-bucket/claude-code/
//   project-a/
//     session1.jsonl
//     session2.jsonl
//   project-b/
//     session1.jsonl

func TestDiscoverFromManifest(t *testing.T) {
	mtime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		files   map[string]manifest.FileEntry
		prefix  string
		want    []string // project names in sorted order
		counts  map[string]int
	}{
		{
			name:   "empty manifest",
			files:  map[string]manifest.FileEntry{},
			prefix: "claude-code/",
			want:   []string{},
			counts: map[string]int{},
		},
		{
			name: "single project with multiple files",
			files: map[string]manifest.FileEntry{
				"claude-code/project-a/session.jsonl":      {Mtime: mtime, Size: 100},
				"claude-code/project-a/logs/2025-01.jsonl": {Mtime: mtime, Size: 200},
			},
			prefix: "claude-code/",
			want:   []string{"project-a"},
			counts: map[string]int{"project-a": 2},
		},
		{
			name: "multiple projects",
			files: map[string]manifest.FileEntry{
				"claude-code/project-a/session.jsonl": {Mtime: mtime, Size: 100},
				"claude-code/project-b/session.jsonl": {Mtime: mtime, Size: 200},
				"claude-code/project-c/logs.jsonl":    {Mtime: mtime, Size: 300},
			},
			prefix: "claude-code/",
			want:   []string{"project-a", "project-b", "project-c"},
			counts: map[string]int{"project-a": 1, "project-b": 1, "project-c": 1},
		},
		{
			name: "prefix without trailing slash",
			files: map[string]manifest.FileEntry{
				"claude-code/project-a/session.jsonl": {Mtime: mtime, Size: 100},
			},
			prefix: "claude-code",
			want:   []string{"project-a"},
			counts: map[string]int{"project-a": 1},
		},
		{
			name: "empty prefix",
			files: map[string]manifest.FileEntry{
				"project-a/session.jsonl": {Mtime: mtime, Size: 100},
				"project-b/logs.jsonl":    {Mtime: mtime, Size: 200},
			},
			prefix: "",
			want:   []string{"project-a", "project-b"},
			counts: map[string]int{"project-a": 1, "project-b": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &manifest.Manifest{
				Version: 1,
				Files:   tt.files,
			}

			got := DiscoverFromManifest(m, tt.prefix)

			if len(got) != len(tt.want) {
				t.Errorf("DiscoverFromManifest() returned %d projects, want %d", len(got), len(tt.want))
			}

			for i, wantName := range tt.want {
				if i >= len(got) {
					break
				}
				if got[i].Name != wantName {
					t.Errorf("DiscoverFromManifest()[%d].Name = %q, want %q", i, got[i].Name, wantName)
				}
				wantCount := tt.counts[wantName]
				if got[i].RemoteCount != wantCount {
					t.Errorf("DiscoverFromManifest()[%d].RemoteCount = %d, want %d", i, got[i].RemoteCount, wantCount)
				}
			}
		})
	}
}

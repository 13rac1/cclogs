package discover

import (
	"testing"
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
// 2. Configure ccls with test credentials
// 3. Run: go test -v -run TestDiscoverRemote_Integration
//
// Example test bucket structure:
// s3://test-bucket/claude-code/
//   project-a/
//     session1.jsonl
//     session2.jsonl
//   project-b/
//     session1.jsonl

package uploader

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/13rac1/cclogs/internal/types"
)

func TestComputeS3Key(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		projectDir string
		relPath    string
		want       string
	}{
		{
			name:       "simple file",
			prefix:     "claude-code/",
			projectDir: "my-project",
			relPath:    "session.jsonl",
			want:       "claude-code/my-project/session.jsonl",
		},
		{
			name:       "nested file",
			prefix:     "claude-code/",
			projectDir: "my-project",
			relPath:    "sessions/2025-01.jsonl",
			want:       "claude-code/my-project/sessions/2025-01.jsonl",
		},
		{
			name:       "prefix without trailing slash",
			prefix:     "claude-code",
			projectDir: "my-project",
			relPath:    "session.jsonl",
			want:       "claude-code/my-project/session.jsonl",
		},
		{
			name:       "empty prefix",
			prefix:     "",
			projectDir: "my-project",
			relPath:    "session.jsonl",
			want:       "my-project/session.jsonl",
		},
		{
			name:       "windows path separators",
			prefix:     "claude-code/",
			projectDir: "my-project",
			relPath:    "sessions\\2025-01.jsonl",
			want:       "claude-code/my-project/sessions/2025-01.jsonl",
		},
		{
			name:       "deeply nested",
			prefix:     "logs/",
			projectDir: "project-a",
			relPath:    "2025/01/15/session.jsonl",
			want:       "logs/project-a/2025/01/15/session.jsonl",
		},
		{
			name:       "project name with hyphens",
			prefix:     "claude-code/",
			projectDir: "my-awesome-project",
			relPath:    "session.jsonl",
			want:       "claude-code/my-awesome-project/session.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeS3Key(tt.prefix, tt.projectDir, tt.relPath)
			if got != tt.want {
				t.Errorf("ComputeS3Key() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDiscoverFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create projects/my-project/session.jsonl
	projectDir := filepath.Join(tmpDir, "my-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	sessionFile := filepath.Join(projectDir, "session.jsonl")
	sessionContent := []byte("test session data")
	if err := os.WriteFile(sessionFile, sessionContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested file: sessions/2025-01.jsonl
	sessionsDir := filepath.Join(projectDir, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		t.Fatal(err)
	}
	nestedFile := filepath.Join(sessionsDir, "2025-01.jsonl")
	nestedContent := []byte("nested session")
	if err := os.WriteFile(nestedFile, nestedContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a non-.jsonl file that should be ignored
	ignoreFile := filepath.Join(projectDir, "readme.txt")
	if err := os.WriteFile(ignoreFile, []byte("ignore me"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test discovery
	cfg := &types.Config{
		Local: types.LocalConfig{ProjectsRoot: tmpDir},
		S3:    types.S3Config{Prefix: "claude-code/"},
	}

	uploader := New(cfg, nil, true)
	files, err := uploader.DiscoverFiles(context.Background())
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Sort files by S3 key for deterministic comparison
	sort.Slice(files, func(i, j int) bool {
		return files[i].S3Key < files[j].S3Key
	})

	// Verify first file (session.jsonl comes before sessions/2025-01.jsonl alphabetically)
	if files[0].ProjectDir != "my-project" {
		t.Errorf("files[0].ProjectDir = %q, want %q", files[0].ProjectDir, "my-project")
	}
	if files[0].S3Key != "claude-code/my-project/session.jsonl" {
		t.Errorf("files[0].S3Key = %q, want %q", files[0].S3Key, "claude-code/my-project/session.jsonl")
	}
	if files[0].Size != int64(len(sessionContent)) {
		t.Errorf("files[0].Size = %d, want %d", files[0].Size, len(sessionContent))
	}

	// Verify second file (nested file)
	if files[1].ProjectDir != "my-project" {
		t.Errorf("files[1].ProjectDir = %q, want %q", files[1].ProjectDir, "my-project")
	}
	if files[1].S3Key != "claude-code/my-project/sessions/2025-01.jsonl" {
		t.Errorf("files[1].S3Key = %q, want %q", files[1].S3Key, "claude-code/my-project/sessions/2025-01.jsonl")
	}
	if files[1].Size != int64(len(nestedContent)) {
		t.Errorf("files[1].Size = %d, want %d", files[1].Size, len(nestedContent))
	}
}

func TestDiscoverFilesMultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project-a with one file
	projectA := filepath.Join(tmpDir, "project-a")
	if err := os.MkdirAll(projectA, 0755); err != nil {
		t.Fatal(err)
	}
	fileA := filepath.Join(projectA, "a.jsonl")
	if err := os.WriteFile(fileA, []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create project-b with two files
	projectB := filepath.Join(tmpDir, "project-b")
	if err := os.MkdirAll(projectB, 0755); err != nil {
		t.Fatal(err)
	}
	fileB1 := filepath.Join(projectB, "b1.jsonl")
	if err := os.WriteFile(fileB1, []byte("b1"), 0644); err != nil {
		t.Fatal(err)
	}
	fileB2 := filepath.Join(projectB, "b2.jsonl")
	if err := os.WriteFile(fileB2, []byte("b2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file in root (should be ignored - not in a project dir)
	rootFile := filepath.Join(tmpDir, "root.jsonl")
	if err := os.WriteFile(rootFile, []byte("root"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &types.Config{
		Local: types.LocalConfig{ProjectsRoot: tmpDir},
		S3:    types.S3Config{Prefix: "logs"},
	}

	uploader := New(cfg, nil, true)
	files, err := uploader.DiscoverFiles(context.Background())
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	// Should find 3 files total (1 from project-a, 2 from project-b)
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	// Count files per project
	projectCounts := make(map[string]int)
	for _, f := range files {
		projectCounts[f.ProjectDir]++
	}

	if projectCounts["project-a"] != 1 {
		t.Errorf("expected 1 file in project-a, got %d", projectCounts["project-a"])
	}
	if projectCounts["project-b"] != 2 {
		t.Errorf("expected 2 files in project-b, got %d", projectCounts["project-b"])
	}
}

func TestDiscoverFilesEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an empty project directory
	projectDir := filepath.Join(tmpDir, "empty-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &types.Config{
		Local: types.LocalConfig{ProjectsRoot: tmpDir},
		S3:    types.S3Config{Prefix: "claude-code/"},
	}

	uploader := New(cfg, nil, true)
	files, err := uploader.DiscoverFiles(context.Background())
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files in empty project, got %d", len(files))
	}
}

func TestDiscoverFilesNonexistentRoot(t *testing.T) {
	cfg := &types.Config{
		Local: types.LocalConfig{ProjectsRoot: "/nonexistent/path"},
		S3:    types.S3Config{Prefix: "claude-code/"},
	}

	uploader := New(cfg, nil, true)
	_, err := uploader.DiscoverFiles(context.Background())
	if err == nil {
		t.Fatal("expected error for nonexistent projects root, got nil")
	}
}

func TestDiscoverFilesCaseInsensitiveExtension(t *testing.T) {
	tmpDir := t.TempDir()

	projectDir := filepath.Join(tmpDir, "test-project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create files with different case extensions
	files := []string{
		"lowercase.jsonl",
		"uppercase.JSONL",
		"mixedcase.JsonL",
	}

	for _, fname := range files {
		fpath := filepath.Join(projectDir, fname)
		if err := os.WriteFile(fpath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &types.Config{
		Local: types.LocalConfig{ProjectsRoot: tmpDir},
		S3:    types.S3Config{Prefix: ""},
	}

	uploader := New(cfg, nil, true)
	discovered, err := uploader.DiscoverFiles(context.Background())
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}

	// All three files should be discovered (case-insensitive extension matching)
	if len(discovered) != 3 {
		t.Errorf("expected 3 files, got %d", len(discovered))
	}
}

// TestUpload validates the upload logic with skip behavior.
// Note: This test focuses on the skip logic and result aggregation.
// Actual S3 upload testing would require integration tests with a mock S3 server.
func TestUpload_SkipLogic(t *testing.T) {
	// Test that files marked as ShouldSkip are properly counted
	files := []FileUpload{
		{
			LocalPath:  "/fake/path1.jsonl",
			S3Key:      "project/file1.jsonl",
			Size:       100,
			ProjectDir: "project",
			ShouldSkip: true,
			SkipReason: "identical",
		},
		{
			LocalPath:  "/fake/path2.jsonl",
			S3Key:      "project/file2.jsonl",
			Size:       200,
			ProjectDir: "project",
			ShouldSkip: true,
			SkipReason: "identical",
		},
	}

	cfg := &types.Config{
		S3: types.S3Config{Bucket: "test-bucket"},
	}
	uploader := New(cfg, nil, true)

	// All files are marked to skip, so no actual upload should be attempted
	result, err := uploader.Upload(context.Background(), files)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if result.Uploaded != 0 {
		t.Errorf("expected 0 files uploaded, got %d", result.Uploaded)
	}
	if result.Skipped != 2 {
		t.Errorf("expected 2 files skipped, got %d", result.Skipped)
	}
	if result.UploadedBytes != 0 {
		t.Errorf("expected 0 bytes uploaded, got %d", result.UploadedBytes)
	}
}

func TestUpload_Empty(t *testing.T) {
	cfg := &types.Config{
		S3: types.S3Config{Bucket: "test-bucket"},
	}
	uploader := New(cfg, nil, true)

	result, err := uploader.Upload(context.Background(), []FileUpload{})
	if err != nil {
		t.Fatalf("Upload of empty list failed: %v", err)
	}

	if result.Uploaded != 0 {
		t.Errorf("expected 0 files uploaded, got %d", result.Uploaded)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 files skipped, got %d", result.Skipped)
	}
}

func TestUpload_ContextCancelled(t *testing.T) {
	cfg := &types.Config{
		S3: types.S3Config{Bucket: "test-bucket"},
	}
	uploader := New(cfg, nil, true)

	files := []FileUpload{
		{
			LocalPath:  "/fake/path.jsonl",
			S3Key:      "test.jsonl",
			Size:       4,
			ProjectDir: "test-project",
			ShouldSkip: false,
		},
	}

	// Cancel context before upload
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uploader.Upload(ctx, files)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

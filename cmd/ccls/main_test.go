package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListCommand(t *testing.T) {
	// Create temporary test environment
	tmpDir := t.TempDir()

	// Create test projects structure
	projectsRoot := filepath.Join(tmpDir, "projects")
	project1 := filepath.Join(projectsRoot, "project1")
	project2 := filepath.Join(projectsRoot, "project2")
	project3 := filepath.Join(projectsRoot, "empty-project")

	if err := os.MkdirAll(project1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project3, 0755); err != nil {
		t.Fatal(err)
	}

	// Create JSONL files
	createFile(t, filepath.Join(project1, "session1.jsonl"))
	createFile(t, filepath.Join(project1, "session2.jsonl"))
	createFile(t, filepath.Join(project2, "session1.jsonl"))

	// Create config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `local:
  projects_root: ` + projectsRoot + `

s3:
  bucket: test-bucket
  region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set command-line args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"ccls", "--config", configPath, "list"}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute command
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	err := rootCmd.Execute()

	// Restore stdout
	if err := w.Close(); err != nil {
		t.Logf("failed to close pipe writer: %v", err)
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Read captured output
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Verify output contains expected projects
	if !strings.Contains(outputStr, "empty-project: 0 JSONL") {
		t.Errorf("expected output to contain 'empty-project: 0 JSONL', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "project1: 2 JSONL") {
		t.Errorf("expected output to contain 'project1: 2 JSONL', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "project2: 1 JSONL file") {
		t.Errorf("expected output to contain 'project2: 1 JSONL file', got: %s", outputStr)
	}
}

func TestListCommandNoProjects(t *testing.T) {
	// Create temporary test environment with empty projects directory
	tmpDir := t.TempDir()
	projectsRoot := filepath.Join(tmpDir, "projects")

	if err := os.MkdirAll(projectsRoot, 0755); err != nil {
		t.Fatal(err)
	}

	// Create config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `local:
  projects_root: ` + projectsRoot + `

s3:
  bucket: test-bucket
  region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set config path flag
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"ccls", "--config", configPath, "list"}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute command
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	err := rootCmd.Execute()

	// Restore stdout
	if err := w.Close(); err != nil {
		t.Logf("failed to close pipe writer: %v", err)
	}
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("list command failed: %v", err)
	}

	// Read captured output
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Verify output contains "no projects found"
	if !strings.Contains(outputStr, "(no projects found)") {
		t.Errorf("expected output to contain '(no projects found)', got: %s", outputStr)
	}
}

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

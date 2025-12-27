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

	// Verify output contains expected projects in table format
	if !strings.Contains(outputStr, "Local Projects") {
		t.Errorf("expected output to contain 'Local Projects', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "empty-project") {
		t.Errorf("expected output to contain 'empty-project', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "project1") {
		t.Errorf("expected output to contain 'project1', got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "project2") {
		t.Errorf("expected output to contain 'project2', got: %s", outputStr)
	}
	// Verify the counts appear in the table
	lines := strings.Split(outputStr, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "project1") && strings.Contains(line, "2") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected project1 with count 2 in output, got: %s", outputStr)
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

	// Verify output contains "No local projects found."
	if !strings.Contains(outputStr, "No local projects found.") {
		t.Errorf("expected output to contain 'No local projects found.', got: %s", outputStr)
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

func TestLoadConfigAutoCreation(t *testing.T) {
	tmpDir := t.TempDir()
	testConfigPath := filepath.Join(tmpDir, ".ccls", "config.yaml")

	oldConfigPath := configPath
	oldDefaultConfigPath := defaultConfigPath
	oldExitFunc := exitFunc
	oldStdout := os.Stdout
	defer func() {
		configPath = oldConfigPath
		defaultConfigPath = oldDefaultConfigPath
		exitFunc = oldExitFunc
		os.Stdout = oldStdout
	}()

	configPath = testConfigPath
	defaultConfigPath = testConfigPath

	r, w, _ := os.Pipe()
	os.Stdout = w

	exitCalled := false
	exitCode := -1
	exitFunc = func(code int) {
		exitCalled = true
		exitCode = code
	}

	_, err := loadConfig()

	if err := w.Close(); err != nil {
		t.Logf("failed to close pipe writer: %v", err)
	}
	os.Stdout = oldStdout

	output := make([]byte, 2048)
	r.Read(output)

	if !exitCalled {
		t.Error("expected exitFunc to be called after creating starter config")
	}

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if err != nil && !strings.Contains(err.Error(), "config file not found") {
		t.Fatalf("loadConfig() unexpected error = %v", err)
	}

	if _, err := os.Stat(testConfigPath); os.IsNotExist(err) {
		t.Error("expected config file to be created, but it does not exist")
	}

	content, err := os.ReadFile(testConfigPath)
	if err != nil {
		t.Fatalf("failed to read created config: %v", err)
	}

	if !strings.Contains(string(content), "YOUR-BUCKET-NAME") {
		t.Error("created config missing expected placeholder content")
	}
}

func TestLoadConfigCustomPathNoAutoCreation(t *testing.T) {
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom-config.yaml")

	defaultConfigPath = filepath.Join(tmpDir, ".ccls", "config.yaml")

	oldConfigPath := configPath
	configPath = customPath
	defer func() { configPath = oldConfigPath }()

	_, err := loadConfig()

	if err == nil {
		t.Error("loadConfig() error = nil, want error for missing custom config")
	}

	if !strings.Contains(err.Error(), "config file not found") {
		t.Errorf("loadConfig() error = %q, want error containing 'config file not found'", err.Error())
	}

	if _, err := os.Stat(customPath); !os.IsNotExist(err) {
		t.Error("custom config path should not be auto-created")
	}
}

func TestPrintWelcomeMessage(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	configPath := "/test/path/config.yaml"
	printWelcomeMessage(configPath)

	if err := w.Close(); err != nil {
		t.Logf("failed to close pipe writer: %v", err)
	}
	os.Stdout = oldStdout

	output := make([]byte, 2048)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	expectedPhrases := []string{
		"Welcome to ccls!",
		configPath,
		"s3.bucket",
		"s3.region",
		"auth.profile",
		"ccls doctor",
		"ccls list",
		"ccls upload",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(outputStr, phrase) {
			t.Errorf("welcome message missing expected phrase: %q", phrase)
		}
	}
}

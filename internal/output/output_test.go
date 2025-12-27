package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/13rac1/ccls/internal/types"
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
				{Name: "project-a", JSONLCount: 5},
				{Name: "project-b", JSONLCount: 12},
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
				{Name: "my-project", JSONLCount: 3},
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
				{Name: "empty-project", JSONLCount: 0},
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
				{Name: "big-project", JSONLCount: 999},
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
		{Name: "test-project", JSONLCount: 5},
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
		{Name: "test", JSONLCount: 1},
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

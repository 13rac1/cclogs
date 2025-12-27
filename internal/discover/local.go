// Package discover handles discovery of local and remote Claude Code projects.
// It scans the local filesystem for project directories and .jsonl files,
// and queries S3 for remote projects and object counts.
package discover

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/13rac1/ccls/internal/types"
)

// DiscoverLocal discovers all local Claude Code projects and counts their .jsonl files.
// It scans the projectsRoot directory, treating each immediate child directory as a project,
// and recursively counts .jsonl files within each project.
//
// Returns an error if projectsRoot doesn't exist, is not a directory, or is not readable.
// Individual project read errors are logged but don't fail the entire operation.
func DiscoverLocal(projectsRoot string) ([]types.Project, error) {
	// Verify projects root exists and is a directory
	info, err := os.Stat(projectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("projects root does not exist: %s", projectsRoot)
		}
		return nil, fmt.Errorf("accessing projects root %s: %w", projectsRoot, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("projects root is not a directory: %s", projectsRoot)
	}

	// Read immediate children of projects root
	entries, err := os.ReadDir(projectsRoot)
	if err != nil {
		return nil, fmt.Errorf("reading projects root %s: %w", projectsRoot, err)
	}

	var projects []types.Project

	// Process each directory as a project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		projectPath := filepath.Join(projectsRoot, projectName)

		count, err := countJSONLFiles(projectPath)
		if err != nil {
			// Log warning but continue with other projects
			fmt.Fprintf(os.Stderr, "Warning: failed to count JSONL files in project %s: %v\n", projectName, err)
			continue
		}

		projects = append(projects, types.Project{
			Name:       projectName,
			LocalPath:  projectPath,
			LocalCount: count,
		})
	}

	// Sort by name for deterministic output
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// countJSONLFiles recursively counts .jsonl files in the given directory.
func countJSONLFiles(root string) (int, error) {
	count := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
			count++
		}

		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("walking directory %s: %w", root, err)
	}

	return count, nil
}

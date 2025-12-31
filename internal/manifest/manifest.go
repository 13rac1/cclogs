// Package manifest provides types and functions for tracking uploaded file metadata.
// The manifest enables efficient deduplication by recording source file modification times,
// allowing the uploader to skip files that haven't changed even when redaction alters content size.
package manifest

import (
	"strings"
	"time"
)

// Manifest tracks uploaded file metadata to enable efficient deduplication.
// It records source file modification times, not uploaded content size.
type Manifest struct {
	Version int                  `json:"version"`
	Files   map[string]FileEntry `json:"files"`
}

// FileEntry records metadata about an uploaded file.
type FileEntry struct {
	Mtime time.Time `json:"mtime"` // Source file modification time (UTC)
	Size  int64     `json:"size"`  // Source file size (for reference only)
}

// New creates an empty manifest with version 1.
func New() *Manifest {
	return &Manifest{
		Version: 1,
		Files:   make(map[string]FileEntry),
	}
}

// CountByProject groups manifest entries by project and returns counts.
// Project is extracted from S3 key: prefix/project/file.jsonl → project
func (m *Manifest) CountByProject(prefix string) map[string]int {
	counts := make(map[string]int)
	for key := range m.Files {
		// Strip prefix, extract first path component as project
		rel := strings.TrimPrefix(key, prefix)
		rel = strings.TrimPrefix(rel, "/")
		parts := strings.SplitN(rel, "/", 2)
		if len(parts) > 0 && parts[0] != "" {
			counts[parts[0]]++
		}
	}
	return counts
}

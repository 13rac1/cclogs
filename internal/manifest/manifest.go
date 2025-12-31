// Package manifest provides types and functions for tracking uploaded file metadata.
// The manifest enables efficient deduplication by recording source file modification times,
// allowing the uploader to skip files that haven't changed even when redaction alters content size.
package manifest

import "time"

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

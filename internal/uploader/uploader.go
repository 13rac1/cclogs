package uploader

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/13rac1/ccls/internal/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FileUpload represents a file to be uploaded to S3.
type FileUpload struct {
	LocalPath  string // Full path to local file
	S3Key      string // Destination S3 key
	Size       int64  // File size in bytes
	ProjectDir string // Project directory name
	ShouldSkip bool   // True if file exists remotely and is identical
	SkipReason string // Reason for skipping (e.g., "identical")
}

// Uploader orchestrates file uploads to S3.
type Uploader struct {
	cfg    *types.Config
	client *s3.Client
}

// New creates a new Uploader with the given configuration and S3 client.
func New(cfg *types.Config, client *s3.Client) *Uploader {
	return &Uploader{
		cfg:    cfg,
		client: client,
	}
}

// DiscoverFiles finds all .jsonl files across all local projects.
// It scans each immediate child directory under projects_root,
// recursively finds all .jsonl files, and computes their S3 keys.
func (u *Uploader) DiscoverFiles(ctx context.Context) ([]FileUpload, error) {
	projectsRoot := u.cfg.Local.ProjectsRoot

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

	var uploads []FileUpload

	// Process each directory as a project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := entry.Name()
		projectPath := filepath.Join(projectsRoot, projectDir)

		// Find all .jsonl files in this project
		projectUploads, err := u.discoverProjectFiles(projectPath, projectDir)
		if err != nil {
			// Log warning but continue with other projects
			fmt.Fprintf(os.Stderr, "Warning: failed to discover files in project %s: %v\n", projectDir, err)
			continue
		}

		uploads = append(uploads, projectUploads...)
	}

	// Check each file against remote to determine if upload is needed
	// Skip remote checking if client is nil (for tests)
	if u.client != nil {
		for i := range uploads {
			shouldUpload, err := ShouldUpload(ctx, u.client, u.cfg.S3.Bucket, uploads[i].S3Key, uploads[i].Size)
			if err != nil {
				// Log warning but continue - default to upload on error
				fmt.Fprintf(os.Stderr, "Warning: failed to check remote file %s: %v\n", uploads[i].S3Key, err)
				uploads[i].ShouldSkip = false
			} else {
				uploads[i].ShouldSkip = !shouldUpload
				if uploads[i].ShouldSkip {
					uploads[i].SkipReason = "identical"
				}
			}
		}
	}

	return uploads, nil
}

// discoverProjectFiles finds all .jsonl files within a single project directory.
func (u *Uploader) discoverProjectFiles(projectPath, projectDir string) ([]FileUpload, error) {
	var uploads []FileUpload

	err := filepath.WalkDir(projectPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Only process .jsonl files
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
			return nil
		}

		// Get file size
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("getting file info for %s: %w", path, err)
		}

		// Compute relative path from project directory
		relPath, err := filepath.Rel(projectPath, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}

		// Compute S3 key
		s3Key := ComputeS3Key(u.cfg.S3.Prefix, projectDir, relPath)

		upload := FileUpload{
			LocalPath:  path,
			S3Key:      s3Key,
			Size:       info.Size(),
			ProjectDir: projectDir,
		}

		uploads = append(uploads, upload)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory %s: %w", projectPath, err)
	}

	return uploads, nil
}

// ComputeS3Key generates the S3 key for a local file.
// Format: <prefix>/<project-dir>/<relative-path>
// The prefix is normalized to have a trailing slash if non-empty.
// Path separators are converted to forward slashes for S3 compatibility.
func ComputeS3Key(prefix, projectDir, relPath string) string {
	// Ensure prefix has trailing slash if non-empty
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	// Convert backslashes to forward slashes (handles Windows paths)
	// filepath.ToSlash only converts the OS-specific separator, so we need
	// to explicitly handle backslashes on Unix systems
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	projectDir = strings.ReplaceAll(projectDir, "\\", "/")
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	// Combine with forward slashes
	var key string
	if prefix != "" {
		key = prefix + projectDir + "/" + relPath
	} else {
		key = projectDir + "/" + relPath
	}

	return key
}

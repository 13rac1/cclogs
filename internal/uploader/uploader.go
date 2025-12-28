// Package uploader handles discovery and upload of JSONL files to S3-compatible storage.
// It discovers all .jsonl files across local projects, computes their S3 keys,
// checks for existing remote files, and uploads new or modified files using multipart uploads.
package uploader

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/13rac1/cclogs/internal/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
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

	// Check files against remote to determine if upload is needed
	// Skip remote checking if client is nil (for tests)
	if u.client != nil {
		// Group files by project to minimize API calls
		projectFiles := make(map[string][]int)
		for i := range uploads {
			projectFiles[uploads[i].ProjectDir] = append(projectFiles[uploads[i].ProjectDir], i)
		}

		// For each project, fetch all remote files once and compare
		for projectDir, indices := range projectFiles {
			// Compute prefix for this project's remote files
			prefix := u.cfg.S3.Prefix
			if prefix != "" && !strings.HasSuffix(prefix, "/") {
				prefix += "/"
			}
			projectPrefix := prefix + projectDir + "/"

			// Fetch all remote files for this project in one API call
			remoteFiles, err := ListRemoteFiles(ctx, u.client, u.cfg.S3.Bucket, projectPrefix)
			if err != nil {
				// Log warning but continue - default to upload on error
				fmt.Fprintf(os.Stderr, "Warning: failed to list remote files for project %s: %v\n", projectDir, err)
				for _, i := range indices {
					uploads[i].ShouldSkip = false
				}
				continue
			}

			// Compare each local file against the remote files map
			for _, i := range indices {
				remoteSize, exists := remoteFiles[uploads[i].S3Key]
				if !exists || remoteSize != uploads[i].Size {
					uploads[i].ShouldSkip = false
				} else {
					uploads[i].ShouldSkip = true
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

// UploadResult contains summary statistics from an upload operation.
type UploadResult struct {
	Uploaded      int   // Number of files uploaded
	Skipped       int   // Number of files skipped
	UploadedBytes int64 // Total bytes uploaded
}

// Upload uploads the provided files to S3, respecting the ShouldSkip field.
// Files marked with ShouldSkip=true are skipped and reported as such.
// Returns summary statistics and any error encountered.
func (u *Uploader) Upload(ctx context.Context, files []FileUpload) (*UploadResult, error) {
	if len(files) == 0 {
		return &UploadResult{}, nil
	}

	// Configure uploader with multipart settings
	uploader := manager.NewUploader(u.client, func(mu *manager.Uploader) {
		mu.Concurrency = 5            // 5 concurrent parts per file
		mu.PartSize = 5 * 1024 * 1024 // 5MB parts
	})

	result := &UploadResult{}
	totalFiles := len(files)

	for i, file := range files {
		fileNum := i + 1

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("upload cancelled: %w", err)
		}

		// Skip files marked as identical
		if file.ShouldSkip {
			fmt.Printf("[%d/%d] Skipping %s (%s)\n", fileNum, totalFiles, file.LocalPath, file.SkipReason)
			result.Skipped++
			continue
		}

		// Upload the file
		fmt.Printf("[%d/%d] Uploading %s (%s)\n", fileNum, totalFiles, file.LocalPath, formatSize(file.Size))

		if err := u.uploadFile(ctx, uploader, file); err != nil {
			return result, fmt.Errorf("uploading %s: %w", file.LocalPath, err)
		}

		result.Uploaded++
		result.UploadedBytes += file.Size
	}

	// Print summary
	fmt.Printf("\nUpload complete: %d uploaded (%s), %d skipped\n",
		result.Uploaded, formatSize(result.UploadedBytes), result.Skipped)

	return result, nil
}

// uploadFile uploads a single file to S3 using the configured uploader.
func (u *Uploader) uploadFile(ctx context.Context, uploader *manager.Uploader, file FileUpload) error {
	// Open the local file
	f, err := os.Open(file.LocalPath)
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			// Log close error but don't override upload error
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", file.LocalPath, closeErr)
		}
	}()

	// Upload to S3
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.cfg.S3.Bucket),
		Key:    aws.String(file.S3Key),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("s3 upload: %w", err)
	}

	return nil
}

// formatSize formats a byte count as a human-readable string.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

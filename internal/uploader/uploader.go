// Package uploader handles discovery and upload of JSONL files to S3-compatible storage.
// It discovers all .jsonl files across local projects, computes their S3 keys,
// checks for existing remote files, and uploads new or modified files using multipart uploads.
package uploader

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/13rac1/cclogs/internal/manifest"
	"github.com/13rac1/cclogs/internal/redactor"
	"github.com/13rac1/cclogs/internal/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// FileUpload represents a file to be uploaded to S3.
type FileUpload struct {
	LocalPath  string    // Full path to local file
	S3Key      string    // Destination S3 key
	Size       int64     // File size in bytes
	ModTime    time.Time // File modification time
	ProjectDir string    // Project directory name
	ShouldSkip bool      // True if file exists remotely and is identical
	SkipReason string    // Reason for skipping (e.g., "unchanged")
}

// Uploader orchestrates file uploads to S3.
type Uploader struct {
	cfg      *types.Config
	client   *s3.Client
	noRedact bool
}

// New creates a new Uploader with the given configuration and S3 client.
func New(cfg *types.Config, client *s3.Client, noRedact bool) *Uploader {
	return &Uploader{
		cfg:      cfg,
		client:   client,
		noRedact: noRedact,
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

	// Check files against manifest to determine if upload is needed
	// Skip manifest checking if client is nil (for tests)
	if u.client != nil {
		// Compute manifest key
		manifestKey := u.cfg.S3.Prefix
		if manifestKey != "" && !strings.HasSuffix(manifestKey, "/") {
			manifestKey += "/"
		}
		manifestKey += ".manifest.json"

		// Load manifest from S3
		m, err := manifest.Load(ctx, u.client, u.cfg.S3.Bucket, manifestKey)
		if err != nil {
			// Log warning but continue - treat as first run
			fmt.Fprintf(os.Stderr, "Warning: failed to load manifest (treating as first run): %v\n", err)
			m = manifest.New()
		}

		// Compare each local file against manifest
		for i := range uploads {
			entry, exists := m.Files[uploads[i].S3Key]
			if !exists {
				// File not in manifest - needs upload
				uploads[i].ShouldSkip = false
				continue
			}

			// Compare modification times (truncate to seconds for filesystem compatibility)
			localMtime := uploads[i].ModTime.Truncate(time.Second)
			remoteMtime := entry.Mtime.Truncate(time.Second)

			if localMtime.Equal(remoteMtime) {
				uploads[i].ShouldSkip = true
				uploads[i].SkipReason = "unchanged"
			} else {
				uploads[i].ShouldSkip = false
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

		// Get file info
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
			ModTime:    info.ModTime().UTC(),
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
	Uploaded       int              // Number of files uploaded
	Skipped        int              // Number of files skipped
	UploadedBytes  int64            // Total bytes uploaded
	RedactionStats *redactor.Stats  // Aggregated redaction statistics
}

// Upload uploads the provided files to S3, respecting the ShouldSkip field.
// Files marked with ShouldSkip=true are skipped and reported as such.
// Returns summary statistics and any error encountered.
func (u *Uploader) Upload(ctx context.Context, files []FileUpload) (*UploadResult, error) {
	if len(files) == 0 {
		return &UploadResult{}, nil
	}

	// Early return for tests with nil client - just count skips
	if u.client == nil {
		result := &UploadResult{}
		for _, file := range files {
			// Check context cancellation
			if err := ctx.Err(); err != nil {
				return result, fmt.Errorf("upload cancelled: %w", err)
			}

			if file.ShouldSkip {
				result.Skipped++
			} else {
				result.Uploaded++
				result.UploadedBytes += file.Size
			}
		}
		return result, nil
	}

	// Compute manifest key
	manifestKey := u.cfg.S3.Prefix
	if manifestKey != "" && !strings.HasSuffix(manifestKey, "/") {
		manifestKey += "/"
	}
	manifestKey += ".manifest.json"

	// Load existing manifest
	m, err := manifest.Load(ctx, u.client, u.cfg.S3.Bucket, manifestKey)
	if err != nil {
		// Log warning but continue with empty manifest
		fmt.Fprintf(os.Stderr, "Warning: failed to load manifest for update: %v\n", err)
		m = manifest.New()
	}

	// Configure uploader with multipart settings
	uploader := manager.NewUploader(u.client, func(mu *manager.Uploader) {
		mu.Concurrency = 5            // 5 concurrent parts per file
		mu.PartSize = 5 * 1024 * 1024 // 5MB parts
	})

	result := &UploadResult{
		RedactionStats: redactor.NewStats(),
	}
	totalFiles := len(files)

	for i, file := range files {
		fileNum := i + 1

		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return result, fmt.Errorf("upload cancelled: %w", err)
		}

		// Skip files marked as unchanged
		if file.ShouldSkip {
			fmt.Printf("[%d/%d] Skipping %s (%s)\n", fileNum, totalFiles, file.LocalPath, file.SkipReason)
			result.Skipped++
			continue
		}

		// Upload the file
		fmt.Printf("[%d/%d] Uploading %s (%s)", fileNum, totalFiles, file.LocalPath, formatSize(file.Size))

		fileStats, err := u.uploadFile(ctx, uploader, file)
		if err != nil {
			fmt.Println() // Complete the line
			return result, fmt.Errorf("uploading %s: %w", file.LocalPath, err)
		}

		// Display per-file redaction stats
		if fileStats != nil && fileStats.TotalMatches > 0 {
			fmt.Printf(" → %s (%.1f%% redacted, %d matches)\n",
				formatSize(fileStats.RedactedBytes),
				fileStats.PercentReduction(),
				fileStats.TotalMatches)
			result.RedactionStats.Add(fileStats)
		} else {
			fmt.Println() // No redaction to report
		}

		// Update manifest entry after successful upload
		m.Files[file.S3Key] = manifest.FileEntry{
			Mtime: file.ModTime,
			Size:  file.Size,
		}

		result.Uploaded++
		result.UploadedBytes += file.Size
	}

	// Save updated manifest if any files were uploaded
	if result.Uploaded > 0 {
		if err := manifest.Save(ctx, u.client, u.cfg.S3.Bucket, manifestKey, m); err != nil {
			// Log warning but don't fail - files were successfully uploaded
			fmt.Fprintf(os.Stderr, "Warning: failed to save manifest (uploads succeeded): %v\n", err)
		}
	}

	// Print summary
	fmt.Printf("\nUpload complete: %d uploaded (%s), %d skipped\n",
		result.Uploaded, formatSize(result.UploadedBytes), result.Skipped)

	// Print redaction summary if any matches were found
	if result.RedactionStats != nil && result.RedactionStats.TotalMatches > 0 {
		fmt.Printf("\nRedaction summary:\n")
		fmt.Printf("  Total: %s → %s (%.1f%% reduction)\n",
			formatSize(result.RedactionStats.OriginalBytes),
			formatSize(result.RedactionStats.RedactedBytes),
			result.RedactionStats.PercentReduction())
		fmt.Printf("  Matches: %d total\n", result.RedactionStats.TotalMatches)

		// Print per-pattern breakdown
		for _, pc := range result.RedactionStats.PatternSummary() {
			fmt.Printf("    %s: %d\n", pc.Pattern, pc.Count)
		}
	}

	return result, nil
}

// uploadFile uploads a single file to S3 using the configured uploader.
// Returns redaction stats if redaction was enabled, nil otherwise.
func (u *Uploader) uploadFile(ctx context.Context, uploader *manager.Uploader, file FileUpload) (*redactor.Stats, error) {
	// Open the local file
	f, err := os.Open(file.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			// Log close error but don't override upload error
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", file.LocalPath, closeErr)
		}
	}()

	// Wrap with redactor unless disabled
	var body io.Reader = f
	var statsCh <-chan *redactor.Stats
	if !u.noRedact {
		body, statsCh = redactor.StreamRedactWithStats(f)
	}

	// Upload to S3
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(u.cfg.S3.Bucket),
		Key:    aws.String(file.S3Key),
		Body:   body,
	})
	if err != nil {
		return nil, fmt.Errorf("s3 upload: %w", err)
	}

	// Wait for stats after upload completes
	if statsCh != nil {
		stats := <-statsCh
		return stats, nil
	}

	return nil, nil
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

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/13rac1/ccls/internal/config"
	"github.com/13rac1/ccls/internal/discover"
	"github.com/13rac1/ccls/internal/doctor"
	"github.com/13rac1/ccls/internal/output"
	"github.com/13rac1/ccls/internal/types"
	"github.com/13rac1/ccls/internal/uploader"
	"github.com/spf13/cobra"
)

var (
	configPath        string
	defaultConfigPath string
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "ccls",
	Short: "Claude Code Log Shipper - upload session logs to S3",
	Long: `ccls discovers Claude Code session logs (*.jsonl files) from ~/.claude/projects/
and uploads them to S3-compatible storage for backup and archival.`,
}

var (
	jsonOutput bool
	dryRun     bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local and remote projects with JSONL counts",
	Long: `Lists all Claude Code projects both locally and in remote storage,
showing the count of .jsonl files for each project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		localProjects, err := discover.DiscoverLocal(cfg.Local.ProjectsRoot)
		if err != nil {
			return fmt.Errorf("discovering local projects: %w", err)
		}

		// Discover remote projects if S3 is configured
		// Gracefully skip remote discovery if S3 is not accessible
		var remoteProjects []types.Project
		if cfg.S3.Bucket != "" {
			s3Client, err := config.NewS3Client(cmd.Context(), cfg)
			if err == nil {
				remoteProjects, err = discover.DiscoverRemote(cmd.Context(), s3Client, cfg.S3.Bucket, cfg.S3.Prefix)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to discover remote projects: %v\n", err)
				}
			}
		}

		// Merge local and remote projects
		merged := mergeProjects(localProjects, remoteProjects)

		if jsonOutput {
			if err := output.PrintJSON(merged, cfg); err != nil {
				return fmt.Errorf("printing JSON output: %w", err)
			}
		} else {
			output.PrintProjects(merged)
		}
		return nil
	},
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload local JSONL logs to remote storage",
	Long: `Discovers all .jsonl files in local Claude Code projects and uploads them
to S3-compatible storage. Safe to run repeatedly from multiple machines.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		ctx := cmd.Context()

		// Create S3 client
		client, err := config.NewS3Client(ctx, cfg)
		if err != nil {
			return fmt.Errorf("creating S3 client: %w", err)
		}

		// Create uploader
		u := uploader.New(cfg, client)

		// Discover files
		files, err := u.DiscoverFiles(ctx)
		if err != nil {
			return fmt.Errorf("discovering files: %w", err)
		}

		// In dry-run mode, just print what would be uploaded
		if dryRun {
			printDryRun(files)
			return nil
		}

		// Actual upload happens in Phase 10
		fmt.Println("Upload not yet implemented (Phase 10)")
		return nil
	},
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate configuration and connectivity",
	Long: `Checks that the configuration is valid, local projects root exists,
and remote S3 connectivity works.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		allPassed := doctor.RunChecks(cfg, configPath, false)
		if !allPassed {
			exitFunc(1)
		}
		return nil
	},
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get home directory: %v\n", err)
		homeDir = "~"
	}
	defaultConfigPath = filepath.Join(homeDir, ".ccls", "config.yaml")

	rootCmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")

	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	uploadCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned uploads without performing them")

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(doctorCmd)
}

var exitFunc = os.Exit

func loadConfig() (*types.Config, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			isDefaultPath := configPath == defaultConfigPath
			if isDefaultPath {
				if err := config.CreateStarterConfig(configPath); err != nil {
					return nil, fmt.Errorf("creating starter config: %w", err)
				}
				printWelcomeMessage(configPath)
				exitFunc(0)
			}
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("loading config from %s: %w", configPath, err)
	}
	return cfg, nil
}

func printWelcomeMessage(configPath string) {
	fmt.Println("Welcome to ccls!")
	fmt.Println()
	fmt.Printf("A starter configuration file has been created at:\n")
	fmt.Printf("  %s\n", configPath)
	fmt.Println()
	fmt.Println("Please edit this file and configure:")
	fmt.Println("  1. s3.bucket - Your S3 bucket name")
	fmt.Println("  2. s3.region - Your AWS region")
	fmt.Println("  3. auth.profile - Your AWS profile (or use static credentials)")
	fmt.Println()
	fmt.Println("For S3-compatible providers (Backblaze B2, MinIO, etc.):")
	fmt.Println("  - Set s3.endpoint to your provider's endpoint URL")
	fmt.Println("  - Set s3.force_path_style: true if required")
	fmt.Println()
	fmt.Println("After configuration, run:")
	fmt.Println("  ccls doctor   # Validate configuration")
	fmt.Println("  ccls list     # List local and remote projects")
	fmt.Println("  ccls upload   # Upload local JSONL files")
}

// mergeProjects combines local and remote projects into a single list.
// Projects with the same name are merged, combining their local and remote counts.
func mergeProjects(local, remote []types.Project) []types.Project {
	projectMap := make(map[string]*types.Project)

	// Add local projects to map
	for _, p := range local {
		projectMap[p.Name] = &types.Project{
			Name:       p.Name,
			LocalPath:  p.LocalPath,
			LocalCount: p.LocalCount,
		}
	}

	// Merge remote projects
	for _, p := range remote {
		if existing, ok := projectMap[p.Name]; ok {
			// Project exists locally and remotely
			existing.RemoteCount = p.RemoteCount
			existing.RemotePath = p.RemotePath
		} else {
			// Remote-only project
			projectMap[p.Name] = &types.Project{
				Name:        p.Name,
				RemotePath:  p.RemotePath,
				RemoteCount: p.RemoteCount,
			}
		}
	}

	// Convert map to sorted slice
	var merged []types.Project
	for _, p := range projectMap {
		merged = append(merged, *p)
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Name < merged[j].Name
	})

	return merged
}

// printDryRun displays planned uploads in a human-readable format.
func printDryRun(files []uploader.FileUpload) {
	if len(files) == 0 {
		fmt.Println("No files to upload")
		return
	}

	fmt.Println("Planned uploads (dry-run mode):")
	fmt.Println()

	// Group files by project
	projectFiles := make(map[string][]uploader.FileUpload)
	for _, f := range files {
		projectFiles[f.ProjectDir] = append(projectFiles[f.ProjectDir], f)
	}

	// Sort project names for deterministic output
	var projectNames []string
	for name := range projectFiles {
		projectNames = append(projectNames, name)
	}
	sort.Strings(projectNames)

	// Print files grouped by project
	var totalFiles int
	var totalSize int64

	for _, projectName := range projectNames {
		fmt.Printf("Project: %s\n", projectName)
		files := projectFiles[projectName]

		// Sort files within project by local path
		sort.Slice(files, func(i, j int) bool {
			return files[i].LocalPath < files[j].LocalPath
		})

		for _, f := range files {
			// Compute relative path from project directory for display
			relPath, err := filepath.Rel(filepath.Join(filepath.Dir(f.LocalPath), ".."), f.LocalPath)
			if err != nil {
				relPath = filepath.Base(f.LocalPath)
			}

			fmt.Printf("  %s -> %s (%s)\n", relPath, f.S3Key, formatSize(f.Size))
			totalFiles++
			totalSize += f.Size
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d files (%s)\n", totalFiles, formatSize(totalSize))
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

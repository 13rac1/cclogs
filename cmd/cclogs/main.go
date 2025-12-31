package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/13rac1/cclogs/internal/config"
	"github.com/13rac1/cclogs/internal/discover"
	"github.com/13rac1/cclogs/internal/doctor"
	"github.com/13rac1/cclogs/internal/manifest"
	"github.com/13rac1/cclogs/internal/output"
	"github.com/13rac1/cclogs/internal/types"
	"github.com/13rac1/cclogs/internal/uploader"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
	Use:     "cclogs",
	Short:   "Claude Code Log Shipper - upload session logs to S3",
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	Long: `cclogs discovers Claude Code session logs (*.jsonl files) from ~/.claude/projects/
and uploads them to S3-compatible storage for backup and archival.`,
}

var (
	jsonOutput bool
	dryRun     bool
	noRedact   bool
	debug      bool
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

		// Discover remote projects from manifest if S3 is configured
		var remoteProjects []types.Project
		if cfg.S3.Bucket != "" {
			s3Client, err := config.NewS3Client(cmd.Context(), cfg)
			if err == nil {
				manifestKey := computeManifestKey(cfg.S3.Prefix)
				m, err := manifest.Load(cmd.Context(), s3Client, cfg.S3.Bucket, manifestKey)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not load manifest: %v\n", err)
					m = manifest.New()
				}
				remoteProjects = discover.DiscoverFromManifest(m, cfg.S3.Prefix)
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

		// Create S3 client (nil for dry-run)
		var client *s3.Client
		if !dryRun {
			client, err = config.NewS3Client(ctx, cfg)
			if err != nil {
				return fmt.Errorf("creating S3 client: %w", err)
			}
		}

		// Create uploader
		u := uploader.New(cfg, client, noRedact, debug)

		// Discover files
		files, err := u.DiscoverFiles(ctx)
		if err != nil {
			return fmt.Errorf("discovering files: %w", err)
		}

		// In dry-run mode, process files with redaction but don't upload
		if dryRun {
			_, err = u.DryRunProcess(ctx, files)
			if err != nil {
				return fmt.Errorf("processing files: %w", err)
			}
			return nil
		}

		// Perform upload
		_, err = u.Upload(ctx, files)
		if err != nil {
			return fmt.Errorf("uploading files: %w", err)
		}

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
	defaultConfigPath = filepath.Join(homeDir, ".cclogs", "config.yaml")

	rootCmd.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "path to config file")

	listCmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	uploadCmd.Flags().BoolVar(&dryRun, "dry-run", false, "process files with redaction but don't upload (shows stats)")
	uploadCmd.Flags().BoolVar(&noRedact, "no-redact", false, "disable PII/secrets redaction (not recommended)")
	uploadCmd.Flags().BoolVar(&debug, "debug", false, "show before/after for each redaction match")

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
	fmt.Println("Welcome to cclogs!")
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
	fmt.Println("  cclogs doctor   # Validate configuration")
	fmt.Println("  cclogs list     # List local and remote projects")
	fmt.Println("  cclogs upload   # Upload local JSONL files")
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

// computeManifestKey returns the S3 key for the manifest file.
func computeManifestKey(prefix string) string {
	if prefix == "" {
		return ".manifest.json"
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	return prefix + ".manifest.json"
}

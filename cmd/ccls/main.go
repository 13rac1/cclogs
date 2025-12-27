package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/13rac1/ccls/internal/config"
	"github.com/13rac1/ccls/internal/discover"
	"github.com/13rac1/ccls/internal/types"
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

		projects, err := discover.DiscoverLocal(cfg.Local.ProjectsRoot)
		if err != nil {
			return fmt.Errorf("discovering local projects: %w", err)
		}

		fmt.Println("Local Projects:")
		if len(projects) == 0 {
			fmt.Println("  (no projects found)")
			return nil
		}

		for _, p := range projects {
			fileWord := "files"
			if p.JSONLCount == 1 {
				fileWord = "file"
			}
			fmt.Printf("  %s: %d JSONL %s\n", p.Name, p.JSONLCount, fileWord)
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

		// TODO: Implement upload functionality
		fmt.Printf("Config loaded successfully:\n")
		fmt.Printf("  Projects root: %s\n", cfg.Local.ProjectsRoot)
		fmt.Printf("  S3 bucket: %s\n", cfg.S3.Bucket)
		fmt.Printf("  S3 region: %s\n", cfg.S3.Region)
		fmt.Printf("  S3 prefix: %s\n", cfg.S3.Prefix)
		fmt.Println("\nUpload functionality will be implemented in Phase 2")
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

		// TODO: Implement doctor functionality
		fmt.Printf("Config loaded successfully:\n")
		fmt.Printf("  Projects root: %s\n", cfg.Local.ProjectsRoot)
		fmt.Printf("  S3 bucket: %s\n", cfg.S3.Bucket)
		fmt.Printf("  S3 region: %s\n", cfg.S3.Region)
		fmt.Printf("  S3 prefix: %s\n", cfg.S3.Prefix)
		fmt.Println("\nDoctor functionality will be implemented in Phase 2")
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

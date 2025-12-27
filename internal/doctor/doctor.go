package doctor

import (
	"fmt"
	"os"

	"github.com/13rac1/ccls/internal/discover"
	"github.com/13rac1/ccls/internal/types"
)

const (
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorReset = "\033[0m"
)

func checkmark() string {
	return colorGreen + "✓" + colorReset
}

func crossmark() string {
	return colorRed + "✗" + colorReset
}

// RunChecks performs all doctor checks and returns whether all passed.
func RunChecks(cfg *types.Config, configPath string) bool {
	fmt.Println("ccls doctor - Configuration and connectivity check")
	fmt.Println()

	allPassed := true

	// Configuration checks
	fmt.Println("Configuration:")
	fmt.Printf("  %s Config file loaded: %s\n", checkmark(), configPath)

	if cfg.S3.Bucket == "" || cfg.S3.Bucket == "YOUR-BUCKET-NAME" {
		fmt.Printf("  %s S3 bucket not configured (still set to placeholder)\n", crossmark())
		fmt.Printf("    → Edit %s and set s3.bucket\n", configPath)
		allPassed = false
	} else {
		fmt.Printf("  %s S3 bucket configured: %s\n", checkmark(), cfg.S3.Bucket)
	}

	if cfg.S3.Region == "" {
		fmt.Printf("  %s S3 region not configured\n", crossmark())
		fmt.Printf("    → Edit %s and set s3.region\n", configPath)
		allPassed = false
	} else {
		fmt.Printf("  %s S3 region configured: %s\n", checkmark(), cfg.S3.Region)
	}

	if cfg.S3.Prefix == "" {
		fmt.Printf("  %s S3 prefix configured: (empty)\n", checkmark())
	} else {
		fmt.Printf("  %s S3 prefix configured: %s\n", checkmark(), cfg.S3.Prefix)
	}

	fmt.Println()

	// Local filesystem checks
	fmt.Println("Local filesystem:")

	info, err := os.Stat(cfg.Local.ProjectsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("  %s Projects root does not exist: %s\n", crossmark(), cfg.Local.ProjectsRoot)
			fmt.Printf("    → Create the directory or update local.projects_root in config\n")
			fmt.Printf("  %s Cannot read projects root\n", crossmark())
			fmt.Printf("  %s No projects found\n", crossmark())
			allPassed = false
			fmt.Println()
			printSummary(allPassed)
			return allPassed
		}
		fmt.Printf("  %s Cannot access projects root: %s\n", crossmark(), cfg.Local.ProjectsRoot)
		fmt.Printf("    → Error: %v\n", err)
		fmt.Printf("  %s Cannot read projects root\n", crossmark())
		fmt.Printf("  %s No projects found\n", crossmark())
		allPassed = false
		fmt.Println()
		printSummary(allPassed)
		return allPassed
	}

	if !info.IsDir() {
		fmt.Printf("  %s Projects root is not a directory: %s\n", crossmark(), cfg.Local.ProjectsRoot)
		fmt.Printf("    → Ensure local.projects_root points to a directory\n")
		fmt.Printf("  %s Cannot read projects root\n", crossmark())
		fmt.Printf("  %s No projects found\n", crossmark())
		allPassed = false
		fmt.Println()
		printSummary(allPassed)
		return allPassed
	}

	fmt.Printf("  %s Projects root exists: %s\n", checkmark(), cfg.Local.ProjectsRoot)

	// Check if projects root is readable
	entries, err := os.ReadDir(cfg.Local.ProjectsRoot)
	if err != nil {
		fmt.Printf("  %s Projects root is not readable\n", crossmark())
		fmt.Printf("    → Error: %v\n", err)
		fmt.Printf("  %s No projects found\n", crossmark())
		allPassed = false
		fmt.Println()
		printSummary(allPassed)
		return allPassed
	}

	fmt.Printf("  %s Projects root is readable\n", checkmark())

	// Count projects with JSONL files
	projects, err := discover.DiscoverLocal(cfg.Local.ProjectsRoot)
	if err != nil {
		fmt.Printf("  %s Failed to discover projects: %v\n", crossmark(), err)
		allPassed = false
		fmt.Println()
		printSummary(allPassed)
		return allPassed
	}

	totalJSONL := 0
	for _, p := range projects {
		totalJSONL += p.JSONLCount
	}

	if len(projects) == 0 {
		// Check if there are any directories at all
		hasDirectories := false
		for _, entry := range entries {
			if entry.IsDir() {
				hasDirectories = true
				break
			}
		}

		if hasDirectories {
			fmt.Printf("  %s Found %d local projects with 0 JSONL files\n", checkmark(), countDirectories(entries))
		} else {
			fmt.Printf("  %s No projects found (no directories in projects root)\n", checkmark())
		}
	} else {
		fileWord := "files"
		if totalJSONL == 1 {
			fileWord = "file"
		}
		projectWord := "projects"
		if len(projects) == 1 {
			projectWord = "project"
		}
		fmt.Printf("  %s Found %d local %s with %d JSONL %s\n", checkmark(), len(projects), projectWord, totalJSONL, fileWord)
	}

	fmt.Println()
	printSummary(allPassed)
	return allPassed
}

func printSummary(allPassed bool) {
	if allPassed {
		fmt.Println("All checks passed! Ready to use ccls.")
	} else {
		fmt.Println("Some checks failed. Please fix the issues above.")
	}
}

func countDirectories(entries []os.DirEntry) int {
	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}
	return count
}

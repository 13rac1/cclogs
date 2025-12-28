package discover

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/13rac1/cclogs/internal/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DiscoverRemote discovers projects in S3 by listing prefixes.
// Each immediate child prefix under bucket/prefix/ is treated as a project.
// For each project, counts .jsonl files (case-insensitive).
func DiscoverRemote(ctx context.Context, client *s3.Client, bucket, prefix string) ([]types.Project, error) {
	// Ensure prefix ends with / for consistent prefix matching
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}

	// Discover project directories
	projectPrefixes, err := listProjectPrefixes(ctx, client, bucket, prefix)
	if err != nil {
		return nil, fmt.Errorf("list project prefixes: %w", err)
	}

	var projects []types.Project

	// Count JSONL files in each project
	for _, projectPrefix := range projectPrefixes {
		projectName := extractProjectName(projectPrefix, prefix)
		if projectName == "" {
			continue
		}

		count, err := countRemoteJSONLFiles(ctx, client, bucket, projectPrefix)
		if err != nil {
			return nil, fmt.Errorf("count JSONL files in %s: %w", projectName, err)
		}

		projects = append(projects, types.Project{
			Name:        projectName,
			RemotePath:  projectPrefix,
			RemoteCount: count,
		})
	}

	// Sort by name for deterministic output
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// listProjectPrefixes returns all immediate child prefixes under bucket/prefix/.
// Uses pagination to handle large buckets.
func listProjectPrefixes(ctx context.Context, client *s3.Client, bucket, prefix string) ([]string, error) {
	var prefixes []string

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket:    &bucket,
		Prefix:    &prefix,
		Delimiter: strPtr("/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list objects: %w", err)
		}

		for _, cp := range page.CommonPrefixes {
			if cp.Prefix != nil {
				prefixes = append(prefixes, *cp.Prefix)
			}
		}
	}

	return prefixes, nil
}

// countRemoteJSONLFiles counts .jsonl files (case-insensitive) under the given prefix.
// Uses pagination to handle projects with many files.
func countRemoteJSONLFiles(ctx context.Context, client *s3.Client, bucket, prefix string) (int, error) {
	count := 0

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: &bucket,
		Prefix: &prefix,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("list objects: %w", err)
		}

		for _, obj := range page.Contents {
			if obj.Key != nil && strings.HasSuffix(strings.ToLower(*obj.Key), ".jsonl") {
				count++
			}
		}
	}

	return count, nil
}

// extractProjectName extracts the project name from an S3 prefix.
// Given basePrefix="claude-code/" and projectPrefix="claude-code/my-project/",
// returns "my-project".
func extractProjectName(projectPrefix, basePrefix string) string {
	// Remove base prefix
	name := strings.TrimPrefix(projectPrefix, basePrefix)

	// Remove trailing slash
	name = strings.TrimSuffix(name, "/")

	// Use path.Base to get the last component
	return path.Base(name)
}

// strPtr returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}

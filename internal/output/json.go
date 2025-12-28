package output

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/13rac1/cclogs/internal/types"
)

// JSONOutput represents the complete JSON output structure.
type JSONOutput struct {
	GeneratedAt    string          `json:"generatedAt"`
	Config         ConfigInfo      `json:"config"`
	LocalProjects  []LocalProject  `json:"localProjects"`
	RemoteProjects []RemoteProject `json:"remoteProjects"`
}

// ConfigInfo holds configuration details for JSON output.
type ConfigInfo struct {
	Bucket   string `json:"bucket"`
	Prefix   string `json:"prefix"`
	Endpoint string `json:"endpoint,omitempty"`
}

// LocalProject represents a local project in JSON output.
type LocalProject struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	JSONLCount int    `json:"jsonlCount"`
}

// RemoteProject represents a remote project in JSON output.
type RemoteProject struct {
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	JSONLCount int    `json:"jsonlCount"`
}

// PrintJSON formats and prints projects as JSON to stdout.
func PrintJSON(projects []types.Project, cfg *types.Config) error {
	output := JSONOutput{
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
		Config:         buildConfigInfo(cfg),
		LocalProjects:  buildLocalProjects(projects),
		RemoteProjects: buildRemoteProjects(projects),
	}

	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// buildConfigInfo extracts config information for JSON output.
func buildConfigInfo(cfg *types.Config) ConfigInfo {
	return ConfigInfo{
		Bucket:   cfg.S3.Bucket,
		Prefix:   cfg.S3.Prefix,
		Endpoint: cfg.S3.Endpoint,
	}
}

// buildLocalProjects extracts local projects from the merged project list.
func buildLocalProjects(projects []types.Project) []LocalProject {
	local := make([]LocalProject, 0)

	for _, p := range projects {
		if p.LocalCount > 0 {
			local = append(local, LocalProject{
				Name:       p.Name,
				Path:       p.LocalPath,
				JSONLCount: p.LocalCount,
			})
		}
	}

	return local
}

// buildRemoteProjects extracts remote projects from the merged project list.
func buildRemoteProjects(projects []types.Project) []RemoteProject {
	remote := make([]RemoteProject, 0)

	for _, p := range projects {
		if p.RemoteCount > 0 {
			remote = append(remote, RemoteProject{
				Name:       p.Name,
				Prefix:     p.RemotePath,
				JSONLCount: p.RemoteCount,
			})
		}
	}

	return remote
}

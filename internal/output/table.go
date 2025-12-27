package output

import (
	"fmt"
	"os"
	"strconv"

	"github.com/13rac1/ccls/internal/types"
	"github.com/olekukonko/tablewriter"
)

// PrintLocalProjects formats and prints local projects as an ASCII table.
func PrintLocalProjects(projects []types.Project) {
	if len(projects) == 0 {
		fmt.Println("No local projects found.")
		return
	}

	fmt.Println("Local Projects")
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Project", "JSONL Files")

	for _, p := range projects {
		table.Append(p.Name, strconv.Itoa(p.LocalCount))
	}

	table.Render()
}

// PrintProjects formats and prints projects with local and remote counts.
func PrintProjects(projects []types.Project) {
	if len(projects) == 0 {
		fmt.Println("No projects found.")
		return
	}

	fmt.Println("Projects")
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Project", "Local", "Remote", "Status")

	for _, p := range projects {
		local := formatCount(p.LocalCount)
		remote := formatCount(p.RemoteCount)
		status := determineStatus(p.LocalCount, p.RemoteCount)

		table.Append(p.Name, local, remote, status)
	}

	table.Render()
}

// formatCount formats a count for display, using "-" for zero values.
func formatCount(count int) string {
	if count == 0 {
		return "-"
	}
	return strconv.Itoa(count)
}

// determineStatus determines the sync status based on local and remote counts.
func determineStatus(localCount, remoteCount int) string {
	hasLocal := localCount > 0
	hasRemote := remoteCount > 0

	if !hasLocal && !hasRemote {
		return "-"
	}

	if hasLocal && !hasRemote {
		return "Local-only"
	}

	if !hasLocal && hasRemote {
		return "Remote-only"
	}

	if localCount == remoteCount {
		return "OK"
	}

	return "Mismatch"
}

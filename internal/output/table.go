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
		table.Append(p.Name, strconv.Itoa(p.JSONLCount))
	}

	table.Render()
}

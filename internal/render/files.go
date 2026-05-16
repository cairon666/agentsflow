package render

import (
	"sort"
	"strings"

	flowmodel "github.com/cairon666/agentsflow/internal/flow"
)

// AgentIDs returns sorted agent ids.
func AgentIDs(flow flowmodel.Flow) []string {
	ids := make([]string, 0, len(flow.Agents))
	for id := range flow.Agents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// HyphenID returns an id suitable for tools that prefer hyphens.
func HyphenID(id string) string {
	return strings.ReplaceAll(id, "_", "-")
}

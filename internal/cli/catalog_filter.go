package cli

import (
	"gymctl/internal/scenario"
)

func filterScaffoldEntries(entries []scenario.CatalogEntry, includeScaffold bool) []scenario.CatalogEntry {
	if includeScaffold {
		return entries
	}

	filtered := make([]scenario.CatalogEntry, 0, len(entries))
	for _, entry := range entries {
		if scenario.IsScaffold(entry.Exercise) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"gymctl/internal/scenario"
)

type authorStatusOptions struct {
	track string
}

func newAuthorStatusCmd() *cobra.Command {
	opts := &authorStatusOptions{}
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report exercise implementation status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := loadCatalogEntries()
			if err != nil {
				return err
			}

			filtered := make([]scenario.CatalogEntry, 0, len(entries))
			for _, entry := range entries {
				if opts.track != "" && !strings.EqualFold(entry.Exercise.Metadata.Track, opts.track) {
					continue
				}
				filtered = append(filtered, entry)
			}

			sort.Slice(filtered, func(i, j int) bool {
				ai := filtered[i].Exercise
				aj := filtered[j].Exercise
				if ai.Metadata.Track != aj.Metadata.Track {
					return ai.Metadata.Track < aj.Metadata.Track
				}
				if ai.Metadata.Order != aj.Metadata.Order {
					return ai.Metadata.Order < aj.Metadata.Order
				}
				return ai.Metadata.Name < aj.Metadata.Name
			})

			counts := map[string]int{
				scenario.ImplementationStatusScaffold: 0,
				scenario.ImplementationStatusDraft:    0,
				scenario.ImplementationStatusReady:    0,
			}

			type row struct {
				name   string
				status string
				issues []string
			}
			rows := make([]row, 0, len(filtered))

			for _, entry := range filtered {
				report := validateExerciseReadiness(entry.Path, entry.Exercise)
				counts[report.Status]++

				var issues []string
				if report.MissingSetup {
					issues = append(issues, "missing setup")
				}
				if report.PlaceholderChecks {
					issues = append(issues, "placeholder checks")
				}
				if len(report.MissingHints) > 0 {
					issues = append(issues, fmt.Sprintf("missing hints: %s", strings.Join(report.MissingHints, ", ")))
				}
				rows = append(rows, row{name: entry.Exercise.Metadata.Name, status: report.Status, issues: issues})
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Exercise authoring status")
			if opts.track != "" {
				fmt.Fprintf(out, " for track %s", opts.track)
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Total: %d  ready: %d  draft: %d  scaffold: %d\n",
				len(filtered),
				counts[scenario.ImplementationStatusReady],
				counts[scenario.ImplementationStatusDraft],
				counts[scenario.ImplementationStatusScaffold],
			)

			for _, row := range rows {
				if len(row.issues) == 0 {
					fmt.Fprintf(out, "  %-48s %-8s ok\n", row.name, row.status)
					continue
				}
				fmt.Fprintf(out, "  %-48s %-8s %s\n", row.name, row.status, strings.Join(row.issues, "; "))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.track, "track", "", "Filter by track")
	return cmd
}

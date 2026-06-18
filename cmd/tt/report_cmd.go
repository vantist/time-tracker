package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/reconcile"
	"github.com/user/tt/internal/report"
)

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().String("project", "", "Filter by project name")
	reportCmd.Flags().String("since", "7d", "Time range: 7d, 30d, or YYYY-MM-DD")
	reportCmd.Flags().String("format", "text", "Output format: text or json")
	reportCmd.Flags().Bool("by-work-item", false, "Group by work item")
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Show usage report",
	RunE: func(cmd *cobra.Command, args []string) error {
		sinceStr, _ := cmd.Flags().GetString("since")
		project, _ := cmd.Flags().GetString("project")
		format, _ := cmd.Flags().GetString("format")
		byWorkItem, _ := cmd.Flags().GetBool("by-work-item")

		since, err := parseSince(sinceStr)
		if err != nil {
			return fmt.Errorf("--since: %w", err)
		}

		conn, err := db.Open()
		if err != nil {
			return err
		}
		defer conn.Close()

		reconcile.MaybeReconcile(conn)

		result, err := report.Query(conn, report.Options{
			Since:      since,
			Project:    project,
			ByWorkItem: byWorkItem,
		})
		if err != nil {
			return err
		}

		if result.Empty {
			fmt.Println("No data for the selected period.")
			return nil
		}

		switch format {
		case "json":
			fmt.Println(report.FormatJSON(result))
		default:
			fmt.Print(report.FormatText(result))
			if byWorkItem && len(result.Groups) > 0 {
				fmt.Println("\nBy work item:")
				for _, g := range result.Groups {
					costStr := "N/A"
					if g.EstimatedCostUSD != nil {
						costStr = fmt.Sprintf("$%.4f", *g.EstimatedCostUSD)
					}
					fmt.Printf("  %-30s  %dh %dm  %s\n",
						g.Label,
						g.AgentTimeSec/3600, (g.AgentTimeSec%3600)/60,
						costStr,
					)
				}
			}
		}
		return nil
	},
}

func parseSince(s string) (time.Time, error) {
	now := time.Now().UTC()
	// Try duration format: 7d, 30d
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return now.AddDate(0, 0, -days), nil
		}
	}
	// Try date format: YYYY-MM-DD
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected NNd or YYYY-MM-DD, got %q", s)
	}
	return t.UTC(), nil
}


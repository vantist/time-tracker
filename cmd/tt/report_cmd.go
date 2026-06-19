package main

import (
	"fmt"
	"strconv"
	"strings"
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
		}
		return nil
	},
}

func parseSince(s string) (time.Time, error) {
	now := time.Now().UTC()
	if strings.HasSuffix(s, "d") {
		if days, err := strconv.Atoi(s[:len(s)-1]); err == nil {
			return now.AddDate(0, 0, -days), nil
		}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected NNd or YYYY-MM-DD, got %q", s)
	}
	return t.UTC(), nil
}


package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/user/tt/internal/db"
	"github.com/user/tt/internal/reconcile"
	"github.com/user/tt/internal/report"
)

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().Int("port", 7890, "HTTP server port")
	serveCmd.Flags().String("since", "7d", "Time range: 7d, 30d, or YYYY-MM-DD")
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start web dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		sinceStr, _ := cmd.Flags().GetString("since")

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

		opts := report.Options{Since: since}
		mux := http.NewServeMux()
		mux.HandleFunc("/", report.HandleDashboard)
		mux.Handle("/api/report", reconcileMiddleware(conn, report.HandleAPIReport(conn, opts)))

		addr := fmt.Sprintf(":%d", port)
		url := fmt.Sprintf("http://localhost:%d", port)
		fmt.Printf("Serving at %s\n", url)
		openBrowser(url)

		if err := http.ListenAndServe(addr, mux); err != nil {
			return fmt.Errorf("listen on port %d: %w", port, err)
		}
		return nil
	},
}

// reconcileMiddleware calls MaybeReconcile before the handler if no active session exists.
func reconcileMiddleware(conn *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !reconcile.HasActiveSession(conn) {
			reconcile.MaybeReconcile(conn)
		}
		next.ServeHTTP(w, r)
	})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	// ponytail: runtime GOOS, add build tags if cross-compile needed
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	cmd.Start() //nolint:errcheck
}

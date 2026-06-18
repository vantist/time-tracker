package recorder

import (
	"database/sql"
	"fmt"
	"os"
)

// RecordPromptSilent calls RecordPrompt and swallows any error to stderr.
// Exit code remains 0 — hooks must never block Claude Code.
func RecordPromptSilent(conn *sql.DB, input PromptInput) error {
	if err := RecordPrompt(conn, input); err != nil {
		fmt.Fprintf(os.Stderr, "tt: record prompt error: %v\n", err)
	}
	return nil
}

// RecordResponseSilent calls RecordResponse and swallows any error to stderr.
func RecordResponseSilent(conn *sql.DB, sessionID, tokensJSON, model string) error {
	if err := RecordResponse(conn, sessionID, tokensJSON, model); err != nil {
		fmt.Fprintf(os.Stderr, "tt: record response error: %v\n", err)
	}
	return nil
}

package recorder_test

import (
	"database/sql"
	"testing"

	"github.com/user/tt/internal/recorder"
)

// Task 3.7: when DB is closed (simulates lock/failure), RecordPrompt/RecordResponse
// return nil error (silent) — callers (hook commands) must not fail loudly.
func TestRecordPromptSilentOnDBError(t *testing.T) {
	conn := openTestDB(t)
	conn.Close() // force all future ops to fail

	err := recorder.RecordPromptSilent(conn, recorder.PromptInput{
		SessionID: "sess-err",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	})
	if err != nil {
		t.Errorf("RecordPromptSilent returned error: %v", err)
	}
}

func TestRecordResponseSilentOnDBError(t *testing.T) {
	conn, _ := sql.Open("sqlite", ":memory:") // no schema → will error
	conn.Close()

	err := recorder.RecordResponseSilent(conn, "sess-err", `{"input_tokens":100}`, "")
	if err != nil {
		t.Errorf("RecordResponseSilent returned error: %v", err)
	}
}

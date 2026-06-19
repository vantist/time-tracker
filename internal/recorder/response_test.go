package recorder_test

import (
	"testing"

	"github.com/user/tt/internal/recorder"
)

// Task 14.1: transcript model 寫入 sessions
func TestRecordResponseWritesModelFromTranscript(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-m1",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "",
	}); err != nil {
		t.Fatal(err)
	}

	tokensJSON := `{"input_tokens":100,"output_tokens":50}`
	if err := recorder.RecordResponse(conn, "sess-m1", tokensJSON, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("RecordResponse: %v", err)
	}

	var model string
	conn.QueryRow("SELECT model FROM sessions WHERE id='sess-m1'").Scan(&model)
	if model != "claude-sonnet-4-6" {
		t.Errorf("sessions.model = %q, want claude-sonnet-4-6", model)
	}
}

// Task 14.2: sessions.model 已有值時 UPDATE 不覆蓋
func TestRecordResponseDoesNotOverwriteExistingModel(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-m2",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "claude-opus-4-8",
	}); err != nil {
		t.Fatal(err)
	}

	tokensJSON := `{"input_tokens":100,"output_tokens":50}`
	if err := recorder.RecordResponse(conn, "sess-m2", tokensJSON, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("RecordResponse: %v", err)
	}

	var model string
	conn.QueryRow("SELECT model FROM sessions WHERE id='sess-m2'").Scan(&model)
	if model != "claude-opus-4-8" {
		t.Errorf("sessions.model = %q, want claude-opus-4-8 (should not be overwritten)", model)
	}
}

// Task 14.3: transcript model 為空時 sessions.model 不變，tokens 正常記錄
func TestRecordResponseEmptyModelNoUpdate(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-m3",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "",
	}); err != nil {
		t.Fatal(err)
	}

	tokensJSON := `{"input_tokens":200,"output_tokens":80}`
	if err := recorder.RecordResponse(conn, "sess-m3", tokensJSON, ""); err != nil {
		t.Fatalf("RecordResponse: %v", err)
	}

	var model string
	conn.QueryRow("SELECT model FROM sessions WHERE id='sess-m3'").Scan(&model)
	if model != "" {
		t.Errorf("sessions.model = %q, want empty", model)
	}

	var inputTok int
	conn.QueryRow("SELECT input_tokens FROM turns WHERE session_id='sess-m3'").Scan(&inputTok)
	if inputTok != 200 {
		t.Errorf("input_tokens = %d, want 200", inputTok)
	}
}

// Task 3.5: RecordResponse updates latest turn token fields and cost
func TestRecordResponseFlatJSON(t *testing.T) {
	conn := openTestDB(t)

	// seed a session + turn via RecordPrompt
	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-r1",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	}); err != nil {
		t.Fatal(err)
	}

	tokensJSON := `{"input_tokens":1000,"output_tokens":200,"cache_read_tokens":500,"cache_creation_tokens":0}`
	if err := recorder.RecordResponse(conn, "sess-r1", tokensJSON, ""); err != nil {
		t.Fatalf("RecordResponse: %v", err)
	}

	var inputTok, outputTok, cacheRead, cacheCreate int
	var cost float64
	var responseAt string
	conn.QueryRow(`
		SELECT input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens,
		       estimated_cost_usd, response_at
		FROM turns WHERE session_id='sess-r1'
	`).Scan(&inputTok, &outputTok, &cacheRead, &cacheCreate, &cost, &responseAt)

	if inputTok != 1000 || outputTok != 200 || cacheRead != 500 || cacheCreate != 0 {
		t.Errorf("tokens wrong: in=%d out=%d cr=%d cc=%d", inputTok, outputTok, cacheRead, cacheCreate)
	}
	// cost = (1000/1e6)*3 + (200/1e6)*15 + (500/1e6)*0.30 = 0.003+0.003+0.00015 = 0.00615
	if cost < 0.006 || cost > 0.007 {
		t.Errorf("cost out of range: %f", cost)
	}
	if responseAt == "" {
		t.Error("response_at not set")
	}
}

func TestRecordResponseNestedJSON(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-r2",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	}); err != nil {
		t.Fatal(err)
	}

	// nested format (some tools wrap under a key)
	tokensJSON := `{"usage":{"input_tokens":500,"output_tokens":100,"cache_read_tokens":0,"cache_creation_tokens":0}}`
	if err := recorder.RecordResponse(conn, "sess-r2", tokensJSON, ""); err != nil {
		t.Fatalf("RecordResponse nested: %v", err)
	}

	var inputTok int
	conn.QueryRow("SELECT input_tokens FROM turns WHERE session_id='sess-r2'").Scan(&inputTok)
	if inputTok != 500 {
		t.Errorf("nested: input_tokens=%d want 500", inputTok)
	}
}

func TestRecordResponseUnknownModelNullCost(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-r3",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "gpt-5-unknown",
	}); err != nil {
		t.Fatal(err)
	}

	if err := recorder.RecordResponse(conn, "sess-r3", `{"input_tokens":100,"output_tokens":50}`, ""); err != nil {
		t.Fatalf("RecordResponse unknown model: %v", err)
	}

	var cost *float64
	conn.QueryRow("SELECT estimated_cost_usd FROM turns WHERE session_id='sess-r3'").Scan(&cost)
	if cost != nil {
		t.Errorf("expected NULL cost for unknown model, got %v", *cost)
	}
}

func TestRecordResponseEmptyTokensNoError(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-r4",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	}); err != nil {
		t.Fatal(err)
	}

	if err := recorder.RecordResponse(conn, "sess-r4", "", ""); err != nil {
		t.Errorf("empty tokens should not error: %v", err)
	}

	var responseAt string
	conn.QueryRow("SELECT response_at FROM turns WHERE session_id='sess-r4'").Scan(&responseAt)
	if responseAt == "" {
		t.Error("response_at must be set even with empty tokens")
	}
}

func TestRecordResponseWritesModelUsages(t *testing.T) {
	conn := openTestDB(t)

	if err := recorder.RecordPrompt(conn, recorder.PromptInput{
		SessionID: "sess-u1",
		Project:   "/proj",
		Tool:      "claude-code",
		Model:     "claude-sonnet-4-6",
	}); err != nil {
		t.Fatal(err)
	}

	tokensJSON := `{"input_tokens":1000,"output_tokens":200,"cache_read_tokens":500,"cache_creation_tokens":0}`
	if err := recorder.RecordResponse(conn, "sess-u1", tokensJSON, "claude-sonnet-4-6"); err != nil {
		t.Fatalf("RecordResponse: %v", err)
	}

	var count int
	conn.QueryRow("SELECT COUNT(*) FROM turn_model_usages").Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 turn_model_usages entry, got %d", count)
	}

	var turnID int64
	var model string
	var isSubagent bool
	var inputTok, outputTok, cacheRead, cacheCreate int
	var cost float64
	err := conn.QueryRow(`
		SELECT turn_id, model, is_subagent, input_tokens, output_tokens, cache_read_tokens, cache_creation_tokens, estimated_cost_usd
		FROM turn_model_usages
	`).Scan(&turnID, &model, &isSubagent, &inputTok, &outputTok, &cacheRead, &cacheCreate, &cost)
	if err != nil {
		t.Fatalf("failed to query turn_model_usages: %v", err)
	}

	if model != "claude-sonnet-4-6" {
		t.Errorf("expected model claude-sonnet-4-6, got %q", model)
	}
	if isSubagent {
		t.Error("expected is_subagent = false, got true")
	}
	if inputTok != 1000 || outputTok != 200 || cacheRead != 500 || cacheCreate != 0 {
		t.Errorf("tokens mismatch in turn_model_usages: %d, %d, %d, %d", inputTok, outputTok, cacheRead, cacheCreate)
	}
	// cost calculation: (1000/1e6)*3.00 + (200/1e6)*15.00 + (500/1e6)*0.30 = 0.00615
	if cost < 0.00614 || cost > 0.00616 {
		t.Errorf("cost mismatch in turn_model_usages: %f", cost)
	}
}

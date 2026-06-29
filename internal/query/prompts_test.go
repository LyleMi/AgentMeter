package query

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestPromptSuggestionsExtractUserPromptShapes(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	sourceID := insertPromptSource(t, conn, "codex", "Codex", now)
	rawEvents := []string{
		`{"type":"user","message":{"role":"user","content":"Run unit tests"}}`,
		`{"type":"message","role":"user","content":[{"type":"input_text","text":"Run unit tests"}]}`,
		`{"role":"user","content":"Run   unit   tests"}`,
		`{"type":"event","payload":{"type":"user_message","text":"Explain prompt clustering"}}`,
		`{"type":"event","payload":{"type":"message","role":"user","content":"Explain prompt clustering"}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}}`,
	}
	for index, raw := range rawEvents {
		started := now.Add(time.Duration(index) * time.Minute)
		sessionID, sourceFileID := insertPromptSession(t, conn, sourceID, fmt.Sprintf("shape-%d", index), "/workspace/api", started)
		insertPromptEvent(t, conn, sessionID, sourceFileID, index+1, started, raw)
	}

	suggestions, err := New(conn).PromptSuggestions(ctx, model.PromptSuggestionFilters{MinCount: 1, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	runTests := findPromptSuggestion(t, suggestions, "Run unit tests")
	if runTests.Count != 3 || runTests.SessionCount != 3 || runTests.VariantCount != 1 {
		t.Fatalf("run tests suggestion = %+v", runTests)
	}
	if runTests.MatchKind != "exact" || runTests.Confidence != 1 {
		t.Fatalf("run tests match metadata = %+v", runTests)
	}
	if len(runTests.Examples) == 0 || runTests.Examples[0].SessionID == 0 || runTests.Examples[0].ProjectPath == "" || runTests.Examples[0].SourceKey == "" {
		t.Fatalf("run tests examples missing context: %+v", runTests.Examples)
	}

	explain := findPromptSuggestion(t, suggestions, "Explain prompt clustering")
	if explain.Count != 2 || explain.SessionCount != 2 {
		t.Fatalf("payload suggestion = %+v", explain)
	}
	assertNoPromptSuggestion(t, suggestions, "ok")
}

func TestPromptSuggestionsFilterInstructionToolAndStructuredDumps(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 30, 10, 30, 0, 0, time.UTC)
	sourceID := insertPromptSource(t, conn, "codex", "Codex", now)
	validPrompt := "Summarize the latest billing changes"
	junkPrompts := []string{
		"# AGENTS.md instructions for D:\\tools\\custom\\AgentMeter\n<INSTRUCTIONS>\n# Repository Guidelines\n## Project Structure & Module Organization\n## Agent-Specific Instructions\n</INSTRUCTIONS>",
		"## Context Usage\n**Model:** claude-opus-4\n### Estimated usage by category\n| Category | Tokens | Percentage |\n### Memory Files\n### Skills\n| Skill | Source | Tokens |",
		"<conversation_history_summary><summary>old work</summary></conversation_history_summary>",
		"[Request interrupted by user]",
		`{"tool_call_id":"toolu_1","type":"tool_result","content":"ok"}`,
		"commit now",
		"commit吧",
		"good, commit",
		strings.Repeat("long prompt block ", 140),
		"```go\npackage main\nfunc main() {}\n```\n" + strings.Repeat("code dump ", 40),
		"diff --git a/file.go b/file.go @@ -1 +1 @@ old new",
	}
	fixtures := append([]string{validPrompt, validPrompt}, junkPrompts...)
	fixtures = append(fixtures, junkPrompts...)

	for index, prompt := range fixtures {
		started := now.Add(time.Duration(index) * time.Minute)
		sessionID, sourceFileID := insertPromptSession(t, conn, sourceID, fmt.Sprintf("junk-%d", index), "/workspace/prompts", started)
		insertPromptEvent(t, conn, sessionID, sourceFileID, index+1, started, promptRaw(t, prompt))
	}

	suggestions, err := New(conn).PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions should only contain valid prompt: %+v", suggestions)
	}
	item := findPromptSuggestion(t, suggestions, validPrompt)
	if item.Count != 2 || item.SessionCount != 2 {
		t.Fatalf("valid prompt suggestion = %+v", item)
	}
	for _, suggestion := range suggestions {
		normalized := normalizePromptText(suggestion.Text)
		if containsAnyPromptTerm(normalized, "agents.md", "context usage", "tool_call_id", "conversation_history_summary", "request interrupted", "diff --git", "```") {
			t.Fatalf("junk prompt leaked into suggestions: %+v", suggestion)
		}
	}
}

func TestPromptSuggestionsClusterNearDuplicatesWithDefaults(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 30, 11, 0, 0, 0, time.UTC)
	sourceID := insertPromptSource(t, conn, "codex", "Codex", now)
	prompts := []string{
		"Review prompt backend implementation",
		"Review prompt backend implementation please",
		"Unrelated one-off",
	}
	for index, prompt := range prompts {
		started := now.Add(time.Duration(index) * time.Minute)
		sessionID, sourceFileID := insertPromptSession(t, conn, sourceID, fmt.Sprintf("near-%d", index), "/workspace/prompts", started)
		insertPromptEvent(t, conn, sessionID, sourceFileID, index+1, started, `{"type":"message","role":"user","content":"`+prompt+`"}`)
	}

	suggestions, err := New(conn).PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions = %+v", suggestions)
	}
	item := suggestions[0]
	if item.Count != 2 || item.SessionCount != 2 || item.VariantCount != 2 || len(item.Variants) != 2 {
		t.Fatalf("near duplicate suggestion = %+v", item)
	}
	if item.MatchKind != "near" || item.Confidence <= 0 || item.Confidence > 1 {
		t.Fatalf("near duplicate match metadata = %+v", item)
	}
	if item.Key == "" || len(item.Examples) == 0 || item.Examples[0].SessionID == 0 {
		t.Fatalf("near duplicate suggestion missing identity/context: %+v", item)
	}
}

func TestPromptSuggestionsFiltersByAgentProjectAndSearch(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)
	codexSourceID := insertPromptSource(t, conn, "codex", "Codex", now)
	claudeSourceID := insertPromptSource(t, conn, "claude", "Claude Code", now)
	fixtures := []struct {
		sourceID int64
		key      string
		project  string
		prompt   string
	}{
		{codexSourceID, "codex-api-a", "/workspace/api", "Deploy the backend service"},
		{codexSourceID, "codex-api-b", "/workspace/api/.", "Deploy the backend service"},
		{codexSourceID, "codex-ui", "/workspace/ui", "Deploy the frontend service"},
		{claudeSourceID, "claude-api", "/workspace/api", "Deploy the backend service"},
	}
	for index, fixture := range fixtures {
		started := now.Add(time.Duration(index) * time.Minute)
		sessionID, sourceFileID := insertPromptSession(t, conn, fixture.sourceID, fixture.key, fixture.project, started)
		insertPromptEvent(t, conn, sessionID, sourceFileID, index+1, started, `{"type":"message","role":"user","content":"`+fixture.prompt+`"}`)
	}

	suggestions, err := New(conn).PromptSuggestions(ctx, model.PromptSuggestionFilters{
		Agent:    sourceInstanceKey(codexSourceID),
		Project:  "/workspace/api",
		Search:   "backend",
		MinCount: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("filtered suggestions = %+v", suggestions)
	}
	item := suggestions[0]
	if item.Count != 2 || item.SessionCount != 2 || item.SourceID != codexSourceID {
		t.Fatalf("filtered suggestion = %+v", item)
	}
	for _, example := range item.Examples {
		if example.SourceID != codexSourceID || projectFilterKey(example.ProjectPath) != projectFilterKey("/workspace/api") {
			t.Fatalf("example ignored filters: %+v", example)
		}
	}
}

func TestPromptSuggestionsFilterIgnoredAndSavedKeys(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 30, 13, 0, 0, 0, time.UTC)
	sourceID := insertPromptSource(t, conn, "codex", "Codex", now)
	for index := 0; index < 2; index++ {
		started := now.Add(time.Duration(index) * time.Minute)
		sessionID, sourceFileID := insertPromptSession(t, conn, sourceID, fmt.Sprintf("filter-%d", index), "/workspace/api", started)
		insertPromptEvent(t, conn, sessionID, sourceFileID, index+1, started, `{"type":"message","role":"user","content":"Create release notes"}`)
	}

	service := New(conn)
	suggestions, err := service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("initial suggestions = %+v", suggestions)
	}
	key := suggestions[0].Key

	if err := service.IgnorePromptSuggestion(ctx, key); err != nil {
		t.Fatal(err)
	}
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("ignored suggestions = %+v", suggestions)
	}
	if err := service.UnignorePromptSuggestion(ctx, key); err != nil {
		t.Fatal(err)
	}
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("unignored suggestions = %+v", suggestions)
	}

	saved, err := service.SavePrompt(ctx, model.SavedPromptInput{
		Title:               "Release notes",
		Content:             "Create release notes",
		SourceSuggestionKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("saved suggestions = %+v", suggestions)
	}
	if err := service.DeleteSavedPrompt(ctx, saved.ID); err != nil {
		t.Fatal(err)
	}
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 {
		t.Fatalf("suggestions after deleting saved prompt = %+v", suggestions)
	}
}

func TestPromptSuggestionsSuppressNearClusterWhenCanonicalVariantChanges(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	now := time.Date(2026, 6, 30, 13, 30, 0, 0, time.UTC)
	sourceID := insertPromptSource(t, conn, "codex", "Codex", now)
	fixtures := []struct {
		key    string
		prompt string
		at     time.Time
	}{
		{"stable-a", "Review prompt backend implementation", now},
		{"stable-b", "Review prompt backend implementation please", now.Add(time.Minute)},
	}
	for index, fixture := range fixtures {
		sessionID, sourceFileID := insertPromptSession(t, conn, sourceID, fixture.key, "/workspace/api", fixture.at)
		insertPromptEvent(t, conn, sessionID, sourceFileID, index+1, fixture.at, `{"type":"message","role":"user","content":"`+fixture.prompt+`"}`)
	}

	service := New(conn)
	suggestions, err := service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 || suggestions[0].VariantCount != 2 {
		t.Fatalf("initial near suggestions = %+v", suggestions)
	}
	oldKey := suggestions[0].Key

	later := now.Add(2 * time.Minute)
	sessionID, sourceFileID := insertPromptSession(t, conn, sourceID, "stable-a-again", "/workspace/api", later)
	insertPromptEvent(t, conn, sessionID, sourceFileID, 3, later, `{"type":"message","role":"user","content":"Review prompt backend implementation"}`)
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 1 || suggestions[0].Key == oldKey {
		t.Fatalf("canonical key should change before suppression regression check: old=%s suggestions=%+v", oldKey, suggestions)
	}

	saved, err := service.SavePrompt(ctx, model.SavedPromptInput{
		Title:               "Review backend",
		Content:             "Review prompt backend implementation please",
		SourceSuggestionKey: oldKey,
	})
	if err != nil {
		t.Fatal(err)
	}
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("saved old variant key should suppress changed canonical cluster: %+v", suggestions)
	}
	if err := service.DeleteSavedPrompt(ctx, saved.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.IgnorePromptSuggestion(ctx, oldKey); err != nil {
		t.Fatal(err)
	}
	suggestions, err = service.PromptSuggestions(ctx, model.PromptSuggestionFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) != 0 {
		t.Fatalf("ignored old variant key should suppress changed canonical cluster: %+v", suggestions)
	}
}

func TestSavedPromptsCRUDAndCopy(t *testing.T) {
	ctx := context.Background()
	conn, err := db.Open(filepath.Join(t.TempDir(), "agentmeter.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	service := New(conn)
	saved, err := service.SavePrompt(ctx, model.SavedPromptInput{Content: "Run go test ./..."})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == 0 || saved.Title != "Run go test ./..." || saved.CopyCount != 0 || saved.LastCopiedAt != nil {
		t.Fatalf("saved prompt = %+v", saved)
	}

	updated, err := service.UpdateSavedPrompt(ctx, saved.ID, model.SavedPromptInput{
		Title:               "Test all packages",
		Content:             "go test ./...",
		SourceSuggestionKey: "prompt:test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Test all packages" || updated.Content != "go test ./..." || updated.SourceSuggestionKey != "prompt:test" {
		t.Fatalf("updated prompt = %+v", updated)
	}

	copied, err := service.RecordPromptCopy(ctx, saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copied.CopyCount != 1 || copied.LastCopiedAt == nil {
		t.Fatalf("copied prompt = %+v", copied)
	}

	list, err := service.SavedPrompts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != saved.ID || list[0].CopyCount != 1 {
		t.Fatalf("saved prompts list = %+v", list)
	}
	if err := service.DeleteSavedPrompt(ctx, saved.ID); err != nil {
		t.Fatal(err)
	}
	list, err = service.SavedPrompts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("saved prompts after delete = %+v", list)
	}
	if err := service.DeleteSavedPrompt(ctx, saved.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second delete err = %v", err)
	}
}

func insertPromptSource(t *testing.T, conn *sql.DB, kind, name string, now time.Time) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO sources
		(kind, name, root_path, sessions_path, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		kind, name, "/workspace/"+kind, "/workspace/"+kind+"/sessions", "test", db.FormatTime(now), db.FormatTime(now))
}

func insertPromptSession(t *testing.T, conn *sql.DB, sourceID int64, key, projectPath string, started time.Time) (int64, int64) {
	t.Helper()
	sourceFileID := insertRow(t, conn, `INSERT INTO source_files
		(source_id, path, size_bytes, modified_at, content_hash, last_scanned_at, scan_status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, fmt.Sprintf("/workspace/prompts/%d/%s.jsonl", sourceID, key), 128, db.FormatTime(started), "hash-"+key, db.FormatTime(started), "indexed", "")
	sessionID := insertRow(t, conn, `INSERT INTO sessions
		(source_id, source_file_id, session_key, codex_session_id, project_path, model, model_provider, originator, thread_source, agent_nickname, agent_role,
		 started_at, ended_at, wall_duration_ms, active_duration_ms, model_duration_ms, tool_duration_ms, idle_duration_ms, event_count, parse_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sourceID, sourceFileID, key, "codex-"+key, projectPath, "gpt-5", "openai", "cli", "local", "", "",
		db.FormatTime(started), db.FormatTime(started.Add(time.Second)), 1000, 1000, 1000, 0, 0, 1, "ok")
	return sessionID, sourceFileID
}

func insertPromptEvent(t *testing.T, conn *sql.DB, sessionID, sourceFileID int64, line int, timestamp time.Time, rawJSON string) int64 {
	t.Helper()
	return insertRow(t, conn, `INSERT INTO events
		(session_id, source_file_id, source_line, timestamp, kind, raw_type, summary, raw_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionID, sourceFileID, line, db.FormatTime(timestamp), "user", "user", "User message", rawJSON)
}

func promptRaw(t *testing.T, text string) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"type":    "message",
		"role":    "user",
		"content": text,
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func findPromptSuggestion(t *testing.T, suggestions []model.PromptSuggestion, text string) model.PromptSuggestion {
	t.Helper()
	normalized := normalizePromptText(text)
	for _, suggestion := range suggestions {
		if normalizePromptText(suggestion.Text) == normalized {
			return suggestion
		}
	}
	t.Fatalf("prompt suggestion %q missing: %+v", text, suggestions)
	return model.PromptSuggestion{}
}

func assertNoPromptSuggestion(t *testing.T, suggestions []model.PromptSuggestion, text string) {
	t.Helper()
	normalized := normalizePromptText(text)
	for _, suggestion := range suggestions {
		if normalizePromptText(suggestion.Text) == normalized {
			t.Fatalf("prompt suggestion %q should be absent: %+v", text, suggestions)
		}
	}
}

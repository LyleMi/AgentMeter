package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileExtractsSessionUsageAndTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-redacted.jsonl")
	content := `{"timestamp":"2026-06-26T10:00:00Z","type":"session_meta","payload":{"session_id":"sess_redacted","cwd":"D:\\workspace\\project","originator":"codex_cli","thread_source":"local","model_provider":"openai"}}
{"timestamp":"2026-06-26T10:00:01Z","type":"turn_context","payload":{"model":"gpt-5.5","cwd":"D:\\workspace\\project"}}
{"timestamp":"2026-06-26T10:00:02Z","type":"event_msg","payload":{"type":"task_started","turn_id":"turn_1","started_at":1782468002}}
{"timestamp":"2026-06-26T10:00:03Z","type":"response_item","payload":{"type":"function_call","id":"fc_1","name":"shell_command","arguments":"{\"command\":\"go test ./...\"}","call_id":"call_1"}}
{"timestamp":"2026-06-26T10:00:05Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_1","output":"ok"}}
{"timestamp":"2026-06-26T10:00:07Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":80,"reasoning_output_tokens":20,"total_tokens":1080},"last_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":80,"reasoning_output_tokens":20,"total_tokens":1080},"model_context_window":258400}}}
{"timestamp":"2026-06-26T10:00:08Z","type":"event_msg","payload":{"type":"task_complete","turn_id":"turn_1","completed_at":1782468008,"duration_ms":6000}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.CodexSessionID != "sess_redacted" {
		t.Fatalf("session id = %q", parsed.Session.CodexSessionID)
	}
	if parsed.Session.Model != "gpt-5.5" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Usage.TotalTokens != 1080 {
		t.Fatalf("total tokens = %d", parsed.Usage.TotalTokens)
	}
	if parsed.Usage.Source != "actual" {
		t.Fatalf("usage source = %q", parsed.Usage.Source)
	}
	if len(parsed.ToolCall) != 1 {
		t.Fatalf("tool calls = %d", len(parsed.ToolCall))
	}
	if parsed.ToolCall[0].ToolName != "shell_command" {
		t.Fatalf("tool name = %q", parsed.ToolCall[0].ToolName)
	}
	if parsed.ToolCall[0].DurationMS != 2000 {
		t.Fatalf("tool duration = %d", parsed.ToolCall[0].DurationMS)
	}
	if len(parsed.ModelCall) != 1 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if parsed.Session.ParseStatus != "ok" {
		t.Fatalf("parse status = %q, warnings: %v", parsed.Session.ParseStatus, parsed.Warnings)
	}
}

func TestParseFileKeepsMalformedLineAsWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "broken.jsonl")
	content := `{"timestamp":"2026-06-26T10:00:00Z","type":"turn_context","payload":{"model":"gpt-5.5","cwd":"D:\\workspace\\project"}}
not-json
{"timestamp":"2026-06-26T10:00:01Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1,"reasoning_output_tokens":0,"total_tokens":2},"last_token_usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1,"reasoning_output_tokens":0,"total_tokens":2}}}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.ParseStatus != "warning" {
		t.Fatalf("parse status = %q", parsed.Session.ParseStatus)
	}
	if len(parsed.Warnings) == 0 {
		t.Fatal("expected malformed line warning")
	}
	if parsed.Usage.TotalTokens != 2 {
		t.Fatalf("total tokens = %d", parsed.Usage.TotalTokens)
	}
}

func TestParseFileBuildsUsageDeltasFromCumulativeTotals(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cumulative.jsonl")
	content := `{"timestamp":"2026-01-02T00:00:00Z","type":"turn_context","payload":{"model":"gpt-5.2"}}
{"timestamp":1767312001000,"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":10,"output_tokens":50,"reasoning_output_tokens":0,"total_tokens":150}}}}
{"timestamp":"2026-01-02T00:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":200,"cached_input_tokens":20,"output_tokens":75,"reasoning_output_tokens":5,"total_tokens":280}}}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Usage.InputTokens != 200 {
		t.Fatalf("input tokens = %d", parsed.Usage.InputTokens)
	}
	if parsed.Usage.CachedInputTokens != 20 {
		t.Fatalf("cached input tokens = %d", parsed.Usage.CachedInputTokens)
	}
	if parsed.Usage.OutputTokens != 75 {
		t.Fatalf("output tokens = %d", parsed.Usage.OutputTokens)
	}
	if parsed.Usage.ReasoningOutputTokens != 5 {
		t.Fatalf("reasoning tokens = %d", parsed.Usage.ReasoningOutputTokens)
	}
	if parsed.Usage.TotalTokens != 280 {
		t.Fatalf("total tokens = %d", parsed.Usage.TotalTokens)
	}
	if len(parsed.ModelCall) != 2 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if parsed.ModelCall[1].InputTokens != 100 || parsed.ModelCall[1].OutputTokens != 25 || parsed.ModelCall[1].ReasoningOutputTokens != 5 || parsed.ModelCall[1].TotalTokens != 130 {
		t.Fatalf("second model call usage = %+v", parsed.ModelCall[1])
	}
}

func TestParseFileSupportsHeadlessUsageRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run.jsonl")
	content := `{"type":"turn.completed","timestamp":"2026-01-02T03:04:05.000Z","model":"gpt-5.2-codex","usage":{"input_tokens":120,"cached_input_tokens":20,"output_tokens":30,"total_tokens":150}}
{"type":"result","data":{"timestamp":"2026-01-02T03:05:05.000Z","model_name":"gpt-5.2-codex","usage":{"prompt_tokens":50,"cached_tokens":5,"completion_tokens":12}}}
{"type":"turn.completed","timestamp":"2026-01-02T03:06:05.000Z","model":"gpt-5.2-codex","usage":{"input_tokens":9,"output_tokens":4,"reasoning_output_tokens":1,"total_tokens":0}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.CodexSessionID != "run" {
		t.Fatalf("session id = %q", parsed.Session.CodexSessionID)
	}
	if parsed.Session.ParseStatus != "ok" {
		t.Fatalf("parse status = %q, warnings: %v", parsed.Session.ParseStatus, parsed.Warnings)
	}
	if parsed.Session.Model != "gpt-5.2-codex" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Usage.InputTokens != 179 || parsed.Usage.CachedInputTokens != 25 || parsed.Usage.OutputTokens != 46 || parsed.Usage.ReasoningOutputTokens != 1 || parsed.Usage.TotalTokens != 226 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if len(parsed.ModelCall) != 3 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
}

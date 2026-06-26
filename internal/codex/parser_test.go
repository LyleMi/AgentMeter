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

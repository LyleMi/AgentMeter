package sessionjsonl

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
{"timestamp":"2026-06-26T10:00:07Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":80,"reasoning_output_tokens":20,"context_compression_tokens":40,"total_tokens":1120},"last_token_usage":{"input_tokens":1000,"cached_input_tokens":200,"output_tokens":80,"reasoning_output_tokens":20,"context_compression_tokens":40,"total_tokens":1120},"model_context_window":258400}}}
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
	if parsed.Session.SessionKey != "sess_redacted" {
		t.Fatalf("session key = %q", parsed.Session.SessionKey)
	}
	if parsed.Session.Model != "gpt-5.5" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Usage.TotalTokens != 1120 {
		t.Fatalf("total tokens = %d", parsed.Usage.TotalTokens)
	}
	if parsed.Usage.ContextCompressionTokens != 40 {
		t.Fatalf("context compression tokens = %d", parsed.Usage.ContextCompressionTokens)
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
	if parsed.ModelCall[0].ContextCompressionTokens != 40 {
		t.Fatalf("model call context compression tokens = %d", parsed.ModelCall[0].ContextCompressionTokens)
	}
	if parsed.Session.ParseStatus != "ok" {
		t.Fatalf("parse status = %q, warnings: %v", parsed.Session.ParseStatus, parsed.Warnings)
	}
}

func TestParseFileAddsCodexCompactedSnapshotToContextCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-compacted.jsonl")
	content := `{"timestamp":"2026-06-26T10:00:00Z","type":"session_meta","payload":{"session_id":"codex_compact_sess","cwd":"D:\\workspace\\project","originator":"codex_cli","thread_source":"local","model_provider":"openai"}}
{"timestamp":"2026-06-26T10:00:01Z","type":"turn_context","payload":{"model":"gpt-5.5","cwd":"D:\\workspace\\project"}}
{"timestamp":"2026-06-26T10:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":240000,"cached_input_tokens":230000,"output_tokens":500,"reasoning_output_tokens":0,"total_tokens":240500},"last_token_usage":{"input_tokens":240000,"cached_input_tokens":230000,"output_tokens":500,"reasoning_output_tokens":0,"total_tokens":240500},"model_context_window":258400}}}
{"timestamp":"2026-06-26T10:00:03Z","type":"compacted","payload":{"window_id":"win_2","previous_window_id":"win_1","replacement_history":[{"type":"message","role":"user","content":[{"type":"input_text","text":"continue"}]},{"type":"compaction","id":"comp_1","encrypted_content":"opaque"}]}}
{"timestamp":"2026-06-26T10:00:04Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":240000,"cached_input_tokens":230000,"output_tokens":500,"reasoning_output_tokens":0,"total_tokens":240500},"last_token_usage":{"input_tokens":0,"cached_input_tokens":0,"output_tokens":0,"reasoning_output_tokens":0,"total_tokens":12000},"model_context_window":258400}}}
{"timestamp":"2026-06-26T10:00:05Z","type":"event_msg","payload":{"type":"context_compacted"}}
{"timestamp":"2026-06-26T10:00:06Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":255000,"cached_input_tokens":238000,"output_tokens":700,"reasoning_output_tokens":0,"total_tokens":255700},"last_token_usage":{"input_tokens":15000,"cached_input_tokens":8000,"output_tokens":200,"reasoning_output_tokens":0,"total_tokens":15200},"model_context_window":258400}}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 10, 20)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Usage.InputTokens != 255000 || parsed.Usage.CachedInputTokens != 238000 || parsed.Usage.OutputTokens != 700 || parsed.Usage.TotalTokens != 255700 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if parsed.Usage.ContextCompressionTokens != 228000 {
		t.Fatalf("context compression tokens = %d", parsed.Usage.ContextCompressionTokens)
	}
	if len(parsed.ModelCall) != 2 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	for _, call := range parsed.ModelCall {
		if call.ContextCompressionTokens != 0 {
			t.Fatalf("model call context compression tokens = %d", call.ContextCompressionTokens)
		}
	}
	if parsed.Events[3].Kind != "session" || parsed.Events[3].Summary != "Context compacted" {
		t.Fatalf("compacted event = %+v", parsed.Events[3])
	}
	if parsed.Events[5].Kind != "session" || parsed.Events[5].Summary != "Context compacted" {
		t.Fatalf("context_compacted event = %+v", parsed.Events[5])
	}
}

func TestParseFileWithHashMatchesHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "run.jsonl")
	content := `{"timestamp":"2026-06-26T10:00:00Z","type":"session_meta","payload":{"session_id":"hash_sess","cwd":"D:\\workspace\\project","originator":"codex_cli","thread_source":"local","model_provider":"openai"}}
{"timestamp":"2026-06-26T10:00:01Z","type":"turn_context","payload":{"model":"gpt-5","cwd":"D:\\workspace\\project"}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, hash, err := ParseFileWithHash(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	wantHash, err := HashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if hash != wantHash {
		t.Fatalf("hash = %q, want %q", hash, wantHash)
	}
	if parsed.Session.SessionKey != "hash_sess" {
		t.Fatalf("session key = %q", parsed.Session.SessionKey)
	}
}

func TestParseFileSupportsClaudeStyleMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-session.jsonl")
	content := `{"type":"user","sessionId":"claude_sess","cwd":"/workspace/project","timestamp":"2026-06-26T10:00:00Z","message":{"role":"user","content":"run tests"}}
{"type":"assistant","sessionId":"claude_sess","cwd":"/workspace/project","timestamp":"2026-06-26T10:00:02Z","message":{"role":"assistant","model":"claude-opus-4","usage":{"input_tokens":100,"cache_creation_input_tokens":10,"cache_read_input_tokens":25,"output_tokens":40},"content":[{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"go test ./..."}}]}}
{"type":"user","sessionId":"claude_sess","cwd":"/workspace/project","timestamp":"2026-06-26T10:00:05Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.SessionKey != "claude_sess" {
		t.Fatalf("session key = %q", parsed.Session.SessionKey)
	}
	if parsed.Session.ProjectPath != "/workspace/project" {
		t.Fatalf("project path = %q", parsed.Session.ProjectPath)
	}
	if parsed.Session.Model != "claude-opus-4" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Usage.InputTokens != 110 || parsed.Usage.CachedInputTokens != 25 || parsed.Usage.OutputTokens != 40 || parsed.Usage.TotalTokens != 175 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if len(parsed.ModelCall) != 1 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if len(parsed.ToolCall) != 1 {
		t.Fatalf("tool calls = %d", len(parsed.ToolCall))
	}
	if parsed.ToolCall[0].ToolName != "Bash" || parsed.ToolCall[0].Status != "completed" || parsed.ToolCall[0].DurationMS != 3000 {
		t.Fatalf("tool call = %+v", parsed.ToolCall[0])
	}
}

func TestParseFileAddsClaudeCompactMetadataToContextCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-compact-session.jsonl")
	content := `{"type":"user","sessionId":"claude_compact_sess","cwd":"/workspace/project","timestamp":"2026-06-26T10:00:00Z","message":{"role":"user","content":"run tests"}}
{"type":"assistant","sessionId":"claude_compact_sess","cwd":"/workspace/project","timestamp":"2026-06-26T10:00:02Z","message":{"role":"assistant","model":"claude-sonnet-4","usage":{"input_tokens":100,"cache_creation_input_tokens":10,"cache_read_input_tokens":25,"output_tokens":40},"content":[{"type":"text","text":"working"}]}}
{"type":"system","sessionId":"claude_compact_sess","cwd":"/workspace/project","timestamp":"2026-06-26T10:00:04Z","compactMetadata":{"trigger":"auto","preTokens":170000,"postTokens":12000,"durationMs":40000},"isCompactSummary":true}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Usage.InputTokens != 110 || parsed.Usage.CachedInputTokens != 25 || parsed.Usage.OutputTokens != 40 || parsed.Usage.TotalTokens != 175 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if parsed.Usage.ContextCompressionTokens != 158000 {
		t.Fatalf("context compression tokens = %d", parsed.Usage.ContextCompressionTokens)
	}
	if parsed.Usage.Source != "actual" {
		t.Fatalf("usage source = %q", parsed.Usage.Source)
	}
	if len(parsed.ModelCall) != 1 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if parsed.ModelCall[0].ContextCompressionTokens != 0 {
		t.Fatalf("model call context compression tokens = %d", parsed.ModelCall[0].ContextCompressionTokens)
	}
}

func TestParseFileSupportsCodeBuddyMessagesAndTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codebuddy-session.jsonl")
	content := `{"id":"msg_1","timestamp":1782468000000,"type":"message","role":"user","content":[{"type":"input_text","text":"run tests"}],"providerData":{"agent":"cli"},"sessionId":"cb_sess","cwd":"D:\\workspace\\project"}
{"id":"call_1","parentId":"msg_1","timestamp":1782468002000,"type":"function_call","providerData":{"model":"claude-sonnet-4.6-1m","usage":{"inputTokens":100,"outputTokens":20,"totalTokens":120,"inputTokensDetails":[{"cached_tokens":10}],"outputTokensDetails":[{"reasoning_tokens":3}]},"agent":"cli","argumentsDisplayText":"go test ./..."},"callId":"toolu_1","name":"Bash","arguments":{"command":"go test ./..."},"sessionId":"cb_sess","cwd":"D:\\workspace\\project"}
{"id":"result_1","parentId":"call_1","timestamp":1782468005000,"type":"function_call_result","name":"Bash","callId":"toolu_1","status":"completed","output":{"type":"text","text":"ok"},"providerData":{"toolResult":{"content":"ok"}},"sessionId":"cb_sess","cwd":"D:\\workspace\\project"}
{"id":"msg_2","parentId":"result_1","timestamp":1782468006000,"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"done"}],"providerData":{"model":"claude-sonnet-4.6-1m","usage":{"inputTokens":50,"outputTokens":5,"totalTokens":55},"agent":"cli"},"sessionId":"cb_sess","cwd":"D:\\workspace\\project"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.SessionKey != "cb_sess" {
		t.Fatalf("session key = %q", parsed.Session.SessionKey)
	}
	if parsed.Session.ProjectPath != "D:\\workspace\\project" {
		t.Fatalf("project path = %q", parsed.Session.ProjectPath)
	}
	if parsed.Session.Model != "claude-sonnet-4.6-1m" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Session.AgentNickname != "cli" {
		t.Fatalf("agent nickname = %q", parsed.Session.AgentNickname)
	}
	if parsed.Usage.InputTokens != 150 || parsed.Usage.CachedInputTokens != 10 || parsed.Usage.OutputTokens != 25 || parsed.Usage.ReasoningOutputTokens != 3 || parsed.Usage.TotalTokens != 175 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if len(parsed.ModelCall) != 2 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if len(parsed.ToolCall) != 1 {
		t.Fatalf("tool calls = %d", len(parsed.ToolCall))
	}
	if parsed.ToolCall[0].ToolName != "Bash" || parsed.ToolCall[0].Status != "completed" || parsed.ToolCall[0].DurationMS != 3000 {
		t.Fatalf("tool call = %+v", parsed.ToolCall[0])
	}
	if parsed.ToolCall[0].InputSummary == "" || parsed.ToolCall[0].OutputSummary != "ok" {
		t.Fatalf("tool summaries = input %q output %q", parsed.ToolCall[0].InputSummary, parsed.ToolCall[0].OutputSummary)
	}
	if parsed.Session.ParseStatus != "ok" {
		t.Fatalf("parse status = %q, warnings: %v", parsed.Session.ParseStatus, parsed.Warnings)
	}
}

func TestParseFileSupportsCodeBuddyCompactReplacementSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codebuddy-compact-replacement.jsonl")
	content := `{"id":"msg_1","timestamp":1782468000000,"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"working"}],"providerData":{"model":"claude-sonnet-4.6-1m","usage":{"inputTokens":180000,"outputTokens":100,"totalTokens":180100},"agent":"cli"},"sessionId":"cb_compact_sess","cwd":"D:\\workspace\\project"}
{"id":"compact_1","parentId":"msg_1","timestamp":1782468001000,"type":"message","role":"user","content":[{"type":"input_text","text":"<conversation_history_summary>summary</conversation_history_summary>"}],"providerData":{"compactType":"pre-message-auto","isCompacted":true,"isCompactInternal":true,"isSummary":true},"sessionId":"cb_compact_sess","cwd":"D:\\workspace\\project"}
{"id":"msg_2","parentId":"compact_1","timestamp":1782468002000,"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"done"}],"providerData":{"model":"claude-sonnet-4.6-1m","usage":{"inputTokens":12000,"outputTokens":30,"totalTokens":12030},"agent":"cli"},"sessionId":"cb_compact_sess","cwd":"D:\\workspace\\project"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Usage.InputTokens != 192000 || parsed.Usage.OutputTokens != 130 || parsed.Usage.TotalTokens != 192130 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if parsed.Usage.ContextCompressionTokens != 168000 {
		t.Fatalf("context compression tokens = %d", parsed.Usage.ContextCompressionTokens)
	}
	if len(parsed.ModelCall) != 2 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if parsed.ModelCall[0].ContextCompressionTokens != 0 || parsed.ModelCall[1].ContextCompressionTokens != 0 {
		t.Fatalf("model call compression tokens = %+v", parsed.ModelCall)
	}
	if parsed.Events[1].Kind != "session" || parsed.Events[1].Summary != "Context compacted" {
		t.Fatalf("compact event = %+v", parsed.Events[1])
	}
}

func TestParseFileSupportsCodeBuddyCompactAgentSummary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "codebuddy-compact-agent.jsonl")
	content := `{"id":"msg_1","timestamp":1782468000000,"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"working"}],"providerData":{"model":"claude-sonnet-4.6-1m","usage":{"inputTokens":90000,"outputTokens":80,"totalTokens":90080},"agent":"cli"},"sessionId":"cb_compact_agent_sess","cwd":"D:\\workspace\\project"}
{"id":"compact_1","parentId":"msg_1","timestamp":1782468001000,"type":"message","role":"assistant","content":[{"type":"output_text","text":"<conversation_history_summary><summary>old work</summary></conversation_history_summary>"}],"providerData":{"agent":"compact","compactType":"user-command"},"sessionId":"cb_compact_agent_sess","cwd":"D:\\workspace\\project"}
{"id":"msg_2","parentId":"compact_1","timestamp":1782468002000,"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"done"}],"providerData":{"model":"claude-sonnet-4.6-1m","usage":{"inputTokens":8000,"outputTokens":20,"totalTokens":8020},"agent":"cli"},"sessionId":"cb_compact_agent_sess","cwd":"D:\\workspace\\project"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Usage.ContextCompressionTokens != 82000 {
		t.Fatalf("context compression tokens = %d", parsed.Usage.ContextCompressionTokens)
	}
	if len(parsed.ModelCall) != 2 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if parsed.Events[1].Kind != "session" || parsed.Events[1].Summary != "Context compacted" {
		t.Fatalf("compact event = %+v", parsed.Events[1])
	}
}

func TestParseFileSupportsCursorRoleMessagesAndTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cursor-session.jsonl")
	content := `{"id":"msg_1","timestampMs":1782468000000,"role":"user","content":"run tests","providerData":{"agent":"cursor"},"metadata":{"source":"cursor"},"session_id":"cursor_sess","cwd":"D:\\workspace\\project"}
{"id":"msg_2","timestampMs":1782468002000,"role":"assistant","model":"gpt-5.2-cursor","usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":10,"total_tokens":130},"content":[{"type":"tool_use","id":"toolu_cursor","name":"terminal","input":{"command":"go test ./..."}}],"providerData":{"agent":"cursor"},"session_id":"cursor_sess","cwd":"D:\\workspace\\project"}
{"id":"msg_3","timestampMs":1782468005000,"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_cursor","content":"ok"}],"providerData":{"agent":"cursor"},"session_id":"cursor_sess","cwd":"D:\\workspace\\project"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.SessionKey != "cursor_sess" {
		t.Fatalf("session key = %q", parsed.Session.SessionKey)
	}
	if parsed.Session.ProjectPath != "D:\\workspace\\project" {
		t.Fatalf("project path = %q", parsed.Session.ProjectPath)
	}
	if parsed.Session.Model != "gpt-5.2-cursor" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Session.AgentNickname != "cursor" {
		t.Fatalf("agent nickname = %q", parsed.Session.AgentNickname)
	}
	if parsed.Session.Originator != "cursor" {
		t.Fatalf("originator = %q", parsed.Session.Originator)
	}
	if parsed.Usage.InputTokens != 100 || parsed.Usage.CachedInputTokens != 20 || parsed.Usage.OutputTokens != 10 || parsed.Usage.TotalTokens != 130 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if len(parsed.ModelCall) != 1 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if len(parsed.ToolCall) != 1 {
		t.Fatalf("tool calls = %d", len(parsed.ToolCall))
	}
	if parsed.ToolCall[0].ToolName != "terminal" || parsed.ToolCall[0].Status != "completed" || parsed.ToolCall[0].DurationMS != 3000 {
		t.Fatalf("tool call = %+v", parsed.ToolCall[0])
	}
	if parsed.Session.ParseStatus != "ok" {
		t.Fatalf("parse status = %q, warnings: %v", parsed.Session.ParseStatus, parsed.Warnings)
	}
}

func TestParseFileSupportsWorkBuddyMessagesAndTools(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workbuddy-session.jsonl")
	content := `{"id":"msg_1","timestamp":1782468000000,"type":"message","role":"user","content":[{"type":"input_text","text":"run tests"}],"providerData":{"agent":"cli"},"sessionId":"wb_sess","cwd":"C:\\workspace\\project"}
{"id":"call_1","parentId":"msg_1","timestamp":1782468002000,"type":"function_call","providerData":{"model":"deepseek-v4-flash","requestModelId":"deepseek-v4-flash","requestModelName":"Deepseek-V4-Flash","agent":"cli","argumentsDisplayText":"go test ./...","usage":{"requests":1,"inputTokens":120,"outputTokens":30,"totalTokens":150,"inputTokensDetails":[{"cached_tokens":40}],"outputTokensDetails":[{"reasoning_tokens":8}]}},"callId":"tool_1","name":"Bash","arguments":"{\"command\":\"go test ./...\"}","sessionId":"wb_sess","message":{"usage":{"input_tokens":120,"output_tokens":30,"total_tokens":150,"cache_read_input_tokens":40}},"cwd":"C:\\workspace\\project"}
{"id":"result_1","parentId":"call_1","timestamp":1782468005000,"type":"function_call_result","name":"Bash","callId":"tool_1","status":"completed","output":{"type":"text","text":"ok"},"providerData":{"toolResult":{"content":"ok"},"agent":"cli"},"sessionId":"wb_sess","cwd":"C:\\workspace\\project"}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Session.SessionKey != "wb_sess" {
		t.Fatalf("session key = %q", parsed.Session.SessionKey)
	}
	if parsed.Session.ProjectPath != "C:\\workspace\\project" {
		t.Fatalf("project path = %q", parsed.Session.ProjectPath)
	}
	if parsed.Session.Model != "deepseek-v4-flash" {
		t.Fatalf("model = %q", parsed.Session.Model)
	}
	if parsed.Session.AgentNickname != "cli" {
		t.Fatalf("agent nickname = %q", parsed.Session.AgentNickname)
	}
	if parsed.Usage.InputTokens != 120 || parsed.Usage.CachedInputTokens != 40 || parsed.Usage.OutputTokens != 30 || parsed.Usage.ReasoningOutputTokens != 8 || parsed.Usage.TotalTokens != 150 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if len(parsed.ModelCall) != 1 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
	if len(parsed.ToolCall) != 1 {
		t.Fatalf("tool calls = %d", len(parsed.ToolCall))
	}
	if parsed.ToolCall[0].ToolName != "Bash" || parsed.ToolCall[0].Status != "completed" || parsed.ToolCall[0].DurationMS != 3000 {
		t.Fatalf("tool call = %+v", parsed.ToolCall[0])
	}
	if parsed.ToolCall[0].InputSummary == "" || parsed.ToolCall[0].OutputSummary != "ok" {
		t.Fatalf("tool summaries = input %q output %q", parsed.ToolCall[0].InputSummary, parsed.ToolCall[0].OutputSummary)
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
	if parsed.Usage.InputTokens != 179 || parsed.Usage.CachedInputTokens != 25 || parsed.Usage.OutputTokens != 46 || parsed.Usage.ReasoningOutputTokens != 1 || parsed.Usage.TotalTokens != 225 {
		t.Fatalf("usage = %+v", parsed.Usage)
	}
	if len(parsed.ModelCall) != 3 {
		t.Fatalf("model calls = %d", len(parsed.ModelCall))
	}
}

func TestUsageFromValueSupportsProviderReasoningShapes(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  struct {
			input       int64
			cached      int64
			output      int64
			reasoning   int64
			compression int64
			total       int64
		}
	}{
		{
			name: "openrouter snake details",
			value: map[string]any{
				"prompt_tokens":     float64(100),
				"completion_tokens": float64(40),
				"output_tokens_details": map[string]any{
					"reasoning_tokens": float64(12),
				},
				"input_tokens_details": map[string]any{
					"cached_tokens": float64(25),
				},
				"total_tokens": float64(140),
			},
			want: struct {
				input       int64
				cached      int64
				output      int64
				reasoning   int64
				compression int64
				total       int64
			}{input: 100, cached: 25, output: 40, reasoning: 12, total: 140},
		},
		{
			name: "anthropic thinking tokens",
			value: map[string]any{
				"input_tokens":    float64(80),
				"output_tokens":   float64(30),
				"thinking_tokens": float64(7),
				"total_tokens":    float64(110),
			},
			want: struct {
				input       int64
				cached      int64
				output      int64
				reasoning   int64
				compression int64
				total       int64
			}{input: 80, output: 30, reasoning: 7, total: 110},
		},
		{
			name: "gemini usage metadata",
			value: map[string]any{
				"usageMetadata": map[string]any{
					"promptTokenCount":        float64(100),
					"candidatesTokenCount":    float64(20),
					"thoughtsTokenCount":      float64(15),
					"cachedContentTokenCount": float64(40),
					"totalTokenCount":         float64(135),
				},
			},
			want: struct {
				input       int64
				cached      int64
				output      int64
				reasoning   int64
				compression int64
				total       int64
			}{input: 100, cached: 40, output: 35, reasoning: 15, total: 135},
		},
		{
			name: "gemini snake usage metadata",
			value: map[string]any{
				"usage_metadata": map[string]any{
					"prompt_token_count":         float64(90),
					"candidates_token_count":     float64(25),
					"thoughts_token_count":       float64(5),
					"cached_content_token_count": float64(10),
					"total_token_count":          float64(120),
				},
			},
			want: struct {
				input       int64
				cached      int64
				output      int64
				reasoning   int64
				compression int64
				total       int64
			}{input: 90, cached: 10, output: 30, reasoning: 5, total: 120},
		},
		{
			name: "context compression tokens",
			value: map[string]any{
				"inputTokens":              float64(120),
				"outputTokens":             float64(40),
				"contextCompressionTokens": float64(15),
			},
			want: struct {
				input       int64
				cached      int64
				output      int64
				reasoning   int64
				compression int64
				total       int64
			}{input: 120, output: 40, compression: 15, total: 175},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usageFromValue(tt.value)
			if got.InputTokens != tt.want.input || got.CachedInputTokens != tt.want.cached || got.OutputTokens != tt.want.output || got.ReasoningOutputTokens != tt.want.reasoning || got.ContextCompressionTokens != tt.want.compression || got.TotalTokens != tt.want.total {
				t.Fatalf("usage = %+v", got)
			}
		})
	}
}

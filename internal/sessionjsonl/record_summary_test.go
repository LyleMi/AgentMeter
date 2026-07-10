package sessionjsonl

import "testing"

func TestClassifyRecord(t *testing.T) {
	tests := []struct {
		name string
		raw  rawRecord
		want string
	}{
		{name: "payload takes precedence", raw: rawRecord{Type: "user", Payload: map[string]any{"type": "function_call"}}, want: "user"},
		{name: "role-only assistant", raw: rawRecord{Role: "assistant"}, want: "model"},
		{name: "message user", raw: rawRecord{Type: "message", Role: "user"}, want: "user"},
		{name: "message unknown role", raw: rawRecord{Type: "message", Role: "tool"}, want: "message"},
		{name: "native tool result", raw: rawRecord{Type: "function_call_result"}, want: "tool"},
		{name: "unknown tool payload", raw: rawRecord{Payload: map[string]any{"type": "future_tool_event"}}, want: "tool"},
		{name: "unknown event", raw: rawRecord{Type: "future_event"}, want: "event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyRecord(tt.raw); got != tt.want {
				t.Fatalf("classifyRecord() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSummarizeRecord(t *testing.T) {
	tests := []struct {
		name string
		raw  rawRecord
		want string
	}{
		{name: "role-only content", raw: rawRecord{Role: "user", Content: "hello"}, want: "user: hello"},
		{name: "reasoning raw content", raw: rawRecord{Type: "reasoning", RawContent: "think"}, want: "Reasoning: think"},
		{name: "title", raw: rawRecord{Type: "ai-title", AITitle: "A title"}, want: "AI title: A title"},
		{name: "successful patch", raw: rawRecord{Payload: map[string]any{"type": "patch_apply_end", "success": true}}, want: "Patch applied"},
		{name: "failed patch", raw: rawRecord{Payload: map[string]any{"type": "patch_apply_end", "success": false}}, want: "Patch failed"},
		{name: "unknown payload", raw: rawRecord{Payload: map[string]any{"type": "future_event"}}, want: "future_event"},
		{name: "unknown top type", raw: rawRecord{Type: "future_event"}, want: "future_event"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summarizeRecord(tt.raw); got != tt.want {
				t.Fatalf("summarizeRecord() = %q, want %q", got, tt.want)
			}
		})
	}
}

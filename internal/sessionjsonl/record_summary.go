package sessionjsonl

import (
	"fmt"
	"strings"
)

var topTypeKinds = map[string]string{
	"session_meta": "session",
	"turn_context": "session",
	"user":         "user",
	"assistant":    "model",
	"system":       "session",
	"summary":      "session",
}

var payloadTypeKinds = map[string]string{
	"user_message":            "user",
	"agent_message":           "model",
	"reasoning":               "model",
	"token_count":             "model",
	"task_started":            "model",
	"task_complete":           "model",
	"context_compacted":       "session",
	"function_call":           "tool",
	"function_call_output":    "tool",
	"custom_tool_call":        "tool",
	"custom_tool_call_output": "tool",
	"web_search_call":         "tool",
	"web_search_end":          "tool",
	"tool_search_call":        "tool",
	"tool_search_output":      "tool",
	"patch_apply_end":         "tool",
	"turn_aborted":            "error",
}

var recordTypeKinds = map[string]string{
	"reasoning":             "model",
	"function_call":         "tool",
	"function_call_result":  "tool",
	"file-history-snapshot": "session",
	"ai-title":              "session",
	"summary":               "session",
	"compacted":             "session",
}

var payloadTypeSummaries = map[string]string{
	"web_search_call":         "Web search",
	"function_call_output":    "Tool output",
	"custom_tool_call_output": "Tool output",
	"tool_search_output":      "Tool output",
	"web_search_end":          "Web search completed",
	"task_started":            "Turn started",
	"task_complete":           "Turn completed",
	"context_compacted":       "Context compacted",
	"user_message":            "User message",
	"reasoning":               "Reasoning",
}

var topTypeSummaries = map[string]string{
	"session_meta": "Session metadata",
	"user":         "User message",
	"assistant":    "Assistant message",
	"system":       "System message",
	"summary":      "Summary",
}

func classifyRecord(raw rawRecord) string {
	if stringValue(raw.Payload, "type") != "" {
		return classify(raw.Type, raw.Payload)
	}
	if isCodeBuddyCompactSummary(raw) {
		return "session"
	}
	role := stringFromAny(raw.Role)
	if raw.Type == "" {
		if kind := kindForRole(role); kind != "" {
			return kind
		}
	}
	if raw.Type == "message" {
		return firstNonEmpty(kindForRole(role), "message")
	}
	if kind, ok := recordTypeKinds[raw.Type]; ok {
		return kind
	}
	return classify(raw.Type, raw.Payload)
}

func kindForRole(role string) string {
	switch role {
	case "user":
		return "user"
	case "assistant":
		return "model"
	case "system":
		return "session"
	default:
		return ""
	}
}

func summarizeRecord(raw rawRecord) string {
	if stringValue(raw.Payload, "type") != "" {
		return summarize(raw.Type, raw.Payload)
	}
	if isCodeBuddyCompactSummary(raw) {
		return "Context compacted"
	}
	role := stringFromAny(raw.Role)
	if raw.Type == "" && role != "" {
		return role + ": " + preview(contentText(raw.Content), 180)
	}
	switch raw.Type {
	case "message":
		return firstNonEmpty(role, "message") + ": " + preview(contentText(raw.Content), 180)
	case "reasoning":
		return "Reasoning: " + preview(firstNonEmpty(contentText(raw.Content), stringFromAny(raw.RawContent)), 180)
	case "function_call":
		return "Tool call: " + recordToolName(raw)
	case "function_call_result":
		return "Tool output: " + recordToolName(raw)
	case "file-history-snapshot":
		return "File history snapshot"
	case "ai-title":
		return "AI title: " + preview(stringFromAny(raw.AITitle), 180)
	case "summary":
		return "Summary: " + preview(stringFromAny(raw.Summary), 180)
	case "compacted":
		return "Context compacted"
	default:
		return summarize(raw.Type, raw.Payload)
	}
}

func classify(topType string, payload map[string]any) string {
	if kind, ok := topTypeKinds[topType]; ok {
		return kind
	}
	payloadType := stringValue(payload, "type")
	if payloadType == "message" {
		if stringValue(payload, "role") == "user" {
			return "user"
		}
		return "message"
	}
	if kind, ok := payloadTypeKinds[payloadType]; ok {
		return kind
	}
	if strings.Contains(payloadType, "tool") || strings.Contains(payloadType, "function") {
		return "tool"
	}
	return "event"
}

func summarize(topType string, payload map[string]any) string {
	if summary, ok := fixedTopTypeSummary(topType); ok {
		return summary
	}
	if topType == "turn_context" {
		return "Turn context: " + firstNonEmpty(stringValue(payload, "model"), "unknown model")
	}
	payloadType := stringValue(payload, "type")
	if summary, ok := payloadTypeSummaries[payloadType]; ok {
		return summary
	}
	return summarizeDynamicPayload(payloadType, payload, topType)
}

func fixedTopTypeSummary(topType string) (string, bool) {
	summary, ok := topTypeSummaries[topType]
	return summary, ok
}

func summarizeDynamicPayload(payloadType string, payload map[string]any, topType string) string {
	switch payloadType {
	case "token_count":
		usage := readUsage(payload, "total_token_usage")
		return fmt.Sprintf("Token usage: %d total", usage.TotalTokens)
	case "function_call", "custom_tool_call", "tool_search_call":
		return "Tool call: " + toolName(payloadType, payload)
	case "patch_apply_end":
		if boolValue(payload, "success") {
			return "Patch applied"
		}
		return "Patch failed"
	case "turn_aborted":
		return "Turn aborted: " + stringValue(payload, "reason")
	case "agent_message":
		return "Agent: " + preview(stringValue(payload, "message"), 180)
	case "message":
		return firstNonEmpty(stringValue(payload, "role"), "message") + ": " + preview(stringValue(payload, "content"), 180)
	default:
		return firstNonEmpty(payloadType, topType)
	}
}

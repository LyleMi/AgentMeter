package sessionjsonl

import (
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func startTool(payload map[string]any, payloadType string, ts time.Time, lineNo int) pendingTool {
	return pendingTool{
		callID:       firstNonEmpty(stringValue(payload, "call_id"), stringValue(payload, "id")),
		name:         toolName(payloadType, payload),
		startedAt:    ts,
		inputSummary: preview(inputSummary(payload), 500),
		rawLine:      lineNo,
		status:       firstNonEmpty(stringValue(payload, "status"), "started"),
	}
}

func startToolRecord(raw rawRecord, ts time.Time, lineNo int) pendingTool {
	return pendingTool{
		callID:       recordCallID(raw),
		name:         recordToolName(raw),
		startedAt:    ts,
		inputSummary: preview(inputSummaryRecord(raw), 500),
		rawLine:      lineNo,
		status:       firstNonEmpty(stringFromAny(raw.Status), "started"),
	}
}

func finishToolCall(call pendingTool, completed completedTool, endedAt time.Time, lineNo int) (model.ToolCall, int64) {
	duration := durationMS(call.startedAt, endedAt)
	return model.ToolCall{
		StartedAt:         call.startedAt,
		EndedAt:           endedAt,
		DurationMS:        duration,
		ToolName:          firstNonEmpty(call.name, completed.name),
		Status:            firstNonEmpty(completed.status, call.status, "completed"),
		InputSummary:      call.inputSummary,
		OutputSummary:     preview(completed.outputSummary, 500),
		Error:             completed.error,
		CallID:            firstNonEmpty(call.callID, completed.callID),
		RawEventLine:      call.rawLine,
		RawStartEventLine: call.rawLine,
		RawEndEventLine:   lineNo,
	}, duration
}

func recordCallID(raw rawRecord) string {
	return firstNonEmpty(
		stringFromAny(raw.CallID),
		stringFromAny(raw.CallIDSnake),
		stringValue(raw.ProviderData, "callId"),
		stringValue(raw.ProviderData, "call_id"),
	)
}

func recordToolName(raw rawRecord) string {
	return firstNonEmpty(stringFromAny(raw.Name), stringValue(raw.ProviderData, "name"), raw.Type, "tool")
}

func inputSummaryRecord(raw rawRecord) string {
	for _, value := range []any{raw.Arguments, raw.ProviderData["argumentsDisplayText"], raw.ProviderData["arguments"]} {
		if summary := valueToString(value); summary != "" {
			return summary
		}
	}
	return ""
}

func outputSummaryRecord(raw rawRecord) string {
	for _, value := range []any{raw.Output, raw.ProviderData["toolResult"], raw.ProviderData["message"]} {
		if summary := outputValueToString(value); summary != "" {
			return summary
		}
	}
	return ""
}

func outputStatusRecord(raw rawRecord) (string, string) {
	status := firstNonEmpty(stringFromAny(raw.Status), stringValue(raw.ProviderData, "status"))
	errorText := firstNonEmpty(stringValue(raw.ProviderData, "error"), stringValue(raw.ProviderData, "stderr"))
	if status == "" && errorText != "" {
		status = "failed"
	}
	return firstNonEmpty(status, "completed"), preview(errorText, 500)
}

func toolName(payloadType string, payload map[string]any) string {
	if name := stringValue(payload, "name"); name != "" {
		return name
	}
	switch payloadType {
	case "web_search_call", "web_search_end":
		return "web_search"
	case "tool_search_call", "tool_search_output":
		return "tool_search"
	case "patch_apply_end":
		return "apply_patch"
	default:
		return firstNonEmpty(payloadType, "tool")
	}
}

func sessionIDFromRecord(raw rawRecord) string {
	message := mapFromAny(raw.Message)
	return firstNonEmpty(
		stringFromAny(raw.SessionID),
		stringFromAny(raw.SessionIDSnake),
		stringValue(raw.Metadata, "sessionId"),
		stringValue(raw.Metadata, "session_id"),
		stringValue(raw.Metadata, "conversation_id"),
		stringValue(message, "sessionId"),
		stringValue(message, "session_id"),
		stringFromAny(raw.UUID),
	)
}

func toolUsesFromMessage(message map[string]any, ts time.Time, lineNo int) []pendingTool {
	var calls []pendingTool
	for _, item := range contentItems(message["content"]) {
		if stringValue(item, "type") != "tool_use" {
			continue
		}
		calls = append(calls, pendingTool{
			callID:       firstNonEmpty(stringValue(item, "id"), stringValue(item, "call_id"), stringValue(item, "tool_use_id")),
			name:         firstNonEmpty(stringValue(item, "name"), "tool"),
			startedAt:    ts,
			inputSummary: preview(valueToString(item["input"]), 500),
			rawLine:      lineNo,
			status:       "started",
		})
	}
	return calls
}

func toolResultsFromMessage(message map[string]any, _ time.Time, _ int) []completedTool {
	var results []completedTool
	for _, item := range contentItems(message["content"]) {
		if stringValue(item, "type") != "tool_result" {
			continue
		}
		status, errorText := messageToolResultStatus(item)
		results = append(results, completedTool{
			callID:        firstNonEmpty(stringValue(item, "tool_use_id"), stringValue(item, "call_id"), stringValue(item, "id")),
			name:          firstNonEmpty(stringValue(item, "name"), "tool"),
			status:        status,
			outputSummary: preview(valueToString(item["content"]), 500),
			error:         errorText,
		})
	}
	return results
}

func messageToolResultStatus(item map[string]any) (string, string) {
	if boolValue(item, "is_error") {
		return "failed", preview(valueToString(item["content"]), 500)
	}
	return "completed", ""
}

func contentItems(value any) []map[string]any {
	switch typed := value.(type) {
	case []any:
		items := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if asMap, ok := item.(map[string]any); ok {
				items = append(items, asMap)
			}
		}
		return items
	case map[string]any:
		return []map[string]any{typed}
	default:
		return nil
	}
}

func contentText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			if asMap, ok := item.(map[string]any); ok {
				if text := firstNonEmpty(stringValue(asMap, "text"), stringValue(asMap, "content"), stringValue(asMap, "summary")); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, " ")
	case map[string]any:
		return firstNonEmpty(stringValue(typed, "text"), stringValue(typed, "content"), stringValue(typed, "summary"))
	default:
		return valueToString(value)
	}
}

func inputSummary(payload map[string]any) string {
	return firstPayloadValue(payload, "arguments", "input", "query", "action")
}

func outputSummary(payload map[string]any) string {
	return firstPayloadValue(payload, "output", "stdout", "summary", "action")
}

func firstPayloadValue(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := payload[key]; ok {
			return valueToString(value)
		}
	}
	return ""
}

func outputValueToString(value any) string {
	if asMap, ok := value.(map[string]any); ok {
		if text := firstNonEmpty(stringValue(asMap, "text"), stringValue(asMap, "content"), stringValue(asMap, "title")); text != "" {
			return text
		}
	}
	return valueToString(value)
}

func outputStatus(payload map[string]any) (string, string) {
	status := stringValue(payload, "status")
	stderr := stringValue(payload, "stderr")
	errorText := firstNonEmpty(stderr, stringValue(payload, "error"))
	if status == "" {
		status = statusFromSuccess(payload["success"])
	}
	if status == "" && errorText != "" {
		status = "failed"
	}
	status = firstNonEmpty(status, "completed")
	if stderr != "" && status == "completed" {
		status = "failed"
	}
	return status, preview(errorText, 500)
}

func statusFromSuccess(value any) string {
	success, ok := value.(bool)
	if !ok {
		return ""
	}
	if success {
		return "completed"
	}
	return "failed"
}

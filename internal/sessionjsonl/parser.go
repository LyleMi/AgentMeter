package sessionjsonl

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type rawRecord struct {
	ID               any            `json:"id"`
	ParentID         any            `json:"parentId"`
	Timestamp        any            `json:"timestamp"`
	TimestampMS      any            `json:"timestamp_ms"`
	TimestampMSCamel any            `json:"timestampMs"`
	CreatedAt        any            `json:"created_at"`
	CreatedAtCamel   any            `json:"createdAt"`
	Type             string         `json:"type"`
	SessionID        any            `json:"sessionId"`
	SessionIDSnake   any            `json:"session_id"`
	UUID             any            `json:"uuid"`
	CWD              any            `json:"cwd"`
	Role             any            `json:"role"`
	Status           any            `json:"status"`
	Name             any            `json:"name"`
	CallID           any            `json:"callId"`
	CallIDSnake      any            `json:"call_id"`
	Arguments        any            `json:"arguments"`
	Output           any            `json:"output"`
	Content          any            `json:"content"`
	RawContent       any            `json:"rawContent"`
	Summary          any            `json:"summary"`
	AITitle          any            `json:"aiTitle"`
	Payload          map[string]any `json:"payload"`
	Data             map[string]any `json:"data"`
	Result           map[string]any `json:"result"`
	Response         map[string]any `json:"response"`
	Message          map[string]any `json:"message"`
	Usage            any            `json:"usage"`
	Model            any            `json:"model"`
	ModelName        any            `json:"model_name"`
	Metadata         map[string]any `json:"metadata"`
	ProviderData     map[string]any `json:"providerData"`
}

type pendingTool struct {
	callID       string
	name         string
	startedAt    time.Time
	inputSummary string
	rawLine      int
	status       string
}

type completedTool struct {
	callID        string
	name          string
	status        string
	outputSummary string
	error         string
}

func HashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func ParseFile(path string, sourceID, sourceFileID int64) (model.ParsedSession, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.ParsedSession{}, err
	}
	defer file.Close()

	accumulator := newParseAccumulator(path, sourceID, sourceFileID)
	reader := newRawRecordReader(path, file)
	for reader.Next() {
		accumulator.handleRecord(reader.Record())
	}
	if err := reader.Err(); err != nil {
		return accumulator.parsed, err
	}
	accumulator.addWarnings(reader.Warnings())
	return accumulator.finalize(), nil
}

func classifyRecord(raw rawRecord) string {
	if stringValue(raw.Payload, "type") != "" {
		return classify(raw.Type, raw.Payload)
	}
	if raw.Type == "" {
		switch stringFromAny(raw.Role) {
		case "user":
			return "user"
		case "assistant":
			return "model"
		case "system":
			return "session"
		}
	}
	switch raw.Type {
	case "message":
		switch stringFromAny(raw.Role) {
		case "user":
			return "user"
		case "assistant":
			return "model"
		default:
			return "message"
		}
	case "reasoning":
		return "model"
	case "function_call", "function_call_result":
		return "tool"
	case "file-history-snapshot", "ai-title", "summary":
		return "session"
	default:
		return classify(raw.Type, raw.Payload)
	}
}

func summarizeRecord(raw rawRecord) string {
	if stringValue(raw.Payload, "type") != "" {
		return summarize(raw.Type, raw.Payload)
	}
	if raw.Type == "" {
		if role := stringFromAny(raw.Role); role != "" {
			return role + ": " + preview(contentText(raw.Content), 180)
		}
	}
	switch raw.Type {
	case "message":
		role := firstNonEmpty(stringFromAny(raw.Role), "message")
		return role + ": " + preview(contentText(raw.Content), 180)
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
	default:
		return summarize(raw.Type, raw.Payload)
	}
}

func classify(topType string, payload map[string]any) string {
	pt := stringValue(payload, "type")
	switch topType {
	case "session_meta", "turn_context":
		return "session"
	case "user":
		return "user"
	case "assistant":
		return "model"
	case "system", "summary":
		return "session"
	}
	switch pt {
	case "user_message":
		return "user"
	case "message":
		if stringValue(payload, "role") == "user" {
			return "user"
		}
		return "message"
	case "agent_message", "reasoning", "token_count", "task_started", "task_complete":
		return "model"
	case "function_call", "function_call_output", "custom_tool_call", "custom_tool_call_output", "web_search_call", "web_search_end", "tool_search_call", "tool_search_output", "patch_apply_end":
		return "tool"
	case "turn_aborted":
		return "error"
	default:
		if strings.Contains(pt, "tool") || strings.Contains(pt, "function") {
			return "tool"
		}
		return "event"
	}
}

func summarize(topType string, payload map[string]any) string {
	pt := stringValue(payload, "type")
	switch topType {
	case "session_meta":
		return "Session metadata"
	case "turn_context":
		return "Turn context: " + firstNonEmpty(stringValue(payload, "model"), "unknown model")
	case "user":
		return "User message"
	case "assistant":
		return "Assistant message"
	case "system":
		return "System message"
	case "summary":
		return "Summary"
	}
	switch pt {
	case "token_count":
		usage := readUsage(payload, "total_token_usage")
		return fmt.Sprintf("Token usage: %d total", usage.TotalTokens)
	case "function_call", "custom_tool_call", "tool_search_call":
		return "Tool call: " + toolName(pt, payload)
	case "web_search_call":
		return "Web search"
	case "function_call_output", "custom_tool_call_output", "tool_search_output":
		return "Tool output"
	case "web_search_end":
		return "Web search completed"
	case "patch_apply_end":
		if boolValue(payload, "success") {
			return "Patch applied"
		}
		return "Patch failed"
	case "task_started":
		return "Turn started"
	case "task_complete":
		return "Turn completed"
	case "turn_aborted":
		return "Turn aborted: " + stringValue(payload, "reason")
	case "agent_message":
		return "Agent: " + preview(stringValue(payload, "message"), 180)
	case "user_message":
		return "User message"
	case "message":
		return firstNonEmpty(stringValue(payload, "role"), "message") + ": " + preview(stringValue(payload, "content"), 180)
	case "reasoning":
		return "Reasoning"
	default:
		if pt != "" {
			return pt
		}
		return topType
	}
}

func startTool(payload map[string]any, payloadType string, ts time.Time, lineNo int) pendingTool {
	callID := firstNonEmpty(stringValue(payload, "call_id"), stringValue(payload, "id"))
	return pendingTool{
		callID:       callID,
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
	callID := firstNonEmpty(call.callID, completed.callID)
	return model.ToolCall{
		StartedAt:         call.startedAt,
		EndedAt:           endedAt,
		DurationMS:        duration,
		ToolName:          firstNonEmpty(call.name, completed.name),
		Status:            firstNonEmpty(completed.status, call.status, "completed"),
		InputSummary:      call.inputSummary,
		OutputSummary:     preview(completed.outputSummary, 500),
		Error:             completed.error,
		CallID:            callID,
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
	for _, value := range []any{
		raw.Arguments,
		raw.ProviderData["argumentsDisplayText"],
		raw.ProviderData["arguments"],
	} {
		if summary := valueToString(value); summary != "" {
			return summary
		}
	}
	return ""
}

func outputSummaryRecord(raw rawRecord) string {
	for _, value := range []any{
		raw.Output,
		raw.ProviderData["toolResult"],
		raw.ProviderData["message"],
	} {
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
	if status == "" {
		status = "completed"
	}
	return status, preview(errorText, 500)
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
	return firstNonEmpty(
		stringFromAny(raw.SessionID),
		stringFromAny(raw.SessionIDSnake),
		stringValue(raw.Metadata, "sessionId"),
		stringValue(raw.Metadata, "session_id"),
		stringValue(raw.Metadata, "conversation_id"),
		stringValue(raw.Message, "sessionId"),
		stringValue(raw.Message, "session_id"),
		stringFromAny(raw.UUID),
	)
}

func toolUsesFromMessage(message map[string]any, ts time.Time, lineNo int) []pendingTool {
	var calls []pendingTool
	for _, item := range contentItems(message["content"]) {
		if stringValue(item, "type") != "tool_use" {
			continue
		}
		callID := firstNonEmpty(stringValue(item, "id"), stringValue(item, "call_id"), stringValue(item, "tool_use_id"))
		calls = append(calls, pendingTool{
			callID:       callID,
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
		callID := firstNonEmpty(stringValue(item, "tool_use_id"), stringValue(item, "call_id"), stringValue(item, "id"))
		status := "completed"
		errorText := ""
		if boolValue(item, "is_error") {
			status = "failed"
			errorText = preview(valueToString(item["content"]), 500)
		}
		results = append(results, completedTool{
			callID:        callID,
			name:          firstNonEmpty(stringValue(item, "name"), "tool"),
			status:        status,
			outputSummary: preview(valueToString(item["content"]), 500),
			error:         errorText,
		})
	}
	return results
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
				text := firstNonEmpty(
					stringValue(asMap, "text"),
					stringValue(asMap, "content"),
					stringValue(asMap, "summary"),
				)
				if text != "" {
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
	for _, key := range []string{"arguments", "input", "query", "action"} {
		if value, ok := payload[key]; ok {
			return valueToString(value)
		}
	}
	return ""
}

func outputSummary(payload map[string]any) string {
	for _, key := range []string{"output", "stdout", "summary", "action"} {
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
	errorText := stringValue(payload, "error")
	if status == "" {
		if v, ok := payload["success"]; ok {
			if asBool, ok := v.(bool); ok {
				if asBool {
					status = "completed"
				} else {
					status = "failed"
				}
			}
		}
	}
	if status == "" && errorText != "" {
		status = "failed"
	}
	if status == "" {
		status = "completed"
	}
	if stderr := stringValue(payload, "stderr"); stderr != "" {
		errorText = stderr
		if status == "completed" {
			status = "failed"
		}
	}
	return status, preview(errorText, 500)
}

func readUsage(payload map[string]any, key string) model.Usage {
	info, _ := payload["info"].(map[string]any)
	if info == nil {
		return model.Usage{}
	}
	return usageFromValue(info[key])
}

func headlessUsage(raw rawRecord) model.Usage {
	if usage := usageFromValue(raw.Usage); hasUsage(usage) {
		return usage
	}
	if raw.ProviderData != nil {
		if usage := usageFromValue(raw.ProviderData["usage"]); hasUsage(usage) {
			return usage
		}
		if usage := usageFromValue(raw.ProviderData["rawUsage"]); hasUsage(usage) {
			return usage
		}
	}
	for _, container := range []map[string]any{raw.Data, raw.Result, raw.Response, raw.Message} {
		if container == nil {
			continue
		}
		if usage := usageFromValue(container["usage"]); hasUsage(usage) {
			return usage
		}
		if usage := usageFromValue(container["usageMetadata"]); hasUsage(usage) {
			return usage
		}
		if usage := usageFromValue(container["usage_metadata"]); hasUsage(usage) {
			return usage
		}
	}
	return model.Usage{}
}

func usageFromValue(value any) model.Usage {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return model.Usage{}
	}
	geminiUsageMetadata := false
	if usageMetadata, ok := firstMap(raw, "usageMetadata", "usage_metadata"); ok {
		raw = usageMetadata
		geminiUsageMetadata = true
	}
	candidateOutput := firstInt64(raw, "candidatesTokenCount", "candidates_token_count")
	if candidateOutput > 0 {
		geminiUsageMetadata = true
	}
	inputIncludesCached := false
	input := firstInt64(raw, "input_tokens", "input", "inputTokens", "promptTokenCount", "prompt_token_count")
	if input > 0 {
		input += firstInt64(raw, "cache_creation_input_tokens", "cache_write_input_tokens", "cacheCreationInputTokens", "cacheWriteInputTokens")
	} else {
		input = firstInt64(raw, "prompt_tokens", "promptTokens")
		inputIncludesCached = input > 0
	}
	cached := firstInt64(raw, "cached_input_tokens", "cache_read_input_tokens", "cached_tokens", "cachedInputTokens", "cacheReadInputTokens", "cachedTokens", "cachedContentTokenCount", "cached_content_token_count")
	cached += nestedInt64(raw["inputTokensDetails"], "cached_tokens", "cachedTokens")
	cached += nestedInt64(raw["input_tokens_details"], "cached_tokens", "cachedTokens")
	cached += nestedInt64(raw["prompt_tokens_details"], "cached_tokens", "cachedTokens")
	cacheRead := firstInt64(raw, "cache_read_input_tokens", "cacheReadInputTokens")
	if cacheRead == 0 && !inputIncludesCached {
		cacheRead = cached
	}
	output := firstInt64(raw, "output_tokens", "completion_tokens", "output", "outputTokens", "completionTokens")
	if candidateOutput > 0 {
		output = candidateOutput
	}
	reasoning := firstInt64(raw, "reasoning_output_tokens", "reasoning_tokens", "reasoningOutputTokens", "reasoningTokens", "completion_thinking_tokens", "thinking_tokens", "thinkingTokens", "thoughtsTokenCount", "thoughts_token_count")
	reasoning += nestedInt64(raw["outputTokensDetails"], "reasoning_tokens", "reasoningTokens")
	reasoning += nestedInt64(raw["output_tokens_details"], "reasoning_tokens", "reasoningTokens")
	reasoning += nestedInt64(raw["completion_tokens_details"], "reasoning_tokens", "reasoningTokens")
	if geminiUsageMetadata && candidateOutput > 0 && reasoning > 0 {
		output += reasoning
	}
	total := firstInt64(raw, "total_tokens", "totalTokens", "totalTokenCount", "total_token_count")
	if total <= 0 && input+cached+output+reasoning > 0 {
		total = input + cacheRead + output
		if reasoning > output {
			total += reasoning
		}
	}
	return model.Usage{
		InputTokens:           input,
		CachedInputTokens:     cached,
		OutputTokens:          output,
		ReasoningOutputTokens: reasoning,
		TotalTokens:           total,
	}
}

func recordTimestamp(raw rawRecord) time.Time {
	for _, value := range []any{raw.Timestamp, raw.TimestampMS, raw.TimestampMSCamel, raw.CreatedAt, raw.CreatedAtCamel} {
		if ts := parseTimestampValue(value); !ts.IsZero() {
			return ts
		}
	}
	for _, container := range []map[string]any{raw.Data, raw.Result, raw.Response, raw.Message, raw.Metadata} {
		if container == nil {
			continue
		}
		for _, key := range []string{"timestamp", "created_at", "createdAt"} {
			if ts := parseTimestampValue(container[key]); !ts.IsZero() {
				return ts
			}
		}
	}
	return time.Time{}
}

func parseTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts
	}
	return time.Time{}
}

func parseTimestampValue(value any) time.Time {
	switch typed := value.(type) {
	case string:
		return parseTimestamp(typed)
	case float64:
		return timestampFromNumber(typed)
	case int64:
		return timestampFromNumber(float64(typed))
	case int:
		return timestampFromNumber(float64(typed))
	case json.Number:
		if n, err := typed.Int64(); err == nil {
			return timestampFromNumber(float64(n))
		}
		if f, err := typed.Float64(); err == nil {
			return timestampFromNumber(f)
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

func timestampFromPayload(payload map[string]any, key string) time.Time {
	raw, ok := payload[key]
	if !ok {
		return time.Time{}
	}
	switch value := raw.(type) {
	case float64:
		return timestampFromNumber(value)
	case string:
		if ts := parseTimestamp(value); !ts.IsZero() {
			return ts
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

func timestampFromNumber(value float64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	if value > 10_000_000_000 {
		sec := int64(value / 1000)
		nsec := int64((value - float64(sec)*1000) * 1_000_000)
		return time.Unix(sec, nsec).UTC()
	}
	sec := int64(value)
	nsec := int64((value - float64(sec)) * 1_000_000_000)
	return time.Unix(sec, nsec).UTC()
}

func durationMS(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}

func modelFromRecord(raw rawRecord) string {
	return firstNonEmpty(
		modelFromParts(raw.Model, raw.ModelName, raw.Metadata),
		modelFromMap(raw.ProviderData),
		modelFromMap(raw.Data),
		modelFromMap(raw.Result),
		modelFromMap(raw.Response),
		modelFromMap(raw.Message),
	)
}

func modelFromPayloadInfo(payload map[string]any) string {
	info, _ := payload["info"].(map[string]any)
	return firstNonEmpty(modelFromMap(payload), modelFromMap(info))
}

func modelFromMap(payload map[string]any) string {
	if payload == nil {
		return ""
	}
	metadata, _ := payload["metadata"].(map[string]any)
	return firstNonEmpty(
		modelFromParts(payload["model"], payload["model_name"], metadata),
		stringFromAny(payload["requestModelId"]),
		stringFromAny(payload["request_model_id"]),
		stringFromAny(payload["requestModelName"]),
	)
}

func modelFromParts(modelValue, modelNameValue any, metadata map[string]any) string {
	return firstNonEmpty(stringFromAny(modelValue), stringFromAny(modelNameValue), stringFromAny(metadata["model"]))
}

func stringValue(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	return stringFromAny(value)
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func boolValue(payload map[string]any, key string) bool {
	value, ok := payload[key].(bool)
	return ok && value
}

func int64Value(payload map[string]any, key string) int64 {
	value, ok := payload[key]
	if !ok || value == nil {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	case json.Number:
		n, _ := typed.Int64()
		return n
	default:
		return 0
	}
}

func firstInt64(payload map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value := int64Value(payload, key); value > 0 {
			return value
		}
	}
	return 0
}

func firstMap(payload map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		value, ok := payload[key].(map[string]any)
		if ok {
			return value, true
		}
	}
	return nil, false
}

func nestedInt64(value any, keys ...string) int64 {
	switch typed := value.(type) {
	case map[string]any:
		var total int64
		for _, key := range keys {
			total += int64Value(typed, key)
		}
		return total
	case []any:
		var total int64
		for _, item := range typed {
			total += nestedInt64(item, keys...)
		}
		return total
	default:
		return 0
	}
}

func hasUsage(usage model.Usage) bool {
	return usage.InputTokens > 0 ||
		usage.CachedInputTokens > 0 ||
		usage.OutputTokens > 0 ||
		usage.ReasoningOutputTokens > 0 ||
		usage.TotalTokens > 0
}

func subtractUsage(current model.Usage, previous *model.Usage) model.Usage {
	if previous == nil {
		return current
	}
	return model.Usage{
		InputTokens:           saturatingSubtract(current.InputTokens, previous.InputTokens),
		CachedInputTokens:     saturatingSubtract(current.CachedInputTokens, previous.CachedInputTokens),
		OutputTokens:          saturatingSubtract(current.OutputTokens, previous.OutputTokens),
		ReasoningOutputTokens: saturatingSubtract(current.ReasoningOutputTokens, previous.ReasoningOutputTokens),
		TotalTokens:           saturatingSubtract(current.TotalTokens, previous.TotalTokens),
	}
}

func addUsage(total *model.Usage, delta model.Usage) {
	if total.Source == "" || total.Source == "unknown" {
		total.Source = firstNonEmpty(delta.Source, "actual")
	}
	if total.Model == "" || total.Model == "unknown" {
		total.Model = delta.Model
	}
	total.InputTokens += delta.InputTokens
	total.CachedInputTokens += delta.CachedInputTokens
	total.OutputTokens += delta.OutputTokens
	total.ReasoningOutputTokens += delta.ReasoningOutputTokens
	total.TotalTokens += delta.TotalTokens
}

func saturatingSubtract(current, previous int64) int64 {
	if current <= previous {
		return 0
	}
	return current - previous
}

func valueToString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case nil:
		return ""
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func preview(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= limit {
		return value
	}
	if limit <= 1 {
		return value[:limit]
	}
	return value[:limit-1] + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

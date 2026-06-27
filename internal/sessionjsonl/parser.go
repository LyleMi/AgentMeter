package sessionjsonl

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type rawRecord struct {
	ID             any            `json:"id"`
	ParentID       any            `json:"parentId"`
	Timestamp      any            `json:"timestamp"`
	CreatedAt      any            `json:"created_at"`
	CreatedAtCamel any            `json:"createdAt"`
	Type           string         `json:"type"`
	SessionID      any            `json:"sessionId"`
	SessionIDSnake any            `json:"session_id"`
	UUID           any            `json:"uuid"`
	CWD            any            `json:"cwd"`
	Role           any            `json:"role"`
	Status         any            `json:"status"`
	Name           any            `json:"name"`
	CallID         any            `json:"callId"`
	CallIDSnake    any            `json:"call_id"`
	Arguments      any            `json:"arguments"`
	Output         any            `json:"output"`
	Content        any            `json:"content"`
	RawContent     any            `json:"rawContent"`
	Summary        any            `json:"summary"`
	AITitle        any            `json:"aiTitle"`
	Payload        map[string]any `json:"payload"`
	Data           map[string]any `json:"data"`
	Result         map[string]any `json:"result"`
	Response       map[string]any `json:"response"`
	Message        map[string]any `json:"message"`
	Usage          any            `json:"usage"`
	Model          any            `json:"model"`
	ModelName      any            `json:"model_name"`
	Metadata       map[string]any `json:"metadata"`
	ProviderData   map[string]any `json:"providerData"`
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

	parsed := model.ParsedSession{
		Session: model.Session{
			SourceID:     sourceID,
			SourceFileID: sourceFileID,
			ParseStatus:  "ok",
		},
		Usage: model.Usage{Source: "unknown"},
	}
	pending := map[string]pendingTool{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var lineNo int
	var firstTime, lastTime time.Time
	var modelBoundary time.Time
	var currentModel string
	var provider string
	var modelDurationMS int64
	var toolDurationMS int64
	var previousTotalUsage *model.Usage
	var hasSessionMeta bool
	var hasHeadlessUsage bool

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var raw rawRecord
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			parsed.Warnings = append(parsed.Warnings, fmt.Sprintf("%s:%d malformed JSON: %v", filepath.Base(path), lineNo, err))
			parsed.Session.ParseStatus = "warning"
			continue
		}
		ts := recordTimestamp(raw)
		if ts.IsZero() {
			ts = timestampFromPayload(raw.Payload, "started_at")
		}
		if ts.IsZero() {
			ts = timestampFromPayload(raw.Payload, "completed_at")
		}
		if ts.IsZero() {
			parsed.Warnings = append(parsed.Warnings, fmt.Sprintf("%s:%d missing timestamp", filepath.Base(path), lineNo))
			parsed.Session.ParseStatus = "warning"
			continue
		}
		if firstTime.IsZero() || ts.Before(firstTime) {
			firstTime = ts
		}
		if lastTime.IsZero() || ts.After(lastTime) {
			lastTime = ts
		}

		payloadType := stringValue(raw.Payload, "type")
		rawType := payloadType
		if rawType == "" {
			rawType = raw.Type
		}
		rawJSON := line
		parsed.Events = append(parsed.Events, model.Event{
			SourceFileID: sourceFileID,
			SourceLine:   lineNo,
			Timestamp:    ts,
			Kind:         classifyRecord(raw),
			RawType:      rawType,
			Summary:      summarizeRecord(raw),
			RawJSON:      rawJSON,
		})

		switch raw.Type {
		case "session_meta":
			hasSessionMeta = true
			parsed.Session.SessionKey = stringValue(raw.Payload, "session_id")
			parsed.Session.SessionKey = firstNonEmpty(parsed.Session.SessionKey, stringValue(raw.Payload, "id"))
			parsed.Session.CodexSessionID = parsed.Session.SessionKey
			parsed.Session.ProjectPath = firstNonEmpty(parsed.Session.ProjectPath, stringValue(raw.Payload, "cwd"))
			parsed.Session.Originator = stringValue(raw.Payload, "originator")
			parsed.Session.ThreadSource = stringValue(raw.Payload, "thread_source")
			provider = firstNonEmpty(provider, stringValue(raw.Payload, "model_provider"))
			parsed.Session.ModelProvider = provider
		case "turn_context":
			currentModel = firstNonEmpty(modelFromMap(raw.Payload), currentModel)
			parsed.Session.Model = firstNonEmpty(currentModel, parsed.Session.Model)
			parsed.Session.ProjectPath = firstNonEmpty(stringValue(raw.Payload, "cwd"), parsed.Session.ProjectPath)
			provider = firstNonEmpty(provider, stringValue(raw.Payload, "model_provider"))
			parsed.Session.ModelProvider = provider
		}

		if id := sessionIDFromRecord(raw); id != "" && parsed.Session.SessionKey == "" {
			parsed.Session.SessionKey = id
		}
		if cwd := firstNonEmpty(stringFromAny(raw.CWD), stringValue(raw.Metadata, "cwd"), stringValue(raw.Message, "cwd")); cwd != "" && parsed.Session.ProjectPath == "" {
			parsed.Session.ProjectPath = cwd
		}
		if agentName := stringValue(raw.ProviderData, "agent"); agentName != "" && parsed.Session.AgentNickname == "" {
			parsed.Session.AgentNickname = agentName
		}
		if boolValue(raw.ProviderData, "isSubAgent") && parsed.Session.AgentRole == "" {
			parsed.Session.AgentRole = "subagent"
		}
		if originator := firstNonEmpty(stringValue(raw.Metadata, "originator"), stringValue(raw.Metadata, "source")); originator != "" && parsed.Session.Originator == "" {
			parsed.Session.Originator = originator
		}
		if provider == "" {
			provider = firstNonEmpty(stringValue(raw.Metadata, "model_provider"), stringValue(raw.Metadata, "provider"))
			parsed.Session.ModelProvider = provider
		}

		if rawModel := modelFromRecord(raw); rawModel != "" {
			currentModel = rawModel
			parsed.Session.Model = firstNonEmpty(parsed.Session.Model, rawModel)
		}

		switch payloadType {
		case "task_started":
			modelBoundary = timestampFromPayload(raw.Payload, "started_at")
			if modelBoundary.IsZero() {
				modelBoundary = ts
			}
		case "token_count":
			total := readUsage(raw.Payload, "total_token_usage")
			last := readUsage(raw.Payload, "last_token_usage")
			eventModel := firstNonEmpty(modelFromPayloadInfo(raw.Payload), currentModel, parsed.Session.Model)
			if eventModel != "" {
				currentModel = eventModel
				parsed.Session.Model = firstNonEmpty(parsed.Session.Model, eventModel)
			}
			callUsage := last
			if !hasUsage(callUsage) && hasUsage(total) {
				callUsage = subtractUsage(total, previousTotalUsage)
			}
			if hasUsage(total) {
				totalCopy := total
				previousTotalUsage = &totalCopy
			}
			if hasUsage(callUsage) {
				callUsage.Model = firstNonEmpty(eventModel, currentModel, parsed.Session.Model)
				callUsage.Source = "actual"
				addUsage(&parsed.Usage, callUsage)
				start := modelBoundary
				if start.IsZero() || start.After(ts) {
					start = ts
				}
				duration := durationMS(start, ts)
				modelDurationMS += duration
				parsed.ModelCall = append(parsed.ModelCall, model.ModelCall{
					StartedAt:             start,
					EndedAt:               ts,
					DurationMS:            duration,
					Model:                 firstNonEmpty(callUsage.Model, currentModel, parsed.Session.Model),
					Provider:              provider,
					Status:                "completed",
					InputTokens:           callUsage.InputTokens,
					CachedInputTokens:     callUsage.CachedInputTokens,
					OutputTokens:          callUsage.OutputTokens,
					ReasoningOutputTokens: callUsage.ReasoningOutputTokens,
					TotalTokens:           callUsage.TotalTokens,
				})
				modelBoundary = ts
			}
		case "function_call", "custom_tool_call", "web_search_call", "tool_search_call":
			call := startTool(raw.Payload, payloadType, ts, lineNo)
			if call.callID != "" {
				pending[call.callID] = call
			}
		case "function_call_output", "custom_tool_call_output", "web_search_end", "web_search_output", "tool_search_output", "patch_apply_end":
			callID := firstNonEmpty(stringValue(raw.Payload, "call_id"), stringValue(raw.Payload, "id"))
			if callID == "" {
				break
			}
			call := pending[callID]
			if call.callID == "" {
				call = pendingTool{
					callID:       callID,
					name:         toolName(payloadType, raw.Payload),
					startedAt:    ts,
					rawLine:      lineNo,
					inputSummary: "",
					status:       "completed",
				}
			}
			duration := durationMS(call.startedAt, ts)
			toolDurationMS += duration
			status, errText := outputStatus(raw.Payload)
			if status == "" {
				status = firstNonEmpty(call.status, "completed")
			}
			parsed.ToolCall = append(parsed.ToolCall, model.ToolCall{
				StartedAt:         call.startedAt,
				EndedAt:           ts,
				DurationMS:        duration,
				ToolName:          call.name,
				Status:            status,
				InputSummary:      call.inputSummary,
				OutputSummary:     preview(outputSummary(raw.Payload), 500),
				Error:             errText,
				CallID:            call.callID,
				RawEventLine:      call.rawLine,
				RawStartEventLine: call.rawLine,
				RawEndEventLine:   lineNo,
			})
			delete(pending, callID)
			modelBoundary = ts
		}

		switch raw.Type {
		case "function_call":
			call := startToolRecord(raw, ts, lineNo)
			if call.callID != "" {
				pending[call.callID] = call
			}
		case "function_call_result":
			callID := recordCallID(raw)
			if callID != "" {
				call := pending[callID]
				if call.callID == "" {
					call = pendingTool{
						callID:       callID,
						name:         recordToolName(raw),
						startedAt:    ts,
						rawLine:      lineNo,
						inputSummary: "",
						status:       "completed",
					}
				}
				duration := durationMS(call.startedAt, ts)
				toolDurationMS += duration
				status, errText := outputStatusRecord(raw)
				if status == "" {
					status = firstNonEmpty(call.status, "completed")
				}
				parsed.ToolCall = append(parsed.ToolCall, model.ToolCall{
					StartedAt:         call.startedAt,
					EndedAt:           ts,
					DurationMS:        duration,
					ToolName:          firstNonEmpty(call.name, recordToolName(raw)),
					Status:            status,
					InputSummary:      call.inputSummary,
					OutputSummary:     preview(outputSummaryRecord(raw), 500),
					Error:             errText,
					CallID:            call.callID,
					RawEventLine:      call.rawLine,
					RawStartEventLine: call.rawLine,
					RawEndEventLine:   lineNo,
				})
				delete(pending, callID)
				modelBoundary = ts
			}
		}

		if raw.Message != nil {
			role := stringValue(raw.Message, "role")
			if messageModel := modelFromMap(raw.Message); messageModel != "" {
				currentModel = messageModel
				parsed.Session.Model = firstNonEmpty(parsed.Session.Model, messageModel)
			}
			if role == "assistant" || raw.Type == "assistant" {
				for _, call := range toolUsesFromMessage(raw.Message, ts, lineNo) {
					if call.callID != "" {
						pending[call.callID] = call
					}
				}
			}
			if role == "user" || raw.Type == "user" {
				for _, result := range toolResultsFromMessage(raw.Message, ts, lineNo) {
					call := pending[result.callID]
					if call.callID == "" {
						call = pendingTool{
							callID:    result.callID,
							name:      result.name,
							startedAt: ts,
							rawLine:   lineNo,
							status:    result.status,
						}
					}
					duration := durationMS(call.startedAt, ts)
					toolDurationMS += duration
					parsed.ToolCall = append(parsed.ToolCall, model.ToolCall{
						StartedAt:         call.startedAt,
						EndedAt:           ts,
						DurationMS:        duration,
						ToolName:          firstNonEmpty(call.name, result.name),
						Status:            firstNonEmpty(result.status, call.status, "completed"),
						InputSummary:      call.inputSummary,
						OutputSummary:     preview(result.outputSummary, 500),
						Error:             result.error,
						CallID:            call.callID,
						RawEventLine:      call.rawLine,
						RawStartEventLine: call.rawLine,
						RawEndEventLine:   lineNo,
					})
					delete(pending, result.callID)
					modelBoundary = ts
				}
			}
		}

		if payloadType == "" {
			usage := headlessUsage(raw)
			if hasUsage(usage) {
				hasHeadlessUsage = true
				eventModel := firstNonEmpty(modelFromRecord(raw), currentModel, parsed.Session.Model, "gpt-5")
				currentModel = eventModel
				parsed.Session.Model = firstNonEmpty(parsed.Session.Model, eventModel)
				usage.Model = eventModel
				usage.Source = "actual"
				addUsage(&parsed.Usage, usage)
				parsed.ModelCall = append(parsed.ModelCall, model.ModelCall{
					StartedAt:             ts,
					EndedAt:               ts,
					DurationMS:            0,
					Model:                 eventModel,
					Provider:              provider,
					Status:                "completed",
					InputTokens:           usage.InputTokens,
					CachedInputTokens:     usage.CachedInputTokens,
					OutputTokens:          usage.OutputTokens,
					ReasoningOutputTokens: usage.ReasoningOutputTokens,
					TotalTokens:           usage.TotalTokens,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return parsed, err
	}
	for _, call := range pending {
		parsed.ToolCall = append(parsed.ToolCall, model.ToolCall{
			StartedAt:         call.startedAt,
			EndedAt:           call.startedAt,
			DurationMS:        0,
			ToolName:          call.name,
			Status:            firstNonEmpty(call.status, "pending"),
			InputSummary:      call.inputSummary,
			CallID:            call.callID,
			RawEventLine:      call.rawLine,
			RawStartEventLine: call.rawLine,
		})
	}

	if parsed.Session.SessionKey == "" {
		parsed.Session.SessionKey = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if !hasHeadlessUsage && !hasSessionMeta {
			parsed.Session.ParseStatus = "warning"
		}
	}
	if parsed.Session.CodexSessionID == "" {
		parsed.Session.CodexSessionID = parsed.Session.SessionKey
	}
	if parsed.Session.ProjectPath == "" {
		parsed.Session.ProjectPath = "unknown"
	}
	if parsed.Session.Model == "" {
		parsed.Session.Model = "unknown"
	}
	if parsed.Session.ModelProvider == "" {
		parsed.Session.ModelProvider = provider
	}
	parsed.Session.StartedAt = firstTime
	parsed.Session.EndedAt = lastTime
	parsed.Session.WallDurationMS = durationMS(firstTime, lastTime)
	parsed.Session.ModelDurationMS = modelDurationMS
	parsed.Session.ToolDurationMS = toolDurationMS
	parsed.Session.ActiveDurationMS = modelDurationMS + toolDurationMS
	parsed.Session.IdleDurationMS = parsed.Session.WallDurationMS - parsed.Session.ActiveDurationMS
	if parsed.Session.IdleDurationMS < 0 {
		parsed.Session.IdleDurationMS = 0
	}
	parsed.Session.EventCount = len(parsed.Events)
	if parsed.Usage.Model == "" {
		parsed.Usage.Model = parsed.Session.Model
	}
	if len(parsed.Events) == 0 {
		parsed.Session.ParseStatus = "warning"
		parsed.Warnings = append(parsed.Warnings, fmt.Sprintf("%s contains no parseable events", filepath.Base(path)))
	}
	return parsed, nil
}

func classifyRecord(raw rawRecord) string {
	if stringValue(raw.Payload, "type") != "" {
		return classify(raw.Type, raw.Payload)
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
	}
	return model.Usage{}
}

func usageFromValue(value any) model.Usage {
	raw, _ := value.(map[string]any)
	if raw == nil {
		return model.Usage{}
	}
	inputIncludesCached := false
	input := firstInt64(raw, "input_tokens", "input", "inputTokens")
	if input > 0 {
		input += firstInt64(raw, "cache_creation_input_tokens", "cache_write_input_tokens", "cacheCreationInputTokens", "cacheWriteInputTokens")
	} else {
		input = firstInt64(raw, "prompt_tokens", "promptTokens")
		inputIncludesCached = input > 0
	}
	cached := firstInt64(raw, "cached_input_tokens", "cache_read_input_tokens", "cached_tokens", "cachedInputTokens", "cacheReadInputTokens", "cachedTokens")
	cached += nestedInt64(raw["inputTokensDetails"], "cached_tokens", "cachedTokens")
	cached += nestedInt64(raw["prompt_tokens_details"], "cached_tokens", "cachedTokens")
	cacheRead := firstInt64(raw, "cache_read_input_tokens", "cacheReadInputTokens")
	if cacheRead == 0 && !inputIncludesCached {
		cacheRead = cached
	}
	output := firstInt64(raw, "output_tokens", "completion_tokens", "output", "outputTokens", "completionTokens")
	reasoning := firstInt64(raw, "reasoning_output_tokens", "reasoning_tokens", "reasoningOutputTokens", "reasoningTokens", "completion_thinking_tokens")
	reasoning += nestedInt64(raw["outputTokensDetails"], "reasoning_tokens", "reasoningTokens")
	reasoning += nestedInt64(raw["completion_tokens_details"], "reasoning_tokens", "reasoningTokens")
	total := firstInt64(raw, "total_tokens", "totalTokens")
	if total <= 0 && input+cached+output+reasoning > 0 {
		total = input + cacheRead + output + reasoning
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
	for _, value := range []any{raw.Timestamp, raw.CreatedAt, raw.CreatedAtCamel} {
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

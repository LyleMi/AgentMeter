package sessionjsonl

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type parseAccumulator struct {
	path               string
	sourceFileID       int64
	parsed             model.ParsedSession
	pending            map[string]pendingTool
	pendingCompaction  *pendingContextCompaction
	firstTime          time.Time
	lastTime           time.Time
	modelBoundary      time.Time
	currentModel       string
	provider           string
	modelDurationMS    int64
	toolDurationMS     int64
	lastPromptTokens   int64
	previousTotalUsage *model.Usage
	hasSessionMeta     bool
	hasHeadlessUsage   bool
}

type pendingContextCompaction struct {
	preTokens int64
}

func newParseAccumulator(path string, sourceID, sourceFileID int64) *parseAccumulator {
	return &parseAccumulator{
		path:         path,
		sourceFileID: sourceFileID,
		parsed: model.ParsedSession{
			Session: model.Session{
				SourceID:     sourceID,
				SourceFileID: sourceFileID,
				ParseStatus:  "ok",
			},
			Usage: model.Usage{Source: "unknown"},
		},
		pending: map[string]pendingTool{},
	}
}

func (a *parseAccumulator) addWarnings(warnings []string) {
	if len(warnings) == 0 {
		return
	}
	a.parsed.Warnings = append(a.parsed.Warnings, warnings...)
	a.parsed.Session.ParseStatus = "warning"
}

func (a *parseAccumulator) handleRecord(record parsedRawRecord) {
	a.addEvent(record)
	a.handleSessionRecord(record.raw)
	a.handleRecordIdentity(record.raw)
	a.handleRecordModel(record.raw)
	a.handleCompactionSignals(record.raw)
	a.handlePayloadRecord(record)
	a.handleTopLevelToolRecord(record)
	a.handleRawMessage(record)
	a.handleHeadlessRecord(record)
}

func (a *parseAccumulator) addEvent(record parsedRawRecord) {
	a.updateTimeBounds(record.ts)
	a.parsed.Events = append(a.parsed.Events, model.Event{
		SourceFileID: a.sourceFileID,
		SourceLine:   record.lineNo,
		Timestamp:    record.ts,
		Kind:         classifyRecord(record.raw),
		RawType:      record.rawType,
		Summary:      summarizeRecord(record.raw),
		RawJSON:      record.line,
	})
}

func (a *parseAccumulator) updateTimeBounds(ts time.Time) {
	if a.firstTime.IsZero() || ts.Before(a.firstTime) {
		a.firstTime = ts
	}
	if a.lastTime.IsZero() || ts.After(a.lastTime) {
		a.lastTime = ts
	}
}

func (a *parseAccumulator) handleSessionRecord(raw rawRecord) {
	switch raw.Type {
	case "session_meta":
		a.hasSessionMeta = true
		a.parsed.Session.SessionKey = stringValue(raw.Payload, "session_id")
		a.parsed.Session.SessionKey = firstNonEmpty(a.parsed.Session.SessionKey, stringValue(raw.Payload, "id"))
		a.parsed.Session.CodexSessionID = a.parsed.Session.SessionKey
		a.parsed.Session.ProjectPath = firstNonEmpty(a.parsed.Session.ProjectPath, stringValue(raw.Payload, "cwd"))
		a.parsed.Session.Originator = stringValue(raw.Payload, "originator")
		a.parsed.Session.ThreadSource = stringValue(raw.Payload, "thread_source")
		a.provider = firstNonEmpty(a.provider, stringValue(raw.Payload, "model_provider"))
		a.parsed.Session.ModelProvider = a.provider
	case "turn_context":
		a.currentModel = firstNonEmpty(modelFromMap(raw.Payload), a.currentModel)
		a.parsed.Session.Model = firstNonEmpty(a.currentModel, a.parsed.Session.Model)
		a.parsed.Session.ProjectPath = firstNonEmpty(stringValue(raw.Payload, "cwd"), a.parsed.Session.ProjectPath)
		a.provider = firstNonEmpty(a.provider, stringValue(raw.Payload, "model_provider"))
		a.parsed.Session.ModelProvider = a.provider
	}
}

func (a *parseAccumulator) handleRecordIdentity(raw rawRecord) {
	if id := sessionIDFromRecord(raw); id != "" && a.parsed.Session.SessionKey == "" {
		a.parsed.Session.SessionKey = id
	}
	if cwd := firstNonEmpty(stringFromAny(raw.CWD), stringValue(raw.Metadata, "cwd"), stringValue(mapFromAny(raw.Message), "cwd")); cwd != "" && a.parsed.Session.ProjectPath == "" {
		a.parsed.Session.ProjectPath = cwd
	}
	if agentName := stringValue(raw.ProviderData, "agent"); agentName != "" && a.parsed.Session.AgentNickname == "" {
		a.parsed.Session.AgentNickname = agentName
	}
	if boolValue(raw.ProviderData, "isSubAgent") && a.parsed.Session.AgentRole == "" {
		a.parsed.Session.AgentRole = "subagent"
	}
	if originator := firstNonEmpty(stringValue(raw.Metadata, "originator"), stringValue(raw.Metadata, "source")); originator != "" && a.parsed.Session.Originator == "" {
		a.parsed.Session.Originator = originator
	}
	if a.provider == "" {
		a.provider = firstNonEmpty(stringValue(raw.Metadata, "model_provider"), stringValue(raw.Metadata, "provider"))
		a.parsed.Session.ModelProvider = a.provider
	}
}

func (a *parseAccumulator) handleRecordModel(raw rawRecord) {
	if rawModel := modelFromRecord(raw); rawModel != "" {
		a.currentModel = rawModel
		a.parsed.Session.Model = firstNonEmpty(a.parsed.Session.Model, rawModel)
	}
}

func (a *parseAccumulator) handleCompactionSignals(raw rawRecord) {
	a.handleCompactMetadata(raw)
	a.handleCodexCompacted(raw)
	a.handleCodeBuddyCompacted(raw)
}

func (a *parseAccumulator) handlePayloadRecord(record parsedRawRecord) {
	switch record.payloadType {
	case "task_started":
		a.modelBoundary = timestampFromPayload(record.raw.Payload, "started_at")
		if a.modelBoundary.IsZero() {
			a.modelBoundary = record.ts
		}
	case "token_count":
		a.handleTokenCount(record.raw, record.ts)
	case "function_call", "custom_tool_call", "web_search_call", "tool_search_call":
		call := startTool(record.raw.Payload, record.payloadType, record.ts, record.lineNo)
		if call.callID != "" {
			a.pending[call.callID] = call
		}
	case "function_call_output", "custom_tool_call_output", "web_search_end", "web_search_output", "tool_search_output", "patch_apply_end":
		a.handlePayloadToolOutput(record.raw, record.payloadType, record.ts, record.lineNo)
	}
}

func (a *parseAccumulator) handleTopLevelToolRecord(record parsedRawRecord) {
	switch record.raw.Type {
	case "function_call":
		call := startToolRecord(record.raw, record.ts, record.lineNo)
		if call.callID != "" {
			a.pending[call.callID] = call
		}
	case "function_call_result":
		a.handleRecordToolResult(record.raw, record.ts, record.lineNo)
	}
}

func (a *parseAccumulator) handleRawMessage(record parsedRawRecord) {
	if record.raw.Message != nil || topLevelRole(record.raw) != "" {
		a.handleMessage(record.raw, record.ts, record.lineNo)
	}
}

func (a *parseAccumulator) handleHeadlessRecord(record parsedRawRecord) {
	if record.payloadType == "" {
		a.handleHeadlessUsage(record.raw, record.ts)
	}
}

func (a *parseAccumulator) handleCompactMetadata(raw rawRecord) {
	tokens := contextCompressionTokensFromCompactMetadata(raw.CompactMetadata)
	if tokens <= 0 {
		return
	}
	usage := model.Usage{
		Model:                    firstNonEmpty(modelFromRecord(raw), a.currentModel, a.parsed.Session.Model),
		ContextCompressionTokens: tokens,
		Source:                   "actual",
	}
	addUsage(&a.parsed.Usage, usage)
}

func (a *parseAccumulator) handleCodexCompacted(raw rawRecord) {
	if raw.Type != "compacted" || codexReplacementHistoryCount(raw) == 0 {
		return
	}
	a.pendingCompaction = &pendingContextCompaction{
		preTokens: a.lastPromptTokens,
	}
}

func (a *parseAccumulator) handleTokenCount(raw rawRecord, ts time.Time) {
	total := readUsage(raw.Payload, "total_token_usage")
	last := readUsage(raw.Payload, "last_token_usage")
	eventModel := firstNonEmpty(modelFromPayloadInfo(raw.Payload), a.currentModel, a.parsed.Session.Model)
	if eventModel != "" {
		a.currentModel = eventModel
		a.parsed.Session.Model = firstNonEmpty(a.parsed.Session.Model, eventModel)
	}
	callUsage := last
	if !hasUsage(callUsage) && hasUsage(total) {
		callUsage = subtractUsage(total, a.previousTotalUsage)
	}
	if hasUsage(total) {
		totalCopy := total
		a.previousTotalUsage = &totalCopy
	}
	if a.handlePendingCodexCompaction(callUsage) {
		a.modelBoundary = ts
		return
	}
	if !hasUsage(callUsage) {
		return
	}
	callUsage.Model = firstNonEmpty(eventModel, a.currentModel, a.parsed.Session.Model)
	callUsage.Source = "actual"
	addUsage(&a.parsed.Usage, callUsage)
	start := a.modelBoundary
	if start.IsZero() || start.After(ts) {
		start = ts
	}
	duration := durationMS(start, ts)
	a.modelDurationMS += duration
	a.parsed.ModelCall = append(a.parsed.ModelCall, model.ModelCall{
		StartedAt:                start,
		EndedAt:                  ts,
		DurationMS:               duration,
		Model:                    firstNonEmpty(callUsage.Model, a.currentModel, a.parsed.Session.Model),
		Provider:                 a.provider,
		Status:                   "completed",
		InputTokens:              callUsage.InputTokens,
		CachedInputTokens:        callUsage.CachedInputTokens,
		OutputTokens:             callUsage.OutputTokens,
		ReasoningOutputTokens:    callUsage.ReasoningOutputTokens,
		ContextCompressionTokens: callUsage.ContextCompressionTokens,
		TotalTokens:              callUsage.TotalTokens,
	})
	if callUsage.InputTokens > 0 {
		a.lastPromptTokens = callUsage.InputTokens
	}
	a.modelBoundary = ts
}

func (a *parseAccumulator) handleCodeBuddyCompacted(raw rawRecord) {
	if !isCodeBuddyCompactSummary(raw) || a.lastPromptTokens <= 0 {
		return
	}
	a.pendingCompaction = &pendingContextCompaction{
		preTokens: a.lastPromptTokens,
	}
}

func (a *parseAccumulator) handlePendingCodexCompaction(usage model.Usage) bool {
	return a.handlePendingCompactionUsage(usage, true)
}

func (a *parseAccumulator) handlePendingCompactionUsage(usage model.Usage, allowSnapshot bool) bool {
	if a.pendingCompaction == nil {
		return false
	}
	if usage.ContextCompressionTokens > 0 {
		a.pendingCompaction = nil
		return false
	}
	if allowSnapshot && isCodexCompactionSnapshotUsage(usage) {
		a.addContextCompactionUsage(usage.TotalTokens)
		return true
	}
	if usage.InputTokens > 0 {
		a.addContextCompactionUsage(usage.InputTokens)
	}
	return false
}

func (a *parseAccumulator) addContextCompactionUsage(postTokens int64) {
	pending := a.pendingCompaction
	a.pendingCompaction = nil
	if pending == nil || pending.preTokens <= 0 || postTokens <= 0 {
		return
	}
	tokens := saturatingSubtract(pending.preTokens, postTokens)
	if tokens <= 0 {
		return
	}
	addUsage(&a.parsed.Usage, model.Usage{
		Model:                    firstNonEmpty(a.currentModel, a.parsed.Session.Model),
		ContextCompressionTokens: tokens,
		Source:                   "actual",
	})
}

func (a *parseAccumulator) handlePayloadToolOutput(raw rawRecord, payloadType string, ts time.Time, lineNo int) {
	callID := firstNonEmpty(stringValue(raw.Payload, "call_id"), stringValue(raw.Payload, "id"))
	if callID == "" {
		return
	}
	call := a.pending[callID]
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
	status, errText := outputStatus(raw.Payload)
	toolCall, duration := finishToolCall(call, completedTool{
		callID:        callID,
		name:          toolName(payloadType, raw.Payload),
		status:        status,
		outputSummary: outputSummary(raw.Payload),
		error:         errText,
	}, ts, lineNo)
	a.toolDurationMS += duration
	a.parsed.ToolCall = append(a.parsed.ToolCall, toolCall)
	delete(a.pending, callID)
	a.modelBoundary = ts
}

func (a *parseAccumulator) handleRecordToolResult(raw rawRecord, ts time.Time, lineNo int) {
	callID := recordCallID(raw)
	if callID == "" {
		return
	}
	call := a.pending[callID]
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
	status, errText := outputStatusRecord(raw)
	toolCall, duration := finishToolCall(call, completedTool{
		callID:        callID,
		name:          recordToolName(raw),
		status:        status,
		outputSummary: outputSummaryRecord(raw),
		error:         errText,
	}, ts, lineNo)
	a.toolDurationMS += duration
	a.parsed.ToolCall = append(a.parsed.ToolCall, toolCall)
	delete(a.pending, callID)
	a.modelBoundary = ts
}

func (a *parseAccumulator) handleMessage(raw rawRecord, ts time.Time, lineNo int) {
	message := mapFromAny(raw.Message)
	if message == nil {
		message = topLevelMessage(raw)
	}
	role := stringValue(message, "role")
	if messageModel := modelFromMap(message); messageModel != "" {
		a.currentModel = messageModel
		a.parsed.Session.Model = firstNonEmpty(a.parsed.Session.Model, messageModel)
	}
	if role == "assistant" || raw.Type == "assistant" {
		for _, call := range toolUsesFromMessage(message, ts, lineNo) {
			if call.callID != "" {
				a.pending[call.callID] = call
			}
		}
	}
	if role != "user" && raw.Type != "user" {
		return
	}
	for _, result := range toolResultsFromMessage(message, ts, lineNo) {
		call := a.pending[result.callID]
		if call.callID == "" {
			call = pendingTool{
				callID:    result.callID,
				name:      result.name,
				startedAt: ts,
				rawLine:   lineNo,
				status:    result.status,
			}
		}
		toolCall, duration := finishToolCall(call, result, ts, lineNo)
		a.toolDurationMS += duration
		a.parsed.ToolCall = append(a.parsed.ToolCall, toolCall)
		delete(a.pending, result.callID)
		a.modelBoundary = ts
	}
}

func topLevelMessage(raw rawRecord) map[string]any {
	role := topLevelRole(raw)
	if role == "" {
		return nil
	}
	message := map[string]any{"role": role}
	if raw.Content != nil {
		message["content"] = raw.Content
	}
	if raw.Model != nil {
		message["model"] = raw.Model
	}
	if raw.ModelName != nil {
		message["model_name"] = raw.ModelName
	}
	if raw.Usage != nil {
		message["usage"] = raw.Usage
	}
	return message
}

func topLevelRole(raw rawRecord) string {
	role := stringFromAny(raw.Role)
	if role != "" {
		return role
	}
	switch raw.Type {
	case "user", "assistant", "system":
		return raw.Type
	default:
		return ""
	}
}

func (a *parseAccumulator) handleHeadlessUsage(raw rawRecord, ts time.Time) {
	usage := headlessUsage(raw)
	if !hasUsage(usage) {
		return
	}
	compactUsage := isCodeBuddyCompactUsage(raw)
	if !compactUsage && a.handlePendingCompactionUsage(usage, false) {
		return
	}
	a.hasHeadlessUsage = true
	eventModel := firstNonEmpty(modelFromRecord(raw), a.currentModel, a.parsed.Session.Model, "gpt-5")
	a.currentModel = eventModel
	a.parsed.Session.Model = firstNonEmpty(a.parsed.Session.Model, eventModel)
	usage.Model = eventModel
	usage.Source = "actual"
	addUsage(&a.parsed.Usage, usage)
	a.parsed.ModelCall = append(a.parsed.ModelCall, model.ModelCall{
		StartedAt:                ts,
		EndedAt:                  ts,
		DurationMS:               0,
		Model:                    eventModel,
		Provider:                 a.provider,
		Status:                   "completed",
		InputTokens:              usage.InputTokens,
		CachedInputTokens:        usage.CachedInputTokens,
		OutputTokens:             usage.OutputTokens,
		ReasoningOutputTokens:    usage.ReasoningOutputTokens,
		ContextCompressionTokens: usage.ContextCompressionTokens,
		TotalTokens:              usage.TotalTokens,
	})
	if !compactUsage && usage.InputTokens > 0 {
		a.lastPromptTokens = usage.InputTokens
	}
}

func (a *parseAccumulator) finalize() model.ParsedSession {
	for _, call := range a.pending {
		a.parsed.ToolCall = append(a.parsed.ToolCall, model.ToolCall{
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

	if a.parsed.Session.SessionKey == "" {
		a.parsed.Session.SessionKey = strings.TrimSuffix(filepath.Base(a.path), filepath.Ext(a.path))
		if !a.hasHeadlessUsage && !a.hasSessionMeta {
			a.parsed.Session.ParseStatus = "warning"
		}
	}
	if a.parsed.Session.CodexSessionID == "" {
		a.parsed.Session.CodexSessionID = a.parsed.Session.SessionKey
	}
	if a.parsed.Session.ProjectPath == "" {
		a.parsed.Session.ProjectPath = "unknown"
	}
	if a.parsed.Session.Model == "" {
		a.parsed.Session.Model = "unknown"
	}
	if a.parsed.Session.ModelProvider == "" {
		a.parsed.Session.ModelProvider = a.provider
	}
	a.parsed.Session.StartedAt = a.firstTime
	a.parsed.Session.EndedAt = a.lastTime
	a.parsed.Session.WallDurationMS = durationMS(a.firstTime, a.lastTime)
	a.parsed.Session.ModelDurationMS = a.modelDurationMS
	a.parsed.Session.ToolDurationMS = a.toolDurationMS
	a.parsed.Session.ActiveDurationMS = a.modelDurationMS + a.toolDurationMS
	a.parsed.Session.IdleDurationMS = a.parsed.Session.WallDurationMS - a.parsed.Session.ActiveDurationMS
	if a.parsed.Session.IdleDurationMS < 0 {
		a.parsed.Session.IdleDurationMS = 0
	}
	a.parsed.Session.EventCount = len(a.parsed.Events)
	if a.parsed.Usage.Model == "" {
		a.parsed.Usage.Model = a.parsed.Session.Model
	}
	if len(a.parsed.Events) == 0 {
		a.parsed.Session.ParseStatus = "warning"
		a.parsed.Warnings = append(a.parsed.Warnings, fmt.Sprintf("%s contains no parseable events", filepath.Base(a.path)))
	}
	return a.parsed
}

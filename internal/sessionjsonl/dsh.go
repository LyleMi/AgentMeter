package sessionjsonl

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/klauspost/compress/zstd"
)

// Hashes cover the physical file; source line numbers refer to decoded JSONL.
func parseEncodedReader(path string, reader io.Reader, sourceID, sourceFileID int64) (model.ParsedSession, error) {
	if strings.HasSuffix(path, ".jsonl.zstd") {
		decoder, err := zstd.NewReader(reader, zstd.WithDecoderConcurrency(1), zstd.WithDecoderMaxMemory(64<<20))
		if err != nil {
			return model.ParsedSession{}, err
		}
		defer decoder.Close()
		reader = decoder
	}
	return parseFromReader(path, reader, sourceID, sourceFileID)
}

func (a *parseAccumulator) handleDSHRecord(record parsedRawRecord) bool {
	raw := record.raw
	if raw.Type == "session" && raw.Version != nil {
		a.dsh = true
		a.dshSeedLength = raw.SeedLength
		a.dshSeeded = raw.IsSeeded
		a.hasSessionMeta = true
		a.parsed.Session.SessionKey = stringFromAny(raw.ID)
		a.parsed.Session.ProjectPath = stringFromAny(raw.CWD)
		a.parsed.Session.Originator = "dsh"
		a.parsed.Session.AgentRole = raw.Origin
		a.addEvent(record)
		return true
	}
	if !a.dsh {
		return false
	}
	a.addEvent(record)
	data := raw.Data
	switch raw.Type {
	case "request/header":
		config := mapFromAny(mapFromAny(data["header"])["config"])
		a.currentModel = firstNonEmpty(stringValue(config, "model"), a.currentModel)
		a.provider = firstNonEmpty(stringValue(config, "provider"), a.provider)
		a.parsed.Session.Model = firstNonEmpty(a.parsed.Session.Model, a.currentModel)
		a.parsed.Session.ModelProvider = a.provider
	case "session/end-seed":
		if boolValue(data, "inherited") {
			// Forks copy prior model/tool events verbatim. Only work after the last
			// tagged boundary belongs to this session, including nested fork seeds.
			a.parsed.Usage = model.Usage{Source: "unknown"}
			a.parsed.ModelCall = nil
			a.parsed.ToolCall = nil
			a.pending = map[string]pendingTool{}
			a.modelDurationMS, a.toolDurationMS = 0, 0
			a.modelBoundary = time.Time{}
			a.dshSeeded = false
			a.clearDSHInheritedEvents()
		}
	}
	if mapFromAny(raw.SurfaceOp) != nil {
		return true
	}
	if raw.Seq < a.dshSeedLength {
		a.clearDSHInheritedEvents()
		return true
	}
	switch raw.Type {
	case "step/start":
		a.modelBoundary = record.ts
	case "assistant/message":
		message := mapFromAny(data["message"])
		if message == nil {
			message = data
		}
		source := mapFromAny(message["source"])
		if source == nil {
			source = mapFromAny(data["provenance"])
		}
		a.currentModel = firstNonEmpty(stringValue(source, "model"), a.currentModel, "unknown")
		a.provider = firstNonEmpty(stringValue(source, "provider"), a.provider)
		a.parsed.Session.Model = firstNonEmpty(a.parsed.Session.Model, a.currentModel)
		a.parsed.Session.ModelProvider = a.provider
		a.recordDSHUsage(mapFromAny(data["usage"]), record.ts, boolValue(data, "interrupted"))
	case "assistant/attempt":
		// Failed attempts can still consume tokens. Embedded stream usage is a
		// snapshot: select the last reported value, never sum stream snapshots.
		var usage map[string]any
		stream, _ := data["stream"].([]any)
		for _, item := range stream {
			chunk := mapFromAny(mapFromAny(item)["chunk"])
			if stringValue(chunk, "type") == "usage" {
				usage = mapFromAny(chunk["usage"])
			}
		}
		a.recordDSHUsage(usage, record.ts, true)
	case "tool/call":
		id := stringValue(data, "callId")
		if id != "" {
			a.pending[id] = pendingTool{callID: id, name: stringValue(data, "name"), startedAt: record.ts, inputSummary: preview(stringValue(data, "arguments"), 500), rawLine: record.lineNo}
		}
	case "tool/result":
		message := mapFromAny(data["message"])
		if message == nil {
			message = data
		}
		source := mapFromAny(message["source"])
		id := firstNonEmpty(stringValue(message, "toolCallId"), stringValue(source, "callId"), stringValue(data, "callId"))
		if id == "" {
			break
		}
		// Formats 0–3 wrap tool results inside a user-message block.
		blocks, _ := message["content"].([]any)
		for _, item := range blocks {
			block := mapFromAny(item)
			if stringValue(block, "type") == "tool-result" {
				message = block
				break
			}
		}
		status := "completed"
		if boolValue(message, "isError") || boolValue(data, "isError") {
			status = "failed"
		}
		a.completeTool(completedTool{callID: id, status: status, outputSummary: preview(contentText(message["content"]), 500), error: stringValue(mapFromAny(data["error"]), "reason")}, record.ts, record.lineNo)
	}
	return true
}

func (a *parseAccumulator) recordDSHUsage(raw map[string]any, ts time.Time, interrupted bool) {
	if raw == nil {
		return
	}
	// DSH input/cache counters are disjoint; AgentMeter stores total input
	// inclusive of cache hits. Cache writes use the normal input estimate.
	cached := int64Value(raw, "cacheReadTokens")
	input := int64Value(raw, "inputTokens") + cached + int64Value(raw, "cacheWriteTokens")
	output := int64Value(raw, "outputTokens")
	total := int64Value(raw, "totalTokens")
	if total == 0 {
		total = input + output
	}
	usage := model.Usage{InputTokens: input, CachedInputTokens: cached, OutputTokens: output, ReasoningOutputTokens: int64Value(raw, "reasoningTokens"), TotalTokens: total}
	a.recordTokenCountCall(usage, firstNonEmpty(a.currentModel, "unknown"), ts)
	if interrupted {
		a.parsed.ModelCall[len(a.parsed.ModelCall)-1].Status = "interrupted"
	}
}

func validateDSHHeader(raw rawRecord) error {
	if raw.Type == "session" && raw.Version != nil && (*raw.Version < 0 || *raw.Version > 4) {
		return fmt.Errorf("unsupported dsh session format version %d", *raw.Version)
	}
	return nil
}

// Retain the child's own header, but do not attribute copied history, audit
// evidence, or elapsed time to its new work. Raw line numbers remain intact.
func (a *parseAccumulator) clearDSHInheritedEvents() {
	if len(a.parsed.Events) == 0 {
		return
	}
	a.parsed.Events = a.parsed.Events[:1]
	a.firstTime = a.parsed.Events[0].Timestamp
	a.lastTime = a.firstTime
}

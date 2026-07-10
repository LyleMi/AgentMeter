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

	"github.com/LyleMi/AgentMeter/internal/model"
)

type rawRecord struct {
	ID                 any            `json:"id"`
	ParentID           any            `json:"parentId"`
	Timestamp          any            `json:"timestamp"`
	TimestampMS        any            `json:"timestamp_ms"`
	TimestampMSCamel   any            `json:"timestampMs"`
	CreatedAt          any            `json:"created_at"`
	CreatedAtCamel     any            `json:"createdAt"`
	Type               string         `json:"type"`
	SessionID          any            `json:"sessionId"`
	SessionIDSnake     any            `json:"session_id"`
	UUID               any            `json:"uuid"`
	CWD                any            `json:"cwd"`
	Role               any            `json:"role"`
	Status             any            `json:"status"`
	Name               any            `json:"name"`
	CallID             any            `json:"callId"`
	CallIDSnake        any            `json:"call_id"`
	Arguments          any            `json:"arguments"`
	Output             any            `json:"output"`
	Content            any            `json:"content"`
	RawContent         any            `json:"rawContent"`
	Summary            any            `json:"summary"`
	AITitle            any            `json:"aiTitle"`
	Payload            map[string]any `json:"payload"`
	Data               map[string]any `json:"data"`
	Result             map[string]any `json:"result"`
	Response           map[string]any `json:"response"`
	Message            any            `json:"message"`
	Usage              any            `json:"usage"`
	CompactMetadata    map[string]any `json:"compactMetadata"`
	IsCompactSummary   any            `json:"isCompactSummary"`
	ReplacementHistory any            `json:"replacement_history"`
	Model              any            `json:"model"`
	ModelName          any            `json:"model_name"`
	Metadata           map[string]any `json:"metadata"`
	ProviderData       map[string]any `json:"providerData"`
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

	return parseFromReader(path, file, sourceID, sourceFileID)
}

func ParseFileWithHash(path string, sourceID, sourceFileID int64) (model.ParsedSession, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.ParsedSession{}, "", err
	}
	defer file.Close()

	hash := sha256.New()
	parsed, err := parseFromReader(path, io.TeeReader(file, hash), sourceID, sourceFileID)
	if err != nil {
		return parsed, "", err
	}
	return parsed, hex.EncodeToString(hash.Sum(nil)), nil
}

func parseFromReader(path string, reader io.Reader, sourceID, sourceFileID int64) (model.ParsedSession, error) {
	accumulator := newParseAccumulator(path, sourceID, sourceFileID)
	records := newRawRecordReader(path, reader)
	for records.Next() {
		accumulator.handleRecord(records.Record())
	}
	if err := records.Err(); err != nil {
		return accumulator.parsed, err
	}
	accumulator.addWarnings(records.Warnings())
	return accumulator.finalize(), nil
}

func codexReplacementHistoryCount(raw rawRecord) int {
	if count := replacementHistoryCount(raw.Payload["replacement_history"]); count > 0 {
		return count
	}
	return replacementHistoryCount(raw.ReplacementHistory)
}

func replacementHistoryCount(value any) int {
	switch value := value.(type) {
	case []any:
		return len(value)
	case nil:
		return 0
	default:
		return 1
	}
}

func isCodexCompactionSnapshotUsage(usage model.Usage) bool {
	return usage.TotalTokens > 0 &&
		usage.InputTokens == 0 &&
		usage.CachedInputTokens == 0 &&
		usage.OutputTokens == 0 &&
		usage.ReasoningOutputTokens == 0 &&
		usage.ContextCompressionTokens == 0
}

func isCodeBuddyCompactSummary(raw rawRecord) bool {
	if raw.ProviderData == nil {
		return false
	}
	if (boolValue(raw.ProviderData, "isCompacted") || boolValue(raw.ProviderData, "isSummary")) &&
		(boolValue(raw.ProviderData, "isCompactInternal") || stringValue(raw.ProviderData, "compactType") != "") {
		return true
	}
	return isCodeBuddyCompactAgent(raw) && hasCodeBuddyCompactSummaryText(codeBuddyRecordText(raw))
}

func isCodeBuddyCompactUsage(raw rawRecord) bool {
	if raw.ProviderData == nil {
		return false
	}
	return isCodeBuddyCompactAgent(raw) ||
		(boolValue(raw.ProviderData, "isCompactInternal") && stringValue(raw.ProviderData, "compactType") != "")
}

func isCodeBuddyCompactAgent(raw rawRecord) bool {
	return strings.EqualFold(stringValue(raw.ProviderData, "agent"), "compact")
}

func hasCodeBuddyCompactSummaryText(text string) bool {
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "<summary>") && strings.Contains(lower, "</summary>")) ||
		(strings.Contains(lower, "<conversation_history_summary>") && strings.Contains(lower, "</conversation_history_summary>"))
}

func codeBuddyRecordText(raw rawRecord) string {
	values := []string{
		contentText(raw.Content),
		contentText(raw.RawContent),
		stringFromAny(raw.Summary),
	}
	if raw.Message != nil {
		message := mapFromAny(raw.Message)
		values = append([]string{contentText(message["content"])}, values...)
	}
	return firstNonEmpty(values...)
}

func recordTimestamp(raw rawRecord) time.Time {
	for _, value := range []any{raw.Timestamp, raw.TimestampMS, raw.TimestampMSCamel, raw.CreatedAt, raw.CreatedAtCamel} {
		if ts := parseTimestampValue(value); !ts.IsZero() {
			return ts
		}
	}
	for _, container := range []map[string]any{raw.Data, raw.Result, raw.Response, mapFromAny(raw.Message), raw.Metadata} {
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
		modelFromMap(mapFromAny(raw.Message)),
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

func mapFromAny(value any) map[string]any {
	asMap, _ := value.(map[string]any)
	return asMap
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

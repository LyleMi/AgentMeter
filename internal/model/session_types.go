package model

import "time"

type SourceFile struct {
	ID            int64     `json:"id"`
	SourceID      int64     `json:"sourceId"`
	Path          string    `json:"path"`
	SizeBytes     int64     `json:"sizeBytes"`
	ModifiedAt    time.Time `json:"modifiedAt"`
	ContentHash   string    `json:"contentHash"`
	LastScannedAt time.Time `json:"lastScannedAt"`
	ScanStatus    string    `json:"scanStatus"`
	Error         string    `json:"error"`
}

type Session struct {
	ID                     int64     `json:"id"`
	SourceID               int64     `json:"sourceId"`
	SourceKey              string    `json:"sourceKey"`
	SourceLabel            string    `json:"sourceLabel"`
	SourceRootPath         string    `json:"sourceRootPath"`
	SourceSessionsPath     string    `json:"sourceSessionsPath"`
	SourceFileID           int64     `json:"sourceFileId"`
	AgentKind              string    `json:"agentKind"`
	AgentName              string    `json:"agentName"`
	SessionKey             string    `json:"sessionKey"`
	CodexSessionID         string    `json:"codexSessionId,omitempty"`
	ProjectPath            string    `json:"projectPath"`
	Model                  string    `json:"model"`
	ModelProvider          string    `json:"modelProvider"`
	Originator             string    `json:"originator"`
	ThreadSource           string    `json:"threadSource"`
	AgentNickname          string    `json:"agentNickname"`
	AgentRole              string    `json:"agentRole"`
	StartedAt              time.Time `json:"startedAt"`
	EndedAt                time.Time `json:"endedAt"`
	WallDurationMS         int64     `json:"wallDurationMs"`
	ActiveDurationMS       int64     `json:"activeDurationMs"`
	ModelDurationMS        int64     `json:"modelDurationMs"`
	ToolDurationMS         int64     `json:"toolDurationMs"`
	IdleDurationMS         int64     `json:"idleDurationMs"`
	EventCount             int       `json:"eventCount"`
	ParseStatus            string    `json:"parseStatus"`
	TokenUsage             Usage     `json:"tokenUsage"`
	EstimatedCostUSD       *float64  `json:"estimatedCostUsd,omitempty"`
	Unpriced               bool      `json:"unpriced"`
	ToolCallCount          int       `json:"toolCallCount"`
	RawSourcePath          string    `json:"rawSourcePath"`
	LastIndexedScanStatus  string    `json:"lastIndexedScanStatus"`
	LastIndexedScanMessage string    `json:"lastIndexedScanMessage"`
}

type Usage struct {
	Model                    string   `json:"model"`
	InputTokens              int64    `json:"inputTokens"`
	CachedInputTokens        int64    `json:"cachedInputTokens"`
	OutputTokens             int64    `json:"outputTokens"`
	ReasoningOutputTokens    int64    `json:"reasoningOutputTokens"`
	ContextCompressionTokens int64    `json:"contextCompressionTokens"`
	TotalTokens              int64    `json:"totalTokens"`
	Source                   string   `json:"source"`
	CostUSD                  *float64 `json:"costUsd,omitempty"`
	Unpriced                 bool     `json:"unpriced"`
}

type Event struct {
	ID           int64     `json:"id"`
	SessionID    int64     `json:"sessionId"`
	SourceFileID int64     `json:"sourceFileId"`
	SourceLine   int       `json:"sourceLine"`
	Timestamp    time.Time `json:"timestamp"`
	Kind         string    `json:"kind"`
	RawType      string    `json:"rawType"`
	Summary      string    `json:"summary"`
	RawJSON      string    `json:"rawJson,omitempty"`
}

type ModelCall struct {
	ID                       int64     `json:"id"`
	SessionID                int64     `json:"sessionId"`
	StartedAt                time.Time `json:"startedAt"`
	EndedAt                  time.Time `json:"endedAt"`
	DurationMS               int64     `json:"durationMs"`
	Model                    string    `json:"model"`
	Provider                 string    `json:"provider"`
	Status                   string    `json:"status"`
	InputTokens              int64     `json:"inputTokens"`
	CachedInputTokens        int64     `json:"cachedInputTokens"`
	OutputTokens             int64     `json:"outputTokens"`
	ReasoningOutputTokens    int64     `json:"reasoningOutputTokens"`
	ContextCompressionTokens int64     `json:"contextCompressionTokens"`
	TotalTokens              int64     `json:"totalTokens"`
	CostUSD                  *float64  `json:"costUsd,omitempty"`
	Unpriced                 bool      `json:"unpriced"`
}

type ToolCall struct {
	ID                   int64     `json:"id"`
	SessionID            int64     `json:"sessionId"`
	SourceID             int64     `json:"sourceId"`
	SourceKey            string    `json:"sourceKey,omitempty"`
	SourceLabel          string    `json:"sourceLabel,omitempty"`
	SourceRootPath       string    `json:"sourceRootPath,omitempty"`
	SourceSessionsPath   string    `json:"sourceSessionsPath,omitempty"`
	StartedAt            time.Time `json:"startedAt"`
	EndedAt              time.Time `json:"endedAt"`
	DurationMS           int64     `json:"durationMs"`
	ToolName             string    `json:"toolName"`
	Status               string    `json:"status"`
	InputSummary         string    `json:"inputSummary"`
	OutputSummary        string    `json:"outputSummary"`
	Error                string    `json:"error"`
	CallID               string    `json:"callId,omitempty"`
	RawEventID           int64     `json:"rawEventId"`
	RawStartEventID      int64     `json:"rawStartEventId,omitempty"`
	RawEndEventID        int64     `json:"rawEndEventId,omitempty"`
	RawEventLine         int       `json:"rawEventLine,omitempty"`
	RawStartEventLine    int       `json:"rawStartEventLine,omitempty"`
	RawEndEventLine      int       `json:"rawEndEventLine,omitempty"`
	RawStartEventType    string    `json:"rawStartEventType,omitempty"`
	RawEndEventType      string    `json:"rawEndEventType,omitempty"`
	RawStartEventSummary string    `json:"rawStartEventSummary,omitempty"`
	RawEndEventSummary   string    `json:"rawEndEventSummary,omitempty"`
	RawStartEventJSON    string    `json:"rawStartEventJson,omitempty"`
	RawEndEventJSON      string    `json:"rawEndEventJson,omitempty"`
	SessionKey           string    `json:"sessionKey,omitempty"`
	CodexSessionID       string    `json:"codexSessionId,omitempty"`
	ProjectPath          string    `json:"projectPath,omitempty"`
	AgentKind            string    `json:"agentKind,omitempty"`
	AgentName            string    `json:"agentName,omitempty"`
	RawSourcePath        string    `json:"rawSourcePath,omitempty"`
	RiskScore            int       `json:"riskScore,omitempty"`
	RiskSeverity         string    `json:"riskSeverity,omitempty"`
	RiskCount            int       `json:"riskCount,omitempty"`
	RiskRuleIDs          []string  `json:"riskRuleIds,omitempty"`
}

type SessionDetail struct {
	Session    Session     `json:"session"`
	Events     []Event     `json:"events"`
	ModelCalls []ModelCall `json:"modelCalls"`
	ToolCalls  []ToolCall  `json:"toolCalls"`
}

type ParsedSession struct {
	Session   Session
	Events    []Event
	Usage     Usage
	ModelCall []ModelCall
	ToolCall  []ToolCall
	Warnings  []string
}

type SessionFilters struct {
	Search string `json:"search"`
	Model  string `json:"model"`
	Agent  string `json:"agent"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

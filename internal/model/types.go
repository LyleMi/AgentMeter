package model

import "time"

type Source struct {
	ID           int64     `json:"id"`
	Kind         string    `json:"kind"`
	Name         string    `json:"name"`
	RootPath     string    `json:"rootPath"`
	SessionsPath string    `json:"sessionsPath"`
	Platform     string    `json:"platform"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type SourceEntry struct {
	Path    string `json:"path"`
	Enabled bool   `json:"enabled"`
}

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
	Model                 string   `json:"model"`
	InputTokens           int64    `json:"inputTokens"`
	CachedInputTokens     int64    `json:"cachedInputTokens"`
	OutputTokens          int64    `json:"outputTokens"`
	ReasoningOutputTokens int64    `json:"reasoningOutputTokens"`
	TotalTokens           int64    `json:"totalTokens"`
	Source                string   `json:"source"`
	CostUSD               *float64 `json:"costUsd,omitempty"`
	Unpriced              bool     `json:"unpriced"`
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
	ID                    int64     `json:"id"`
	SessionID             int64     `json:"sessionId"`
	StartedAt             time.Time `json:"startedAt"`
	EndedAt               time.Time `json:"endedAt"`
	DurationMS            int64     `json:"durationMs"`
	Model                 string    `json:"model"`
	Provider              string    `json:"provider"`
	Status                string    `json:"status"`
	InputTokens           int64     `json:"inputTokens"`
	CachedInputTokens     int64     `json:"cachedInputTokens"`
	OutputTokens          int64     `json:"outputTokens"`
	ReasoningOutputTokens int64     `json:"reasoningOutputTokens"`
	TotalTokens           int64     `json:"totalTokens"`
	CostUSD               *float64  `json:"costUsd,omitempty"`
	Unpriced              bool      `json:"unpriced"`
}

type ToolCall struct {
	ID                   int64     `json:"id"`
	SessionID            int64     `json:"sessionId"`
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
}

type PricingModel struct {
	ID               int64     `json:"id"`
	Model            string    `json:"model"`
	NormalizedModel  string    `json:"normalizedModel"`
	InputPer1M       float64   `json:"inputPer1m"`
	CachedInputPer1M float64   `json:"cachedInputPer1m"`
	OutputPer1M      float64   `json:"outputPer1m"`
	Source           string    `json:"source"`
	EffectiveFrom    time.Time `json:"effectiveFrom"`
}

type DailyUsage struct {
	Date             string   `json:"date"`
	SessionCount     int      `json:"sessionCount"`
	TotalTokens      int64    `json:"totalTokens"`
	InputTokens      int64    `json:"inputTokens"`
	OutputTokens     int64    `json:"outputTokens"`
	ToolCalls        int      `json:"toolCalls"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
}

type ModelUsage struct {
	Model            string   `json:"model"`
	SessionCount     int      `json:"sessionCount"`
	TotalTokens      int64    `json:"totalTokens"`
	InputTokens      int64    `json:"inputTokens"`
	OutputTokens     int64    `json:"outputTokens"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
	Unpriced         bool     `json:"unpriced"`
}

type AgentUsage struct {
	AgentKind        string   `json:"agentKind"`
	AgentName        string   `json:"agentName"`
	SessionCount     int      `json:"sessionCount"`
	TotalTokens      int64    `json:"totalTokens"`
	InputTokens      int64    `json:"inputTokens"`
	OutputTokens     int64    `json:"outputTokens"`
	ToolCalls        int      `json:"toolCalls"`
	EstimatedCostUSD *float64 `json:"estimatedCostUsd,omitempty"`
	Unpriced         bool     `json:"unpriced"`
}

type ToolTimeUsage struct {
	ToolName         string  `json:"toolName"`
	Calls            int     `json:"calls"`
	SuccessCalls     int     `json:"successCalls"`
	FailedCalls      int     `json:"failedCalls"`
	TotalDurationMS  int64   `json:"totalDurationMs"`
	AvgDurationMS    float64 `json:"avgDurationMs"`
	MaxDurationMS    int64   `json:"maxDurationMs"`
	SuspectedNetwork bool    `json:"suspectedNetwork"`
}

type AgentTimeUsage struct {
	AgentKind                      string `json:"agentKind"`
	AgentName                      string `json:"agentName"`
	SessionCount                   int    `json:"sessionCount"`
	ToolCalls                      int    `json:"toolCalls"`
	WallDurationMS                 int64  `json:"wallDurationMs"`
	ActiveDurationMS               int64  `json:"activeDurationMs"`
	ModelDurationMS                int64  `json:"modelDurationMs"`
	ToolDurationMS                 int64  `json:"toolDurationMs"`
	IdleDurationMS                 int64  `json:"idleDurationMs"`
	SuspectedNetworkToolDurationMS int64  `json:"suspectedNetworkToolDurationMs"`
}

type ModelTimeUsage struct {
	Model            string `json:"model"`
	SessionCount     int    `json:"sessionCount"`
	TotalTokens      int64  `json:"totalTokens"`
	WallDurationMS   int64  `json:"wallDurationMs"`
	ActiveDurationMS int64  `json:"activeDurationMs"`
	ModelDurationMS  int64  `json:"modelDurationMs"`
	ToolDurationMS   int64  `json:"toolDurationMs"`
	IdleDurationMS   int64  `json:"idleDurationMs"`
}

type Overview struct {
	TotalSessions                  int              `json:"totalSessions"`
	TotalInputTokens               int64            `json:"totalInputTokens"`
	TotalCachedInputTokens         int64            `json:"totalCachedInputTokens"`
	TotalOutputTokens              int64            `json:"totalOutputTokens"`
	TotalReasoningTokens           int64            `json:"totalReasoningTokens"`
	TotalTokens                    int64            `json:"totalTokens"`
	EstimatedCostUSD               *float64         `json:"estimatedCostUsd,omitempty"`
	UnpricedSessions               int              `json:"unpricedSessions"`
	TotalWallDurationMS            int64            `json:"totalWallDurationMs"`
	TotalActiveDurationMS          int64            `json:"totalActiveDurationMs"`
	TotalModelDurationMS           int64            `json:"totalModelDurationMs"`
	TotalToolDurationMS            int64            `json:"totalToolDurationMs"`
	TotalIdleDurationMS            int64            `json:"totalIdleDurationMs"`
	TotalToolCalls                 int              `json:"totalToolCalls"`
	SuspectedNetworkToolDurationMS int64            `json:"suspectedNetworkToolDurationMs"`
	SuspectedNetworkToolCalls      int              `json:"suspectedNetworkToolCalls"`
	DailyUsage                     []DailyUsage     `json:"dailyUsage"`
	ModelUsage                     []ModelUsage     `json:"modelUsage"`
	AgentUsage                     []AgentUsage     `json:"agentUsage"`
	ToolTimeLeaders                []ToolTimeUsage  `json:"toolTimeLeaders"`
	AgentTimeUsage                 []AgentTimeUsage `json:"agentTimeUsage"`
	ModelTimeUsage                 []ModelTimeUsage `json:"modelTimeUsage"`
	RecentSessions                 []Session        `json:"recentSessions"`
	SlowSessions                   []Session        `json:"slowSessions"`
}

type ToolStat struct {
	ToolName        string  `json:"toolName"`
	Calls           int     `json:"calls"`
	SuccessCalls    int     `json:"successCalls"`
	FailedCalls     int     `json:"failedCalls"`
	TotalDurationMS int64   `json:"totalDurationMs"`
	AvgDurationMS   float64 `json:"avgDurationMs"`
}

type AuditFinding struct {
	ID             int64     `json:"id"`
	SessionID      int64     `json:"sessionId"`
	ToolCallID     int64     `json:"toolCallId"`
	SourceFileID   int64     `json:"sourceFileId"`
	RawEventID     int64     `json:"rawEventId"`
	SourceLine     int       `json:"sourceLine"`
	Timestamp      time.Time `json:"timestamp"`
	Source         string    `json:"source"`
	EventType      string    `json:"eventType"`
	Category       string    `json:"category"`
	Severity       string    `json:"severity"`
	RuleID         string    `json:"ruleId"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Evidence       string    `json:"evidence"`
	Command        string    `json:"command"`
	ShellFamily    string    `json:"shellFamily"`
	Platform       string    `json:"platform"`
	Decision       string    `json:"decision"`
	CreatedAt      time.Time `json:"createdAt"`
	SessionKey     string    `json:"sessionKey,omitempty"`
	CodexSessionID string    `json:"codexSessionId,omitempty"`
	ProjectPath    string    `json:"projectPath,omitempty"`
	AgentKind      string    `json:"agentKind,omitempty"`
	AgentName      string    `json:"agentName,omitempty"`
	RawSourcePath  string    `json:"rawSourcePath,omitempty"`
}

type AuditSummary struct {
	TotalFindings        int            `json:"totalFindings"`
	CriticalFindings     int            `json:"criticalFindings"`
	HighFindings         int            `json:"highFindings"`
	MediumFindings       int            `json:"mediumFindings"`
	LowFindings          int            `json:"lowFindings"`
	CommandFindings      int            `json:"commandFindings"`
	PrivacyFindings      int            `json:"privacyFindings"`
	EgressFindings       int            `json:"egressFindings"`
	FileFindings         int            `json:"fileFindings"`
	SessionsWithFindings int            `json:"sessionsWithFindings"`
	RecentFindings       []AuditFinding `json:"recentFindings"`
}

type ToolFilters struct {
	Agent string `json:"agent"`
}

type SessionDetail struct {
	Session    Session     `json:"session"`
	Events     []Event     `json:"events"`
	ModelCalls []ModelCall `json:"modelCalls"`
	ToolCalls  []ToolCall  `json:"toolCalls"`
}

type Settings struct {
	SourcePath         string         `json:"sourcePath"`
	SourcePaths        []string       `json:"sourcePaths"`
	SourceEntries      []SourceEntry  `json:"sourceEntries"`
	DefaultSourcePath  string         `json:"defaultSourcePath"`
	DefaultSourcePaths []string       `json:"defaultSourcePaths"`
	DatabasePath       string         `json:"databasePath"`
	PricingModels      []PricingModel `json:"pricingModels"`
	LastIndexStartedAt *time.Time     `json:"lastIndexStartedAt,omitempty"`
	LastIndexResult    *IndexResult   `json:"lastIndexResult,omitempty"`
}

type IndexResult struct {
	SourcePath  string   `json:"sourcePath"`
	SourcePaths []string `json:"sourcePaths"`
	Database    string   `json:"database"`
	FilesSeen   int      `json:"filesSeen"`
	Indexed     int      `json:"indexed"`
	Skipped     int      `json:"skipped"`
	Failed      int      `json:"failed"`
	Sessions    int      `json:"sessions"`
	Warnings    []string `json:"warnings"`
	DurationMS  int64    `json:"durationMs"`
	Rebuild     bool     `json:"rebuild"`
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

type ToolCallFilters struct {
	ToolName    string `json:"toolName"`
	Agent       string `json:"agent"`
	StartedFrom string `json:"startedFrom"`
	StartedTo   string `json:"startedTo"`
	Sort        string `json:"sort"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
}

type AuditFindingFilters struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	ShellFamily string `json:"shellFamily"`
	Search      string `json:"search"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
}

type PrivacyConfigStatus struct {
	Target     string                 `json:"target"`
	Name       string                 `json:"name"`
	ConfigPath string                 `json:"configPath"`
	Exists     bool                   `json:"exists"`
	Summary    PrivacyConfigSummary   `json:"summary"`
	Settings   []PrivacyConfigSetting `json:"settings"`
	Warnings   []string               `json:"warnings"`
}

type PrivacyConfigSummary struct {
	Score     int `json:"score"`
	Total     int `json:"total"`
	Hardened  int `json:"hardened"`
	Attention int `json:"attention"`
	Implicit  int `json:"implicit"`
}

type PrivacyConfigSetting struct {
	ID            string `json:"id"`
	Group         string `json:"group"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Key           string `json:"key"`
	DesiredValue  any    `json:"desiredValue"`
	StrictValue   any    `json:"strictValue"`
	ValueType     string `json:"valueType"`
	Configured    bool   `json:"configured"`
	SupportsUnset bool   `json:"supportsUnset"`
	CurrentValue  any    `json:"currentValue"`
	Status        string `json:"status"`
	Impact        string `json:"impact"`
	CanApply      bool   `json:"canApply"`
}

type PrivacyConfigEdit struct {
	ID    string `json:"id"`
	Op    string `json:"op"`
	Value any    `json:"value,omitempty"`
}

type PrivacyConfigChange struct {
	ID     string `json:"id"`
	Key    string `json:"key"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

type PrivacyConfigApplyResult struct {
	Status     PrivacyConfigStatus   `json:"status"`
	Changed    []PrivacyConfigChange `json:"changed"`
	BackupPath string                `json:"backupPath"`
	Warnings   []string              `json:"warnings"`
}

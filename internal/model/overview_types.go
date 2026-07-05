package model

type DailyUsage struct {
	Date                     string   `json:"date"`
	SessionCount             int      `json:"sessionCount"`
	TotalTokens              int64    `json:"totalTokens"`
	InputTokens              int64    `json:"inputTokens"`
	CachedInputTokens        int64    `json:"cachedInputTokens"`
	OutputTokens             int64    `json:"outputTokens"`
	ContextCompressionTokens int64    `json:"contextCompressionTokens"`
	ToolCalls                int      `json:"toolCalls"`
	CacheUtilizationRate     float64  `json:"cacheUtilizationRate"`
	EstimatedCostUSD         *float64 `json:"estimatedCostUsd,omitempty"`
}

type CacheHitTrendPoint struct {
	Date                        string  `json:"date"`
	SessionCount                int     `json:"sessionCount"`
	TotalTokens                 int64   `json:"totalTokens"`
	InputTokens                 int64   `json:"inputTokens"`
	CachedInputTokens           int64   `json:"cachedInputTokens"`
	CacheUtilizationRate        float64 `json:"cacheUtilizationRate"`
	RollingCacheUtilizationRate float64 `json:"rollingCacheUtilizationRate"`
	LowInputVolume              bool    `json:"lowInputVolume"`
	HasUsage                    bool    `json:"hasUsage"`
}

type ModelUsage struct {
	Model                    string   `json:"model"`
	SessionCount             int      `json:"sessionCount"`
	TotalTokens              int64    `json:"totalTokens"`
	InputTokens              int64    `json:"inputTokens"`
	CachedInputTokens        int64    `json:"cachedInputTokens"`
	OutputTokens             int64    `json:"outputTokens"`
	ReasoningOutputTokens    int64    `json:"reasoningOutputTokens"`
	ContextCompressionTokens int64    `json:"contextCompressionTokens"`
	EstimatedCostUSD         *float64 `json:"estimatedCostUsd,omitempty"`
	Unpriced                 bool     `json:"unpriced"`
}

type AgentUsage struct {
	SourceID                 int64    `json:"sourceId"`
	SourceKey                string   `json:"sourceKey"`
	SourceLabel              string   `json:"sourceLabel"`
	SourceRootPath           string   `json:"sourceRootPath"`
	SourceSessionsPath       string   `json:"sourceSessionsPath"`
	AgentKind                string   `json:"agentKind"`
	AgentName                string   `json:"agentName"`
	SessionCount             int      `json:"sessionCount"`
	TotalTokens              int64    `json:"totalTokens"`
	InputTokens              int64    `json:"inputTokens"`
	CachedInputTokens        int64    `json:"cachedInputTokens"`
	OutputTokens             int64    `json:"outputTokens"`
	ReasoningOutputTokens    int64    `json:"reasoningOutputTokens"`
	ContextCompressionTokens int64    `json:"contextCompressionTokens"`
	CacheUtilizationRate     float64  `json:"cacheUtilizationRate"`
	ToolCalls                int      `json:"toolCalls"`
	EstimatedCostUSD         *float64 `json:"estimatedCostUsd,omitempty"`
	Unpriced                 bool     `json:"unpriced"`
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
	SourceID                       int64  `json:"sourceId"`
	SourceKey                      string `json:"sourceKey"`
	SourceLabel                    string `json:"sourceLabel"`
	SourceRootPath                 string `json:"sourceRootPath"`
	SourceSessionsPath             string `json:"sourceSessionsPath"`
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

type AnalyticsFilters struct {
	Agent       string `json:"agent"`
	Model       string `json:"model"`
	Project     string `json:"project"`
	StartedFrom string `json:"startedFrom"`
	StartedTo   string `json:"startedTo"`
}

type Overview struct {
	TotalSessions                  int                  `json:"totalSessions"`
	TotalInputTokens               int64                `json:"totalInputTokens"`
	TotalCachedInputTokens         int64                `json:"totalCachedInputTokens"`
	TotalOutputTokens              int64                `json:"totalOutputTokens"`
	TotalReasoningTokens           int64                `json:"totalReasoningTokens"`
	TotalContextCompressionTokens  int64                `json:"totalContextCompressionTokens"`
	TotalTokens                    int64                `json:"totalTokens"`
	EstimatedCostUSD               *float64             `json:"estimatedCostUsd,omitempty"`
	UnpricedSessions               int                  `json:"unpricedSessions"`
	TotalWallDurationMS            int64                `json:"totalWallDurationMs"`
	TotalActiveDurationMS          int64                `json:"totalActiveDurationMs"`
	TotalModelDurationMS           int64                `json:"totalModelDurationMs"`
	TotalToolDurationMS            int64                `json:"totalToolDurationMs"`
	TotalIdleDurationMS            int64                `json:"totalIdleDurationMs"`
	TotalToolCalls                 int                  `json:"totalToolCalls"`
	SuspectedNetworkToolDurationMS int64                `json:"suspectedNetworkToolDurationMs"`
	SuspectedNetworkToolCalls      int                  `json:"suspectedNetworkToolCalls"`
	DailyUsage                     []DailyUsage         `json:"dailyUsage"`
	CacheHitTrend                  []CacheHitTrendPoint `json:"cacheHitTrend"`
	ModelUsage                     []ModelUsage         `json:"modelUsage"`
	AgentUsage                     []AgentUsage         `json:"agentUsage"`
	ToolTimeLeaders                []ToolTimeUsage      `json:"toolTimeLeaders"`
	AgentTimeUsage                 []AgentTimeUsage     `json:"agentTimeUsage"`
	ModelTimeUsage                 []ModelTimeUsage     `json:"modelTimeUsage"`
	RecentSessions                 []Session            `json:"recentSessions"`
	SlowSessions                   []Session            `json:"slowSessions"`
}

type TokenAnalytics struct {
	TotalSessions                 int                  `json:"totalSessions"`
	TotalInputTokens              int64                `json:"totalInputTokens"`
	TotalCachedInputTokens        int64                `json:"totalCachedInputTokens"`
	TotalOutputTokens             int64                `json:"totalOutputTokens"`
	TotalReasoningTokens          int64                `json:"totalReasoningTokens"`
	TotalContextCompressionTokens int64                `json:"totalContextCompressionTokens"`
	TotalTokens                   int64                `json:"totalTokens"`
	CacheUtilizationRate          float64              `json:"cacheUtilizationRate"`
	EstimatedCostUSD              *float64             `json:"estimatedCostUsd,omitempty"`
	UnpricedCount                 int                  `json:"unpricedCount"`
	CacheHitTrend                 []CacheHitTrendPoint `json:"cacheHitTrend"`
	ModelUsage                    []ModelUsage         `json:"modelUsage"`
	AgentUsage                    []AgentUsage         `json:"agentUsage"`
	RecentSessions                []Session            `json:"recentSessions"`
	HighTokenSessions             []Session            `json:"highTokenSessions"`
}

type UsageBreakdown struct {
	GroupBy string                 `json:"groupBy"`
	Buckets []UsageBreakdownBucket `json:"buckets"`
}

type UsageBreakdownBucket struct {
	SourceID                 int64    `json:"sourceId,omitempty"`
	SourceKey                string   `json:"sourceKey,omitempty"`
	SourceLabel              string   `json:"sourceLabel,omitempty"`
	SourceRootPath           string   `json:"sourceRootPath,omitempty"`
	SourceSessionsPath       string   `json:"sourceSessionsPath,omitempty"`
	AgentKind                string   `json:"agentKind,omitempty"`
	AgentName                string   `json:"agentName,omitempty"`
	Model                    string   `json:"model,omitempty"`
	ProjectPath              string   `json:"projectPath,omitempty"`
	Date                     string   `json:"date,omitempty"`
	SessionCount             int      `json:"sessionCount"`
	TotalTokens              int64    `json:"totalTokens"`
	InputTokens              int64    `json:"inputTokens"`
	CachedInputTokens        int64    `json:"cachedInputTokens"`
	OutputTokens             int64    `json:"outputTokens"`
	ReasoningOutputTokens    int64    `json:"reasoningOutputTokens"`
	ContextCompressionTokens int64    `json:"contextCompressionTokens"`
	CacheUtilizationRate     float64  `json:"cacheUtilizationRate"`
	EstimatedCostUSD         *float64 `json:"estimatedCostUsd,omitempty"`
	Unpriced                 bool     `json:"unpriced"`
}

type ToolStat struct {
	ToolName        string  `json:"toolName"`
	Calls           int     `json:"calls"`
	SuccessCalls    int     `json:"successCalls"`
	FailedCalls     int     `json:"failedCalls"`
	TotalDurationMS int64   `json:"totalDurationMs"`
	AvgDurationMS   float64 `json:"avgDurationMs"`
}

type ToolFilters struct {
	Agent string `json:"agent"`
}

type ToolCallFilters struct {
	ToolName    string `json:"toolName"`
	Agent       string `json:"agent"`
	StartedFrom string `json:"startedFrom"`
	StartedTo   string `json:"startedTo"`
	Sort        string `json:"sort"`
	Shell       bool   `json:"shell"`
	RiskOnly    bool   `json:"riskOnly"`
	IncludeRisk bool   `json:"includeRisk"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
}

type ToolCallRiskFilters struct {
	Agent       string `json:"agent"`
	StartedFrom string `json:"startedFrom"`
	StartedTo   string `json:"startedTo"`
	Limit       int    `json:"limit"`
}

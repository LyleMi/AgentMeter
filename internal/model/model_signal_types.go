package model

import "time"

type ModelSignals struct {
	TotalSessions                        int                          `json:"totalSessions"`
	TotalModelCalls                      int                          `json:"totalModelCalls"`
	TotalToolCalls                       int                          `json:"totalToolCalls"`
	FailedToolCalls                      int                          `json:"failedToolCalls"`
	ToolFailureRate                      float64                      `json:"toolFailureRate"`
	ToolDependencyRate                   float64                      `json:"toolDependencyRate"`
	AvgModelCallsPerSession              float64                      `json:"avgModelCallsPerSession"`
	OutputExpansionRate                  float64                      `json:"outputExpansionRate"`
	ReasoningTokenShare                  float64                      `json:"reasoningTokenShare"`
	ReasoningOverheadRate                float64                      `json:"reasoningOverheadRate"`
	VisibleOutputTokens                  int64                        `json:"visibleOutputTokens"`
	BillableOutputTokens                 int64                        `json:"billableOutputTokens"`
	CacheMissRate                        float64                      `json:"cacheMissRate"`
	ModelThroughputTokensPerSecond       float64                      `json:"modelThroughputTokensPerSecond"`
	ModelThroughputOutputTokensPerSecond float64                      `json:"modelThroughputOutputTokensPerSecond"`
	Trend                                []ModelSignalsTrendPoint     `json:"trend"`
	ModelBreakdown                       []ModelSignalsBreakdown      `json:"modelBreakdown"`
	AnomalySessions                      []ModelSignalsAnomalySession `json:"anomalySessions"`
	HealthSummary                        ModelSignalsHealthSummary    `json:"healthSummary"`
	Cohorts                              []ModelSignalsCohort         `json:"cohorts"`
	Matrix                               []ModelSignalsMatrixRow      `json:"matrix"`
	ProjectHotspots                      []ModelSignalsProjectHotspot `json:"projectHotspots"`
	DailyMetrics                         []ModelSignalsDailyMetric    `json:"dailyMetrics"`
	ProjectMetrics                       []ModelSignalsProjectMetric  `json:"projectMetrics"`
}

type ModelSignalsWindow struct {
	From         string `json:"from"`
	To           string `json:"to"`
	SessionCount int    `json:"sessionCount"`
	ModelCalls   int    `json:"modelCalls"`
}

type ModelSignalsMetricSet struct {
	SessionCount                         int      `json:"sessionCount"`
	ModelCalls                           int      `json:"modelCalls"`
	FailedModelCalls                     int      `json:"failedModelCalls"`
	ToolCalls                            int      `json:"toolCalls"`
	FailedToolCalls                      int      `json:"failedToolCalls"`
	TotalTokens                          int64    `json:"totalTokens"`
	InputTokens                          int64    `json:"inputTokens"`
	CachedInputTokens                    int64    `json:"cachedInputTokens"`
	OutputTokens                         int64    `json:"outputTokens"`
	ReasoningOutputTokens                int64    `json:"reasoningOutputTokens"`
	VisibleOutputTokens                  int64    `json:"visibleOutputTokens"`
	BillableOutputTokens                 int64    `json:"billableOutputTokens"`
	WallDurationMS                       int64    `json:"wallDurationMs"`
	ActiveDurationMS                     int64    `json:"activeDurationMs"`
	ModelDurationMS                      int64    `json:"modelDurationMs"`
	ToolDurationMS                       int64    `json:"toolDurationMs"`
	IdleDurationMS                       int64    `json:"idleDurationMs"`
	EstimatedCostUSD                     *float64 `json:"estimatedCostUsd,omitempty"`
	UnpricedSessionCount                 int      `json:"unpricedSessionCount"`
	CacheSavingsUSD                      *float64 `json:"cacheSavingsUsd,omitempty"`
	AvgModelCallsPerSession              float64  `json:"avgModelCallsPerSession"`
	OutputExpansionRate                  float64  `json:"outputExpansionRate"`
	ReasoningTokenShare                  float64  `json:"reasoningTokenShare"`
	ReasoningOverheadRate                float64  `json:"reasoningOverheadRate"`
	CacheMissRate                        float64  `json:"cacheMissRate"`
	CostPerSession                       *float64 `json:"costPerSession,omitempty"`
	CostPerActiveHour                    *float64 `json:"costPerActiveHour,omitempty"`
	CostPer1kTokens                      *float64 `json:"costPer1kTokens,omitempty"`
	FailurePressure                      float64  `json:"failurePressure"`
	DegradationRiskScore                 float64  `json:"degradationRiskScore"`
	ModelThroughputTokensPerSecond       float64  `json:"modelThroughputTokensPerSecond"`
	ModelThroughputOutputTokensPerSecond float64  `json:"modelThroughputOutputTokensPerSecond"`
	ModelLatencyMsPer1kOutputTokens      float64  `json:"modelLatencyMsPer1kOutputTokens"`
	ToolFailureRate                      float64  `json:"toolFailureRate"`
	ToolDependencyRate                   float64  `json:"toolDependencyRate"`
	P50ModelLatencyMsPer1kOutputTokens   float64  `json:"p50ModelLatencyMsPer1kOutputTokens"`
	P90ModelLatencyMsPer1kOutputTokens   float64  `json:"p90ModelLatencyMsPer1kOutputTokens"`
	P50ModelThroughputTokensPerSecond    float64  `json:"p50ModelThroughputTokensPerSecond"`
	P10ModelThroughputTokensPerSecond    float64  `json:"p10ModelThroughputTokensPerSecond"`
}

type ModelSignalsDriftMetric struct {
	Key       string  `json:"key"`
	Label     string  `json:"label"`
	Direction string  `json:"direction"`
	Severity  string  `json:"severity"`
	Current   float64 `json:"current"`
	Baseline  float64 `json:"baseline"`
	Delta     float64 `json:"delta"`
	DeltaPct  float64 `json:"deltaPct"`
}

type ModelSignalsDrift struct {
	Severity   string                    `json:"severity"`
	Confidence string                    `json:"confidence"`
	SampleNote string                    `json:"sampleNote"`
	Reasons    []string                  `json:"reasons"`
	Metrics    []ModelSignalsDriftMetric `json:"metrics"`
}

type ModelSignalsHealthSummary struct {
	CurrentWindow        ModelSignalsWindow `json:"currentWindow"`
	BaselineWindow       ModelSignalsWindow `json:"baselineWindow"`
	Severity             string             `json:"severity"`
	CohortCount          int                `json:"cohortCount"`
	WarningCohorts       int                `json:"warningCohorts"`
	CriticalCohorts      int                `json:"criticalCohorts"`
	LowConfidenceCohorts int                `json:"lowConfidenceCohorts"`
	TopReasons           []string           `json:"topReasons"`
}

type ModelSignalsCohort struct {
	SourceID           int64  `json:"sourceId"`
	SourceKey          string `json:"sourceKey"`
	SourceLabel        string `json:"sourceLabel"`
	SourceRootPath     string `json:"sourceRootPath"`
	SourceSessionsPath string `json:"sourceSessionsPath"`
	AgentKind          string `json:"agentKind"`
	AgentName          string `json:"agentName"`
	ModelProvider      string `json:"modelProvider"`
	Model              string `json:"model"`
	ProjectPath        string `json:"projectPath"`
	CohortKey          string `json:"cohortKey"`
	ModelSignalsMetricSet
	Current  ModelSignalsMetricSet `json:"current"`
	Baseline ModelSignalsMetricSet `json:"baseline"`
	Drift    ModelSignalsDrift     `json:"drift"`
}

type ModelSignalsMatrixRow struct {
	SourceID           int64                    `json:"sourceId"`
	SourceKey          string                   `json:"sourceKey"`
	SourceLabel        string                   `json:"sourceLabel"`
	SourceRootPath     string                   `json:"sourceRootPath"`
	SourceSessionsPath string                   `json:"sourceSessionsPath"`
	AgentKind          string                   `json:"agentKind"`
	AgentName          string                   `json:"agentName"`
	Cells              []ModelSignalsMatrixCell `json:"cells"`
}

type ModelSignalsMatrixCell struct {
	ModelProvider string                `json:"modelProvider"`
	Model         string                `json:"model"`
	CohortCount   int                   `json:"cohortCount"`
	SessionCount  int                   `json:"sessionCount"`
	ModelCalls    int                   `json:"modelCalls"`
	TotalTokens   int64                 `json:"totalTokens"`
	Severity      string                `json:"severity"`
	Confidence    string                `json:"confidence"`
	KeyReason     string                `json:"keyReason"`
	Drift         ModelSignalsDrift     `json:"drift"`
	Current       ModelSignalsMetricSet `json:"current"`
	Baseline      ModelSignalsMetricSet `json:"baseline"`
}

type ModelSignalsProjectHotspot struct {
	ProjectPath string `json:"projectPath"`
	ModelCount  int    `json:"modelCount"`
	SourceCount int    `json:"sourceCount"`
	ModelSignalsMetricSet
	Current  ModelSignalsMetricSet `json:"current"`
	Baseline ModelSignalsMetricSet `json:"baseline"`
	Drift    ModelSignalsDrift     `json:"drift"`
}

type ModelSignalsDailyMetric struct {
	Date string `json:"date"`
	ModelSignalsMetricSet
	Baseline  ModelSignalsMetricSet `json:"baseline"`
	LowSample bool                  `json:"lowSample"`
	Drift     ModelSignalsDrift     `json:"drift"`
	KeyReason string                `json:"keyReason"`
}

type ModelSignalsProjectMetric struct {
	ProjectPath           string  `json:"projectPath"`
	ModelCount            int     `json:"modelCount"`
	SourceCount           int     `json:"sourceCount"`
	DominantModelProvider string  `json:"dominantModelProvider"`
	DominantModel         string  `json:"dominantModel"`
	DominantModelShare    float64 `json:"dominantModelShare"`
	ModelSignalsMetricSet
	Current  ModelSignalsMetricSet `json:"current"`
	Baseline ModelSignalsMetricSet `json:"baseline"`
	Drift    ModelSignalsDrift     `json:"drift"`
}

type ModelSignalsTrendPoint struct {
	Date                                  string  `json:"date"`
	SessionCount                          int     `json:"sessionCount"`
	ModelCalls                            int     `json:"modelCalls"`
	ToolCalls                             int     `json:"toolCalls"`
	FailedToolCalls                       int     `json:"failedToolCalls"`
	TotalTokens                           int64   `json:"totalTokens"`
	InputTokens                           int64   `json:"inputTokens"`
	CachedInputTokens                     int64   `json:"cachedInputTokens"`
	OutputTokens                          int64   `json:"outputTokens"`
	ReasoningOutputTokens                 int64   `json:"reasoningOutputTokens"`
	VisibleOutputTokens                   int64   `json:"visibleOutputTokens"`
	BillableOutputTokens                  int64   `json:"billableOutputTokens"`
	ModelDurationMS                       int64   `json:"modelDurationMs"`
	OutputExpansionRate                   float64 `json:"outputExpansionRate"`
	ReasoningTokenShare                   float64 `json:"reasoningTokenShare"`
	ReasoningOverheadRate                 float64 `json:"reasoningOverheadRate"`
	CacheMissRate                         float64 `json:"cacheMissRate"`
	ModelThroughputTokensPerSecond        float64 `json:"modelThroughputTokensPerSecond"`
	ModelThroughputOutputTokensPerSecond  float64 `json:"modelThroughputOutputTokensPerSecond"`
	ToolFailureRate                       float64 `json:"toolFailureRate"`
	ToolDependencyRate                    float64 `json:"toolDependencyRate"`
	RollingModelThroughputTokensPerSecond float64 `json:"rollingModelThroughputTokensPerSecond"`
	RollingToolFailureRate                float64 `json:"rollingToolFailureRate"`
	LowSample                             bool    `json:"lowSample"`
}

type ModelSignalsBreakdown struct {
	Model                                string  `json:"model"`
	SessionCount                         int     `json:"sessionCount"`
	ModelCalls                           int     `json:"modelCalls"`
	ToolCalls                            int     `json:"toolCalls"`
	FailedToolCalls                      int     `json:"failedToolCalls"`
	TotalTokens                          int64   `json:"totalTokens"`
	InputTokens                          int64   `json:"inputTokens"`
	CachedInputTokens                    int64   `json:"cachedInputTokens"`
	OutputTokens                         int64   `json:"outputTokens"`
	ReasoningOutputTokens                int64   `json:"reasoningOutputTokens"`
	VisibleOutputTokens                  int64   `json:"visibleOutputTokens"`
	BillableOutputTokens                 int64   `json:"billableOutputTokens"`
	ModelDurationMS                      int64   `json:"modelDurationMs"`
	ToolFailureRate                      float64 `json:"toolFailureRate"`
	ToolDependencyRate                   float64 `json:"toolDependencyRate"`
	AvgModelCallsPerSession              float64 `json:"avgModelCallsPerSession"`
	OutputExpansionRate                  float64 `json:"outputExpansionRate"`
	ReasoningTokenShare                  float64 `json:"reasoningTokenShare"`
	ReasoningOverheadRate                float64 `json:"reasoningOverheadRate"`
	CacheMissRate                        float64 `json:"cacheMissRate"`
	ModelThroughputTokensPerSecond       float64 `json:"modelThroughputTokensPerSecond"`
	ModelThroughputOutputTokensPerSecond float64 `json:"modelThroughputOutputTokensPerSecond"`
}

type ModelSignalsAnomalySession struct {
	SessionID                            int64     `json:"sessionId"`
	SourceID                             int64     `json:"sourceId"`
	SourceKey                            string    `json:"sourceKey"`
	SourceLabel                          string    `json:"sourceLabel"`
	SourceRootPath                       string    `json:"sourceRootPath"`
	SourceSessionsPath                   string    `json:"sourceSessionsPath"`
	AgentKind                            string    `json:"agentKind"`
	AgentName                            string    `json:"agentName"`
	SessionKey                           string    `json:"sessionKey"`
	CodexSessionID                       string    `json:"codexSessionId,omitempty"`
	ProjectPath                          string    `json:"projectPath"`
	Model                                string    `json:"model"`
	StartedAt                            time.Time `json:"startedAt"`
	RawSourcePath                        string    `json:"rawSourcePath"`
	ModelCalls                           int       `json:"modelCalls"`
	ToolCalls                            int       `json:"toolCalls"`
	FailedToolCalls                      int       `json:"failedToolCalls"`
	TotalTokens                          int64     `json:"totalTokens"`
	InputTokens                          int64     `json:"inputTokens"`
	CachedInputTokens                    int64     `json:"cachedInputTokens"`
	OutputTokens                         int64     `json:"outputTokens"`
	ReasoningOutputTokens                int64     `json:"reasoningOutputTokens"`
	VisibleOutputTokens                  int64     `json:"visibleOutputTokens"`
	BillableOutputTokens                 int64     `json:"billableOutputTokens"`
	ModelDurationMS                      int64     `json:"modelDurationMs"`
	OutputExpansionRate                  float64   `json:"outputExpansionRate"`
	ReasoningTokenShare                  float64   `json:"reasoningTokenShare"`
	ReasoningOverheadRate                float64   `json:"reasoningOverheadRate"`
	CacheMissRate                        float64   `json:"cacheMissRate"`
	ModelThroughputTokensPerSecond       float64   `json:"modelThroughputTokensPerSecond"`
	ModelThroughputOutputTokensPerSecond float64   `json:"modelThroughputOutputTokensPerSecond"`
	ToolFailureRate                      float64   `json:"toolFailureRate"`
	ReasonLabels                         []string  `json:"reasons"`
	Score                                float64   `json:"score"`
}

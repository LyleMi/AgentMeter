package model

import "time"

type AttentionResponse struct {
	Window   AttentionWindow   `json:"window"`
	Snapshot AttentionSnapshot `json:"snapshot"`
	Counts   AttentionCounts   `json:"counts"`
	Items    []AttentionItem   `json:"items"`
}

type AttentionWindow struct {
	Current  AttentionPeriod `json:"current"`
	Baseline AttentionPeriod `json:"baseline"`
}

type AttentionPeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type AttentionSnapshot struct {
	Sessions   AttentionMetric `json:"sessions"`
	Tokens     AttentionMetric `json:"tokens"`
	CostUSD    AttentionMetric `json:"costUsd"`
	ActiveTime AttentionMetric `json:"activeTime"`
	ToolCalls  AttentionMetric `json:"toolCalls"`
}

type AttentionMetric struct {
	Current   float64  `json:"current"`
	Baseline  float64  `json:"baseline"`
	ChangePct *float64 `json:"changePct,omitempty"`
}

type AttentionCounts struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Total    int `json:"total"`
}

type AttentionItem struct {
	Key         string                   `json:"key"`
	Kind        string                   `json:"kind"`
	Severity    string                   `json:"severity"`
	Confidence  float64                  `json:"confidence"`
	Subject     string                   `json:"subject"`
	Reason      string                   `json:"reason"`
	Metric      *AttentionItemMetric     `json:"metric,omitempty"`
	OccurredAt  *time.Time               `json:"occurredAt,omitempty"`
	Destination AttentionItemDestination `json:"destination"`
}

type AttentionItemMetric struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type AttentionItemDestination struct {
	Kind           string `json:"kind"`
	SessionID      int64  `json:"sessionId,omitempty"`
	AuditFindingID int64  `json:"auditFindingId,omitempty"`
	Agent          string `json:"agent,omitempty"`
	Model          string `json:"model,omitempty"`
	Project        string `json:"project,omitempty"`
	PrivacyTarget  string `json:"privacyTarget,omitempty"`
	SettingsPanel  string `json:"settingsPanel,omitempty"`
}

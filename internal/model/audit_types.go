package model

import "time"

type AuditFinding struct {
	ID                 int64     `json:"id"`
	SessionID          int64     `json:"sessionId"`
	SourceID           int64     `json:"sourceId"`
	SourceKey          string    `json:"sourceKey,omitempty"`
	SourceLabel        string    `json:"sourceLabel,omitempty"`
	SourceRootPath     string    `json:"sourceRootPath,omitempty"`
	SourceSessionsPath string    `json:"sourceSessionsPath,omitempty"`
	ToolCallID         int64     `json:"toolCallId"`
	SourceFileID       int64     `json:"sourceFileId"`
	RawEventID         int64     `json:"rawEventId"`
	SourceLine         int       `json:"sourceLine"`
	Timestamp          time.Time `json:"timestamp"`
	Source             string    `json:"source"`
	EventType          string    `json:"eventType"`
	Category           string    `json:"category"`
	Severity           string    `json:"severity"`
	RuleID             string    `json:"ruleId"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Evidence           string    `json:"evidence"`
	Command            string    `json:"command"`
	ShellFamily        string    `json:"shellFamily"`
	Platform           string    `json:"platform"`
	Decision           string    `json:"decision"`
	CreatedAt          time.Time `json:"createdAt"`
	SessionKey         string    `json:"sessionKey,omitempty"`
	CodexSessionID     string    `json:"codexSessionId,omitempty"`
	ProjectPath        string    `json:"projectPath,omitempty"`
	AgentKind          string    `json:"agentKind,omitempty"`
	AgentName          string    `json:"agentName,omitempty"`
	RawSourcePath      string    `json:"rawSourcePath,omitempty"`
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

type ToolCallRiskSummary struct {
	ToolCallID int64    `json:"toolCallId"`
	Severity   string   `json:"severity"`
	RiskScore  int      `json:"riskScore"`
	RiskCount  int      `json:"riskCount"`
	RuleIDs    []string `json:"ruleIds"`
}

type AuditFindingFilters struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	ShellFamily string `json:"shellFamily"`
	Agent       string `json:"agent"`
	Search      string `json:"search"`
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
}

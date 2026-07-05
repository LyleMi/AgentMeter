package model

import "time"

type PromptSuggestion struct {
	Key                string          `json:"key"`
	Text               string          `json:"text"`
	Count              int             `json:"count"`
	SessionCount       int             `json:"sessionCount"`
	VariantCount       int             `json:"variantCount"`
	FirstUsedAt        time.Time       `json:"firstUsedAt"`
	LastUsedAt         time.Time       `json:"lastUsedAt"`
	MatchKind          string          `json:"matchKind"`
	Confidence         float64         `json:"confidence"`
	Examples           []PromptExample `json:"examples"`
	Variants           []PromptVariant `json:"variants"`
	SourceID           int64           `json:"sourceId,omitempty"`
	SourceKey          string          `json:"sourceKey,omitempty"`
	SourceLabel        string          `json:"sourceLabel,omitempty"`
	SourceRootPath     string          `json:"sourceRootPath,omitempty"`
	SourceSessionsPath string          `json:"sourceSessionsPath,omitempty"`
	AgentKind          string          `json:"agentKind,omitempty"`
	AgentName          string          `json:"agentName,omitempty"`
}

type PromptVariant struct {
	Key          string    `json:"key"`
	Text         string    `json:"text"`
	Count        int       `json:"count"`
	SessionCount int       `json:"sessionCount"`
	FirstUsedAt  time.Time `json:"firstUsedAt"`
	LastUsedAt   time.Time `json:"lastUsedAt"`
}

type PromptExample struct {
	Text               string    `json:"text"`
	EventID            int64     `json:"eventId"`
	SourceLine         int       `json:"sourceLine"`
	Timestamp          time.Time `json:"timestamp"`
	SessionID          int64     `json:"sessionId"`
	SessionKey         string    `json:"sessionKey"`
	CodexSessionID     string    `json:"codexSessionId,omitempty"`
	ProjectPath        string    `json:"projectPath"`
	SourceID           int64     `json:"sourceId"`
	SourceKey          string    `json:"sourceKey"`
	SourceLabel        string    `json:"sourceLabel"`
	SourceRootPath     string    `json:"sourceRootPath,omitempty"`
	SourceSessionsPath string    `json:"sourceSessionsPath,omitempty"`
	AgentKind          string    `json:"agentKind"`
	AgentName          string    `json:"agentName"`
	RawSourcePath      string    `json:"rawSourcePath,omitempty"`
}

type PromptSuggestionFilters struct {
	Agent    string `json:"agent"`
	Project  string `json:"project"`
	Search   string `json:"search"`
	Limit    int    `json:"limit"`
	MinCount int    `json:"minCount"`
}

type SavedPrompt struct {
	ID                  int64      `json:"id"`
	Title               string     `json:"title"`
	Content             string     `json:"content"`
	SourceSuggestionKey string     `json:"sourceSuggestionKey,omitempty"`
	CopyCount           int        `json:"copyCount"`
	LastCopiedAt        *time.Time `json:"lastCopiedAt,omitempty"`
	CreatedAt           time.Time  `json:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

type SavedPromptInput struct {
	Title               string `json:"title"`
	Content             string `json:"content"`
	SourceSuggestionKey string `json:"sourceSuggestionKey,omitempty"`
}

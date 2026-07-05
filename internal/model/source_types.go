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
	Label   string `json:"label,omitempty"`
}

type SourceIdentity struct {
	SourceID           int64  `json:"sourceId"`
	SourceKey          string `json:"sourceKey"`
	SourceLabel        string `json:"sourceLabel"`
	SourceRootPath     string `json:"sourceRootPath"`
	SourceSessionsPath string `json:"sourceSessionsPath"`
	AgentKind          string `json:"agentKind"`
	AgentName          string `json:"agentName"`
}

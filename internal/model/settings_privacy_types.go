package model

import "time"

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

type SourceStorage struct {
	TotalSizeBytes int64                    `json:"totalSizeBytes"`
	TotalFileCount int                      `json:"totalFileCount"`
	Directories    []SourceDirectoryStorage `json:"directories"`
	ScannedAt      time.Time                `json:"scannedAt"`
}

type SourceDirectoryStorage struct {
	Path      string `json:"path"`
	Label     string `json:"label,omitempty"`
	Enabled   bool   `json:"enabled"`
	Exists    bool   `json:"exists"`
	SizeBytes int64  `json:"sizeBytes"`
	FileCount int    `json:"fileCount"`
	Error     string `json:"error,omitempty"`
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

type PrivacyConfigStatus struct {
	Target            string                      `json:"target"`
	Name              string                      `json:"name"`
	ConfigPath        string                      `json:"configPath"`
	Exists            bool                        `json:"exists"`
	Profiles          []PrivacyConfigProfile      `json:"profiles"`
	Summary           PrivacyConfigSummary        `json:"summary"`
	Settings          []PrivacyConfigSetting      `json:"settings"`
	Warnings          []string                    `json:"warnings"`
	SourceOptions     []PrivacyConfigSourceOption `json:"sourceOptions,omitempty"`
	SelectedSourceKey string                      `json:"selectedSourceKey,omitempty"`
}

type PrivacyConfigSourceOption struct {
	SourceID   int64  `json:"sourceId,omitempty"`
	SourceKey  string `json:"sourceKey"`
	Label      string `json:"label"`
	RootPath   string `json:"rootPath"`
	ConfigPath string `json:"configPath"`
}

type PrivacyConfigProfile struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type PrivacyConfigSummary struct {
	Score     int `json:"score"`
	Total     int `json:"total"`
	Hardened  int `json:"hardened"`
	Attention int `json:"attention"`
	Implicit  int `json:"implicit"`
}

type PrivacyConfigSetting struct {
	ID            string                      `json:"id"`
	Group         string                      `json:"group"`
	Title         string                      `json:"title"`
	Description   string                      `json:"description"`
	Key           string                      `json:"key"`
	DesiredValue  any                         `json:"desiredValue"`
	StrictValue   any                         `json:"strictValue"`
	ProfileValues []PrivacyConfigProfileValue `json:"profileValues"`
	ValueType     string                      `json:"valueType"`
	Configured    bool                        `json:"configured"`
	SupportsUnset bool                        `json:"supportsUnset"`
	CurrentValue  any                         `json:"currentValue"`
	Status        string                      `json:"status"`
	Impact        string                      `json:"impact"`
	CanApply      bool                        `json:"canApply"`
}

type PrivacyConfigProfileValue struct {
	Profile string `json:"profile"`
	Op      string `json:"op"`
	Value   any    `json:"value"`
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

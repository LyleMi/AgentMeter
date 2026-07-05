package model

import "time"

type AgentResourceOverview struct {
	Agents     []AgentResourceAgent     `json:"agents"`
	Skills     []AgentSkillResource     `json:"skills"`
	MCPServers []AgentMCPServerResource `json:"mcpServers"`
	Memories   []AgentMemoryResource    `json:"memories"`
	Warnings   []string                 `json:"warnings"`
}

type AgentResourceAgent struct {
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	RootPath    string   `json:"rootPath"`
	Exists      bool     `json:"exists"`
	ConfigPath  string   `json:"configPath"`
	Warnings    []string `json:"warnings"`
	Supports    []string `json:"supports,omitempty"`
	Unsupported []string `json:"unsupported,omitempty"`
}

type AgentSkillResource struct {
	AgentKind    string    `json:"agentKind"`
	ResourceType string    `json:"resourceType,omitempty"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Path         string    `json:"path"`
	RelativePath string    `json:"relativePath"`
	System       bool      `json:"system"`
	Enabled      bool      `json:"enabled"`
	CanToggle    bool      `json:"canToggle"`
	Status       string    `json:"status"`
	Warning      string    `json:"warning,omitempty"`
	SizeBytes    int64     `json:"sizeBytes"`
	ModifiedAt   time.Time `json:"modifiedAt"`
}

type AgentMCPServerResource struct {
	AgentKind  string   `json:"agentKind"`
	Name       string   `json:"name"`
	Command    string   `json:"command"`
	Args       []string `json:"args"`
	EnvKeys    []string `json:"envKeys"`
	ConfigPath string   `json:"configPath"`
	Enabled    bool     `json:"enabled"`
	CanToggle  bool     `json:"canToggle"`
	Status     string   `json:"status"`
	Warning    string   `json:"warning,omitempty"`
}

type AgentMemoryResource struct {
	AgentKind    string    `json:"agentKind"`
	Name         string    `json:"name"`
	Title        string    `json:"title"`
	Path         string    `json:"path"`
	RelativePath string    `json:"relativePath"`
	Kind         string    `json:"kind"`
	Preview      string    `json:"preview"`
	CanEdit      bool      `json:"canEdit"`
	SizeBytes    int64     `json:"sizeBytes"`
	ModifiedAt   time.Time `json:"modifiedAt"`
}

type AgentResourceToggleRequest struct {
	AgentKind    string `json:"agentKind"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	RelativePath string `json:"relativePath"`
	Enabled      bool   `json:"enabled"`
}

type AgentResourceOperationResult struct {
	Overview AgentResourceOverview `json:"overview"`
	Warnings []string              `json:"warnings"`
}

type AgentMemoryDetail struct {
	AgentMemoryResource
	Content string `json:"content"`
}

type AgentMemoryUpdateRequest struct {
	AgentKind    string `json:"agentKind"`
	Path         string `json:"path"`
	RelativePath string `json:"relativePath"`
	Content      string `json:"content"`
}

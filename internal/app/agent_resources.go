package app

import (
	"github.com/LyleMi/AgentMeter/internal/agentresources"
	"github.com/LyleMi/AgentMeter/internal/model"
)

func (a *App) GetAgentResources() (model.AgentResourceOverview, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentResourceOverview{}, err
	}
	return agentresources.Overview(a.ctx)
}

func (a *App) SetAgentSkillEnabled(request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return agentresources.SetSkillEnabled(a.ctx, request)
}

func (a *App) SetAgentMCPServerEnabled(request model.AgentResourceToggleRequest) (model.AgentResourceOperationResult, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentResourceOperationResult{}, err
	}
	return agentresources.SetMCPServerEnabled(a.ctx, request)
}

func (a *App) GetAgentMemoryDetail(agentKind, path, relativePath string) (model.AgentMemoryDetail, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return agentresources.MemoryDetail(a.ctx, agentKind, path, relativePath)
}

func (a *App) UpdateAgentMemory(request model.AgentMemoryUpdateRequest) (model.AgentMemoryDetail, error) {
	if err := a.ensureReady(); err != nil {
		return model.AgentMemoryDetail{}, err
	}
	return agentresources.UpdateMemory(a.ctx, request)
}

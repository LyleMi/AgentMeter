package app

import (
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
)

func (a *App) GetOverview() (model.Overview, error) {
	return a.GetOverviewWithFilters(model.AnalyticsFilters{})
}

func (a *App) GetOverviewWithFilters(filters model.AnalyticsFilters) (model.Overview, error) {
	if err := a.ensureReady(); err != nil {
		return model.Overview{}, err
	}
	return a.query.OverviewWithFilters(a.ctx, filters)
}

func (a *App) GetTokenAnalytics() (model.TokenAnalytics, error) {
	return a.GetTokenAnalyticsWithFilters(model.AnalyticsFilters{})
}

func (a *App) GetTokenAnalyticsWithFilters(filters model.AnalyticsFilters) (model.TokenAnalytics, error) {
	if err := a.ensureReady(); err != nil {
		return model.TokenAnalytics{}, err
	}
	return a.query.TokenAnalyticsWithFilters(a.ctx, filters)
}

func (a *App) GetModelSignalsWithFilters(filters model.AnalyticsFilters) (model.ModelSignals, error) {
	if err := a.ensureReady(); err != nil {
		return model.ModelSignals{}, err
	}
	return a.query.ModelSignalsWithFilters(a.ctx, filters)
}

func (a *App) GetUsageBreakdown(groupBy string, filters model.AnalyticsFilters) (model.UsageBreakdown, error) {
	if err := a.ensureReady(); err != nil {
		return model.UsageBreakdown{}, err
	}
	return a.query.UsageBreakdown(a.ctx, groupBy, filters)
}

func (a *App) ListSessions(filters model.SessionFilters) ([]model.Session, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.Sessions(a.ctx, filters)
}

func (a *App) GetSessionDetail(id int64) (model.SessionDetail, error) {
	if err := a.ensureReady(); err != nil {
		return model.SessionDetail{}, err
	}
	return a.query.SessionDetail(a.ctx, id)
}

func (a *App) GetTools() ([]model.ToolStat, error) {
	return a.ListTools(model.ToolFilters{})
}

func (a *App) ListTools(filters model.ToolFilters) ([]model.ToolStat, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.Tools(a.ctx, filters)
}

func (a *App) ListToolCalls(filters model.ToolCallFilters) ([]model.ToolCall, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.ToolCalls(a.ctx, filters)
}

func (a *App) ListToolCallRisks(filters model.ToolCallRiskFilters) ([]model.ToolCallRiskSummary, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.ToolCallRisks(a.ctx, filters)
}

func (a *App) PromptSuggestions(filters model.PromptSuggestionFilters) ([]model.PromptSuggestion, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.PromptSuggestions(a.ctx, filters)
}

func (a *App) SavedPrompts() ([]model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.SavedPrompts(a.ctx)
}

func (a *App) SavePrompt(input model.SavedPromptInput) (model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return model.SavedPrompt{}, err
	}
	return a.query.SavePrompt(a.ctx, input)
}

func (a *App) UpdateSavedPrompt(id int64, input model.SavedPromptInput) (model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return model.SavedPrompt{}, err
	}
	return a.query.UpdateSavedPrompt(a.ctx, id, input)
}

func (a *App) DeleteSavedPrompt(id int64) error {
	if err := a.ensureReady(); err != nil {
		return err
	}
	return a.query.DeleteSavedPrompt(a.ctx, id)
}

func (a *App) RecordPromptCopy(id int64) (model.SavedPrompt, error) {
	if err := a.ensureReady(); err != nil {
		return model.SavedPrompt{}, err
	}
	return a.query.RecordPromptCopy(a.ctx, id)
}

func (a *App) IgnorePromptSuggestion(key string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}
	return a.query.IgnorePromptSuggestion(a.ctx, key)
}

func (a *App) UnignorePromptSuggestion(key string) error {
	if err := a.ensureReady(); err != nil {
		return err
	}
	return a.query.UnignorePromptSuggestion(a.ctx, key)
}

func (a *App) GetAuditSummary() (model.AuditSummary, error) {
	return a.GetAuditSummaryWithFilters(model.AuditFindingFilters{})
}

func (a *App) GetAuditSummaryWithFilters(filters model.AuditFindingFilters) (model.AuditSummary, error) {
	if err := a.ensureReady(); err != nil {
		return model.AuditSummary{}, err
	}
	return a.query.AuditSummaryWithFilters(a.ctx, filters)
}

func (a *App) ListAuditFindings(filters model.AuditFindingFilters) ([]model.AuditFinding, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.AuditFindings(a.ctx, filters)
}

func (a *App) GetAuditFinding(id int64) (model.AuditFinding, error) {
	if err := a.ensureReady(); err != nil {
		return model.AuditFinding{}, err
	}
	return a.query.AuditFinding(a.ctx, id)
}

func (a *App) GetPricingModels() ([]model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return pricing.List(a.ctx, a.conn)
}

func (a *App) SavePricingModel(input model.PricingModelInput) (model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return model.PricingModel{}, err
	}
	return pricing.UpsertCustom(a.ctx, a.conn, input)
}

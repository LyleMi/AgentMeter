package app

import (
	"time"

	"github.com/LyleMi/AgentMeter/internal/attention"
	"github.com/LyleMi/AgentMeter/internal/model"
)

func (a *App) GetAttention(filters model.AnalyticsFilters) (model.AttentionResponse, error) {
	if err := a.ensureReady(); err != nil {
		return model.AttentionResponse{}, err
	}
	window, currentFilters, baselineFilters := attention.ResolveWindow(time.Now(), filters)
	current, err := a.query.OverviewWithFilters(a.ctx, currentFilters)
	if err != nil {
		return model.AttentionResponse{}, err
	}
	baseline, err := a.query.OverviewWithFilters(a.ctx, baselineFilters)
	if err != nil {
		return model.AttentionResponse{}, err
	}
	signals, err := a.query.ModelSignalsWithFilters(a.ctx, currentFilters)
	if err != nil {
		return model.AttentionResponse{}, err
	}
	findings, err := a.query.AuditFindings(a.ctx, model.AuditFindingFilters{
		Agent:       currentFilters.Agent,
		Model:       currentFilters.Model,
		Project:     currentFilters.Project,
		StartedFrom: currentFilters.StartedFrom,
		StartedTo:   currentFilters.StartedTo,
		Limit:       100,
	})
	if err != nil {
		return model.AttentionResponse{}, err
	}
	settings, err := a.GetSettings()
	if err != nil {
		return model.AttentionResponse{}, err
	}
	privacyStatuses, privacyErr := a.GetPrivacyConfigs()
	privacyWarning := ""
	if privacyErr != nil {
		privacyWarning = privacyErr.Error()
	}
	return attention.Build(attention.Input{
		Window: window, Current: current, Baseline: baseline, Signals: signals,
		AuditFindings: findings, Privacy: privacyStatuses, PrivacyWarning: privacyWarning, Settings: settings,
	}), nil
}

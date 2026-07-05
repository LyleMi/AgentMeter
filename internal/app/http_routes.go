package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func registerSettingsHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetSettings()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SourceEntries []model.SourceEntry `json:"sourceEntries"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.SaveSourceSettings(body.SourceEntries)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/index", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Rebuild bool `json:"rebuild"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		value, err := service.IndexNow(body.Rebuild)
		writeJSON(w, value, err)
	})
}

func registerAgentResourceHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/agent-resources", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetAgentResources()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/agent-resources/skills/enabled", func(w http.ResponseWriter, r *http.Request) {
		var body model.AgentResourceToggleRequest
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.SetAgentSkillEnabled(body)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/agent-resources/mcp/enabled", func(w http.ResponseWriter, r *http.Request) {
		var body model.AgentResourceToggleRequest
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.SetAgentMCPServerEnabled(body)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/agent-resources/memories/detail", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.GetAgentMemoryDetail(query.Get("agentKind"), query.Get("path"), query.Get("relativePath"))
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/agent-resources/memories/detail", func(w http.ResponseWriter, r *http.Request) {
		var body model.AgentMemoryUpdateRequest
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.UpdateAgentMemory(body)
		writeJSON(w, value, err)
	})
}

func registerPrivacyHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/privacy/{target}", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}
		value, err := service.GetPrivacyConfigForSource(target, r.URL.Query().Get("sourceKey"))
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/apply", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}

		var body struct {
			SettingIDs []string `json:"settingIds"`
			SourceKey  string   `json:"sourceKey"`
		}
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.ApplyPrivacyConfigForSource(target, firstNonEmpty(body.SourceKey, r.URL.Query().Get("sourceKey")), body.SettingIDs)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/profile", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}

		var body struct {
			Profile   string `json:"profile"`
			SourceKey string `json:"sourceKey"`
		}
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.Profile) == "" {
			writeJSON(w, nil, errors.New("privacy profile is required"))
			return
		}
		value, err := service.ApplyPrivacyProfileForSource(target, firstNonEmpty(body.SourceKey, r.URL.Query().Get("sourceKey")), body.Profile)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/changes", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}

		var body struct {
			Changes   []model.PrivacyConfigEdit `json:"changes"`
			SourceKey string                    `json:"sourceKey"`
		}
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.ApplyPrivacyConfigChangesForSource(target, firstNonEmpty(body.SourceKey, r.URL.Query().Get("sourceKey")), body.Changes)
		writeJSON(w, value, err)
	})
}

func registerAnalyticsHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/overview", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetOverviewWithFilters(analyticsFilters(r))
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/tokens", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetTokenAnalyticsWithFilters(analyticsFilters(r))
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/model-signals", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetModelSignalsWithFilters(analyticsFilters(r))
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/usage/breakdown", func(w http.ResponseWriter, r *http.Request) {
		groupBy := strings.TrimSpace(r.URL.Query().Get("groupBy"))
		if groupBy == "" {
			groupBy = "model"
		}
		value, err := service.GetUsageBreakdown(groupBy, analyticsFilters(r))
		writeJSON(w, value, err)
	})
}

func registerSessionHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.ListSessions(model.SessionFilters{
			Search: query.Get("search"),
			Model:  query.Get("model"),
			Agent:  query.Get("agent"),
			Limit:  queryInt(r, "limit"),
			Offset: queryInt(r, "offset"),
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/sessions/"), 10, 64)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.GetSessionDetail(id)
		writeJSON(w, value, err)
	})
}

func registerToolHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/tools", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.ListTools(model.ToolFilters{
			Agent: query.Get("agent"),
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/tool-calls", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.ListToolCalls(model.ToolCallFilters{
			ToolName:    query.Get("tool"),
			Agent:       query.Get("agent"),
			StartedFrom: query.Get("from"),
			StartedTo:   query.Get("to"),
			Sort:        query.Get("sort"),
			Shell:       queryBool(r, "shell"),
			RiskOnly:    queryBool(r, "riskOnly"),
			IncludeRisk: queryBool(r, "includeRisk"),
			Limit:       queryInt(r, "limit"),
			Offset:      queryInt(r, "offset"),
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/tool-call-risks", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.ListToolCallRisks(model.ToolCallRiskFilters{
			Agent:       query.Get("agent"),
			StartedFrom: query.Get("from"),
			StartedTo:   query.Get("to"),
			Limit:       queryInt(r, "limit"),
		})
		writeJSON(w, value, err)
	})
}

func registerPromptHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/prompts/suggestions", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.PromptSuggestions(model.PromptSuggestionFilters{
			Agent:    query.Get("agent"),
			Project:  query.Get("project"),
			Search:   query.Get("search"),
			Limit:    queryInt(r, "limit"),
			MinCount: queryInt(r, "minCount"),
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/prompts/saved", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.SavedPrompts()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/prompts/saved", func(w http.ResponseWriter, r *http.Request) {
		var body model.SavedPromptInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.SavePrompt(body)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("PUT /api/prompts/saved/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		var body model.SavedPromptInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.UpdateSavedPrompt(id, body)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("DELETE /api/prompts/saved/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		err = service.DeleteSavedPrompt(id)
		writeJSON(w, map[string]bool{"ok": err == nil}, err)
	})
	mux.HandleFunc("POST /api/prompts/saved/{id}/copy", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.RecordPromptCopy(id)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/prompts/ignored", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SuggestionKey string `json:"suggestionKey"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, nil, err)
			return
		}
		err := service.IgnorePromptSuggestion(body.SuggestionKey)
		writeJSON(w, map[string]bool{"ok": err == nil}, err)
	})
	mux.HandleFunc("DELETE /api/prompts/ignored/{key}", func(w http.ResponseWriter, r *http.Request) {
		err := service.UnignorePromptSuggestion(r.PathValue("key"))
		writeJSON(w, map[string]bool{"ok": err == nil}, err)
	})
}

func registerAuditHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/audit/summary", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.GetAuditSummaryWithFilters(model.AuditFindingFilters{
			Agent: query.Get("agent"),
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/audit/findings", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		value, err := service.ListAuditFindings(model.AuditFindingFilters{
			Category:    query.Get("category"),
			Severity:    query.Get("severity"),
			ShellFamily: query.Get("shell"),
			Agent:       query.Get("agent"),
			Search:      query.Get("search"),
			Limit:       queryInt(r, "limit"),
			Offset:      queryInt(r, "offset"),
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/audit/findings/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.GetAuditFinding(id)
		writeJSON(w, value, err)
	})
}

func registerPricingHandlers(mux *http.ServeMux, service *App) {
	mux.HandleFunc("GET /api/pricing", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetPricingModels()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/pricing", func(w http.ResponseWriter, r *http.Request) {
		var body model.PricingModelInput
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.SavePricingModel(body)
		writeJSON(w, value, err)
	})
}

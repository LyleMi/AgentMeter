package app

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
)

func writeJSON(w http.ResponseWriter, value any, err error) {
	if err != nil {
		writeJSONError(w, statusForError(err), err.Error())
		return
	}
	writeJSONResponse(w, http.StatusOK, value)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, map[string]string{"error": message})
}

func writeJSONResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	if status != http.StatusOK {
		w.WriteHeader(status)
	}
	_ = json.NewEncoder(w).Encode(value)
}

func statusForError(err error) int {
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound
	}
	if errors.Is(err, pricing.ErrInvalidRate) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func decodeOptionalJSONBody(r *http.Request, value any) error {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func analyticsFilters(r *http.Request) model.AnalyticsFilters {
	query := r.URL.Query()
	return model.AnalyticsFilters{
		Agent:       query.Get("agent"),
		Model:       query.Get("model"),
		Project:     query.Get("project"),
		StartedFrom: query.Get("from"),
		StartedTo:   query.Get("to"),
	}
}

func queryInt(r *http.Request, key string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(key))
	return value
}

func requirePrivacyTarget(w http.ResponseWriter, r *http.Request, service *App) (string, bool) {
	target := r.PathValue("target")
	if !service.SupportsPrivacyTarget(target) {
		writeJSONError(w, http.StatusNotFound, "unsupported privacy target: "+target)
		return "", false
	}
	return target, true
}

func decodeOptionalJSONBodyOrWrite(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := decodeOptionalJSONBody(r, value); err != nil {
		writeJSON(w, nil, err)
		return false
	}
	return true
}

func RegisterHTTPHandlers(mux *http.ServeMux, service *App, staticFS fs.FS) {

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetSettings()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/privacy/{target}", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}
		value, err := service.GetPrivacyConfig(target)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/apply", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}

		var body struct {
			SettingIDs []string `json:"settingIds"`
		}
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.ApplyPrivacyConfig(target, body.SettingIDs)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/profile", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}

		var body struct {
			Profile string `json:"profile"`
		}
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		if strings.TrimSpace(body.Profile) == "" {
			writeJSON(w, nil, errors.New("privacy profile is required"))
			return
		}
		value, err := service.ApplyPrivacyProfile(target, body.Profile)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/changes", func(w http.ResponseWriter, r *http.Request) {
		target, ok := requirePrivacyTarget(w, r, service)
		if !ok {
			return
		}

		var body struct {
			Changes []model.PrivacyConfigEdit `json:"changes"`
		}
		if !decodeOptionalJSONBodyOrWrite(w, r, &body) {
			return
		}
		value, err := service.ApplyPrivacyConfigChanges(target, body.Changes)
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
			Limit:       queryInt(r, "limit"),
			Offset:      queryInt(r, "offset"),
		})
		writeJSON(w, value, err)
	})
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

	if _, err := fs.Stat(staticFS, "index.html"); err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend build not found; run `cd frontend && npm run build`, or use `npm run dev` during development", http.StatusServiceUnavailable)
		})
		return
	}

	fileServer := http.FileServer(http.FS(staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") {
			writeJSONError(w, http.StatusNotFound, "api route not found: "+r.URL.Path)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(staticFS, path); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

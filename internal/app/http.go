package app

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"AgentMeter/internal/model"
)

func RegisterHTTPHandlers(mux *http.ServeMux, service *App, staticFS fs.FS) {
	writeJSON := func(w http.ResponseWriter, value any, err error) {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(value)
	}

	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetSettings()
		writeJSON(w, value, err)
	})
	writePrivacyTargetError := func(w http.ResponseWriter, target string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported privacy target: " + target})
	}
	mux.HandleFunc("GET /api/privacy/{target}", func(w http.ResponseWriter, r *http.Request) {
		switch target := r.PathValue("target"); target {
		case "codex":
			value, err := service.GetCodexPrivacyConfig()
			writeJSON(w, value, err)
		case "gemini":
			value, err := service.GetGeminiPrivacyConfig()
			writeJSON(w, value, err)
		default:
			writePrivacyTargetError(w, target)
		}
	})
	mux.HandleFunc("POST /api/privacy/{target}/apply", func(w http.ResponseWriter, r *http.Request) {
		var apply func([]string) (model.PrivacyConfigApplyResult, error)
		switch target := r.PathValue("target"); target {
		case "codex":
			apply = service.ApplyCodexPrivacyConfig
		case "gemini":
			apply = service.ApplyGeminiPrivacyConfig
		default:
			writePrivacyTargetError(w, target)
			return
		}

		var body struct {
			SettingIDs []string `json:"settingIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, nil, err)
			return
		}
		value, err := apply(body.SettingIDs)
		writeJSON(w, value, err)
	})
	mux.HandleFunc("POST /api/privacy/{target}/changes", func(w http.ResponseWriter, r *http.Request) {
		var applyChanges func([]model.PrivacyConfigEdit) (model.PrivacyConfigApplyResult, error)
		switch target := r.PathValue("target"); target {
		case "codex":
			applyChanges = service.ApplyCodexPrivacyConfigChanges
		case "gemini":
			applyChanges = service.ApplyGeminiPrivacyConfigChanges
		default:
			writePrivacyTargetError(w, target)
			return
		}

		var body struct {
			Changes []model.PrivacyConfigEdit `json:"changes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			writeJSON(w, nil, err)
			return
		}
		value, err := applyChanges(body.Changes)
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
		value, err := service.GetOverview()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/sessions", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		limit, _ := strconv.Atoi(query.Get("limit"))
		offset, _ := strconv.Atoi(query.Get("offset"))
		value, err := service.ListSessions(model.SessionFilters{
			Search: query.Get("search"),
			Model:  query.Get("model"),
			Agent:  query.Get("agent"),
			Limit:  limit,
			Offset: offset,
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
		limit, _ := strconv.Atoi(query.Get("limit"))
		offset, _ := strconv.Atoi(query.Get("offset"))
		value, err := service.ListToolCalls(model.ToolCallFilters{
			ToolName:    query.Get("tool"),
			Agent:       query.Get("agent"),
			StartedFrom: query.Get("from"),
			StartedTo:   query.Get("to"),
			Sort:        query.Get("sort"),
			Limit:       limit,
			Offset:      offset,
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/audit/summary", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetAuditSummary()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/audit/findings", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		limit, _ := strconv.Atoi(query.Get("limit"))
		offset, _ := strconv.Atoi(query.Get("offset"))
		value, err := service.ListAuditFindings(model.AuditFindingFilters{
			Category:    query.Get("category"),
			Severity:    query.Get("severity"),
			ShellFamily: query.Get("shell"),
			Search:      query.Get("search"),
			Limit:       limit,
			Offset:      offset,
		})
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/pricing", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetPricingModels()
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "api route not found: " + r.URL.Path})
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

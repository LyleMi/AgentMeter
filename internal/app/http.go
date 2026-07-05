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

	"github.com/LyleMi/AgentMeter/internal/agentresources"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
	"github.com/LyleMi/AgentMeter/internal/query"
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
	var sourceKeyErr privacySourceKeyError
	if errors.As(err, &sourceKeyErr) {
		return http.StatusBadRequest
	}
	var sourceNotFoundErr privacySourceNotFoundError
	if errors.As(err, &sourceNotFoundErr) {
		return http.StatusNotFound
	}
	var sourceUnsupportedErr privacySourceUnsupportedError
	if errors.As(err, &sourceUnsupportedErr) {
		return http.StatusBadRequest
	}
	var resourceErr agentresources.ResourceError
	if errors.As(err, &resourceErr) {
		if resourceErr.Status > 0 {
			return resourceErr.Status
		}
		return http.StatusBadRequest
	}
	if errors.Is(err, pricing.ErrInvalidRate) {
		return http.StatusBadRequest
	}
	if errors.Is(err, query.ErrInvalidPrompt) {
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

func queryBool(r *http.Request, key string) bool {
	value := strings.ToLower(strings.TrimSpace(r.URL.Query().Get(key)))
	return value == "1" || value == "true" || value == "yes" || value == "on"
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func RegisterHTTPHandlers(mux *http.ServeMux, service *App, staticFS fs.FS) {
	registerSettingsHandlers(mux, service)
	registerAgentResourceHandlers(mux, service)
	registerPrivacyHandlers(mux, service)
	registerAnalyticsHandlers(mux, service)
	registerSessionHandlers(mux, service)
	registerToolHandlers(mux, service)
	registerPromptHandlers(mux, service)
	registerAuditHandlers(mux, service)
	registerPricingHandlers(mux, service)
	registerStaticHandlers(mux, staticFS)
}

func registerStaticHandlers(mux *http.ServeMux, staticFS fs.FS) {
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

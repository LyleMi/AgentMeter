package app

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"

	"AgentMeter/internal/model"
)

func RegisterHTTPHandlers(mux *http.ServeMux, service *App, assets embed.FS) {
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
	mux.HandleFunc("POST /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			SourcePath string `json:"sourcePath"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, nil, err)
			return
		}
		value, err := service.SaveSettings(body.SourcePath)
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
		value, err := service.GetTools()
		writeJSON(w, value, err)
	})
	mux.HandleFunc("GET /api/pricing", func(w http.ResponseWriter, r *http.Request) {
		value, err := service.GetPricingModels()
		writeJSON(w, value, err)
	})

	staticFS, err := fs.Sub(assets, "frontend/dist")
	if err == nil {
		fileServer := http.FileServer(http.FS(staticFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
}

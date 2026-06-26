package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"AgentMeter/internal/db"
	"AgentMeter/internal/ingest"
	"AgentMeter/internal/model"
	"AgentMeter/internal/platform"
	"AgentMeter/internal/pricing"
	"AgentMeter/internal/query"
)

type App struct {
	ctx        context.Context
	dbPath     string
	conn       *sql.DB
	indexer    *ingest.Indexer
	query      *query.Service
	startupMu  sync.Mutex
	indexMu    sync.Mutex
	lastResult *model.IndexResult
	lastStart  *time.Time
}

func New() (*App, error) {
	dbPath, err := platform.DefaultDatabasePath()
	if err != nil {
		return nil, err
	}
	return &App{dbPath: dbPath}, nil
}

func (a *App) Startup(ctx context.Context) error {
	a.startupMu.Lock()
	defer a.startupMu.Unlock()
	if a.conn != nil {
		a.ctx = ctx
		return nil
	}
	conn, err := db.Open(a.dbPath)
	if err != nil {
		return err
	}
	a.ctx = ctx
	a.conn = conn
	a.indexer = ingest.New(conn, a.dbPath)
	a.query = query.New(conn)
	if _, ok, err := db.GetConfig(ctx, conn, "source_path"); err != nil {
		return err
	} else if !ok {
		return db.SetConfig(ctx, conn, "source_path", platform.DefaultCodexSourcePath())
	}
	return nil
}

func (a *App) Shutdown(_ context.Context) {
	_ = db.Close(a.conn)
}

func (a *App) GetSettings() (model.Settings, error) {
	if err := a.ensureReady(); err != nil {
		return model.Settings{}, err
	}
	sourcePath, ok, err := db.GetConfig(a.ctx, a.conn, "source_path")
	if err != nil {
		return model.Settings{}, err
	}
	if !ok || sourcePath == "" {
		sourcePath = platform.DefaultCodexSourcePath()
	}
	models, err := pricing.List(a.ctx, a.conn)
	if err != nil {
		return model.Settings{}, err
	}
	return model.Settings{
		SourcePath:         sourcePath,
		DefaultSourcePath:  platform.DefaultCodexSourcePath(),
		DatabasePath:       a.dbPath,
		PricingModels:      models,
		LastIndexStartedAt: a.lastStart,
		LastIndexResult:    a.lastResult,
	}, nil
}

func (a *App) SaveSettings(sourcePath string) (model.Settings, error) {
	if err := a.ensureReady(); err != nil {
		return model.Settings{}, err
	}
	cleaned := filepath.Clean(sourcePath)
	if err := db.SetConfig(a.ctx, a.conn, "source_path", cleaned); err != nil {
		return model.Settings{}, err
	}
	return a.GetSettings()
}

func (a *App) IndexNow(rebuild bool) (model.IndexResult, error) {
	if err := a.ensureReady(); err != nil {
		return model.IndexResult{}, err
	}
	a.indexMu.Lock()
	defer a.indexMu.Unlock()

	settings, err := a.GetSettings()
	if err != nil {
		return model.IndexResult{}, err
	}
	start := time.Now().UTC()
	a.lastStart = &start
	result, err := a.indexer.Index(a.ctx, settings.SourcePath, rebuild)
	if err != nil {
		return result, err
	}
	a.lastResult = &result
	encoded, _ := json.Marshal(result)
	_ = db.SetConfig(a.ctx, a.conn, "last_index_result", string(encoded))
	return result, nil
}

func (a *App) GetOverview() (model.Overview, error) {
	if err := a.ensureReady(); err != nil {
		return model.Overview{}, err
	}
	return a.query.Overview(a.ctx)
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
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return a.query.Tools(a.ctx)
}

func (a *App) GetPricingModels() ([]model.PricingModel, error) {
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	return pricing.List(a.ctx, a.conn)
}

func (a *App) ensureReady() error {
	if a.conn == nil {
		if a.ctx == nil {
			a.ctx = context.Background()
		}
		return a.Startup(a.ctx)
	}
	if a.query == nil || a.indexer == nil {
		return errors.New("app services are not initialized")
	}
	return nil
}

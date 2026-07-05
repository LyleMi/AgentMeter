package app

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/ingest"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/platform"
	"github.com/LyleMi/AgentMeter/internal/query"
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
	if err := a.ensureSourcePathsConfig(ctx, conn); err != nil {
		return err
	}
	return nil
}

func (a *App) Shutdown(_ context.Context) {
	_ = db.Close(a.conn)
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

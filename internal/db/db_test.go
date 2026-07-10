package db

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsureMigrationColumnsUpgradesLegacyTables(t *testing.T) {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	ctx := context.Background()
	legacyTables := []string{
		`CREATE TABLE sessions (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE tool_calls (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE token_usage (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE model_calls (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE source_files (id INTEGER PRIMARY KEY)`,
		`CREATE TABLE pricing_models (id INTEGER PRIMARY KEY)`,
	}
	for _, statement := range legacyTables {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	if err := ensureMigrationColumns(ctx, conn); err != nil {
		t.Fatal(err)
	}
	for _, column := range migrationColumns {
		if !hasTableColumn(t, ctx, conn, column.table, column.name) {
			t.Errorf("column %s.%s was not added", column.table, column.name)
		}
	}
}

func hasTableColumn(t *testing.T, ctx context.Context, conn *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := conn.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return false
}

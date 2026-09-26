package ingest

import (
	"context"
	"database/sql"
	"path/filepath"
	"strconv"
	"strings"
)

// DSH preserves old generations during migrations. Index only the highest
// canonical generation in each session directory, preferring compressed data
// if the same generation has both encodings.
func currentDSHFiles(files []string) []string {
	selected := map[string]string{}
	versions := map[string]int{}
	var result []string
	for _, path := range files {
		version, ok := dshGeneration(filepath.Base(path))
		if !ok {
			continue
		}
		dir := filepath.Dir(path)
		previous, exists := selected[dir]
		if !exists || version > versions[dir] || (version == versions[dir] && strings.HasSuffix(path, ".zstd") && !strings.HasSuffix(previous, ".zstd")) {
			selected[dir], versions[dir] = path, version
		}
	}
	for _, path := range files {
		if selected[filepath.Dir(path)] == path {
			result = append(result, path)
		}
	}
	return result
}

func dshGeneration(name string) (int, bool) {
	name = strings.TrimSuffix(name, ".zstd")
	if name == "session.jsonl" {
		return 0, true
	}
	if !strings.HasPrefix(name, "session.v") || !strings.HasSuffix(name, ".jsonl") {
		return 0, false
	}
	digits := strings.TrimSuffix(strings.TrimPrefix(name, "session.v"), ".jsonl")
	version, err := strconv.Atoi(digits)
	return version, err == nil && version > 0 && strconv.Itoa(version) == digits
}

// Remove prior indexed generations only after the replacement has parsed, in
// the same transaction as its insertion. A broken newer file retains old data.
func clearOlderDSHGenerations(ctx context.Context, tx *sql.Tx, sourceID, fileID int64, key string) error {
	var current string
	if err := tx.QueryRowContext(ctx, `SELECT path FROM source_files WHERE id = ?`, fileID).Scan(&current); err != nil {
		return err
	}
	version, ok := dshGeneration(filepath.Base(current))
	if !ok {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT sf.id, sf.path FROM source_files sf JOIN sessions s ON s.source_file_id = sf.id WHERE sf.source_id = ? AND s.session_key = ? AND sf.id <> ?`, sourceID, key, fileID)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			rows.Close()
			return err
		}
		older, valid := dshGeneration(filepath.Base(path))
		if valid && older <= version && filepath.Dir(path) == filepath.Dir(current) {
			ids = append(ids, id)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := clearParsedSessionRows(ctx, tx, id); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM source_files WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

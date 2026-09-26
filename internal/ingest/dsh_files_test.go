package ingest

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/klauspost/compress/zstd"
)

func TestDSHIndexMigrationAndPricing(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".dsh")
	dir := filepath.Join(root, "sessions", "project", "example")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	log := `{"type":"session","version":0,"id":"example","createdAt":1780000000000,"cwd":"/work"}
{"type":"assistant/message","seq":0,"time":1780000001000,"data":{"message":{"source":{"model":"deepseek-flash","provider":"deepseek"}},"usage":{"inputTokens":1000000,"cacheReadTokens":1000000,"outputTokens":1000000}}}
`
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(log), 0600); err != nil {
		t.Fatal(err)
	}
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	index := New(conn, "test.sqlite")
	ctx := context.Background()
	for pass := 0; pass < 3; pass++ {
		if pass == 1 {
			enc, err := zstd.NewWriter(nil)
			if err != nil {
				t.Fatal(err)
			}
			currentLog := strings.Replace(log, `"version":0`, `"version":4,"isSeeded":false`, 1)
			data := enc.EncodeAll([]byte(currentLog), nil)
			enc.Close()
			if err := os.WriteFile(filepath.Join(dir, "session.v4.jsonl.zstd"), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		result, err := index.Index(ctx, root, false)
		if err != nil || result.Failed != 0 || result.FilesSeen != 1 {
			t.Fatalf("index: %+v %v", result, err)
		}
		if pass == 2 && result.Skipped != 1 {
			t.Fatalf("unchanged compressed log not skipped: %+v", result)
		}
		var sessions, usages int
		if err := conn.QueryRow(`SELECT count(*) FROM sessions`).Scan(&sessions); err != nil {
			t.Fatal(err)
		}
		if err := conn.QueryRow(`SELECT count(*) FROM token_usage`).Scan(&usages); err != nil {
			t.Fatal(err)
		}
		if sessions != 1 || usages != 1 {
			t.Fatalf("migration duplicated usage: sessions=%d usages=%d", sessions, usages)
		}
		var kind string
		if err := conn.QueryRow(`SELECT kind FROM sources`).Scan(&kind); err != nil || kind != "dsh" {
			t.Fatalf("source: %s %v", kind, err)
		}
		var cost float64
		if err := conn.QueryRow(`SELECT cost_usd FROM model_calls`).Scan(&cost); err != nil {
			t.Fatal(err)
		}
		if math.Abs(cost-0.753) > 1e-9 {
			t.Fatalf("cost = %v, want 0.753", cost)
		}
	}
	// A future, unsupported generation must not delete the last good index.
	future := strings.Replace(log, `"version":0`, `"version":5`, 1)
	if err := os.WriteFile(filepath.Join(dir, "session.v5.jsonl"), []byte(future), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := index.Index(ctx, root, false)
	if err != nil || result.Failed != 1 {
		t.Fatalf("unsupported generation: %+v %v", result, err)
	}
	var count int
	if err := conn.QueryRow(`SELECT count(*) FROM sessions`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("last good session lost: %d %v", count, err)
	}
}

func TestDSHGenerationSelection(t *testing.T) {
	files := []string{"a/session.jsonl", "a/session.v3.jsonl", "a/session.v4.jsonl", "a/session.v4.jsonl.zstd", "b/session.jsonl", "other.jsonl"}
	got := currentDSHFiles(files)
	if len(got) != 2 || got[0] != "a/session.v4.jsonl.zstd" || got[1] != "b/session.jsonl" {
		t.Fatalf("files: %v", got)
	}
	for _, name := range []string{"session.v0.jsonl", "session.v04.jsonl", "session.v-1.jsonl", "session.v4.jsonl.tmp"} {
		if _, ok := dshGeneration(name); ok {
			t.Fatalf("accepted noncanonical name %s", name)
		}
	}
}

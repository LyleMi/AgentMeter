package pricing

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"AgentMeter/internal/model"

	_ "modernc.org/sqlite"
)

func TestComputeKnownRegistryRows(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	tests := []struct {
		name  string
		usage model.Usage
		want  float64
	}{
		{
			name: "gpt5 alias",
			usage: model.Usage{
				Model:        "gpt5.5",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 35,
		},
		{
			name: "glm",
			usage: model.Usage{
				Model:             "glm-5.2",
				InputTokens:       1_000_000,
				CachedInputTokens: 200_000,
				OutputTokens:      500_000,
			},
			want: 3.372,
		},
		{
			name: "kimi",
			usage: model.Usage{
				Model:             "kimi-k2.6",
				InputTokens:       1_000_000,
				CachedInputTokens: 200_000,
				OutputTokens:      500_000,
			},
			want: 2.792,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, unpriced := Compute(conn, tt.usage)
			if unpriced {
				t.Fatalf("usage marked unpriced")
			}
			if cost == nil {
				t.Fatal("cost is nil")
			}
			if math.Abs(*cost-tt.want) > 0.000001 {
				t.Fatalf("cost = %f, want %f", *cost, tt.want)
			}
		})
	}
}

func TestComputeDoesNotMarkEmptyUsageUnpriced(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	cost, unpriced := Compute(conn, model.Usage{Model: "unknown", Source: "unknown"})
	if cost != nil || unpriced {
		t.Fatalf("empty usage cost=%v unpriced=%v", cost, unpriced)
	}

	_, unpriced = Compute(conn, model.Usage{Model: "unknown", TotalTokens: 1})
	if !unpriced {
		t.Fatal("unknown model with tokens should be unpriced")
	}
}

func TestCalculatorReusesLoadedRates(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	calculator, err := LoadCalculator(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	cost, unpriced := calculator.Compute(model.Usage{
		Model:             "openai/gpt5.5",
		InputTokens:       1_000_000,
		CachedInputTokens: 200_000,
		OutputTokens:      500_000,
	})
	if unpriced {
		t.Fatal("usage marked unpriced")
	}
	if cost == nil {
		t.Fatal("cost is nil")
	}
	if math.Abs(*cost-19.1) > 0.000001 {
		t.Fatalf("cost = %f, want %f", *cost, 19.1)
	}
}

func openSeededPricingDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	_, err = conn.Exec(`CREATE TABLE pricing_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		model TEXT NOT NULL,
		normalized_model TEXT NOT NULL UNIQUE,
		input_per_1m REAL NOT NULL,
		cached_input_per_1m REAL NOT NULL,
		output_per_1m REAL NOT NULL,
		source TEXT NOT NULL,
		effective_from TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := Seed(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	return conn
}

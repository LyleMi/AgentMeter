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
			name: "glm alias",
			usage: model.Usage{
				Model:        "glm-5.1",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 5.8,
		},
		{
			name: "claude vendor order alias",
			usage: model.Usage{
				Model:        "claude-4.6-opus",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 30,
		},
		{
			name: "deepseek suffix fallback",
			usage: model.Usage{
				Model:        "deepseek-v4-flash-custom",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 0.42,
		},
		{
			name: "deepseek compound suffix fallback",
			usage: model.Usage{
				Model:        "deepseek-v4-flash-custom-tier",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 0.42,
		},
		{
			name: "hy3 custom suffix fallback",
			usage: model.Usage{
				Model:        "hy3-preview-custom",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 0.77,
		},
		{
			name: "gemini 3.1 pro preview",
			usage: model.Usage{
				Model:        "gemini-3.1-pro-preview",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 14,
		},
		{
			name: "gemini 3.1 pro stable alias",
			usage: model.Usage{
				Model:        "gemini-3.1-pro",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 14,
		},
		{
			name: "registered suffix wins before fallback",
			usage: model.Usage{
				Model:        "gpt-5.4-long-context-variant",
				InputTokens:  1_000_000,
				OutputTokens: 1_000_000,
			},
			want: 27.5,
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

func TestNormalizeModelPreservesUnregisteredQualifiers(t *testing.T) {
	if got := NormalizeModel("deepseek-v4-flash-custom"); got != "deepseek-v4-flash-custom" {
		t.Fatalf("NormalizeModel() = %q, want %q", got, "deepseek-v4-flash-custom")
	}
}

func TestComputeHandlesCacheReadSeparateFromInput(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	cost, unpriced := Compute(conn, model.Usage{
		Model:             "claude-4.6-opus",
		InputTokens:       1_000,
		CachedInputTokens: 10_000,
		OutputTokens:      1_000,
	})
	if unpriced {
		t.Fatal("usage marked unpriced")
	}
	if cost == nil {
		t.Fatal("cost is nil")
	}
	if math.Abs(*cost-0.035) > 0.000001 {
		t.Fatalf("cost = %f, want %f", *cost, 0.035)
	}
}

func TestComputeBillsSeparateReasoningOutputWithoutDoubleCounting(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	cost, unpriced := Compute(conn, model.Usage{
		Model:                 "gemini-2.5-flash",
		InputTokens:           1_000,
		OutputTokens:          200,
		ReasoningOutputTokens: 300,
		TotalTokens:           1_500,
	})
	if unpriced {
		t.Fatal("usage marked unpriced")
	}
	if cost == nil {
		t.Fatal("cost is nil")
	}
	if math.Abs(*cost-0.00155) > 0.000001 {
		t.Fatalf("cost = %f, want %f", *cost, 0.00155)
	}

	cost, unpriced = Compute(conn, model.Usage{
		Model:                 "gpt-5",
		InputTokens:           1_000,
		OutputTokens:          500,
		ReasoningOutputTokens: 300,
		TotalTokens:           1_500,
	})
	if unpriced {
		t.Fatal("usage marked unpriced")
	}
	if cost == nil {
		t.Fatal("cost is nil")
	}
	if math.Abs(*cost-0.00625) > 0.000001 {
		t.Fatalf("cost = %f, want %f", *cost, 0.00625)
	}

	cost, unpriced = Compute(conn, model.Usage{
		Model:                 "claude-4.6-opus",
		InputTokens:           1_000,
		CachedInputTokens:     10_000,
		OutputTokens:          1_000,
		ReasoningOutputTokens: 100,
		TotalTokens:           12_000,
	})
	if unpriced {
		t.Fatal("usage marked unpriced")
	}
	if cost == nil {
		t.Fatal("cost is nil")
	}
	if math.Abs(*cost-0.035) > 0.000001 {
		t.Fatalf("cost = %f, want %f", *cost, 0.035)
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

func TestCalculatorCacheSavings(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	calculator, err := LoadCalculator(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	savings := calculator.CacheSavings(model.Usage{
		Model:             "openai/gpt5.5",
		InputTokens:       1_000_000,
		CachedInputTokens: 200_000,
		OutputTokens:      500_000,
	})
	if savings == nil {
		t.Fatal("savings is nil")
	}
	if math.Abs(*savings-0.9) > 0.000001 {
		t.Fatalf("savings = %f, want %f", *savings, 0.9)
	}
	savings = calculator.CacheSavings(model.Usage{
		Model:             "openai/gpt5.5-custom",
		CachedInputTokens: 200_000,
		TotalTokens:       200_000,
	})
	if savings == nil {
		t.Fatal("suffix fallback savings is nil")
	}
	if math.Abs(*savings-0.9) > 0.000001 {
		t.Fatalf("suffix fallback savings = %f, want %f", *savings, 0.9)
	}
	if got := calculator.CacheSavings(model.Usage{Model: "unknown-model", CachedInputTokens: 200_000, TotalTokens: 200_000}); got != nil {
		t.Fatalf("unknown model savings = %v", got)
	}
	if got := calculator.CacheSavings(model.Usage{Model: "command", CachedInputTokens: 200_000, TotalTokens: 200_000}); got != nil {
		t.Fatalf("non-discounted cache savings = %v", got)
	}
}

func TestUpsertCustomPricingOverridesSeedAndSurvivesSeed(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	saved, err := UpsertCustom(context.Background(), conn, model.PricingModelInput{
		Model:            "codex-auto-review",
		InputPer1M:       9,
		CachedInputPer1M: 1,
		OutputPer1M:      20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.IsCustom || saved.NormalizedModel != "codex-auto-review" {
		t.Fatalf("saved custom pricing = %+v", saved)
	}
	cost, unpriced := Compute(conn, model.Usage{
		Model:        "codex-auto-review",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	})
	if unpriced || cost == nil || math.Abs(*cost-29) > 0.000001 {
		t.Fatalf("custom cost = %v unpriced=%v, want 29 priced", cost, unpriced)
	}

	if _, err := UpsertCustom(context.Background(), conn, model.PricingModelInput{
		Model:            "gpt-5",
		InputPer1M:       9,
		CachedInputPer1M: 1,
		OutputPer1M:      20,
	}); err != nil {
		t.Fatal(err)
	}
	if err := Seed(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	cost, unpriced = Compute(conn, model.Usage{
		Model:        "gpt-5",
		InputTokens:  1_000_000,
		OutputTokens: 1_000_000,
	})
	if unpriced || cost == nil || math.Abs(*cost-29) > 0.000001 {
		t.Fatalf("custom seed override cost = %v unpriced=%v, want 29 priced", cost, unpriced)
	}
}

func TestUpsertCustomPricingRejectsInvalidInput(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()

	if _, err := UpsertCustom(context.Background(), conn, model.PricingModelInput{Model: "", InputPer1M: 1}); err == nil {
		t.Fatal("empty model should fail")
	}
	if _, err := UpsertCustom(context.Background(), conn, model.PricingModelInput{Model: "custom", InputPer1M: -1}); err == nil {
		t.Fatal("negative price should fail")
	}
	if _, err := UpsertCustom(context.Background(), conn, model.PricingModelInput{Model: "custom", InputPer1M: math.Inf(1)}); err == nil {
		t.Fatal("infinite price should fail")
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
		effective_from TEXT NOT NULL,
		is_custom INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := Seed(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	return conn
}

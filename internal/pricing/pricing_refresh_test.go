package pricing

import (
	"context"
	"math"
	"testing"

	"github.com/LyleMi/AgentMeter/internal/model"
)

func TestSeptemberPricingRefresh(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()
	calc, err := LoadCalculator(context.Background(), conn)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		model                 string
		input, cached, output float64
	}{
		{"openai/gpt-6-astra", 10, 1, 50},
		{"gpt-6-sol", 2, .2, 10},
		{"gpt-6-luna", .1, .01, .5},
		{"gpt-6-sol-long-context", 4, .4, 15},
		{"gpt-5.6-sol", 4, .4, 20},
		{"gpt-5.6-terra", 2, .2, 12},
		{"gpt-5.6-luna", .2, .02, 1.2},
		{"anthropic/claude-fable-5-1-20260901", 10, .25, 50},
		{"claude-mythos-5.1", 10, .25, 50},
		{"claude-opus-5-5", 4, .2, 20},
		{"deepseek/deepseek-flash", .3, .006, 1.2},
		{"deepseek-v4-flash-vision-exp", .3, .006, 1.2},
		{"deepseek-v4-pro-0813", 1.32, .044, 3.96},
		{"deepseek-flash-off-peak", .15, .003, .6},
		{"deepseek-v4-pro-off-peak", .66, .022, 1.98},
		{"glm-5", 1, .2, 3.2},
		{"glm-5.1", 1.4, .26, 4.4},
		{"glm-5.3-flashx", .37, .075, 1.25},
		{"grok-4.7", 2, .5, 6},
		{"gemini-3.8-flash", .75, .075, 3.75},
	} {
		t.Run(tc.model, func(t *testing.T) {
			// Each class is checked independently so cache and output errors cannot cancel.
			for _, sample := range []struct {
				usage model.Usage
				want  float64
			}{
				{model.Usage{Model: tc.model, InputTokens: 1_000_000}, tc.input},
				{model.Usage{Model: tc.model, InputTokens: 1_000_000, CachedInputTokens: 1_000_000}, tc.cached},
				{model.Usage{Model: tc.model, OutputTokens: 1_000_000}, tc.output},
			} {
				cost, unpriced := Compute(conn, sample.usage)
				cachedCost, cachedUnpriced := calc.Compute(sample.usage)
				if unpriced || cachedUnpriced || cost == nil || cachedCost == nil || math.Abs(*cost-sample.want) > 1e-9 || math.Abs(*cachedCost-sample.want) > 1e-9 {
					t.Fatalf("pricing %s: %v %v, want %v", tc.model, cost, cachedCost, sample.want)
				}
			}
		})
	}
}

func TestSeedRefreshesExistingRates(t *testing.T) {
	conn := openSeededPricingDB(t)
	defer conn.Close()
	if _, err := conn.Exec(`UPDATE pricing_models SET input_per_1m = 99 WHERE normalized_model = 'gpt-5.6-sol'`); err != nil {
		t.Fatal(err)
	}
	if err := Seed(context.Background(), conn); err != nil {
		t.Fatal(err)
	}
	cost, unpriced := Compute(conn, model.Usage{Model: "gpt-5.6-sol", InputTokens: 1_000_000})
	if unpriced || cost == nil || *cost != 4 {
		t.Fatalf("stale seeded price: %v", cost)
	}
	seen := map[string]bool{}
	for _, rate := range seedRates() {
		if seen[rate.NormalizedModel] {
			t.Fatalf("duplicate seed %s", rate.NormalizedModel)
		}
		seen[rate.NormalizedModel] = true
	}
}

package pricing

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

type Rate struct {
	Model            string
	NormalizedModel  string
	InputPer1M       float64
	CachedInputPer1M float64
	OutputPer1M      float64
	Source           string
	EffectiveFrom    time.Time
}

func Seed(ctx context.Context, conn *sql.DB) error {
	// These are API list-price estimates used only when a local Codex model name
	// can be matched. Codex subscription usage may not map one-to-one to API cost.
	effective := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	rates := []Rate{
		{Model: "gpt-5.5", NormalizedModel: "gpt-5.5", InputPer1M: 5.00, CachedInputPer1M: 0.50, OutputPer1M: 30.00, Source: "OpenAI API pricing page, standard estimate", EffectiveFrom: effective},
		{Model: "gpt-5.5-short-context", NormalizedModel: "gpt-5.5-short-context", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: "OpenAI API pricing page, short-context estimate", EffectiveFrom: effective},
		{Model: "gpt-5.4", NormalizedModel: "gpt-5.4", InputPer1M: 2.50, CachedInputPer1M: 0.25, OutputPer1M: 15.00, Source: "OpenAI API pricing page, standard estimate", EffectiveFrom: effective},
		{Model: "gpt-5", NormalizedModel: "gpt-5", InputPer1M: 1.25, CachedInputPer1M: 0.125, OutputPer1M: 10.00, Source: "OpenAI API pricing page, built-in estimate", EffectiveFrom: effective},
		{Model: "gpt-5-mini", NormalizedModel: "gpt-5-mini", InputPer1M: 0.25, CachedInputPer1M: 0.025, OutputPer1M: 2.00, Source: "OpenAI API pricing page, built-in estimate", EffectiveFrom: effective},
		{Model: "gpt-5-nano", NormalizedModel: "gpt-5-nano", InputPer1M: 0.05, CachedInputPer1M: 0.005, OutputPer1M: 0.40, Source: "OpenAI API pricing page, built-in estimate", EffectiveFrom: effective},
	}
	for _, rate := range rates {
		_, err := conn.ExecContext(ctx, `INSERT INTO pricing_models
			(model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(normalized_model) DO UPDATE SET
				model = excluded.model,
				input_per_1m = excluded.input_per_1m,
				cached_input_per_1m = excluded.cached_input_per_1m,
				output_per_1m = excluded.output_per_1m,
				source = excluded.source,
				effective_from = excluded.effective_from`,
			rate.Model, rate.NormalizedModel, rate.InputPer1M, rate.CachedInputPer1M, rate.OutputPer1M, rate.Source, rate.EffectiveFrom.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
	}
	return nil
}

func NormalizeModel(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "openai/")
	normalized = strings.TrimPrefix(normalized, "models/")
	return normalized
}

func Compute(conn *sql.DB, usage model.Usage) (*float64, bool) {
	if usage.Model == "" || usage.Source == "unknown" {
		return nil, true
	}
	var inputRate, cachedRate, outputRate float64
	err := conn.QueryRow(`SELECT input_per_1m, cached_input_per_1m, output_per_1m FROM pricing_models WHERE normalized_model = ?`, NormalizeModel(usage.Model)).
		Scan(&inputRate, &cachedRate, &outputRate)
	if err == sql.ErrNoRows {
		return nil, true
	}
	if err != nil {
		return nil, true
	}
	uncachedInput := usage.InputTokens - usage.CachedInputTokens
	if uncachedInput < 0 {
		uncachedInput = 0
	}
	cost := (float64(uncachedInput)*inputRate + float64(usage.CachedInputTokens)*cachedRate + float64(usage.OutputTokens)*outputRate) / 1_000_000
	return &cost, false
}

func List(ctx context.Context, conn *sql.DB) ([]model.PricingModel, error) {
	rows, err := conn.QueryContext(ctx, `SELECT id, model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from FROM pricing_models ORDER BY normalized_model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.PricingModel
	for rows.Next() {
		var item model.PricingModel
		var effective string
		if err := rows.Scan(&item.ID, &item.Model, &item.NormalizedModel, &item.InputPer1M, &item.CachedInputPer1M, &item.OutputPer1M, &item.Source, &effective); err != nil {
			return nil, err
		}
		item.EffectiveFrom, _ = time.Parse(time.RFC3339Nano, effective)
		result = append(result, item)
	}
	return result, rows.Err()
}

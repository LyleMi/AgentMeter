package pricing

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

type Rate struct {
	Model            string
	NormalizedModel  string
	InputPer1M       float64
	CachedInputPer1M float64
	OutputPer1M      float64
	Source           string
	EffectiveFrom    time.Time
	IsCustom         bool
}

type Calculator struct {
	rates map[string]Rate
}

var ErrInvalidRate = errors.New("invalid pricing model")

func Seed(ctx context.Context, conn *sql.DB) error {
	for _, rate := range seedRates() {
		if err := seedRate(ctx, conn, rate); err != nil {
			return err
		}
	}
	return nil
}

func seedRate(ctx context.Context, conn *sql.DB, rate Rate) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO pricing_models
		(model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0)
		ON CONFLICT(normalized_model) DO UPDATE SET
			model = excluded.model,
			input_per_1m = excluded.input_per_1m,
			cached_input_per_1m = excluded.cached_input_per_1m,
			output_per_1m = excluded.output_per_1m,
			source = excluded.source,
			effective_from = excluded.effective_from,
			is_custom = 0
		WHERE pricing_models.is_custom = 0`,
		rate.Model,
		rate.NormalizedModel,
		rate.InputPer1M,
		rate.CachedInputPer1M,
		rate.OutputPer1M,
		rate.Source,
		rate.EffectiveFrom.Format(time.RFC3339Nano),
	)
	return err
}

func NormalizeModel(value string) string {
	return normalizeModelAlias(normalizeModelName(value))
}

func normalizeModelName(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "models/")
	for _, prefix := range []string{
		"openai/",
		"anthropic/",
		"google/",
		"google-gemini/",
		"gemini/",
		"deepseek/",
		"mistral/",
		"xai/",
		"x-ai/",
		"tencent/",
		"tencent-hunyuan/",
		"hunyuan/",
		"cohere/",
		"qwen/",
		"dashscope/",
		"alibaba/",
	} {
		normalized = strings.TrimPrefix(normalized, prefix)
	}
	if strings.HasPrefix(normalized, "gpt") && len(normalized) > 3 && normalized[3] >= '0' && normalized[3] <= '9' {
		normalized = "gpt-" + normalized[3:]
	}
	return normalized
}

func normalizeModelAlias(normalized string) string {
	if alias, ok := modelAliases[normalized]; ok {
		return alias
	}
	return normalized
}

func normalizedModelCandidates(value string) []string {
	normalized := normalizeModelName(value)
	candidates := make([]string, 0, 4)
	seen := map[string]bool{}
	add := func(candidate string) {
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}

	for {
		add(normalizeModelAlias(normalized))
		index := strings.LastIndex(normalized, "-")
		if index <= 0 {
			break
		}
		normalized = normalized[:index]
	}
	return candidates
}

var modelAliases = map[string]string{
	"claude-4.5-haiku":    "claude-haiku-4.5",
	"claude-haiku-4-5":    "claude-haiku-4.5",
	"claude-4.6-opus":     "claude-opus-4.6",
	"claude-4.6-sonnet":   "claude-sonnet-4.6",
	"claude-4.7-opus":     "claude-opus-4.7",
	"claude-opus-4-8":     "claude-opus-4.8",
	"claude-opus-4.6-1m":  "claude-opus-4.6",
	"claude-sonnet-4-6":   "claude-sonnet-4.6",
	"glm-5":               "glm-5.2",
	"glm-5.1":             "glm-5.2",
	"gpt-5.1-codex-mini":  "gpt-5-mini",
	"gpt-5.6":             "gpt-5.6-sol",
	"hy3":                 "hy3-preview",
	"hunyuan-hy3":         "hy3-preview",
	"hunyuan-hy3-preview": "hy3-preview",
}

func Compute(conn *sql.DB, usage model.Usage) (*float64, bool) {
	rate, ok := rateForUsage(conn, usage)
	return computeWithRate(usage, rate, ok)
}

func LoadCalculator(ctx context.Context, conn *sql.DB) (Calculator, error) {
	rows, err := conn.QueryContext(ctx, `SELECT model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from FROM pricing_models`)
	if err != nil {
		return Calculator{}, err
	}
	defer rows.Close()

	calc := Calculator{rates: map[string]Rate{}}
	for rows.Next() {
		var rate Rate
		var effective string
		if err := rows.Scan(&rate.Model, &rate.NormalizedModel, &rate.InputPer1M, &rate.CachedInputPer1M, &rate.OutputPer1M, &rate.Source, &effective); err != nil {
			return Calculator{}, err
		}
		rate.EffectiveFrom, _ = time.Parse(time.RFC3339Nano, effective)
		calc.rates[rate.NormalizedModel] = rate
	}
	return calc, rows.Err()
}

func UpsertCustom(ctx context.Context, conn *sql.DB, input model.PricingModelInput) (model.PricingModel, error) {
	rate, err := customRate(input)
	if err != nil {
		return model.PricingModel{}, err
	}
	if _, err := conn.ExecContext(ctx, `INSERT INTO pricing_models
		(model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT(normalized_model) DO UPDATE SET
			model = excluded.model,
			input_per_1m = excluded.input_per_1m,
			cached_input_per_1m = excluded.cached_input_per_1m,
			output_per_1m = excluded.output_per_1m,
			source = excluded.source,
			effective_from = excluded.effective_from,
			is_custom = 1`,
		rate.Model,
		rate.NormalizedModel,
		rate.InputPer1M,
		rate.CachedInputPer1M,
		rate.OutputPer1M,
		rate.Source,
		rate.EffectiveFrom.Format(time.RFC3339Nano),
	); err != nil {
		return model.PricingModel{}, err
	}
	return get(ctx, conn, rate.NormalizedModel)
}

func customRate(input model.PricingModelInput) (Rate, error) {
	modelName := strings.TrimSpace(input.Model)
	if modelName == "" {
		return Rate{}, ErrInvalidRate
	}
	normalized := NormalizeModel(modelName)
	if normalized == "" || normalized == "unknown" {
		return Rate{}, ErrInvalidRate
	}
	for _, value := range []float64{input.InputPer1M, input.CachedInputPer1M, input.OutputPer1M} {
		if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
			return Rate{}, ErrInvalidRate
		}
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "Custom pricing"
	}
	return Rate{
		Model:            modelName,
		NormalizedModel:  normalized,
		InputPer1M:       input.InputPer1M,
		CachedInputPer1M: input.CachedInputPer1M,
		OutputPer1M:      input.OutputPer1M,
		Source:           source,
		EffectiveFrom:    time.Now().UTC(),
		IsCustom:         true,
	}, nil
}

func (c Calculator) Compute(usage model.Usage) (*float64, bool) {
	if !hasBillableUsage(usage) {
		return nil, false
	}
	normalized := NormalizeModel(usage.Model)
	if usage.Model == "" || normalized == "unknown" {
		return nil, true
	}
	rate, ok := c.rateForModel(usage.Model)
	return computeWithRate(usage, rate, ok)
}

func (c Calculator) CacheSavings(usage model.Usage) *float64 {
	if !hasBillableUsage(usage) || usage.CachedInputTokens <= 0 {
		return nil
	}
	normalized := NormalizeModel(usage.Model)
	if usage.Model == "" || normalized == "unknown" {
		return nil
	}
	rate, ok := c.rateForModel(usage.Model)
	if !ok || rate.InputPer1M <= rate.CachedInputPer1M {
		return nil
	}
	savings := float64(usage.CachedInputTokens) * (rate.InputPer1M - rate.CachedInputPer1M) / 1_000_000
	if savings <= 0 {
		return nil
	}
	return &savings
}

func (c Calculator) rateForModel(value string) (Rate, bool) {
	for _, candidate := range normalizedModelCandidates(value) {
		if rate, ok := c.rates[candidate]; ok {
			return rate, true
		}
	}
	return Rate{}, false
}

func rateForUsage(conn *sql.DB, usage model.Usage) (Rate, bool) {
	if !hasBillableUsage(usage) {
		return Rate{}, false
	}
	normalized := NormalizeModel(usage.Model)
	if usage.Model == "" || normalized == "unknown" {
		return Rate{}, false
	}
	for _, candidate := range normalizedModelCandidates(usage.Model) {
		var rate Rate
		err := conn.QueryRow(`SELECT input_per_1m, cached_input_per_1m, output_per_1m FROM pricing_models WHERE normalized_model = ?`, candidate).
			Scan(&rate.InputPer1M, &rate.CachedInputPer1M, &rate.OutputPer1M)
		if err == nil {
			return rate, true
		}
		if err != sql.ErrNoRows {
			return Rate{}, false
		}
	}
	return Rate{}, false
}

func computeWithRate(usage model.Usage, rate Rate, hasRate bool) (*float64, bool) {
	if !hasBillableUsage(usage) {
		return nil, false
	}
	if usage.Model == "" || NormalizeModel(usage.Model) == "unknown" || !hasRate {
		return nil, true
	}
	uncachedInput, cachedInput := billableInputTokens(usage)
	outputTokens := billableOutputTokens(usage)
	cost := (float64(uncachedInput)*rate.InputPer1M + float64(cachedInput)*rate.CachedInputPer1M + float64(outputTokens)*rate.OutputPer1M) / 1_000_000
	return &cost, false
}

func billableInputTokens(usage model.Usage) (int64, int64) {
	inputTokens := usage.InputTokens
	if inputTokens < 0 {
		inputTokens = 0
	}
	cachedInputTokens := usage.CachedInputTokens
	if cachedInputTokens < 0 {
		cachedInputTokens = 0
	}
	if cachedInputTokens > inputTokens {
		return inputTokens, cachedInputTokens
	}
	return inputTokens - cachedInputTokens, cachedInputTokens
}

func billableOutputTokens(usage model.Usage) int64 {
	outputTokens := usage.OutputTokens
	if outputTokens < 0 {
		outputTokens = 0
	}
	reasoningTokens := usage.ReasoningOutputTokens
	if reasoningTokens <= 0 {
		return outputTokens
	}
	if usageReasoningOutputAppearsSeparate(usage, outputTokens, reasoningTokens) {
		return outputTokens + reasoningTokens
	}
	return outputTokens
}

func usageReasoningOutputAppearsSeparate(usage model.Usage, outputTokens, reasoningTokens int64) bool {
	if outputTokens <= 0 || reasoningTokens <= 0 || usage.TotalTokens <= 0 {
		return false
	}
	if reasoningTokens > outputTokens {
		return true
	}
	if !strings.Contains(NormalizeModel(usage.Model), "gemini") {
		return false
	}
	promptTokens := usage.InputTokens
	if usage.CachedInputTokens > usage.InputTokens {
		promptTokens += usage.CachedInputTokens
	}
	return usage.TotalTokens >= promptTokens+outputTokens+reasoningTokens
}

func hasBillableUsage(usage model.Usage) bool {
	return usage.InputTokens > 0 ||
		usage.CachedInputTokens > 0 ||
		usage.OutputTokens > 0 ||
		usage.ReasoningOutputTokens > 0 ||
		usage.TotalTokens > 0
}

func List(ctx context.Context, conn *sql.DB) ([]model.PricingModel, error) {
	rows, err := conn.QueryContext(ctx, `SELECT id, model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom FROM pricing_models ORDER BY normalized_model`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []model.PricingModel
	for rows.Next() {
		item, err := scanPricingModel(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func get(ctx context.Context, conn *sql.DB, normalizedModel string) (model.PricingModel, error) {
	row := conn.QueryRowContext(ctx, `SELECT id, model, normalized_model, input_per_1m, cached_input_per_1m, output_per_1m, source, effective_from, is_custom
		FROM pricing_models WHERE normalized_model = ?`, normalizedModel)
	return scanPricingModel(row)
}

func scanPricingModel(row interface{ Scan(dest ...any) error }) (model.PricingModel, error) {
	var item model.PricingModel
	var effective string
	var isCustom int
	if err := row.Scan(&item.ID, &item.Model, &item.NormalizedModel, &item.InputPer1M, &item.CachedInputPer1M, &item.OutputPer1M, &item.Source, &effective, &isCustom); err != nil {
		return model.PricingModel{}, err
	}
	item.EffectiveFrom, _ = time.Parse(time.RFC3339Nano, effective)
	item.IsCustom = isCustom != 0
	return item, nil
}

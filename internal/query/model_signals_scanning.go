package query

import (
	"context"
	"math"
	"strconv"
	"strings"

	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/pricing"
)

type modelSignalSessionMetric struct {
	SessionID          int64
	SourceID           int64
	SourceRootPath     string
	SourceSessionsPath string
	AgentKind          string
	AgentName          string
	SessionKey         string
	CodexSessionID     string
	ProjectPath        string
	Model              string
	ModelProvider      string
	StartedAt          string
	Day                string
	RawSourcePath      string

	InputTokens           int64
	CachedInputTokens     int64
	OutputTokens          int64
	ReasoningOutputTokens int64
	VisibleOutputTokens   int64
	BillableOutputTokens  int64
	TotalTokens           int64
	WallDurationMS        int64
	ActiveDurationMS      int64
	ModelDurationMS       int64
	ToolDurationMS        int64
	IdleDurationMS        int64
	EstimatedCostUSD      *float64
	Unpriced              bool
	CacheSavingsUSD       *float64
	ModelCalls            int
	FailedModelCalls      int
	ToolCalls             int
	FailedToolCalls       int
	LatencySamples        []float64
	ThroughputSamples     []float64
}

func (s *Service) modelSignalSessionMetrics(ctx context.Context, filters model.AnalyticsFilters) ([]modelSignalSessionMetric, error) {
	where, args := analyticsSessionWhere(filters)
	calculator := s.pricingCalculator(ctx)
	rows, err := s.conn.QueryContext(ctx, modelSignalSessionMetricsSQL(where), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []modelSignalSessionMetric
	for rows.Next() {
		item, latencySamples, throughputSamples, err := scanModelSignalSessionMetric(rows)
		if err != nil {
			return nil, err
		}
		enrichModelSignalSessionMetric(&item, latencySamples, throughputSamples, calculator)
		result = append(result, item)
	}
	return result, rows.Err()
}

func modelSignalSessionMetricsSQL(where []string) string {
	return `WITH model_call_stats AS (
		SELECT
			session_id,
			COUNT(*) AS model_calls,
			SUM(CASE WHEN status NOT IN ('completed', 'success') THEN 1 ELSE 0 END) AS failed_model_calls,
			SUM(CASE WHEN duration_ms > 0 THEN duration_ms ELSE 0 END) AS model_duration_ms,
			GROUP_CONCAT(CASE WHEN duration_ms > 0 AND output_tokens > 0 THEN (CAST(duration_ms AS REAL) * 1000.0 / CAST(output_tokens AS REAL)) END) AS latency_samples,
			GROUP_CONCAT(CASE WHEN duration_ms > 0 AND total_tokens > 0 THEN (CAST(total_tokens AS REAL) / (CAST(duration_ms AS REAL) / 1000.0)) END) AS throughput_samples
		FROM model_calls
		GROUP BY session_id
	), tool_call_stats AS (
		SELECT
			session_id,
			COUNT(*) AS tool_calls,
			SUM(CASE WHEN status NOT IN ('completed', 'success') THEN 1 ELSE 0 END) AS failed_tool_calls
		FROM tool_calls
		GROUP BY session_id
	)
	SELECT
		s.id,
		s.source_id,
		src.root_path,
		src.sessions_path,
		src.kind,
		src.name,
		COALESCE(NULLIF(s.session_key, ''), s.codex_session_id),
		s.codex_session_id,
		s.project_path,
		` + usageSessionModelExpr + `,
		s.model_provider,
		s.started_at,
		substr(s.started_at, 1, 10),
		sf.path,
		COALESCE(tu.input_tokens, 0),
		COALESCE(tu.cached_input_tokens, 0),
		COALESCE(tu.output_tokens, 0),
		COALESCE(tu.reasoning_output_tokens, 0),
		COALESCE(tu.total_tokens, 0),
		COALESCE(s.wall_duration_ms, 0),
		COALESCE(s.active_duration_ms, 0),
		CASE
			WHEN COALESCE(mcs.model_calls, 0) > 0 THEN COALESCE(mcs.model_duration_ms, 0)
			WHEN s.model_duration_ms > 0 THEN s.model_duration_ms
			ELSE 0
		END,
		COALESCE(s.tool_duration_ms, 0),
		COALESCE(s.idle_duration_ms, 0),
		COALESCE(mcs.model_calls, 0),
		COALESCE(mcs.failed_model_calls, 0),
		COALESCE(mcs.latency_samples, ''),
		COALESCE(mcs.throughput_samples, ''),
		COALESCE(tcs.tool_calls, 0),
		COALESCE(tcs.failed_tool_calls, 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN source_files sf ON sf.id = s.source_file_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		LEFT JOIN model_call_stats mcs ON mcs.session_id = s.id
		LEFT JOIN tool_call_stats tcs ON tcs.session_id = s.id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY s.started_at ASC, s.id ASC`
}

type modelSignalMetricScanner interface {
	Scan(dest ...any) error
}

func scanModelSignalSessionMetric(scanner modelSignalMetricScanner) (modelSignalSessionMetric, string, string, error) {
	var item modelSignalSessionMetric
	var latencySamples string
	var throughputSamples string
	err := scanner.Scan(
		&item.SessionID,
		&item.SourceID,
		&item.SourceRootPath,
		&item.SourceSessionsPath,
		&item.AgentKind,
		&item.AgentName,
		&item.SessionKey,
		&item.CodexSessionID,
		&item.ProjectPath,
		&item.Model,
		&item.ModelProvider,
		&item.StartedAt,
		&item.Day,
		&item.RawSourcePath,
		&item.InputTokens,
		&item.CachedInputTokens,
		&item.OutputTokens,
		&item.ReasoningOutputTokens,
		&item.TotalTokens,
		&item.WallDurationMS,
		&item.ActiveDurationMS,
		&item.ModelDurationMS,
		&item.ToolDurationMS,
		&item.IdleDurationMS,
		&item.ModelCalls,
		&item.FailedModelCalls,
		&latencySamples,
		&throughputSamples,
		&item.ToolCalls,
		&item.FailedToolCalls,
	)
	return item, latencySamples, throughputSamples, err
}

func enrichModelSignalSessionMetric(item *modelSignalSessionMetric, latencySamples, throughputSamples string, calculator pricing.Calculator) {
	if item.AgentName == "" {
		item.AgentName = item.AgentKind
	}
	usage := model.Usage{
		Model:                 item.Model,
		InputTokens:           item.InputTokens,
		CachedInputTokens:     item.CachedInputTokens,
		OutputTokens:          item.OutputTokens,
		ReasoningOutputTokens: item.ReasoningOutputTokens,
		TotalTokens:           item.TotalTokens,
	}
	item.EstimatedCostUSD, item.Unpriced = calculator.Compute(usage)
	item.CacheSavingsUSD = calculator.CacheSavings(usage)
	item.VisibleOutputTokens, item.BillableOutputTokens, _, _ = reasoningOutputSemantics(
		item.InputTokens,
		item.CachedInputTokens,
		item.OutputTokens,
		item.ReasoningOutputTokens,
		item.TotalTokens,
		item.Model,
	)
	item.LatencySamples = modelSignalSamplesOrFallback(latencySamples, modelLatencyMSPer1kOutputTokens(item.OutputTokens, item.ModelDurationMS))
	item.ThroughputSamples = modelSignalSamplesOrFallback(throughputSamples, throughputPerSecond(item.TotalTokens, item.ModelDurationMS))
}

func modelSignalSamplesOrFallback(value string, fallback float64) []float64 {
	samples := parseModelSignalSamples(value)
	if len(samples) > 0 || fallback <= 0 {
		return samples
	}
	return []float64{fallback}
}

func parseModelSignalSamples(value string) []float64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	samples := make([]float64, 0, len(parts))
	for _, part := range parts {
		sample, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || sample <= 0 || math.IsNaN(sample) || math.IsInf(sample, 0) {
			continue
		}
		samples = append(samples, sample)
	}
	return samples
}

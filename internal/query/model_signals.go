package query

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"AgentMeter/internal/db"
	"AgentMeter/internal/model"
	"AgentMeter/internal/sourcepath"
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

func (s *Service) ModelSignals(ctx context.Context) (model.ModelSignals, error) {
	return s.ModelSignalsWithFilters(ctx, model.AnalyticsFilters{})
}

func (s *Service) ModelSignalsWithFilters(ctx context.Context, filters model.AnalyticsFilters) (model.ModelSignals, error) {
	metrics, err := s.modelSignalSessionMetrics(ctx, filters)
	if err != nil {
		return model.ModelSignals{}, err
	}
	result := buildModelSignals(metrics)
	normalizeModelSignalsSlices(&result)
	return result, nil
}

func (s *Service) modelSignalSessionMetrics(ctx context.Context, filters model.AnalyticsFilters) ([]modelSignalSessionMetric, error) {
	where, args := analyticsSessionWhere(filters)
	calculator := s.pricingCalculator(ctx)
	rows, err := s.conn.QueryContext(ctx, `SELECT
		s.id,
		s.source_id,
		src.root_path,
		src.sessions_path,
		src.kind,
		src.name,
		COALESCE(NULLIF(s.session_key, ''), s.codex_session_id),
		s.codex_session_id,
		s.project_path,
		`+usageSessionModelExpr+`,
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
			WHEN COALESCE((SELECT COUNT(*) FROM model_calls mc WHERE mc.session_id = s.id), 0) > 0
				THEN COALESCE((SELECT SUM(CASE WHEN mc.duration_ms > 0 THEN mc.duration_ms ELSE 0 END) FROM model_calls mc WHERE mc.session_id = s.id), 0)
			WHEN s.model_duration_ms > 0 THEN s.model_duration_ms
			ELSE 0
		END,
		COALESCE(s.tool_duration_ms, 0),
		COALESCE(s.idle_duration_ms, 0),
		COALESCE((SELECT COUNT(*) FROM model_calls mc WHERE mc.session_id = s.id), 0),
		COALESCE((SELECT COUNT(*) FROM model_calls mc WHERE mc.session_id = s.id AND mc.status NOT IN ('completed', 'success')), 0),
		COALESCE((SELECT GROUP_CONCAT(CASE WHEN mc.duration_ms > 0 AND mc.output_tokens > 0 THEN (CAST(mc.duration_ms AS REAL) * 1000.0 / CAST(mc.output_tokens AS REAL)) END) FROM model_calls mc WHERE mc.session_id = s.id), ''),
		COALESCE((SELECT GROUP_CONCAT(CASE WHEN mc.duration_ms > 0 AND mc.total_tokens > 0 THEN (CAST(mc.total_tokens AS REAL) / (CAST(mc.duration_ms AS REAL) / 1000.0)) END) FROM model_calls mc WHERE mc.session_id = s.id), ''),
		COALESCE((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id), 0),
		COALESCE((SELECT COUNT(*) FROM tool_calls tc WHERE tc.session_id = s.id AND tc.status NOT IN ('completed', 'success')), 0)
		FROM sessions s
		JOIN sources src ON src.id = s.source_id
		JOIN source_files sf ON sf.id = s.source_file_id
		LEFT JOIN token_usage tu ON tu.owner_kind = 'session' AND tu.owner_id = s.id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY s.started_at ASC, s.id ASC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []modelSignalSessionMetric
	for rows.Next() {
		var item modelSignalSessionMetric
		var latencySamples string
		var throughputSamples string
		if err := rows.Scan(
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
		); err != nil {
			return nil, err
		}
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
		item.LatencySamples = parseModelSignalSamples(latencySamples)
		item.ThroughputSamples = parseModelSignalSamples(throughputSamples)
		if len(item.LatencySamples) == 0 {
			if latency := modelLatencyMSPer1kOutputTokens(item.OutputTokens, item.ModelDurationMS); latency > 0 {
				item.LatencySamples = append(item.LatencySamples, latency)
			}
		}
		if len(item.ThroughputSamples) == 0 {
			if throughput := throughputPerSecond(item.TotalTokens, item.ModelDurationMS); throughput > 0 {
				item.ThroughputSamples = append(item.ThroughputSamples, throughput)
			}
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func buildModelSignals(metrics []modelSignalSessionMetric) model.ModelSignals {
	var result model.ModelSignals
	breakdowns := map[string]*model.ModelSignalsBreakdown{}
	breakdownSessionsWithTools := map[string]int{}
	trendByDay := map[string]*model.ModelSignalsTrendPoint{}
	trendSessionsWithTools := map[string]int{}
	sessionsWithTools := 0

	for _, metric := range metrics {
		result.TotalSessions++
		result.TotalModelCalls += metric.ModelCalls
		result.TotalToolCalls += metric.ToolCalls
		result.FailedToolCalls += metric.FailedToolCalls
		if metric.ToolCalls > 0 {
			sessionsWithTools++
		}

		breakdown := breakdowns[metric.Model]
		if breakdown == nil {
			breakdown = &model.ModelSignalsBreakdown{Model: metric.Model}
			breakdowns[metric.Model] = breakdown
		}
		breakdown.SessionCount++
		breakdown.ModelCalls += metric.ModelCalls
		breakdown.ToolCalls += metric.ToolCalls
		breakdown.FailedToolCalls += metric.FailedToolCalls
		if metric.ToolCalls > 0 {
			breakdownSessionsWithTools[metric.Model]++
		}
		addTokenAndDurationTotals(
			&breakdown.TotalTokens,
			&breakdown.InputTokens,
			&breakdown.CachedInputTokens,
			&breakdown.OutputTokens,
			&breakdown.ReasoningOutputTokens,
			&breakdown.ModelDurationMS,
			metric,
		)

		if metric.Day != "" {
			point := trendByDay[metric.Day]
			if point == nil {
				point = &model.ModelSignalsTrendPoint{Date: metric.Day}
				trendByDay[metric.Day] = point
			}
			point.SessionCount++
			point.ModelCalls += metric.ModelCalls
			point.ToolCalls += metric.ToolCalls
			point.FailedToolCalls += metric.FailedToolCalls
			point.TotalTokens += metric.TotalTokens
			point.InputTokens += metric.InputTokens
			point.CachedInputTokens += metric.CachedInputTokens
			point.OutputTokens += metric.OutputTokens
			point.ReasoningOutputTokens += metric.ReasoningOutputTokens
			point.ModelDurationMS += metric.ModelDurationMS
			if metric.ToolCalls > 0 {
				trendSessionsWithTools[metric.Day]++
			}
		}
	}

	var totalTokens, inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, modelDurationMS int64
	for _, metric := range metrics {
		totalTokens += metric.TotalTokens
		inputTokens += metric.InputTokens
		cachedInputTokens += metric.CachedInputTokens
		outputTokens += metric.OutputTokens
		reasoningOutputTokens += metric.ReasoningOutputTokens
		modelDurationMS += metric.ModelDurationMS
	}
	result.ToolFailureRate = safeRateInt(result.FailedToolCalls, result.TotalToolCalls)
	result.ToolDependencyRate = safeRateInt(sessionsWithTools, result.TotalSessions)
	result.AvgModelCallsPerSession = safeRateInt(result.TotalModelCalls, result.TotalSessions)
	result.OutputExpansionRate = safeRate(outputTokens, inputTokens)
	result.ReasoningTokenShare = clamp01(safeRate(reasoningOutputTokens, outputTokens))
	result.CacheMissRate = cacheMissRate(inputTokens, cachedInputTokens)
	result.ModelThroughputTokensPerSecond = throughputPerSecond(totalTokens, modelDurationMS)
	result.ModelThroughputOutputTokensPerSecond = throughputPerSecond(outputTokens, modelDurationMS)

	for _, breakdown := range breakdowns {
		applyModelSignalsBreakdownRates(breakdown, breakdownSessionsWithTools[breakdown.Model])
		result.ModelBreakdown = append(result.ModelBreakdown, *breakdown)
	}
	sort.Slice(result.ModelBreakdown, func(i, j int) bool {
		left := result.ModelBreakdown[i]
		right := result.ModelBreakdown[j]
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		return left.Model < right.Model
	})

	for _, point := range trendByDay {
		point.ToolDependencyRate = safeRateInt(trendSessionsWithTools[point.Date], point.SessionCount)
		applyModelSignalsTrendRates(point)
		result.Trend = append(result.Trend, *point)
	}
	sort.Slice(result.Trend, func(i, j int) bool { return result.Trend[i].Date < result.Trend[j].Date })
	if len(result.Trend) > 30 {
		result.Trend = result.Trend[len(result.Trend)-30:]
	}
	result.Trend = fillModelSignalsTrendGaps(result.Trend)
	applyModelSignalsRollingRates(result.Trend)

	result.AnomalySessions = rankModelSignalAnomalies(metrics, 8)
	result.DailyMetrics = buildModelSignalDailyMetrics(metrics)
	result.HealthSummary, result.Cohorts, result.Matrix, result.ProjectHotspots, result.ProjectMetrics = buildModelSignalHealthReadModels(metrics)
	return result
}

const (
	modelSignalCurrentWindowDuration  = 24 * time.Hour
	modelSignalBaselineWindowDuration = 31 * 24 * time.Hour

	modelSignalSeverityCritical = "critical"
	modelSignalSeverityWarning  = "warning"
	modelSignalSeverityUnknown  = "unknown"
	modelSignalSeverityHealthy  = "healthy"

	modelSignalConfidenceLow  = "low"
	modelSignalConfidenceHigh = "high"
)

type modelSignalMetricWindow int

const (
	modelSignalWindowOutside modelSignalMetricWindow = iota
	modelSignalWindowBaseline
	modelSignalWindowCurrent
)

type modelSignalMetricAccumulator struct {
	set               model.ModelSignalsMetricSet
	sessionsWithTools int
	latencySamples    []float64
	throughputSamples []float64
}

type modelSignalCohortAggregate struct {
	SourceID           int64
	SourceKey          string
	SourceLabel        string
	SourceRootPath     string
	SourceSessionsPath string
	AgentKind          string
	AgentName          string
	ModelProvider      string
	Model              string
	ProjectPath        string
	CohortKey          string
	total              modelSignalMetricAccumulator
	current            modelSignalMetricAccumulator
	baseline           modelSignalMetricAccumulator
}

type modelSignalMatrixCellAggregate struct {
	SourceID           int64
	SourceKey          string
	SourceLabel        string
	SourceRootPath     string
	SourceSessionsPath string
	AgentKind          string
	AgentName          string
	ModelProvider      string
	Model              string
	cohortKeys         map[string]struct{}
	total              modelSignalMetricAccumulator
	current            modelSignalMetricAccumulator
	baseline           modelSignalMetricAccumulator
}

type modelSignalProjectAggregate struct {
	ProjectPath        string
	modelKeys          map[string]struct{}
	modelIdentities    map[string]modelSignalModelIdentity
	modelSessionCounts map[string]int
	sourceIDs          map[int64]struct{}
	total              modelSignalMetricAccumulator
	current            modelSignalMetricAccumulator
	baseline           modelSignalMetricAccumulator
}

type modelSignalModelIdentity struct {
	Provider string
	Model    string
}

func buildModelSignalHealthReadModels(metrics []modelSignalSessionMetric) (model.ModelSignalsHealthSummary, []model.ModelSignalsCohort, []model.ModelSignalsMatrixRow, []model.ModelSignalsProjectHotspot, []model.ModelSignalsProjectMetric) {
	health := model.ModelSignalsHealthSummary{
		Severity:   modelSignalSeverityUnknown,
		TopReasons: []string{},
	}
	cohorts := []model.ModelSignalsCohort{}
	matrix := []model.ModelSignalsMatrixRow{}
	hotspots := []model.ModelSignalsProjectHotspot{}
	projectMetrics := []model.ModelSignalsProjectMetric{}

	anchor := latestModelSignalMetricStart(metrics)
	if anchor.IsZero() {
		return health, cohorts, matrix, hotspots, projectMetrics
	}

	currentFrom := anchor.Add(-modelSignalCurrentWindowDuration)
	baselineFrom := anchor.Add(-modelSignalBaselineWindowDuration)
	health.CurrentWindow = model.ModelSignalsWindow{
		From: db.FormatTime(currentFrom),
		To:   db.FormatTime(anchor),
	}
	health.BaselineWindow = model.ModelSignalsWindow{
		From: db.FormatTime(baselineFrom),
		To:   db.FormatTime(currentFrom),
	}

	cohortAggregates := map[string]*modelSignalCohortAggregate{}
	matrixAggregates := map[string]*modelSignalMatrixCellAggregate{}
	projectAggregates := map[string]*modelSignalProjectAggregate{}
	var currentTotal modelSignalMetricAccumulator
	var baselineTotal modelSignalMetricAccumulator

	for _, metric := range metrics {
		started := db.ParseTime(metric.StartedAt)
		window := modelSignalMetricWindowFor(started, baselineFrom, currentFrom, anchor)

		cohort := modelSignalCohortAggregateFor(cohortAggregates, metric)
		cohort.total.add(metric)
		if window == modelSignalWindowCurrent {
			cohort.current.add(metric)
			currentTotal.add(metric)
		} else if window == modelSignalWindowBaseline {
			cohort.baseline.add(metric)
			baselineTotal.add(metric)
		}

		cell := modelSignalMatrixAggregateFor(matrixAggregates, metric, cohort.CohortKey)
		cell.total.add(metric)
		if window == modelSignalWindowCurrent {
			cell.current.add(metric)
		} else if window == modelSignalWindowBaseline {
			cell.baseline.add(metric)
		}

		project := modelSignalProjectAggregateFor(projectAggregates, metric)
		project.total.add(metric)
		if window == modelSignalWindowCurrent {
			project.current.add(metric)
		} else if window == modelSignalWindowBaseline {
			project.baseline.add(metric)
		}
	}

	currentSet := currentTotal.metricSet()
	baselineSet := baselineTotal.metricSet()
	health.CurrentWindow.SessionCount = currentSet.SessionCount
	health.CurrentWindow.ModelCalls = currentSet.ModelCalls
	health.BaselineWindow.SessionCount = baselineSet.SessionCount
	health.BaselineWindow.ModelCalls = baselineSet.ModelCalls

	globalDrift := compareModelSignalDrift(currentSet, baselineSet)
	health.Severity = globalDrift.Severity
	reasonCounts := map[string]int{}
	addModelSignalReasonCounts(reasonCounts, globalDrift.Reasons)

	for _, aggregate := range cohortAggregates {
		totalSet := aggregate.total.metricSet()
		currentSet := aggregate.current.metricSet()
		baselineSet := aggregate.baseline.metricSet()
		drift := compareModelSignalDrift(currentSet, baselineSet)
		cohorts = append(cohorts, model.ModelSignalsCohort{
			SourceID:              aggregate.SourceID,
			SourceKey:             aggregate.SourceKey,
			SourceLabel:           aggregate.SourceLabel,
			SourceRootPath:        aggregate.SourceRootPath,
			SourceSessionsPath:    aggregate.SourceSessionsPath,
			AgentKind:             aggregate.AgentKind,
			AgentName:             aggregate.AgentName,
			ModelProvider:         aggregate.ModelProvider,
			Model:                 aggregate.Model,
			ProjectPath:           aggregate.ProjectPath,
			CohortKey:             aggregate.CohortKey,
			ModelSignalsMetricSet: totalSet,
			Current:               currentSet,
			Baseline:              baselineSet,
			Drift:                 drift,
		})
		health.CohortCount++
		if drift.Severity == modelSignalSeverityWarning {
			health.WarningCohorts++
		}
		if drift.Severity == modelSignalSeverityCritical {
			health.CriticalCohorts++
		}
		if drift.Confidence == modelSignalConfidenceLow {
			health.LowConfidenceCohorts++
		}
		health.Severity = worseModelSignalSeverity(health.Severity, drift.Severity)
		addModelSignalReasonCounts(reasonCounts, drift.Reasons)
	}
	sortModelSignalCohorts(cohorts)

	matrix = buildModelSignalMatrixRows(matrixAggregates)
	hotspots = buildModelSignalProjectHotspots(projectAggregates)
	projectMetrics = buildModelSignalProjectMetrics(projectAggregates)
	health.TopReasons = topModelSignalReasons(reasonCounts, 5)
	return health, cohorts, matrix, hotspots, projectMetrics
}

func latestModelSignalMetricStart(metrics []modelSignalSessionMetric) time.Time {
	var anchor time.Time
	for _, metric := range metrics {
		started := db.ParseTime(metric.StartedAt)
		if started.After(anchor) {
			anchor = started
		}
	}
	return anchor
}

func modelSignalMetricWindowFor(started, baselineFrom, currentFrom, anchor time.Time) modelSignalMetricWindow {
	if started.IsZero() || anchor.IsZero() || started.After(anchor) {
		return modelSignalWindowOutside
	}
	if !started.Before(currentFrom) {
		return modelSignalWindowCurrent
	}
	if !started.Before(baselineFrom) && started.Before(currentFrom) {
		return modelSignalWindowBaseline
	}
	return modelSignalWindowOutside
}

func modelSignalCohortAggregateFor(aggregates map[string]*modelSignalCohortAggregate, metric modelSignalSessionMetric) *modelSignalCohortAggregate {
	provider := modelSignalProvider(metric.ModelProvider)
	modelName := modelSignalModelName(metric.Model)
	projectPath := modelSignalProjectPath(metric.ProjectPath)
	projectKey := projectFilterKey(metric.ProjectPath)
	sourceKey, sourceLabel := sourceIdentity(metric.SourceID, metric.AgentName, metric.AgentKind)
	key := sourceKey + "\x00" + provider + "\x00" + modelName + "\x00" + projectKey
	aggregate := aggregates[key]
	if aggregate == nil {
		aggregate = &modelSignalCohortAggregate{
			SourceID:           metric.SourceID,
			SourceKey:          sourceKey,
			SourceLabel:        sourceLabel,
			SourceRootPath:     metric.SourceRootPath,
			SourceSessionsPath: metric.SourceSessionsPath,
			AgentKind:          metric.AgentKind,
			AgentName:          metric.AgentName,
			ModelProvider:      provider,
			Model:              modelName,
			ProjectPath:        projectPath,
			CohortKey:          key,
		}
		aggregates[key] = aggregate
	}
	return aggregate
}

func modelSignalMatrixAggregateFor(aggregates map[string]*modelSignalMatrixCellAggregate, metric modelSignalSessionMetric, cohortKey string) *modelSignalMatrixCellAggregate {
	provider := modelSignalProvider(metric.ModelProvider)
	modelName := modelSignalModelName(metric.Model)
	sourceKey, sourceLabel := sourceIdentity(metric.SourceID, metric.AgentName, metric.AgentKind)
	key := sourceKey + "\x00" + provider + "\x00" + modelName
	aggregate := aggregates[key]
	if aggregate == nil {
		aggregate = &modelSignalMatrixCellAggregate{
			SourceID:           metric.SourceID,
			SourceKey:          sourceKey,
			SourceLabel:        sourceLabel,
			SourceRootPath:     metric.SourceRootPath,
			SourceSessionsPath: metric.SourceSessionsPath,
			AgentKind:          metric.AgentKind,
			AgentName:          metric.AgentName,
			ModelProvider:      provider,
			Model:              modelName,
			cohortKeys:         map[string]struct{}{},
		}
		aggregates[key] = aggregate
	}
	aggregate.cohortKeys[cohortKey] = struct{}{}
	return aggregate
}

func modelSignalProjectAggregateFor(aggregates map[string]*modelSignalProjectAggregate, metric modelSignalSessionMetric) *modelSignalProjectAggregate {
	projectPath := modelSignalProjectPath(metric.ProjectPath)
	key := projectFilterKey(metric.ProjectPath)
	aggregate := aggregates[key]
	if aggregate == nil {
		aggregate = &modelSignalProjectAggregate{
			ProjectPath:        projectPath,
			modelKeys:          map[string]struct{}{},
			modelIdentities:    map[string]modelSignalModelIdentity{},
			modelSessionCounts: map[string]int{},
			sourceIDs:          map[int64]struct{}{},
		}
		aggregates[key] = aggregate
	}
	provider := modelSignalProvider(metric.ModelProvider)
	modelName := modelSignalModelName(metric.Model)
	modelKey := provider + "\x00" + modelName
	aggregate.modelKeys[modelKey] = struct{}{}
	aggregate.modelIdentities[modelKey] = modelSignalModelIdentity{Provider: provider, Model: modelName}
	aggregate.modelSessionCounts[modelKey]++
	aggregate.sourceIDs[metric.SourceID] = struct{}{}
	return aggregate
}

func modelSignalProvider(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func modelSignalModelName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func modelSignalProjectPath(value string) string {
	normalized := sourcepath.Normalize(value)
	if normalized == "" {
		return ""
	}
	normalized = strings.ReplaceAll(normalized, "\\", "/")
	return strings.TrimRight(normalized, "/.")
}

func (a *modelSignalMetricAccumulator) add(metric modelSignalSessionMetric) {
	a.set.SessionCount++
	a.set.ModelCalls += metric.ModelCalls
	a.set.FailedModelCalls += metric.FailedModelCalls
	a.set.ToolCalls += metric.ToolCalls
	a.set.FailedToolCalls += metric.FailedToolCalls
	a.set.TotalTokens += metric.TotalTokens
	a.set.InputTokens += metric.InputTokens
	a.set.CachedInputTokens += metric.CachedInputTokens
	a.set.OutputTokens += metric.OutputTokens
	a.set.ReasoningOutputTokens += metric.ReasoningOutputTokens
	a.set.WallDurationMS += metric.WallDurationMS
	a.set.ActiveDurationMS += metric.ActiveDurationMS
	a.set.ModelDurationMS += metric.ModelDurationMS
	a.set.ToolDurationMS += metric.ToolDurationMS
	a.set.IdleDurationMS += metric.IdleDurationMS
	if metric.EstimatedCostUSD != nil {
		addCost(&a.set.EstimatedCostUSD, *metric.EstimatedCostUSD)
	}
	if metric.Unpriced {
		a.set.UnpricedSessionCount++
	}
	if metric.CacheSavingsUSD != nil {
		addCost(&a.set.CacheSavingsUSD, *metric.CacheSavingsUSD)
	}
	for _, latency := range metric.LatencySamples {
		if latency > 0 {
			a.latencySamples = append(a.latencySamples, latency)
		}
	}
	for _, throughput := range metric.ThroughputSamples {
		if throughput > 0 {
			a.throughputSamples = append(a.throughputSamples, throughput)
		}
	}
	if metric.ToolCalls > 0 {
		a.sessionsWithTools++
	}
}

func (a modelSignalMetricAccumulator) metricSet() model.ModelSignalsMetricSet {
	item := a.set
	item.ToolFailureRate = safeRateInt(item.FailedToolCalls, item.ToolCalls)
	item.ToolDependencyRate = safeRateInt(a.sessionsWithTools, item.SessionCount)
	item.AvgModelCallsPerSession = safeRateInt(item.ModelCalls, item.SessionCount)
	item.OutputExpansionRate = safeRate(item.OutputTokens, item.InputTokens)
	item.ReasoningTokenShare = clamp01(safeRate(item.ReasoningOutputTokens, item.OutputTokens))
	item.CacheMissRate = cacheMissRate(item.InputTokens, item.CachedInputTokens)
	item.FailurePressure = safeRateInt(item.FailedModelCalls+item.FailedToolCalls, item.SessionCount)
	item.ModelThroughputTokensPerSecond = throughputPerSecond(item.TotalTokens, item.ModelDurationMS)
	item.ModelThroughputOutputTokensPerSecond = throughputPerSecond(item.OutputTokens, item.ModelDurationMS)
	item.ModelLatencyMsPer1kOutputTokens = modelLatencyMSPer1kOutputTokens(item.OutputTokens, item.ModelDurationMS)
	if item.EstimatedCostUSD != nil && item.UnpricedSessionCount == 0 && item.SessionCount > 0 {
		value := *item.EstimatedCostUSD / float64(item.SessionCount)
		item.CostPerSession = &value
	}
	if item.EstimatedCostUSD != nil && item.UnpricedSessionCount == 0 && item.ActiveDurationMS > 0 {
		value := *item.EstimatedCostUSD / (float64(item.ActiveDurationMS) / 3_600_000)
		item.CostPerActiveHour = &value
	}
	if item.EstimatedCostUSD != nil && item.UnpricedSessionCount == 0 && item.TotalTokens > 0 {
		value := *item.EstimatedCostUSD / (float64(item.TotalTokens) / 1_000)
		item.CostPer1kTokens = &value
	}
	item.P50ModelLatencyMsPer1kOutputTokens = percentileNearest(a.latencySamples, 0.50)
	item.P90ModelLatencyMsPer1kOutputTokens = percentileNearest(a.latencySamples, 0.90)
	item.P50ModelThroughputTokensPerSecond = percentileNearest(a.throughputSamples, 0.50)
	item.P10ModelThroughputTokensPerSecond = percentileNearest(a.throughputSamples, 0.10)
	return item
}

func compareModelSignalDrift(current, baseline model.ModelSignalsMetricSet) model.ModelSignalsDrift {
	drift := model.ModelSignalsDrift{
		Severity:   modelSignalSeverityHealthy,
		Confidence: modelSignalConfidenceHigh,
		Reasons:    []string{},
		Metrics:    []model.ModelSignalsDriftMetric{},
	}
	if note := modelSignalSampleNote(current, baseline); note != "" {
		drift.Severity = modelSignalSeverityUnknown
		drift.Confidence = modelSignalConfidenceLow
		drift.SampleNote = note
		drift.Reasons = append(drift.Reasons, note)
		return drift
	}

	addRelativeIncreaseDriftMetric(&drift, "p90ModelLatencyMsPer1kOutputTokens", "p90 model latency per 1k output tokens", "higher_worse", "model latency increased", current.P90ModelLatencyMsPer1kOutputTokens, baseline.P90ModelLatencyMsPer1kOutputTokens, 0.5, 1.0, 250)
	addRelativeIncreaseDriftMetric(&drift, "modelLatencyMsPer1kOutputTokens", "model latency per 1k output tokens", "higher_worse", "model latency increased", current.ModelLatencyMsPer1kOutputTokens, baseline.ModelLatencyMsPer1kOutputTokens, 0.5, 1.0, 250)
	addRelativeDecreaseDriftMetric(&drift, "p10ModelThroughputTokensPerSecond", "p10 model throughput", "lower_worse", "output throughput dropped", current.P10ModelThroughputTokensPerSecond, baseline.P10ModelThroughputTokensPerSecond, 0.25, 0.5, 25)
	addRelativeDecreaseDriftMetric(&drift, "modelThroughputOutputTokensPerSecond", "model output throughput", "lower_worse", "output throughput dropped", current.ModelThroughputOutputTokensPerSecond, baseline.ModelThroughputOutputTokensPerSecond, 0.25, 0.5, 25)
	addAbsoluteIncreaseDriftMetric(&drift, "toolFailureRate", "tool failure rate", "higher_downstream_symptom", "tool failure rate increased", current.ToolFailureRate, baseline.ToolFailureRate, 0.10, 0.25, 0.10)
	addAbsoluteIncreaseDriftMetric(&drift, "failurePressure", "failure pressure", "higher_worse", "failure pressure increased", current.FailurePressure, baseline.FailurePressure, 0.10, 0.25, 0.10)
	addAbsoluteIncreaseDriftMetric(&drift, "cacheMissRate", "cache miss rate", "higher_symptom", "cache miss rate increased", current.CacheMissRate, baseline.CacheMissRate, 0.20, 0.40, 0.50)
	addRelativeIncreaseDriftMetric(&drift, "avgModelCallsPerSession", "model calls per session", "higher_retry_loop_symptom", "model calls per session increased", current.AvgModelCallsPerSession, baseline.AvgModelCallsPerSession, 0.5, 1.0, 0.5)
	addRelativeIncreaseDriftMetric(&drift, "outputExpansionRate", "output expansion rate", "behavior_higher", "output expansion increased", current.OutputExpansionRate, baseline.OutputExpansionRate, 1.0, 2.0, 1.0)
	addAbsoluteIncreaseDriftMetric(&drift, "reasoningTokenShare", "reasoning token share", "behavior_higher", "reasoning token share increased", current.ReasoningTokenShare, baseline.ReasoningTokenShare, 0.20, 0.40, 0.30)

	return drift
}

func modelSignalSampleNote(current, baseline model.ModelSignalsMetricSet) string {
	switch {
	case current.SessionCount == 0 && baseline.SessionCount == 0:
		return "missing current and baseline windows"
	case current.SessionCount == 0:
		return "missing current window"
	case baseline.SessionCount == 0:
		return "missing baseline window"
	case current.ModelCalls == 0 || baseline.ModelCalls == 0:
		return "missing model call samples"
	case current.SessionCount < 2 || baseline.SessionCount < 2:
		return "low current or baseline sample"
	default:
		return ""
	}
}

func addRelativeIncreaseDriftMetric(drift *model.ModelSignalsDrift, key, label, direction, reason string, current, baseline, warningPct, criticalPct, minDelta float64) {
	if baseline <= 0 || current <= baseline {
		return
	}
	delta := current - baseline
	deltaPct := safeDeltaPct(current, baseline)
	severity := ""
	if deltaPct >= criticalPct && delta >= minDelta {
		severity = modelSignalSeverityCritical
	} else if deltaPct >= warningPct && delta >= minDelta {
		severity = modelSignalSeverityWarning
	}
	if severity == "" {
		return
	}
	appendModelSignalDriftMetric(drift, model.ModelSignalsDriftMetric{
		Key:       key,
		Label:     label,
		Direction: direction,
		Severity:  severity,
		Current:   current,
		Baseline:  baseline,
		Delta:     delta,
		DeltaPct:  deltaPct,
	}, reason)
}

func addRelativeDecreaseDriftMetric(drift *model.ModelSignalsDrift, key, label, direction, reason string, current, baseline, warningPct, criticalPct, minDelta float64) {
	if baseline <= 0 || current >= baseline {
		return
	}
	delta := current - baseline
	deltaPct := safeDeltaPct(current, baseline)
	severity := ""
	if deltaPct <= -criticalPct && -delta >= minDelta {
		severity = modelSignalSeverityCritical
	} else if deltaPct <= -warningPct && -delta >= minDelta {
		severity = modelSignalSeverityWarning
	}
	if severity == "" {
		return
	}
	appendModelSignalDriftMetric(drift, model.ModelSignalsDriftMetric{
		Key:       key,
		Label:     label,
		Direction: direction,
		Severity:  severity,
		Current:   current,
		Baseline:  baseline,
		Delta:     delta,
		DeltaPct:  deltaPct,
	}, reason)
}

func addAbsoluteIncreaseDriftMetric(drift *model.ModelSignalsDrift, key, label, direction, reason string, current, baseline, warningDelta, criticalDelta, minCurrent float64) {
	if current <= baseline || current < minCurrent {
		return
	}
	delta := current - baseline
	severity := ""
	if delta >= criticalDelta {
		severity = modelSignalSeverityCritical
	} else if delta >= warningDelta {
		severity = modelSignalSeverityWarning
	}
	if severity == "" {
		return
	}
	appendModelSignalDriftMetric(drift, model.ModelSignalsDriftMetric{
		Key:       key,
		Label:     label,
		Direction: direction,
		Severity:  severity,
		Current:   current,
		Baseline:  baseline,
		Delta:     delta,
		DeltaPct:  safeDeltaPct(current, baseline),
	}, reason)
}

func appendModelSignalDriftMetric(drift *model.ModelSignalsDrift, metric model.ModelSignalsDriftMetric, reason string) {
	drift.Metrics = append(drift.Metrics, metric)
	drift.Reasons = appendUniqueString(drift.Reasons, reason)
	drift.Severity = worseModelSignalSeverity(drift.Severity, metric.Severity)
}

func buildModelSignalMatrixRows(aggregates map[string]*modelSignalMatrixCellAggregate) []model.ModelSignalsMatrixRow {
	rowsBySource := map[string]*model.ModelSignalsMatrixRow{}
	for _, aggregate := range aggregates {
		totalSet := aggregate.total.metricSet()
		currentSet := aggregate.current.metricSet()
		baselineSet := aggregate.baseline.metricSet()
		drift := compareModelSignalDrift(currentSet, baselineSet)
		row := rowsBySource[aggregate.SourceKey]
		if row == nil {
			row = &model.ModelSignalsMatrixRow{
				SourceID:           aggregate.SourceID,
				SourceKey:          aggregate.SourceKey,
				SourceLabel:        aggregate.SourceLabel,
				SourceRootPath:     aggregate.SourceRootPath,
				SourceSessionsPath: aggregate.SourceSessionsPath,
				AgentKind:          aggregate.AgentKind,
				AgentName:          aggregate.AgentName,
				Cells:              []model.ModelSignalsMatrixCell{},
			}
			rowsBySource[aggregate.SourceKey] = row
		}
		row.Cells = append(row.Cells, model.ModelSignalsMatrixCell{
			ModelProvider: aggregate.ModelProvider,
			Model:         aggregate.Model,
			CohortCount:   len(aggregate.cohortKeys),
			SessionCount:  totalSet.SessionCount,
			ModelCalls:    totalSet.ModelCalls,
			TotalTokens:   totalSet.TotalTokens,
			Severity:      drift.Severity,
			Confidence:    drift.Confidence,
			KeyReason:     firstModelSignalReason(drift.Reasons),
			Current:       currentSet,
			Baseline:      baselineSet,
		})
	}

	rows := make([]model.ModelSignalsMatrixRow, 0, len(rowsBySource))
	for _, row := range rowsBySource {
		sortModelSignalMatrixCells(row.Cells)
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		left := rows[i]
		right := rows[j]
		leftSeverity := modelSignalMatrixRowSeverityRank(left)
		rightSeverity := modelSignalMatrixRowSeverityRank(right)
		if leftSeverity != rightSeverity {
			return leftSeverity < rightSeverity
		}
		leftTokens := modelSignalMatrixRowTotalTokens(left)
		rightTokens := modelSignalMatrixRowTotalTokens(right)
		if leftTokens != rightTokens {
			return leftTokens > rightTokens
		}
		if left.SourceLabel != right.SourceLabel {
			return left.SourceLabel < right.SourceLabel
		}
		if left.AgentName != right.AgentName {
			return left.AgentName < right.AgentName
		}
		return left.SourceKey < right.SourceKey
	})
	return rows
}

func modelSignalMatrixRowSeverityRank(row model.ModelSignalsMatrixRow) int {
	rank := modelSignalSeverityRank(modelSignalSeverityHealthy)
	for _, cell := range row.Cells {
		if cellRank := modelSignalSeverityRank(cell.Severity); cellRank < rank {
			rank = cellRank
		}
	}
	return rank
}

func modelSignalMatrixRowTotalTokens(row model.ModelSignalsMatrixRow) int64 {
	var total int64
	for _, cell := range row.Cells {
		total += cell.TotalTokens
	}
	return total
}

func buildModelSignalProjectHotspots(aggregates map[string]*modelSignalProjectAggregate) []model.ModelSignalsProjectHotspot {
	hotspots := make([]model.ModelSignalsProjectHotspot, 0, len(aggregates))
	for _, aggregate := range aggregates {
		totalSet := aggregate.total.metricSet()
		currentSet := aggregate.current.metricSet()
		baselineSet := aggregate.baseline.metricSet()
		hotspots = append(hotspots, model.ModelSignalsProjectHotspot{
			ProjectPath:           aggregate.ProjectPath,
			ModelCount:            len(aggregate.modelKeys),
			SourceCount:           len(aggregate.sourceIDs),
			ModelSignalsMetricSet: totalSet,
			Current:               currentSet,
			Baseline:              baselineSet,
			Drift:                 compareModelSignalDrift(currentSet, baselineSet),
		})
	}
	sortModelSignalProjectHotspots(hotspots)
	return hotspots
}

func buildModelSignalDailyMetrics(metrics []modelSignalSessionMetric) []model.ModelSignalsDailyMetric {
	metricsByDay := map[string][]modelSignalSessionMetric{}
	for _, metric := range metrics {
		if metric.Day == "" {
			continue
		}
		metricsByDay[metric.Day] = append(metricsByDay[metric.Day], metric)
	}
	if len(metricsByDay) == 0 {
		return []model.ModelSignalsDailyMetric{}
	}

	dates := make([]string, 0, len(metricsByDay))
	for date := range metricsByDay {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	result := make([]model.ModelSignalsDailyMetric, 0, len(dates))
	for _, date := range dates {
		var current modelSignalMetricAccumulator
		for _, metric := range metricsByDay[date] {
			current.add(metric)
		}

		var baseline modelSignalMetricAccumulator
		observedDays := 0
		day, err := time.Parse(analyticsDateOnlyLayout, date)
		for offset := 1; err == nil && offset <= 7; offset++ {
			previous := metricsByDay[day.AddDate(0, 0, -offset).Format(analyticsDateOnlyLayout)]
			if len(previous) == 0 {
				continue
			}
			for _, metric := range previous {
				baseline.add(metric)
			}
			observedDays++
		}

		currentSet := current.metricSet()
		baselineSet := baseline.metricSet()
		drift := compareModelSignalDrift(currentSet, baselineSet)
		if observedDays < 7 && drift.Confidence != modelSignalConfidenceLow {
			drift.Confidence = modelSignalConfidenceLow
			drift.SampleNote = "insufficient baseline days"
			drift.Reasons = appendUniqueString(drift.Reasons, drift.SampleNote)
			if drift.Severity == modelSignalSeverityWarning || drift.Severity == modelSignalSeverityCritical {
				drift.Severity = modelSignalSeverityUnknown
			}
		}
		result = append(result, model.ModelSignalsDailyMetric{
			Date:                  date,
			ModelSignalsMetricSet: currentSet,
			Baseline:              baselineSet,
			LowSample:             drift.Confidence == modelSignalConfidenceLow || modelSignalMetricSetLowSample(currentSet),
			Drift:                 drift,
			KeyReason:             firstModelSignalReason(drift.Reasons),
		})
	}
	return result
}

func buildModelSignalProjectMetrics(aggregates map[string]*modelSignalProjectAggregate) []model.ModelSignalsProjectMetric {
	projectMetrics := make([]model.ModelSignalsProjectMetric, 0, len(aggregates))
	for _, aggregate := range aggregates {
		totalSet := aggregate.total.metricSet()
		currentSet := aggregate.current.metricSet()
		baselineSet := aggregate.baseline.metricSet()
		dominantProvider, dominantModel, dominantShare := modelSignalProjectDominantModel(aggregate, totalSet.SessionCount)
		projectMetrics = append(projectMetrics, model.ModelSignalsProjectMetric{
			ProjectPath:           aggregate.ProjectPath,
			ModelCount:            len(aggregate.modelKeys),
			SourceCount:           len(aggregate.sourceIDs),
			DominantModelProvider: dominantProvider,
			DominantModel:         dominantModel,
			DominantModelShare:    dominantShare,
			ModelSignalsMetricSet: totalSet,
			Current:               currentSet,
			Baseline:              baselineSet,
			Drift:                 compareModelSignalDrift(currentSet, baselineSet),
		})
	}
	sortModelSignalProjectMetrics(projectMetrics)
	return projectMetrics
}

func modelSignalProjectDominantModel(aggregate *modelSignalProjectAggregate, sessionCount int) (string, string, float64) {
	if aggregate == nil || sessionCount <= 0 || len(aggregate.modelSessionCounts) == 0 {
		return "", "", 0
	}
	bestCount := 0
	bestProvider := ""
	bestModel := ""
	for key, count := range aggregate.modelSessionCounts {
		identity := aggregate.modelIdentities[key]
		if count > bestCount ||
			(count == bestCount && (identity.Provider < bestProvider || (identity.Provider == bestProvider && identity.Model < bestModel))) ||
			bestProvider == "" {
			bestCount = count
			bestProvider = identity.Provider
			bestModel = identity.Model
		}
	}
	return bestProvider, bestModel, safeRateInt(bestCount, sessionCount)
}

func modelSignalMetricSetLowSample(item model.ModelSignalsMetricSet) bool {
	return item.SessionCount > 0 && (item.SessionCount < 3 || item.ModelCalls < 3 || item.ModelDurationMS <= 0)
}

func sortModelSignalCohorts(cohorts []model.ModelSignalsCohort) {
	sort.Slice(cohorts, func(i, j int) bool {
		left := cohorts[i]
		right := cohorts[j]
		if modelSignalSeverityRank(left.Drift.Severity) != modelSignalSeverityRank(right.Drift.Severity) {
			return modelSignalSeverityRank(left.Drift.Severity) < modelSignalSeverityRank(right.Drift.Severity)
		}
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		if left.SourceLabel != right.SourceLabel {
			return left.SourceLabel < right.SourceLabel
		}
		if left.ModelProvider != right.ModelProvider {
			return left.ModelProvider < right.ModelProvider
		}
		if left.Model != right.Model {
			return left.Model < right.Model
		}
		return left.ProjectPath < right.ProjectPath
	})
}

func sortModelSignalMatrixCells(cells []model.ModelSignalsMatrixCell) {
	sort.Slice(cells, func(i, j int) bool {
		left := cells[i]
		right := cells[j]
		if modelSignalSeverityRank(left.Severity) != modelSignalSeverityRank(right.Severity) {
			return modelSignalSeverityRank(left.Severity) < modelSignalSeverityRank(right.Severity)
		}
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		if left.ModelProvider != right.ModelProvider {
			return left.ModelProvider < right.ModelProvider
		}
		return left.Model < right.Model
	})
}

func sortModelSignalProjectHotspots(hotspots []model.ModelSignalsProjectHotspot) {
	sort.Slice(hotspots, func(i, j int) bool {
		left := hotspots[i]
		right := hotspots[j]
		if modelSignalSeverityRank(left.Drift.Severity) != modelSignalSeverityRank(right.Drift.Severity) {
			return modelSignalSeverityRank(left.Drift.Severity) < modelSignalSeverityRank(right.Drift.Severity)
		}
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		return left.ProjectPath < right.ProjectPath
	})
}

func sortModelSignalProjectMetrics(projectMetrics []model.ModelSignalsProjectMetric) {
	sort.Slice(projectMetrics, func(i, j int) bool {
		left := projectMetrics[i]
		right := projectMetrics[j]
		if modelSignalSeverityRank(left.Drift.Severity) != modelSignalSeverityRank(right.Drift.Severity) {
			return modelSignalSeverityRank(left.Drift.Severity) < modelSignalSeverityRank(right.Drift.Severity)
		}
		if left.TotalTokens != right.TotalTokens {
			return left.TotalTokens > right.TotalTokens
		}
		return left.ProjectPath < right.ProjectPath
	})
}

func modelSignalSeverityRank(severity string) int {
	switch severity {
	case modelSignalSeverityCritical:
		return 0
	case modelSignalSeverityWarning:
		return 1
	case modelSignalSeverityUnknown:
		return 2
	case modelSignalSeverityHealthy:
		return 3
	default:
		return 4
	}
}

func worseModelSignalSeverity(left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}
	if modelSignalSeverityRank(right) < modelSignalSeverityRank(left) {
		return right
	}
	return left
}

func addModelSignalReasonCounts(counts map[string]int, reasons []string) {
	for _, reason := range reasons {
		if strings.TrimSpace(reason) == "" {
			continue
		}
		counts[reason]++
	}
}

func topModelSignalReasons(counts map[string]int, limit int) []string {
	if len(counts) == 0 || limit <= 0 {
		return []string{}
	}
	reasons := make([]string, 0, len(counts))
	for reason := range counts {
		reasons = append(reasons, reason)
	}
	sort.Slice(reasons, func(i, j int) bool {
		if counts[reasons[i]] != counts[reasons[j]] {
			return counts[reasons[i]] > counts[reasons[j]]
		}
		return reasons[i] < reasons[j]
	})
	if len(reasons) > limit {
		reasons = reasons[:limit]
	}
	return reasons
}

func firstModelSignalReason(reasons []string) string {
	if len(reasons) == 0 {
		return ""
	}
	return reasons[0]
}

func appendUniqueString(values []string, value string) []string {
	for _, candidate := range values {
		if candidate == value {
			return values
		}
	}
	return append(values, value)
}

func addTokenAndDurationTotals(totalTokens, inputTokens, cachedInputTokens, outputTokens, reasoningOutputTokens, modelDurationMS *int64, metric modelSignalSessionMetric) {
	*totalTokens += metric.TotalTokens
	*inputTokens += metric.InputTokens
	*cachedInputTokens += metric.CachedInputTokens
	*outputTokens += metric.OutputTokens
	*reasoningOutputTokens += metric.ReasoningOutputTokens
	*modelDurationMS += metric.ModelDurationMS
}

func applyModelSignalsBreakdownRates(item *model.ModelSignalsBreakdown, sessionsWithTools int) {
	item.ToolFailureRate = safeRateInt(item.FailedToolCalls, item.ToolCalls)
	item.ToolDependencyRate = safeRateInt(sessionsWithTools, item.SessionCount)
	item.AvgModelCallsPerSession = safeRateInt(item.ModelCalls, item.SessionCount)
	item.OutputExpansionRate = safeRate(item.OutputTokens, item.InputTokens)
	item.ReasoningTokenShare = clamp01(safeRate(item.ReasoningOutputTokens, item.OutputTokens))
	item.CacheMissRate = cacheMissRate(item.InputTokens, item.CachedInputTokens)
	item.ModelThroughputTokensPerSecond = throughputPerSecond(item.TotalTokens, item.ModelDurationMS)
	item.ModelThroughputOutputTokensPerSecond = throughputPerSecond(item.OutputTokens, item.ModelDurationMS)
}

func applyModelSignalsTrendRates(point *model.ModelSignalsTrendPoint) {
	point.ToolFailureRate = safeRateInt(point.FailedToolCalls, point.ToolCalls)
	point.OutputExpansionRate = safeRate(point.OutputTokens, point.InputTokens)
	point.ReasoningTokenShare = clamp01(safeRate(point.ReasoningOutputTokens, point.OutputTokens))
	point.CacheMissRate = cacheMissRate(point.InputTokens, point.CachedInputTokens)
	point.ModelThroughputTokensPerSecond = throughputPerSecond(point.TotalTokens, point.ModelDurationMS)
	point.ModelThroughputOutputTokensPerSecond = throughputPerSecond(point.OutputTokens, point.ModelDurationMS)
	point.LowSample = point.SessionCount > 0 && (point.SessionCount < 3 || point.ModelCalls < 3 || point.ModelDurationMS <= 0)
}

func applyModelSignalsRollingRates(points []model.ModelSignalsTrendPoint) {
	for index := range points {
		start := index - 6
		if start < 0 {
			start = 0
		}
		var totalTokens int64
		var modelDurationMS int64
		var toolCalls int
		var failedToolCalls int
		for cursor := start; cursor <= index; cursor++ {
			totalTokens += points[cursor].TotalTokens
			modelDurationMS += points[cursor].ModelDurationMS
			toolCalls += points[cursor].ToolCalls
			failedToolCalls += points[cursor].FailedToolCalls
		}
		points[index].RollingModelThroughputTokensPerSecond = throughputPerSecond(totalTokens, modelDurationMS)
		points[index].RollingToolFailureRate = safeRateInt(failedToolCalls, toolCalls)
	}
}

func fillModelSignalsTrendGaps(points []model.ModelSignalsTrendPoint) []model.ModelSignalsTrendPoint {
	if len(points) <= 1 {
		return points
	}
	start, err := time.Parse(analyticsDateOnlyLayout, points[0].Date)
	if err != nil {
		return points
	}
	end, err := time.Parse(analyticsDateOnlyLayout, points[len(points)-1].Date)
	if err != nil || end.Before(start) {
		return points
	}
	spanDays := int(end.Sub(start).Hours()/24) + 1
	if spanDays <= len(points) || spanDays > 62 {
		return points
	}
	byDate := make(map[string]model.ModelSignalsTrendPoint, len(points))
	for _, point := range points {
		byDate[point.Date] = point
	}
	filled := make([]model.ModelSignalsTrendPoint, 0, spanDays)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		date := day.Format(analyticsDateOnlyLayout)
		if point, ok := byDate[date]; ok {
			filled = append(filled, point)
		} else {
			filled = append(filled, model.ModelSignalsTrendPoint{Date: date})
		}
	}
	return filled
}

func rankModelSignalAnomalies(metrics []modelSignalSessionMetric, limit int) []model.ModelSignalsAnomalySession {
	anomalies := make([]model.ModelSignalsAnomalySession, 0, len(metrics))
	for _, metric := range metrics {
		item := modelSignalsAnomalyFromMetric(metric)
		if len(item.ReasonLabels) == 0 {
			continue
		}
		anomalies = append(anomalies, item)
	}
	sort.Slice(anomalies, func(i, j int) bool {
		if anomalies[i].Score != anomalies[j].Score {
			return anomalies[i].Score > anomalies[j].Score
		}
		if !anomalies[i].StartedAt.Equal(anomalies[j].StartedAt) {
			return anomalies[i].StartedAt.After(anomalies[j].StartedAt)
		}
		return anomalies[i].SessionID > anomalies[j].SessionID
	})
	if len(anomalies) > limit {
		anomalies = anomalies[:limit]
	}
	return anomalies
}

func modelSignalsAnomalyFromMetric(metric modelSignalSessionMetric) model.ModelSignalsAnomalySession {
	sourceKey, sourceLabel := sourceIdentity(metric.SourceID, metric.AgentName, metric.AgentKind)
	item := model.ModelSignalsAnomalySession{
		SessionID:                            metric.SessionID,
		SourceID:                             metric.SourceID,
		SourceKey:                            sourceKey,
		SourceLabel:                          sourceLabel,
		SourceRootPath:                       metric.SourceRootPath,
		SourceSessionsPath:                   metric.SourceSessionsPath,
		AgentKind:                            metric.AgentKind,
		AgentName:                            metric.AgentName,
		SessionKey:                           metric.SessionKey,
		CodexSessionID:                       metric.CodexSessionID,
		ProjectPath:                          metric.ProjectPath,
		Model:                                metric.Model,
		StartedAt:                            db.ParseTime(metric.StartedAt),
		RawSourcePath:                        metric.RawSourcePath,
		ModelCalls:                           metric.ModelCalls,
		ToolCalls:                            metric.ToolCalls,
		FailedToolCalls:                      metric.FailedToolCalls,
		TotalTokens:                          metric.TotalTokens,
		InputTokens:                          metric.InputTokens,
		CachedInputTokens:                    metric.CachedInputTokens,
		OutputTokens:                         metric.OutputTokens,
		ReasoningOutputTokens:                metric.ReasoningOutputTokens,
		ModelDurationMS:                      metric.ModelDurationMS,
		OutputExpansionRate:                  safeRate(metric.OutputTokens, metric.InputTokens),
		ReasoningTokenShare:                  clamp01(safeRate(metric.ReasoningOutputTokens, metric.OutputTokens)),
		CacheMissRate:                        cacheMissRate(metric.InputTokens, metric.CachedInputTokens),
		ModelThroughputTokensPerSecond:       throughputPerSecond(metric.TotalTokens, metric.ModelDurationMS),
		ModelThroughputOutputTokensPerSecond: throughputPerSecond(metric.OutputTokens, metric.ModelDurationMS),
		ToolFailureRate:                      safeRateInt(metric.FailedToolCalls, metric.ToolCalls),
		ReasonLabels:                         []string{},
	}
	if item.ReasoningTokenShare >= 0.5 {
		item.ReasonLabels = append(item.ReasonLabels, "high reasoning share")
		item.Score += item.ReasoningTokenShare * 2
	}
	if item.OutputExpansionRate >= 3 {
		item.ReasonLabels = append(item.ReasonLabels, "high output/input ratio")
		item.Score += math.Min(item.OutputExpansionRate/3, 5)
	}
	if item.TotalTokens > 0 && item.ModelDurationMS > 0 && item.ModelThroughputTokensPerSecond < 500 {
		item.ReasonLabels = append(item.ReasonLabels, "slow model throughput")
		item.Score += (500 - item.ModelThroughputTokensPerSecond) / 500
	}
	if item.FailedToolCalls >= 2 || (item.ToolCalls > 0 && item.ToolFailureRate >= 0.5) {
		item.ReasonLabels = append(item.ReasonLabels, "failed tool calls")
		item.Score += float64(item.FailedToolCalls) + item.ToolFailureRate
	}
	if item.InputTokens > 0 && item.CacheMissRate >= 0.8 {
		item.ReasonLabels = append(item.ReasonLabels, "high cache miss")
		item.Score += item.CacheMissRate
	}
	return item
}

func normalizeModelSignalsSlices(result *model.ModelSignals) {
	if result.Trend == nil {
		result.Trend = []model.ModelSignalsTrendPoint{}
	}
	if result.ModelBreakdown == nil {
		result.ModelBreakdown = []model.ModelSignalsBreakdown{}
	}
	if result.AnomalySessions == nil {
		result.AnomalySessions = []model.ModelSignalsAnomalySession{}
	}
	if result.HealthSummary.Severity == "" {
		result.HealthSummary.Severity = modelSignalSeverityUnknown
	}
	if result.HealthSummary.TopReasons == nil {
		result.HealthSummary.TopReasons = []string{}
	}
	if result.Cohorts == nil {
		result.Cohorts = []model.ModelSignalsCohort{}
	}
	for index := range result.Cohorts {
		normalizeModelSignalsDrift(&result.Cohorts[index].Drift)
	}
	if result.Matrix == nil {
		result.Matrix = []model.ModelSignalsMatrixRow{}
	}
	for rowIndex := range result.Matrix {
		if result.Matrix[rowIndex].Cells == nil {
			result.Matrix[rowIndex].Cells = []model.ModelSignalsMatrixCell{}
		}
	}
	if result.ProjectHotspots == nil {
		result.ProjectHotspots = []model.ModelSignalsProjectHotspot{}
	}
	for index := range result.ProjectHotspots {
		normalizeModelSignalsDrift(&result.ProjectHotspots[index].Drift)
	}
	if result.DailyMetrics == nil {
		result.DailyMetrics = []model.ModelSignalsDailyMetric{}
	}
	for index := range result.DailyMetrics {
		normalizeModelSignalsDrift(&result.DailyMetrics[index].Drift)
	}
	if result.ProjectMetrics == nil {
		result.ProjectMetrics = []model.ModelSignalsProjectMetric{}
	}
	for index := range result.ProjectMetrics {
		normalizeModelSignalsDrift(&result.ProjectMetrics[index].Drift)
	}
}

func normalizeModelSignalsDrift(drift *model.ModelSignalsDrift) {
	if drift.Severity == "" {
		drift.Severity = modelSignalSeverityUnknown
	}
	if drift.Confidence == "" {
		drift.Confidence = modelSignalConfidenceLow
	}
	if drift.Reasons == nil {
		drift.Reasons = []string{}
	}
	if drift.Metrics == nil {
		drift.Metrics = []model.ModelSignalsDriftMetric{}
	}
}

func safeRate(numerator, denominator int64) float64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func safeRateInt(numerator, denominator int) float64 {
	if denominator <= 0 || numerator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func cacheMissRate(inputTokens, cachedInputTokens int64) float64 {
	if inputTokens <= 0 {
		return 0
	}
	return clamp01(float64(inputTokens-cachedInputTokens) / float64(inputTokens))
}

func throughputPerSecond(tokens, durationMS int64) float64 {
	if tokens <= 0 || durationMS <= 0 {
		return 0
	}
	return float64(tokens) / (float64(durationMS) / 1000)
}

func modelLatencyMSPer1kOutputTokens(outputTokens, durationMS int64) float64 {
	if outputTokens <= 0 || durationMS <= 0 {
		return 0
	}
	return float64(durationMS) / float64(outputTokens) * 1000
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

func percentileNearest(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, 0, len(values))
	for _, value := range values {
		if value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0) {
			sorted = append(sorted, value)
		}
	}
	if len(sorted) == 0 {
		return 0
	}
	sort.Float64s(sorted)
	if percentile <= 0 {
		return sorted[0]
	}
	if percentile >= 1 {
		return sorted[len(sorted)-1]
	}
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func safeDeltaPct(current, baseline float64) float64 {
	if baseline <= 0 {
		return 0
	}
	value := (current - baseline) / baseline
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func clamp01(value float64) float64 {
	if value < 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

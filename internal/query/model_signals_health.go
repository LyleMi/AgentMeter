package query

import (
	"sort"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/db"
	"github.com/LyleMi/AgentMeter/internal/model"
	"github.com/LyleMi/AgentMeter/internal/sourcepath"
)

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

type modelSignalHealthWindows struct {
	anchor       time.Time
	currentFrom  time.Time
	baselineFrom time.Time
	current      model.ModelSignalsWindow
	baseline     model.ModelSignalsWindow
}

type modelSignalReadModelAggregates struct {
	cohorts       map[string]*modelSignalCohortAggregate
	matrix        map[string]*modelSignalMatrixCellAggregate
	projects      map[string]*modelSignalProjectAggregate
	currentTotal  modelSignalMetricAccumulator
	baselineTotal modelSignalMetricAccumulator
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

	windows := modelSignalHealthWindowBounds(anchor)
	health.CurrentWindow = windows.current
	health.BaselineWindow = windows.baseline

	aggregates := newModelSignalReadModelAggregates()
	aggregates.addMetrics(metrics, windows)

	currentSet := aggregates.currentTotal.metricSet()
	baselineSet := aggregates.baselineTotal.metricSet()
	setModelSignalHealthWindowCounts(&health, currentSet, baselineSet)

	globalDrift := compareModelSignalDrift(currentSet, baselineSet)
	health.Severity = globalDrift.Severity
	reasonCounts := map[string]int{}
	addModelSignalReasonCounts(reasonCounts, globalDrift.Reasons)

	cohorts = buildModelSignalCohorts(aggregates.cohorts, &health, reasonCounts)
	matrix = buildModelSignalMatrixRows(aggregates.matrix)
	hotspots = buildModelSignalProjectHotspots(aggregates.projects)
	projectMetrics = buildModelSignalProjectMetrics(aggregates.projects)
	health.TopReasons = topModelSignalReasons(reasonCounts, 5)
	return health, cohorts, matrix, hotspots, projectMetrics
}

func modelSignalHealthWindowBounds(anchor time.Time) modelSignalHealthWindows {
	currentFrom := anchor.Add(-modelSignalCurrentWindowDuration)
	baselineFrom := anchor.Add(-modelSignalBaselineWindowDuration)
	return modelSignalHealthWindows{
		anchor:       anchor,
		currentFrom:  currentFrom,
		baselineFrom: baselineFrom,
		current: model.ModelSignalsWindow{
			From: db.FormatTime(currentFrom),
			To:   db.FormatTime(anchor),
		},
		baseline: model.ModelSignalsWindow{
			From: db.FormatTime(baselineFrom),
			To:   db.FormatTime(currentFrom),
		},
	}
}

func newModelSignalReadModelAggregates() modelSignalReadModelAggregates {
	return modelSignalReadModelAggregates{
		cohorts:  map[string]*modelSignalCohortAggregate{},
		matrix:   map[string]*modelSignalMatrixCellAggregate{},
		projects: map[string]*modelSignalProjectAggregate{},
	}
}

func (a *modelSignalReadModelAggregates) addMetrics(metrics []modelSignalSessionMetric, windows modelSignalHealthWindows) {
	for _, metric := range metrics {
		a.addMetric(metric, modelSignalMetricWindowFor(db.ParseTime(metric.StartedAt), windows.baselineFrom, windows.currentFrom, windows.anchor))
	}
}

func (a *modelSignalReadModelAggregates) addMetric(metric modelSignalSessionMetric, window modelSignalMetricWindow) {
	cohort := modelSignalCohortAggregateFor(a.cohorts, metric)
	cohort.total.add(metric)
	cell := modelSignalMatrixAggregateFor(a.matrix, metric, cohort.CohortKey)
	cell.total.add(metric)
	project := modelSignalProjectAggregateFor(a.projects, metric)
	project.total.add(metric)

	switch window {
	case modelSignalWindowCurrent:
		cohort.current.add(metric)
		cell.current.add(metric)
		project.current.add(metric)
		a.currentTotal.add(metric)
	case modelSignalWindowBaseline:
		cohort.baseline.add(metric)
		cell.baseline.add(metric)
		project.baseline.add(metric)
		a.baselineTotal.add(metric)
	}
}

func setModelSignalHealthWindowCounts(health *model.ModelSignalsHealthSummary, currentSet, baselineSet model.ModelSignalsMetricSet) {
	health.CurrentWindow.SessionCount = currentSet.SessionCount
	health.CurrentWindow.ModelCalls = currentSet.ModelCalls
	health.BaselineWindow.SessionCount = baselineSet.SessionCount
	health.BaselineWindow.ModelCalls = baselineSet.ModelCalls
}

func buildModelSignalCohorts(aggregates map[string]*modelSignalCohortAggregate, health *model.ModelSignalsHealthSummary, reasonCounts map[string]int) []model.ModelSignalsCohort {
	cohorts := make([]model.ModelSignalsCohort, 0, len(aggregates))
	for _, aggregate := range aggregates {
		cohort, drift := modelSignalCohortFromAggregate(aggregate)
		cohorts = append(cohorts, cohort)
		updateModelSignalHealthCohortCounts(health, drift)
		addModelSignalReasonCounts(reasonCounts, drift.Reasons)
	}
	sortModelSignalCohorts(cohorts)
	return cohorts
}

func modelSignalCohortFromAggregate(aggregate *modelSignalCohortAggregate) (model.ModelSignalsCohort, model.ModelSignalsDrift) {
	totalSet := aggregate.total.metricSet()
	currentSet := aggregate.current.metricSet()
	baselineSet := aggregate.baseline.metricSet()
	drift := compareModelSignalDrift(currentSet, baselineSet)
	return model.ModelSignalsCohort{
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
	}, drift
}

func updateModelSignalHealthCohortCounts(health *model.ModelSignalsHealthSummary, drift model.ModelSignalsDrift) {
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
	a.addCounts(metric)
	a.addTokens(metric)
	a.addDurations(metric)
	a.addCosts(metric)
	a.addSamples(metric)
}

func (a *modelSignalMetricAccumulator) addCounts(metric modelSignalSessionMetric) {
	a.set.SessionCount++
	a.set.ModelCalls += metric.ModelCalls
	a.set.FailedModelCalls += metric.FailedModelCalls
	a.set.ToolCalls += metric.ToolCalls
	a.set.FailedToolCalls += metric.FailedToolCalls
	if metric.ToolCalls > 0 {
		a.sessionsWithTools++
	}
}

func (a *modelSignalMetricAccumulator) addTokens(metric modelSignalSessionMetric) {
	a.set.TotalTokens += metric.TotalTokens
	a.set.InputTokens += metric.InputTokens
	a.set.CachedInputTokens += metric.CachedInputTokens
	a.set.OutputTokens += metric.OutputTokens
	a.set.ReasoningOutputTokens += metric.ReasoningOutputTokens
	a.set.VisibleOutputTokens += metric.VisibleOutputTokens
	a.set.BillableOutputTokens += metric.BillableOutputTokens
}

func (a *modelSignalMetricAccumulator) addDurations(metric modelSignalSessionMetric) {
	a.set.WallDurationMS += metric.WallDurationMS
	a.set.ActiveDurationMS += metric.ActiveDurationMS
	a.set.ModelDurationMS += metric.ModelDurationMS
	a.set.ToolDurationMS += metric.ToolDurationMS
	a.set.IdleDurationMS += metric.IdleDurationMS
}

func (a *modelSignalMetricAccumulator) addCosts(metric modelSignalSessionMetric) {
	if metric.EstimatedCostUSD != nil {
		addCost(&a.set.EstimatedCostUSD, *metric.EstimatedCostUSD)
	}
	if metric.Unpriced {
		a.set.UnpricedSessionCount++
	}
	if metric.CacheSavingsUSD != nil {
		addCost(&a.set.CacheSavingsUSD, *metric.CacheSavingsUSD)
	}
}

func (a *modelSignalMetricAccumulator) addSamples(metric modelSignalSessionMetric) {
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
}

func (a modelSignalMetricAccumulator) metricSet() model.ModelSignalsMetricSet {
	item := a.set
	a.applyRates(&item)
	applyModelSignalCostRates(&item)
	a.applyPercentiles(&item)
	item.DegradationRiskScore = modelSignalDegradationRiskScore(item)
	return item
}

func (a modelSignalMetricAccumulator) applyRates(item *model.ModelSignalsMetricSet) {
	item.ToolFailureRate = safeRateInt(item.FailedToolCalls, item.ToolCalls)
	item.ToolDependencyRate = safeRateInt(a.sessionsWithTools, item.SessionCount)
	item.AvgModelCallsPerSession = safeRateInt(item.ModelCalls, item.SessionCount)
	item.OutputExpansionRate = safeRate(item.OutputTokens, item.InputTokens)
	if item.BillableOutputTokens <= 0 && item.OutputTokens > 0 {
		item.VisibleOutputTokens, item.BillableOutputTokens = reasoningOutputDenominators(item.OutputTokens, item.ReasoningOutputTokens, false)
	}
	item.ReasoningTokenShare, item.ReasoningOverheadRate = reasoningRates(item.ReasoningOutputTokens, item.VisibleOutputTokens, item.BillableOutputTokens)
	item.CacheMissRate = cacheMissRate(item.InputTokens, item.CachedInputTokens)
	item.FailurePressure = safeRateInt(item.FailedModelCalls+item.FailedToolCalls, item.SessionCount)
	item.ModelThroughputTokensPerSecond = throughputPerSecond(item.TotalTokens, item.ModelDurationMS)
	item.ModelThroughputOutputTokensPerSecond = throughputPerSecond(item.OutputTokens, item.ModelDurationMS)
	item.ModelLatencyMsPer1kOutputTokens = modelLatencyMSPer1kOutputTokens(item.OutputTokens, item.ModelDurationMS)
}

func applyModelSignalCostRates(item *model.ModelSignalsMetricSet) {
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
}

func (a modelSignalMetricAccumulator) applyPercentiles(item *model.ModelSignalsMetricSet) {
	item.P50ModelLatencyMsPer1kOutputTokens = percentileNearest(a.latencySamples, 0.50)
	item.P90ModelLatencyMsPer1kOutputTokens = percentileNearest(a.latencySamples, 0.90)
	item.P50ModelThroughputTokensPerSecond = percentileNearest(a.throughputSamples, 0.50)
	item.P10ModelThroughputTokensPerSecond = percentileNearest(a.throughputSamples, 0.10)
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
			Drift:         drift,
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

func sortModelSignalCohorts(cohorts []model.ModelSignalsCohort) {
	sort.Slice(cohorts, func(i, j int) bool {
		left := cohorts[i]
		right := cohorts[j]
		if order := compareModelSignalSeverityTokens(left.Drift.Severity, right.Drift.Severity, left.TotalTokens, right.TotalTokens); order != 0 {
			return order < 0
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
		if order := compareModelSignalSeverityTokens(left.Severity, right.Severity, left.TotalTokens, right.TotalTokens); order != 0 {
			return order < 0
		}
		if left.ModelProvider != right.ModelProvider {
			return left.ModelProvider < right.ModelProvider
		}
		return left.Model < right.Model
	})
}

func compareModelSignalSeverityTokens(leftSeverity, rightSeverity string, leftTokens, rightTokens int64) int {
	leftRank := modelSignalSeverityRank(leftSeverity)
	rightRank := modelSignalSeverityRank(rightSeverity)
	if leftRank != rightRank {
		if leftRank < rightRank {
			return -1
		}
		return 1
	}
	if leftTokens != rightTokens {
		if leftTokens > rightTokens {
			return -1
		}
		return 1
	}
	return 0
}

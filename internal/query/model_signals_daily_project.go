package query

import (
	"sort"
	"time"

	"github.com/LyleMi/AgentMeter/internal/model"
)

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

func sortModelSignalProjectHotspots(hotspots []model.ModelSignalsProjectHotspot) {
	sort.Slice(hotspots, func(i, j int) bool {
		left := hotspots[i]
		right := hotspots[j]
		if order := compareModelSignalSeverityTokens(left.Drift.Severity, right.Drift.Severity, left.TotalTokens, right.TotalTokens); order != 0 {
			return order < 0
		}
		return left.ProjectPath < right.ProjectPath
	})
}

func sortModelSignalProjectMetrics(projectMetrics []model.ModelSignalsProjectMetric) {
	sort.Slice(projectMetrics, func(i, j int) bool {
		left := projectMetrics[i]
		right := projectMetrics[j]
		if order := compareModelSignalSeverityTokens(left.Drift.Severity, right.Drift.Severity, left.TotalTokens, right.TotalTokens); order != 0 {
			return order < 0
		}
		return left.ProjectPath < right.ProjectPath
	})
}

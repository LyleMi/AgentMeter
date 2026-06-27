package viewmodel

import (
	"math"

	"AgentMeter/internal/model"
)

type Metric struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
	Note  string `json:"note"`
}

// DerivedOverviewMetrics mirrors the dashboard's lightweight derived signals.
func DerivedOverviewMetrics(overview model.Overview) []Metric {
	if overview.TotalSessions <= 0 {
		return nil
	}

	sessions := float64(overview.TotalSessions)
	inputTokens := math.Max(float64(overview.TotalInputTokens), 0)
	totalTokens := math.Max(float64(overview.TotalTokens), 0)
	activeHours := float64(overview.TotalActiveDurationMS) / 3_600_000
	hasCompletePricing := overview.EstimatedCostUSD != nil && overview.UnpricedSessions == 0

	costPerThousand := "unpriced"
	costNote := "Needs pricing for all sessions"
	if hasCompletePricing {
		costPerThousand = FormatCostPerThousand(overview.EstimatedCostUSD, overview.TotalTokens)
		costNote = "Uses complete pricing coverage"
	}

	tokensPerActiveHour := "-"
	if activeHours > 0 {
		tokensPerActiveHour = FormatNumber(int64(math.Round(totalTokens / activeHours)))
	}

	return []Metric{
		{
			Key:   "avgTokensPerSession",
			Label: "Avg tokens / session",
			Value: FormatNumber(int64(math.Round(totalTokens / sessions))),
			Note:  "Total tokens divided by sessions",
		},
		{
			Key:   "avgWallPerSession",
			Label: "Avg wall / session",
			Value: FormatDuration(float64(overview.TotalWallDurationMS) / sessions),
			Note:  "First to last timestamp",
		},
		{
			Key:   "activeShare",
			Label: "Active share",
			Value: FormatPercent(float64(overview.TotalActiveDurationMS) / math.Max(float64(overview.TotalWallDurationMS), 1)),
			Note:  "Measured model and tool time",
		},
		{
			Key:   "toolsPerSession",
			Label: "Tools / session",
			Value: FormatRatio(float64(overview.TotalToolCalls) / sessions),
			Note:  "Tool invocations per session",
		},
		{
			Key:   "cacheHitRate",
			Label: "Cache hit rate",
			Value: FormatPercent(float64(overview.TotalCachedInputTokens) / math.Max(inputTokens, 1)),
			Note:  "Cached input over input tokens",
		},
		{
			Key:   "outputInputRatio",
			Label: "Output / input",
			Value: FormatRatio(float64(overview.TotalOutputTokens)/math.Max(inputTokens, 1)) + "x",
			Note:  "Output token density",
		},
		{
			Key:   "costPerThousandTokens",
			Label: "Cost / 1K tokens",
			Value: costPerThousand,
			Note:  costNote,
		},
		{
			Key:   "tokensPerActiveHour",
			Label: "Tokens / active hour",
			Value: tokensPerActiveHour,
			Note:  "Token throughput during measured work",
		},
	}
}

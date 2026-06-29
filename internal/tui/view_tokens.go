package tui

import (
	"fmt"
	"sort"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) tokenViewportLines() []string {
	lines := tokenLines(s.tokens, s.breakdown, s.width, s.tokensTab, s.tokenBreakdownGroup)
	height := s.contentHeight()
	if s.scroll >= len(lines) {
		s.scroll = len(lines) - 1
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
	end := s.scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return lines[s.scroll:end]
}

func tokenLines(tokens agentmodel.TokenAnalytics, breakdown agentmodel.UsageBreakdown, width int, tab tokensTab, group string) []string {
	lines := []string{
		bold("Tokens"),
		tokenTabLine(tab, width),
	}
	if tokens.TotalSessions == 0 {
		return append(lines, "", "No token usage indexed yet. Press i to update the index.")
	}

	lines = appendTokenKpiLines(lines, tokens)
	switch tab {
	case tokensTabTrends:
		lines = appendTokenTrendLines(lines, tokens.CacheHitTrend, width)
	case tokensTabBreakdown:
		lines = appendTokenBreakdownLines(lines, tokens, breakdown, width, group)
	case tokensTabSessions:
		lines = appendTokenSessionLines(lines, tokens.HighTokenSessions, width)
	default:
		lines = appendTokenMixLines(lines, tokens)
		lines = appendSourceCacheLines(lines, tokens.AgentUsage, width)
	}
	return lines
}

func tokenTabLine(active tokensTab, width int) string {
	labels := make([]string, 0, len(tokensTabs))
	for _, tab := range tokensTabs {
		label := tab.title()
		if tab == active {
			label = inverse(" " + label + " ")
		}
		labels = append(labels, label)
	}
	return fit("Tabs: "+strings.Join(labels, "  "), width)
}

func appendTokenKpiLines(lines []string, tokens agentmodel.TokenAnalytics) []string {
	return append(lines, "",
		bold("Summary"),
		fmt.Sprintf("Total tokens: %-14s Sessions: %-10s Cache utilization: %-8s Cost: %s",
			formatInt(tokens.TotalTokens),
			formatInt(int64(tokens.TotalSessions)),
			formatPercent(tokens.CacheUtilizationRate),
			formatCost(tokens.EstimatedCostUSD),
		),
		fmt.Sprintf("Input: %-15s Cached input: %-12s Output: %-12s Reasoning: %s",
			formatInt(tokens.TotalInputTokens),
			formatInt(tokens.TotalCachedInputTokens),
			formatInt(tokens.TotalOutputTokens),
			formatInt(tokens.TotalReasoningTokens),
		),
		fmt.Sprintf("Context compression: %-12s Compression share: %s",
			formatInt(tokens.TotalContextCompressionTokens),
			formatPercent(ratio(float64(tokens.TotalContextCompressionTokens), float64(tokens.TotalTokens))),
		),
		fmt.Sprintf("Output/input: %-8sx Unpriced rows: %-8s Recent sessions: %-8s High-token sessions: %s",
			formatSignalRate(ratio(float64(tokens.TotalOutputTokens), float64(tokens.TotalInputTokens)), 2),
			formatInt(int64(tokens.UnpricedCount)),
			formatInt(int64(len(tokens.RecentSessions))),
			formatInt(int64(len(tokens.HighTokenSessions))),
		),
	)
}

func appendTokenMixLines(lines []string, tokens agentmodel.TokenAnalytics) []string {
	total := float64(tokens.TotalInputTokens + tokens.TotalCachedInputTokens + tokens.TotalOutputTokens + tokens.TotalReasoningTokens + tokens.TotalContextCompressionTokens)
	lines = append(lines, "", bold("Token Mix"))
	lines = append(lines,
		tokenMixLine("Input", tokens.TotalInputTokens, total),
		tokenMixLine("Cached input", tokens.TotalCachedInputTokens, total),
		tokenMixLine("Output", tokens.TotalOutputTokens, total),
		tokenMixLine("Reasoning overhead", tokens.TotalReasoningTokens, total),
		tokenMixLine("Context compression", tokens.TotalContextCompressionTokens, total),
	)
	return lines
}

func tokenMixLine(label string, value int64, total float64) string {
	return fmt.Sprintf("  %-22s %14s %8s", label, formatInt(value), formatPercent(ratio(float64(value), total)))
}

func appendSourceCacheLines(lines []string, rows []agentmodel.AgentUsage, width int) []string {
	rows = rankedAgentUsageByCache(rows)
	lines = append(lines, "", bold("Source Cache Hit Rate"))
	if len(rows) == 0 {
		return append(lines, "No source cache rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-18s %-25s %12s %12s %8s %10s",
		"Source", "Family/Path", "Input", "Cached", "Rate", "Tokens"), width))
	for _, row := range limitSlice(rows, 12) {
		rate := row.CacheUtilizationRate
		if rate == 0 && row.InputTokens > 0 {
			rate = ratio(float64(row.CachedInputTokens), float64(row.InputTokens))
		}
		lines = append(lines, fit(fmt.Sprintf("  %-18s %-25s %12s %12s %8s %10s",
			truncate(agentUsageSourceName(row), 18),
			truncate(agentUsageContext(row), 25),
			formatInt(row.InputTokens),
			formatInt(row.CachedInputTokens),
			formatPercent(rate),
			formatInt(row.TotalTokens),
		), width))
	}
	return lines
}

func appendTokenTrendLines(lines []string, rows []agentmodel.CacheHitTrendPoint, width int) []string {
	lines = append(lines, "", bold("Cache Hit Trend"))
	if len(rows) == 0 {
		return append(lines, "No cache trend rows.")
	}
	latest := latestCacheTrendPoint(rows)
	lowVolume := 0
	for _, row := range rows {
		if row.LowInputVolume {
			lowVolume++
		}
	}
	lines = append(lines,
		fmt.Sprintf("Latest hit rate: %-8s Date: %-10s Input: %-12s Rolling 7-day: %s",
			formatPercent(latest.CacheUtilizationRate),
			empty(latest.Date, "-"),
			formatInt(latest.InputTokens),
			formatPercent(latest.RollingCacheUtilizationRate),
		),
		fmt.Sprintf("Low-volume days: %s", formatInt(int64(lowVolume))),
		fit(fmt.Sprintf("  %-10s %9s %12s %12s %8s %8s %s",
			"Date", "Sessions", "Input", "Cached", "Hit", "Rolling", "Note"), width),
	)
	for _, row := range recentCacheTrendPoints(rows, 16) {
		note := ""
		if row.LowInputVolume {
			note = "low volume"
		}
		lines = append(lines, fit(fmt.Sprintf("  %-10s %9s %12s %12s %8s %8s %s",
			truncate(row.Date, 10),
			formatInt(int64(row.SessionCount)),
			formatInt(row.InputTokens),
			formatInt(row.CachedInputTokens),
			formatPercent(row.CacheUtilizationRate),
			formatPercent(row.RollingCacheUtilizationRate),
			note,
		), width))
	}
	return lines
}

func appendTokenBreakdownLines(lines []string, tokens agentmodel.TokenAnalytics, breakdown agentmodel.UsageBreakdown, width int, group string) []string {
	rows := tokenBreakdownRows(tokens, breakdown, group)
	lines = append(lines, "", bold("Usage Breakdown"), "Group: "+tokenBreakdownGroupTitle(group)+"  (press d to cycle)")
	if len(rows) == 0 {
		lines = append(lines, "No usage rows match the current scope.")
	} else {
		lines = appendTokenBreakdownRows(lines, rows, width, group, 16)
	}

	lines = append(lines, "", bold("Model Breakdown"))
	if len(tokens.ModelUsage) == 0 {
		lines = append(lines, "No model usage rows.")
	} else {
		lines = append(lines, fit(fmt.Sprintf("  %-26s %8s %12s %12s %12s %10s",
			"Model", "Sessions", "Tokens", "Cached", "Reasoning", "Cost"), width))
		for _, row := range limitSlice(tokens.ModelUsage, 8) {
			lines = append(lines, fit(fmt.Sprintf("  %-26s %8s %12s %12s %12s %10s",
				truncate(empty(row.Model, "unknown"), 26),
				formatInt(int64(row.SessionCount)),
				formatInt(row.TotalTokens),
				formatInt(row.CachedInputTokens),
				formatInt(row.ReasoningOutputTokens),
				formatCost(row.EstimatedCostUSD),
			), width))
		}
	}

	lines = append(lines, "", bold("Source Breakdown"))
	if len(tokens.AgentUsage) == 0 {
		return append(lines, "No source usage rows.")
	}
	lines = append(lines, fit(fmt.Sprintf("  %-18s %-25s %8s %12s %12s %8s %10s",
		"Source", "Family/Path", "Sessions", "Tokens", "Cached", "Tools", "Cost"), width))
	for _, row := range limitSlice(tokens.AgentUsage, 8) {
		lines = append(lines, fit(fmt.Sprintf("  %-18s %-25s %8s %12s %12s %8s %10s",
			truncate(agentUsageSourceName(row), 18),
			truncate(agentUsageContext(row), 25),
			formatInt(int64(row.SessionCount)),
			formatInt(row.TotalTokens),
			formatInt(row.CachedInputTokens),
			formatInt(int64(row.ToolCalls)),
			formatCost(row.EstimatedCostUSD),
		), width))
	}
	return lines
}

func appendTokenBreakdownRows(lines []string, rows []agentmodel.UsageBreakdownBucket, width int, group string, limit int) []string {
	lines = append(lines, fit(fmt.Sprintf("  %-30s %8s %12s %12s %12s %12s %12s %8s %10s",
		tokenBreakdownGroupTitle(group), "Sessions", "Tokens", "Input", "Cached", "Output", "Compress", "Cache", "Cost"), width))
	for _, row := range limitSlice(rows, limit) {
		lines = append(lines, fit(fmt.Sprintf("  %-30s %8s %12s %12s %12s %12s %12s %8s %10s",
			truncate(tokenBreakdownScope(row, group), 30),
			formatInt(int64(row.SessionCount)),
			formatInt(row.TotalTokens),
			formatInt(row.InputTokens),
			formatInt(row.CachedInputTokens),
			formatInt(row.OutputTokens),
			formatInt(row.ContextCompressionTokens),
			formatPercent(row.CacheUtilizationRate),
			formatCost(row.EstimatedCostUSD),
		), width))
	}
	return lines
}

func appendTokenSessionLines(lines []string, rows []agentmodel.Session, width int) []string {
	lines = append(lines, "", bold("High Token Sessions"))
	if len(rows) == 0 {
		return append(lines, "No high-token sessions.")
	}
	lines = append(lines, sessionHeader(width))
	for _, item := range limitSlice(rows, 16) {
		lines = append(lines, sessionRow(item, false, width))
	}
	return lines
}

func tokenBreakdownRows(tokens agentmodel.TokenAnalytics, breakdown agentmodel.UsageBreakdown, group string) []agentmodel.UsageBreakdownBucket {
	if group == tokenBreakdownGlobal {
		return []agentmodel.UsageBreakdownBucket{{
			SessionCount:             tokens.TotalSessions,
			TotalTokens:              tokens.TotalTokens,
			InputTokens:              tokens.TotalInputTokens,
			CachedInputTokens:        tokens.TotalCachedInputTokens,
			OutputTokens:             tokens.TotalOutputTokens,
			ReasoningOutputTokens:    tokens.TotalReasoningTokens,
			ContextCompressionTokens: tokens.TotalContextCompressionTokens,
			CacheUtilizationRate:     tokens.CacheUtilizationRate,
			EstimatedCostUSD:         tokens.EstimatedCostUSD,
			Unpriced:                 tokens.UnpricedCount > 0,
		}}
	}
	return breakdown.Buckets
}

func tokenBreakdownScope(row agentmodel.UsageBreakdownBucket, group string) string {
	switch group {
	case "agent":
		return sourceDisplayName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey)
	case "model":
		return empty(row.Model, "unknown")
	case "agent,model":
		source := sourceDisplayName(row.SourceLabel, row.AgentName, row.AgentKind, row.SourceKey)
		return source + " / " + empty(row.Model, "unknown")
	case "project":
		return shortPath(row.ProjectPath, 30)
	case "day":
		return empty(row.Date, "unknown")
	default:
		return "Global"
	}
}

func rankedAgentUsageByCache(rows []agentmodel.AgentUsage) []agentmodel.AgentUsage {
	result := append([]agentmodel.AgentUsage(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool {
		leftRate := result[i].CacheUtilizationRate
		if leftRate == 0 && result[i].InputTokens > 0 {
			leftRate = ratio(float64(result[i].CachedInputTokens), float64(result[i].InputTokens))
		}
		rightRate := result[j].CacheUtilizationRate
		if rightRate == 0 && result[j].InputTokens > 0 {
			rightRate = ratio(float64(result[j].CachedInputTokens), float64(result[j].InputTokens))
		}
		if leftRate == rightRate {
			if result[i].InputTokens == result[j].InputTokens {
				return agentUsageSourceName(result[i]) < agentUsageSourceName(result[j])
			}
			return result[i].InputTokens > result[j].InputTokens
		}
		return leftRate > rightRate
	})
	return result
}

func latestCacheTrendPoint(rows []agentmodel.CacheHitTrendPoint) agentmodel.CacheHitTrendPoint {
	for i := len(rows) - 1; i >= 0; i-- {
		if rows[i].HasUsage || rows[i].InputTokens > 0 {
			return rows[i]
		}
	}
	return rows[len(rows)-1]
}

func recentCacheTrendPoints(rows []agentmodel.CacheHitTrendPoint, limit int) []agentmodel.CacheHitTrendPoint {
	result := append([]agentmodel.CacheHitTrendPoint(nil), rows...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Date > result[j].Date })
	return limitSlice(result, limit)
}

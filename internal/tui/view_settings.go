package tui

import (
	"fmt"
	"strings"

	agentmodel "AgentMeter/internal/model"
)

func (s *state) settingsViewportLines() []string {
	lines := settingsLines(s.settings, s.width)
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

func settingsLines(settings agentmodel.Settings, width int) []string {
	lines := []string{
		bold("Settings"),
		"Database: " + empty(settings.DatabasePath, "unknown"),
		"",
		bold("Source Paths"),
	}
	if len(settings.SourceEntries) == 0 {
		lines = append(lines, "No source paths configured.")
	} else {
		for _, entry := range settings.SourceEntries {
			state := "disabled"
			if entry.Enabled {
				state = "enabled "
			}
			label := strings.TrimSpace(entry.Label)
			if label == "" {
				lines = append(lines, fmt.Sprintf("[%s] %s", state, entry.Path))
				continue
			}
			lines = append(lines, fmt.Sprintf("[%s] %s -> %s", state, label, entry.Path))
		}
	}
	lines = append(lines, "", bold("Last Index"))
	if settings.LastIndexStartedAt == nil {
		lines = append(lines, "No index run recorded.")
	} else {
		lines = append(lines, "Started: "+formatFullTime(*settings.LastIndexStartedAt))
	}
	if settings.LastIndexResult != nil {
		result := settings.LastIndexResult
		lines = append(lines, fmt.Sprintf("Files seen: %s  Indexed: %s  Skipped: %s  Failed: %s  Sessions: %s  Duration: %s",
			formatInt(int64(result.FilesSeen)),
			formatInt(int64(result.Indexed)),
			formatInt(int64(result.Skipped)),
			formatInt(int64(result.Failed)),
			formatInt(int64(result.Sessions)),
			formatDuration(result.DurationMS),
		))
		if len(result.Warnings) > 0 {
			lines = append(lines, "Warnings:")
			for _, warning := range result.Warnings {
				lines = append(lines, fit("- "+warning, width))
			}
		}
	}
	lines = append(lines, "", bold("Pricing Models"))
	if len(settings.PricingModels) == 0 {
		lines = append(lines, "No pricing models configured.")
		return lines
	}
	lines = append(lines, fmt.Sprintf("%-28s %12s %12s %12s", "Model", "Input/1M", "Cached/1M", "Output/1M"))
	for _, item := range settings.PricingModels {
		lines = append(lines, fmt.Sprintf("%-28s %12.4f %12.4f %12.4f",
			truncate(item.Model, 28),
			item.InputPer1M,
			item.CachedInputPer1M,
			item.OutputPer1M,
		))
	}
	return lines
}

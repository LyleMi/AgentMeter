package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) privacyViewportLines() []string {
	lines := privacySummaryLines(s.privacy, s.privacyTarget, s.privacyPending, s.width)
	status := s.selectedPrivacyStatus()
	if status == nil {
		return lines
	}
	detail := privacyDetailLines(*status, s.width)
	return append(lines, s.viewportLinesWithHeight(detail, s.privacyDetailHeight())...)
}

func (s *state) privacyLines() []string {
	return privacyLines(s.privacy, s.privacyTarget, s.privacyPending, s.width)
}

func privacyLines(statuses []agentmodel.PrivacyConfigStatus, selected int, pending *privacyProfileAction, width int) []string {
	lines := privacySummaryLines(statuses, selected, pending, width)
	if len(statuses) == 0 {
		return lines
	}
	if selected < 0 || selected >= len(statuses) {
		selected = 0
	}
	return append(lines, privacyDetailLines(statuses[selected], width)...)
}

func privacySummaryLines(statuses []agentmodel.PrivacyConfigStatus, selected int, pending *privacyProfileAction, width int) []string {
	lines := []string{bold("Agent Privacy")}
	if len(statuses) == 0 {
		return append(lines, "No privacy targets loaded.")
	}
	if selected < 0 || selected >= len(statuses) {
		selected = 0
	}
	selectedStatus := statuses[selected]
	lines = append(lines,
		"User-level config controls for supported agents. Profile writes require confirmation.",
		fmt.Sprintf("Selected: %s (%d/%d)  Score: %s  Next: Enter recommended, A strict, u defaults",
			privacyDisplayName(selectedStatus),
			selected+1,
			len(statuses),
			privacyScoreLabel(selectedStatus),
		),
	)
	if pending != nil {
		lines = append(lines, accent(fmt.Sprintf("Pending: apply %s profile to %s; Enter writes config, Esc cancels.", pending.profile, pending.targetName)))
	}
	lines = append(lines,
		"",
		bold("Targets"),
		fmt.Sprintf("  %-18s %-13s %7s %9s %4s %9s %s", "Agent", "State", "Score", "Attention", "Warn", "Config", "Path"),
	)
	for i, status := range statuses {
		summary := status.Summary
		prefix := "  "
		if i == selected {
			prefix = "> "
		}
		lines = append(lines, fit(fmt.Sprintf("%s%-18s %-13s %7s %9d %4d %9s %s",
			prefix,
			truncate(privacyDisplayName(status), 18),
			privacyStateLabel(status),
			fmt.Sprintf("%d%%", summary.Score),
			summary.Attention,
			len(status.Warnings),
			privacyConfigExistsLabel(status),
			shortPath(status.ConfigPath, width-72),
		), width))
	}
	return lines
}

func privacyDetailLines(status agentmodel.PrivacyConfigStatus, width int) []string {
	summary := status.Summary
	lines := []string{
		"",
		bold("Selected Target"),
		fmt.Sprintf("Target: %s  Config: %s  Safe: %d/%d (%d%%)  Hardened: %d  Default-safe: %d  Attention: %d",
			empty(status.Target, "unknown"),
			privacyConfigExistsLabel(status),
			summary.Hardened+summary.Implicit,
			summary.Total,
			summary.Score,
			summary.Hardened,
			summary.Implicit,
			summary.Attention,
		),
		"Path: " + empty(status.ConfigPath, "unknown"),
		"Profiles: " + strings.Join(privacyProfileNames(status), ", "),
	}
	if len(status.Warnings) > 0 {
		lines = append(lines, "", bold("Warnings"))
		for _, warning := range status.Warnings {
			lines = append(lines, fit("- "+warning, width))
		}
	}
	lines = append(lines, "", bold("Settings"))
	if len(status.Settings) == 0 {
		return append(lines, "No settings reported.")
	}
	profiles := privacyProfileNames(status)
	for _, setting := range status.Settings {
		lines = append(lines, fit(fmt.Sprintf("[%s] %-28s %-14s %s",
			empty(setting.Status, "unknown"),
			truncate(empty(setting.Title, setting.ID), 28),
			privacyConfigState(setting),
			empty(setting.Key, setting.ID),
		), width))
		profileParts := []string{"current=" + formatPrivacyValue(setting.CurrentValue)}
		for _, profile := range profiles {
			profileParts = append(profileParts, profile+"="+formatPrivacyValue(privacyProfileValue(setting, profile)))
		}
		valueLine := "    " + strings.Join(profileParts, "  ")
		if setting.Impact != "" {
			valueLine += "  impact=" + setting.Impact
		}
		lines = append(lines, fit(valueLine, width))
	}
	return lines
}

func (s *state) privacyDetailHeight() int {
	height := s.contentHeight() - len(privacySummaryLines(s.privacy, s.privacyTarget, s.privacyPending, s.width))
	if height < 1 {
		return 1
	}
	return height
}

func (s *state) privacyMaxScroll() int {
	status := s.selectedPrivacyStatus()
	if status == nil {
		return 0
	}
	max := len(privacyDetailLines(*status, s.width)) - s.privacyDetailHeight()
	if max < 0 {
		return 0
	}
	return max
}

func privacyStateLabel(status agentmodel.PrivacyConfigStatus) string {
	if status.Summary.Attention > 0 {
		return "needs review"
	}
	if len(status.Warnings) > 0 {
		return "warning"
	}
	if status.Summary.Total == 0 {
		return "no status"
	}
	return "ready"
}

func privacyScoreLabel(status agentmodel.PrivacyConfigStatus) string {
	summary := status.Summary
	if summary.Total == 0 {
		return "no status"
	}
	return fmt.Sprintf("%d/%d safe (%d%%)", summary.Hardened+summary.Implicit, summary.Total, summary.Score)
}

func privacyConfigExistsLabel(status agentmodel.PrivacyConfigStatus) string {
	if status.Exists {
		return "exists"
	}
	return "missing"
}

func privacyConfigState(setting agentmodel.PrivacyConfigSetting) string {
	if setting.Configured {
		return "configured"
	}
	if setting.Status == "implicit" {
		return "default-safe"
	}
	if !setting.CanApply {
		return "read-only"
	}
	return "not configured"
}

func privacyStrictValue(setting agentmodel.PrivacyConfigSetting) any {
	if setting.StrictValue != nil {
		return setting.StrictValue
	}
	return setting.DesiredValue
}

func privacyProfileValue(setting agentmodel.PrivacyConfigSetting, profile string) any {
	profile = strings.ToLower(strings.TrimSpace(profile))
	for _, value := range setting.ProfileValues {
		if strings.ToLower(strings.TrimSpace(value.Profile)) != profile {
			continue
		}
		if strings.EqualFold(value.Op, "set") {
			return value.Value
		}
		return nil
	}
	switch profile {
	case "default":
		return nil
	case "recommended":
		return setting.DesiredValue
	case "strict":
		return privacyStrictValue(setting)
	default:
		return nil
	}
}

func privacyProfileNames(status agentmodel.PrivacyConfigStatus) []string {
	names := make([]string, 0, len(status.Profiles))
	for _, profile := range status.Profiles {
		names = append(names, profile.ID)
	}
	if len(names) == 0 {
		for _, setting := range status.Settings {
			for _, value := range setting.ProfileValues {
				names = append(names, value.Profile)
			}
		}
	}
	if len(names) == 0 {
		names = []string{"default", "recommended", "strict"}
	}
	return orderProfileNames(names)
}

func orderProfileNames(names []string) []string {
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			seen[name] = true
		}
	}
	ordered := make([]string, 0, len(seen))
	for _, name := range []string{"default", "recommended", "strict"} {
		if seen[name] {
			ordered = append(ordered, name)
			delete(seen, name)
		}
	}
	extras := make([]string, 0, len(seen))
	for name := range seen {
		extras = append(extras, name)
	}
	sort.Strings(extras)
	return append(ordered, extras...)
}

func formatPrivacyValue(value any) string {
	if value == nil {
		return "unset"
	}
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return `""`
		}
		return typed
	case fmt.Stringer:
		return typed.String()
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(encoded)
}

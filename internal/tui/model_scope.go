package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) analyticsFilters() agentmodel.AnalyticsFilters {
	filters := agentmodel.AnalyticsFilters{
		Agent:   strings.TrimSpace(s.usageAgent),
		Model:   strings.TrimSpace(s.usageModel),
		Project: strings.TrimSpace(s.usageProject),
	}
	switch s.usageRange {
	case usageRangeDay:
		filters.StartedFrom = time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	case usageRangeWeek:
		filters.StartedFrom = time.Now().AddDate(0, 0, -7).UTC().Format(time.RFC3339)
	case usageRangeMonth:
		filters.StartedFrom = time.Now().AddDate(0, 0, -30).UTC().Format(time.RFC3339)
	}
	return filters
}

func (s *state) loadUsageScopeOptions(filters agentmodel.AnalyticsFilters, fallback agentmodel.Overview) (agentmodel.Overview, agentmodel.UsageBreakdown) {
	overview := fallback
	var projects agentmodel.UsageBreakdown
	if hasAnalyticsFilters(filters) {
		if value, err := s.service.GetOverview(); err == nil {
			overview = value
		}
	}
	if len(overview.AgentUsage) == 0 && len(overview.ModelUsage) == 0 && len(overview.RecentSessions) == 0 {
		if value, err := s.service.GetOverview(); err == nil {
			overview = value
		}
	}
	if value, err := s.service.GetUsageBreakdown("project", agentmodel.AnalyticsFilters{}); err == nil {
		projects = value
	}
	return overview, projects
}

func hasAnalyticsFilters(filters agentmodel.AnalyticsFilters) bool {
	return strings.TrimSpace(filters.Agent) != "" ||
		strings.TrimSpace(filters.Model) != "" ||
		strings.TrimSpace(filters.Project) != "" ||
		strings.TrimSpace(filters.StartedFrom) != "" ||
		strings.TrimSpace(filters.StartedTo) != ""
}

func (s *state) mergeScopeOptions(overview agentmodel.Overview, projects agentmodel.UsageBreakdown) {
	if len(overview.AgentUsage) > 0 || len(overview.ModelUsage) > 0 || len(overview.RecentSessions) > 0 || len(overview.SlowSessions) > 0 {
		s.scopeOverview = overview
	}
	if len(projects.Buckets) > 0 {
		s.scopeProjects = projects
	}
}

func (s *state) isUsageScopePage() bool {
	switch s.page {
	case pageOverview, pageTime, pageTokens, pageModelSignals, pageModelRisk:
		return true
	default:
		return false
	}
}

func (s *state) cycleUsageAgent() command {
	options := usageAgentOptions(s.scopeOverview)
	next, label := cycleStringOption(s.usageAgent, options)
	s.usageAgent = next
	s.selected = 0
	s.scroll = 0
	s.status = "source filter: " + label
	return s.load(s.page)
}

func (s *state) cycleUsageModel() command {
	options := usageModelOptions(s.scopeOverview)
	next, label := cycleStringOption(s.usageModel, options)
	s.usageModel = next
	s.selected = 0
	s.scroll = 0
	s.status = "model filter: " + label
	return s.load(s.page)
}

func (s *state) cycleUsageProject() command {
	options := usageProjectOptions(s.scopeProjects, s.scopeOverview)
	next, label := cycleStringOption(s.usageProject, options)
	s.usageProject = next
	s.selected = 0
	s.scroll = 0
	s.status = "project filter: " + label
	return s.load(s.page)
}

func (s *state) cycleUsageRange() command {
	s.usageRange = cycleTab(s.usageRange, usageRanges, 1)
	s.selected = 0
	s.scroll = 0
	s.status = "range filter: " + s.usageRange.title()
	return s.load(s.page)
}

func (s *state) clearUsageScope() command {
	if s.usageAgent == "" && s.usageModel == "" && s.usageProject == "" && s.usageRange == usageRangeAll {
		s.status = "usage scope already clear"
		return nil
	}
	s.usageAgent = ""
	s.usageModel = ""
	s.usageProject = ""
	s.usageRange = usageRangeAll
	s.selected = 0
	s.scroll = 0
	s.status = "usage scope cleared"
	return s.load(s.page)
}

type stringOption struct {
	value string
	label string
}

func cycleStringOption(current string, options []stringOption) (string, string) {
	if len(options) == 0 {
		return "", "All"
	}
	all := append([]stringOption{{label: "All"}}, options...)
	index := 0
	for i, option := range all {
		if option.value == current {
			index = i
			break
		}
	}
	next := all[(index+1)%len(all)]
	return next.value, next.label
}

func usageAgentOptions(overview agentmodel.Overview) []stringOption {
	seen := map[string]string{}
	for _, row := range overview.AgentUsage {
		addSourceOption(seen, row.SourceID, row.SourceKey, row.SourceLabel, row.AgentKind, row.AgentName, row.SourceRootPath, row.SourceSessionsPath)
	}
	for _, session := range append(append([]agentmodel.Session{}, overview.RecentSessions...), overview.SlowSessions...) {
		addSourceOption(seen, session.SourceID, session.SourceKey, session.SourceLabel, session.AgentKind, session.AgentName, session.SourceRootPath, session.SourceSessionsPath)
	}
	return sortedStringOptions(seen)
}

func usageModelOptions(overview agentmodel.Overview) []stringOption {
	seen := map[string]string{}
	for _, row := range overview.ModelUsage {
		addValueOption(seen, row.Model, empty(row.Model, "unknown"))
	}
	for _, session := range append(append([]agentmodel.Session{}, overview.RecentSessions...), overview.SlowSessions...) {
		addValueOption(seen, session.Model, empty(session.Model, "unknown"))
	}
	return sortedStringOptions(seen)
}

func usageProjectOptions(projects agentmodel.UsageBreakdown, overview agentmodel.Overview) []stringOption {
	seen := map[string]string{}
	for _, row := range projects.Buckets {
		addValueOption(seen, row.ProjectPath, shortPath(row.ProjectPath, 36))
	}
	for _, session := range append(append([]agentmodel.Session{}, overview.RecentSessions...), overview.SlowSessions...) {
		addValueOption(seen, session.ProjectPath, shortPath(session.ProjectPath, 36))
	}
	return sortedStringOptions(seen)
}

func addSourceOption(seen map[string]string, sourceID int64, sourceKey, sourceLabel, agentKind, agentName, rootPath, sessionsPath string) {
	value := strings.TrimSpace(sourceKey)
	if value == "" && sourceID > 0 {
		value = fmt.Sprintf("source:%d", sourceID)
	}
	if value == "" {
		value = strings.TrimSpace(agentKind)
	}
	if value == "" {
		value = strings.TrimSpace(agentName)
	}
	if value == "" {
		return
	}
	label := sourceDisplayName(sourceLabel, agentName, agentKind, sourceKey)
	context := sourceContext(agentKind, agentName, rootPath, sessionsPath)
	if context != "" && context != label {
		label += " (" + context + ")"
	}
	seen[value] = label
}

func addValueOption(seen map[string]string, value, label string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	label = strings.TrimSpace(label)
	if label == "" {
		label = value
	}
	seen[value] = label
}

func sortedStringOptions(seen map[string]string) []stringOption {
	options := make([]stringOption, 0, len(seen))
	for value, label := range seen {
		options = append(options, stringOption{value: value, label: label})
	}
	sort.SliceStable(options, func(i, j int) bool {
		if options[i].label == options[j].label {
			return options[i].value < options[j].value
		}
		return options[i].label < options[j].label
	})
	return options
}

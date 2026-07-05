package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

type fakeService struct {
	overview  agentmodel.Overview
	tokens    agentmodel.TokenAnalytics
	breakdown map[string]agentmodel.UsageBreakdown
	signals   agentmodel.ModelSignals
	sessions  []agentmodel.Session
	detail    agentmodel.SessionDetail
	tools     []agentmodel.ToolStat
	toolCalls []agentmodel.ToolCall
	audit     agentmodel.AuditSummary
	findings  []agentmodel.AuditFinding
	settings  agentmodel.Settings
	privacy   []agentmodel.PrivacyConfigStatus
	index     agentmodel.IndexResult

	indexCalls          []bool
	overviewFilters     []agentmodel.AnalyticsFilters
	tokenFilters        []agentmodel.AnalyticsFilters
	breakdownFilters    []breakdownFilterCall
	signalFilters       []agentmodel.AnalyticsFilters
	toolFilters         []agentmodel.ToolFilters
	toolCallFilters     []agentmodel.ToolCallFilters
	auditSummaryFilters []agentmodel.AuditFindingFilters
	auditFindingFilters []agentmodel.AuditFindingFilters
	privacyApply        agentmodel.PrivacyConfigApplyResult
	privacyApplyErr     error
	privacyApplyCall    []privacyApplyCall
}

type breakdownFilterCall struct {
	groupBy string
	filters agentmodel.AnalyticsFilters
}

type privacyApplyCall struct {
	target  string
	profile string
}

func (f *fakeService) GetOverview() (agentmodel.Overview, error) {
	return f.overview, nil
}

func (f *fakeService) GetOverviewWithFilters(filters agentmodel.AnalyticsFilters) (agentmodel.Overview, error) {
	f.overviewFilters = append(f.overviewFilters, filters)
	return f.overview, nil
}

func (f *fakeService) GetTokenAnalyticsWithFilters(filters agentmodel.AnalyticsFilters) (agentmodel.TokenAnalytics, error) {
	f.tokenFilters = append(f.tokenFilters, filters)
	return f.tokens, nil
}

func (f *fakeService) GetUsageBreakdown(groupBy string, filters agentmodel.AnalyticsFilters) (agentmodel.UsageBreakdown, error) {
	f.breakdownFilters = append(f.breakdownFilters, breakdownFilterCall{groupBy: groupBy, filters: filters})
	if f.breakdown != nil {
		if value, ok := f.breakdown[groupBy]; ok {
			return value, nil
		}
	}
	return agentmodel.UsageBreakdown{GroupBy: groupBy, Buckets: []agentmodel.UsageBreakdownBucket{}}, nil
}

func (f *fakeService) GetModelSignalsWithFilters(filters agentmodel.AnalyticsFilters) (agentmodel.ModelSignals, error) {
	f.signalFilters = append(f.signalFilters, filters)
	return f.signals, nil
}

func (f *fakeService) ListSessions(_ agentmodel.SessionFilters) ([]agentmodel.Session, error) {
	return f.sessions, nil
}

func (f *fakeService) GetSessionDetail(_ int64) (agentmodel.SessionDetail, error) {
	return f.detail, nil
}

func (f *fakeService) ListTools(filters agentmodel.ToolFilters) ([]agentmodel.ToolStat, error) {
	f.toolFilters = append(f.toolFilters, filters)
	if strings.TrimSpace(filters.Agent) == "" {
		return f.tools, nil
	}
	stats := map[string]*agentmodel.ToolStat{}
	for _, call := range f.toolCalls {
		if !toolCallMatchesAgent(call, filters.Agent) {
			continue
		}
		toolName := call.ToolName
		if toolName == "" {
			toolName = "unknown"
		}
		stat := stats[toolName]
		if stat == nil {
			stat = &agentmodel.ToolStat{ToolName: toolName}
			stats[toolName] = stat
		}
		stat.Calls++
		switch strings.ToLower(strings.TrimSpace(call.Status)) {
		case "completed", "success":
			stat.SuccessCalls++
		default:
			stat.FailedCalls++
		}
		stat.TotalDurationMS += call.DurationMS
	}
	result := make([]agentmodel.ToolStat, 0, len(stats))
	for _, stat := range stats {
		if stat.Calls > 0 {
			stat.AvgDurationMS = float64(stat.TotalDurationMS) / float64(stat.Calls)
		}
		result = append(result, *stat)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Calls == result[j].Calls {
			return result[i].ToolName < result[j].ToolName
		}
		return result[i].Calls > result[j].Calls
	})
	return result, nil
}

func (f *fakeService) ListToolCalls(filters agentmodel.ToolCallFilters) ([]agentmodel.ToolCall, error) {
	f.toolCallFilters = append(f.toolCallFilters, filters)
	result := make([]agentmodel.ToolCall, 0, len(f.toolCalls))
	for _, call := range f.toolCalls {
		if strings.TrimSpace(filters.ToolName) != "" && call.ToolName != filters.ToolName {
			continue
		}
		if filters.Shell && !isShellToolName(call.ToolName) {
			continue
		}
		if filters.RiskOnly && call.RiskCount <= 0 {
			continue
		}
		if strings.TrimSpace(filters.Agent) != "" && !toolCallMatchesAgent(call, filters.Agent) {
			continue
		}
		if strings.TrimSpace(filters.StartedFrom) != "" && call.StartedAt.Before(parseTestTime(filters.StartedFrom)) {
			continue
		}
		result = append(result, call)
	}
	switch filters.Sort {
	case "duration_desc":
		sort.Slice(result, func(i, j int) bool {
			if result[i].DurationMS == result[j].DurationMS {
				return result[i].StartedAt.After(result[j].StartedAt)
			}
			return result[i].DurationMS > result[j].DurationMS
		})
	case "duration_asc":
		sort.Slice(result, func(i, j int) bool {
			if result[i].DurationMS == result[j].DurationMS {
				return result[i].StartedAt.After(result[j].StartedAt)
			}
			return result[i].DurationMS < result[j].DurationMS
		})
	case "risk_desc":
		sort.Slice(result, func(i, j int) bool {
			left, right := result[i].RiskScore, result[j].RiskScore
			if left == 0 {
				left = 1
			}
			if right == 0 {
				right = 1
			}
			if left == right {
				return result[i].StartedAt.After(result[j].StartedAt)
			}
			return left > right
		})
	case "risk_asc":
		sort.Slice(result, func(i, j int) bool {
			left, right := result[i].RiskScore, result[j].RiskScore
			if left == 0 {
				left = 1
			}
			if right == 0 {
				right = 1
			}
			if left == right {
				return result[i].StartedAt.After(result[j].StartedAt)
			}
			return left < right
		})
	default:
		sort.Slice(result, func(i, j int) bool {
			return result[i].StartedAt.After(result[j].StartedAt)
		})
	}
	if filters.Offset > 0 && filters.Offset < len(result) {
		result = result[filters.Offset:]
	}
	if filters.Offset >= len(result) {
		result = []agentmodel.ToolCall{}
	}
	if filters.Limit > 0 && filters.Limit < len(result) {
		result = result[:filters.Limit]
	}
	return result, nil
}

func toolCallMatchesAgent(call agentmodel.ToolCall, filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return true
	}
	values := []string{
		call.SourceKey,
		call.SourceLabel,
		call.AgentKind,
		call.AgentName,
	}
	if call.SourceID > 0 {
		values = append(values, fmt.Sprintf("source:%d", call.SourceID))
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == filter {
			return true
		}
	}
	return false
}

func parseTestTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func (f *fakeService) GetAuditSummaryWithFilters(filters agentmodel.AuditFindingFilters) (agentmodel.AuditSummary, error) {
	f.auditSummaryFilters = append(f.auditSummaryFilters, filters)
	return f.audit, nil
}

func (f *fakeService) ListAuditFindings(filters agentmodel.AuditFindingFilters) ([]agentmodel.AuditFinding, error) {
	f.auditFindingFilters = append(f.auditFindingFilters, filters)
	result := make([]agentmodel.AuditFinding, 0, len(f.findings))
	search := strings.ToLower(strings.TrimSpace(filters.Search))
	for _, finding := range f.findings {
		if strings.TrimSpace(filters.Agent) != "" && !auditFindingMatchesAgent(finding, filters.Agent) {
			continue
		}
		if filters.Category != "" && !strings.EqualFold(finding.Category, filters.Category) {
			continue
		}
		if filters.Severity != "" && !strings.EqualFold(finding.Severity, filters.Severity) {
			continue
		}
		if filters.ShellFamily != "" && !strings.EqualFold(finding.ShellFamily, filters.ShellFamily) {
			continue
		}
		if search != "" {
			haystack := strings.ToLower(strings.Join([]string{finding.Title, finding.Command, finding.Evidence, finding.ProjectPath}, " "))
			if !strings.Contains(haystack, search) {
				continue
			}
		}
		result = append(result, finding)
	}
	if filters.Offset > 0 && filters.Offset < len(result) {
		result = result[filters.Offset:]
	}
	if filters.Offset >= len(result) {
		result = []agentmodel.AuditFinding{}
	}
	if filters.Limit > 0 && filters.Limit < len(result) {
		result = result[:filters.Limit]
	}
	return result, nil
}

func (f *fakeService) GetAuditFinding(id int64) (agentmodel.AuditFinding, error) {
	for _, finding := range f.findings {
		if finding.ID == id {
			return finding, nil
		}
	}
	return agentmodel.AuditFinding{}, fmt.Errorf("audit finding %d not found", id)
}

func (f *fakeService) GetSettings() (agentmodel.Settings, error) {
	return f.settings, nil
}

func (f *fakeService) GetPrivacyConfigs() ([]agentmodel.PrivacyConfigStatus, error) {
	return f.privacy, nil
}

func (f *fakeService) IndexNow(rebuild bool) (agentmodel.IndexResult, error) {
	f.indexCalls = append(f.indexCalls, rebuild)
	return f.index, nil
}

func (f *fakeService) ApplyPrivacyProfile(target, profile string) (agentmodel.PrivacyConfigApplyResult, error) {
	f.privacyApplyCall = append(f.privacyApplyCall, privacyApplyCall{target: target, profile: profile})
	if f.privacyApplyErr != nil {
		return agentmodel.PrivacyConfigApplyResult{}, f.privacyApplyErr
	}
	if f.privacyApply.Status.Target == "" {
		for _, status := range f.privacy {
			if status.Target == target {
				result := f.privacyApply
				result.Status = status
				return result, nil
			}
		}
	}
	return f.privacyApply, nil
}

func TestOverviewLoadsAndRenders(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 32)

	cmd := st.init()
	_, quit := st.update(runCommand(t, cmd))
	if quit {
		t.Fatal("unexpected quit")
	}

	view := st.view()
	assertContains(t, view, "Overview")
	assertContains(t, view, "Sessions: 2")
	assertContains(t, view, "gpt-5-codex")
	assertContains(t, view, "Work Codex")
	assertContains(t, view, "codex @")
	assertContains(t, view, "Recent Sessions")
}

func TestUsageScopeFiltersAreAppliedAcrossAnalyticsPages(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 150, 40)

	cmd := st.init()
	st.update(runCommand(t, cmd))

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'u'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAnalyticsFilter(t, svc.overviewFilters).Agent; got != "source:7" {
		t.Fatalf("overview agent filter = %q, want source:7", got)
	}
	assertContains(t, st.view(), "scope source Work Codex")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'v'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAnalyticsFilter(t, svc.overviewFilters).Model; got != "gpt-5-codex" {
		t.Fatalf("overview model filter = %q, want gpt-5-codex", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'w'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAnalyticsFilter(t, svc.overviewFilters).Project; got != `D:\tools\custom\AgentMeter` {
		t.Fatalf("overview project filter = %q, want project path", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'e'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAnalyticsFilter(t, svc.overviewFilters).StartedFrom; got == "" {
		t.Fatal("overview range filter StartedFrom is empty")
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: '3'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	tokenFilter := lastAnalyticsFilter(t, svc.tokenFilters)
	if tokenFilter.Agent != "source:7" || tokenFilter.Model != "gpt-5-codex" || tokenFilter.Project == "" || tokenFilter.StartedFrom == "" {
		t.Fatalf("token filters = %+v, want inherited usage scope", tokenFilter)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: '4'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	signalFilter := lastAnalyticsFilter(t, svc.signalFilters)
	if signalFilter.Agent != "source:7" || signalFilter.Model != "gpt-5-codex" || signalFilter.Project == "" || signalFilter.StartedFrom == "" {
		t.Fatalf("signal filters = %+v, want inherited usage scope", signalFilter)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'U'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	cleared := lastAnalyticsFilter(t, svc.signalFilters)
	if cleared.Agent != "" || cleared.Model != "" || cleared.Project != "" || cleared.StartedFrom != "" {
		t.Fatalf("cleared signal filters = %+v, want empty", cleared)
	}
}

func TestSessionsOpenDetail(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 40)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 's'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Session Detail")
	assertContains(t, view, "Source: Work Codex  Family: codex  Agent: Codex")
	assertContains(t, view, `Source root: D:\sessions\codex-work`)
	assertContains(t, view, `Raw source: D:\sessions\codex-work\sessions\2026\06\27\session-42.jsonl`)
	assertContains(t, view, "Tool Calls")
	assertContains(t, view, "shell_command")
	assertContains(t, view, "loaded repository")
}

func TestIndexNowRefreshesCurrentPage(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 24)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'g'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'i'})
	if quit {
		t.Fatal("unexpected quit")
	}
	if !st.indexing {
		t.Fatal("expected indexing state")
	}
	msg := runCommand(t, cmd)
	cmd, quit = st.update(msg)
	if quit {
		t.Fatal("unexpected quit")
	}
	if len(svc.indexCalls) != 1 || svc.indexCalls[0] {
		t.Fatalf("index calls = %v, want one non-rebuild call", svc.indexCalls)
	}
	if !strings.Contains(st.status, "index complete") {
		t.Fatalf("status = %q, want index completion", st.status)
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Settings")
	assertContains(t, view, "Database:")
	assertContains(t, view, `Work Codex -> D:\sessions\codex-work`)
	assertContains(t, view, "Indexed: 3")
}

func TestToolsOpenToolCallsAndDetail(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 120, 30)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 't'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Tools")
	assertContains(t, view, "shell_command")

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageToolCalls {
		t.Fatalf("page = %v, want tool calls", st.page)
	}
	if len(svc.toolCallFilters) == 0 || svc.toolCallFilters[len(svc.toolCallFilters)-1].ToolName != "shell_command" {
		t.Fatalf("tool call filters = %+v, want shell_command filter", svc.toolCallFilters)
	}
	view = st.view()
	assertContains(t, view, "Tool Calls")
	assertContains(t, view, "Scope: shell_command")
	assertContains(t, view, "rg --files")
	assertContains(t, view, "Work Codex")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'd'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := svc.toolCallFilters[len(svc.toolCallFilters)-1].Sort; got != "duration_desc" {
		t.Fatalf("sort = %q, want duration_desc", got)
	}
	assertContains(t, st.view(), "Sort: duration high to low")

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("tool call detail should not load")
	}
	view = st.view()
	assertContains(t, view, "Tool Call Detail")
	assertContains(t, view, "Tool: shell_command")
	assertContains(t, view, "Project: D:\\tools\\custom\\AgentMeter")
	assertContains(t, view, "Raw source: D:\\sessions\\codex-work\\sessions\\2026\\06\\27\\session-42.jsonl")
	assertContains(t, view, "Input")
	assertContains(t, view, "rg --files")
	assertContains(t, view, "Output")
	assertContains(t, view, "repository file list")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'b'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageToolCalls {
		t.Fatalf("page = %v, want tool calls after back", st.page)
	}
}

func TestToolsTabsAndFiltersRender(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 140, 45)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 't'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Tools")
	assertContains(t, view, "Activity Summary")
	assertContains(t, view, "Top Tools")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	view = st.view()
	assertContains(t, view, "Tool Summary")
	assertContains(t, view, "shell_command")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	view = st.view()
	assertContains(t, view, "Shell Commands")
	assertContains(t, view, "shell_command")
	assertNotContains(t, view, "web_fetch")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'd'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastToolCallFilter(t, svc.toolCallFilters).Sort; got != "duration_desc" {
		t.Fatalf("tool sort filter = %q, want duration_desc", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'v'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	assertContains(t, st.view(), "command rg")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'u'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastToolFilter(t, svc.toolFilters).Agent; got != "source:7" {
		t.Fatalf("tool summary source filter = %q, want source:7", got)
	}
	if got := lastToolCallFilter(t, svc.toolCallFilters).Agent; got != "source:7" {
		t.Fatalf("tool source filter = %q, want source:7", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'e'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastToolCallFilter(t, svc.toolCallFilters).StartedFrom; got == "" {
		t.Fatal("tool range filter StartedFrom is empty")
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'U'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	cleared := lastToolCallFilter(t, svc.toolCallFilters)
	if cleared.Agent != "" || cleared.StartedFrom != "" {
		t.Fatalf("cleared tool filters = %+v, want empty agent/range", cleared)
	}
}

func TestToolsShellTabUsesSharedShellToolCallQuery(t *testing.T) {
	svc := sampleService()
	base := time.Date(2026, 6, 27, 9, 30, 0, 0, time.UTC)
	shellCall := svc.toolCalls[0]
	shellCall.StartedAt = base
	shellCall.EndedAt = base.Add(250 * time.Millisecond)
	svc.toolCalls = []agentmodel.ToolCall{shellCall}
	for i := 0; i < 600; i++ {
		call := svc.toolCalls[0]
		call.ID = int64(1000 + i)
		call.ToolName = "web_fetch"
		call.InputSummary = fmt.Sprintf("https://example.test/%d", i)
		call.StartedAt = base.Add(time.Duration(i+1) * time.Minute)
		call.EndedAt = call.StartedAt.Add(500 * time.Millisecond)
		call.DurationMS = 500
		svc.toolCalls = append(svc.toolCalls, call)
	}

	st := newState(svc, 140, 35)
	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 't'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Shell Commands")
	assertContains(t, view, "shell_command")
	assertContains(t, view, "rg --files")
	filters := lastToolCallFilter(t, svc.toolCallFilters)
	if !filters.Shell || !filters.IncludeRisk || filters.ToolName != "" {
		t.Fatalf("tool call filters = %+v, want shared shell risk query", filters)
	}
}

func TestToolsTabDetailReturnsToOriginTab(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 140, 35)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 't'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	for i := 0; i < 2; i++ {
		cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
		if quit {
			t.Fatal("unexpected quit")
		}
		st.update(runCommand(t, cmd))
	}
	if st.toolsTab != toolsTabShell {
		t.Fatalf("tools tab = %v, want shell", st.toolsTab)
	}

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("tool detail open from tools tab should not load")
	}
	if st.page != pageToolCallDetail {
		t.Fatalf("page = %v, want tool call detail", st.page)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'b'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageTools || st.toolsTab != toolsTabShell {
		t.Fatalf("page/tab = %v/%v, want tools/shell", st.page, st.toolsTab)
	}
}

func TestInvokedToolCommandMatchesStructuredShellInput(t *testing.T) {
	call := agentmodel.ToolCall{
		ToolName:          "shell_command",
		InputSummary:      "fallback should not win",
		RawStartEventJSON: `{"payload":{"type":"function_call","name":"shell_command","arguments":"{\"command\":\"cd src && sudo -u agent rg --files\"}"}}`,
	}
	if got := invokedToolCommand(call); got != "rg" {
		t.Fatalf("invoked command = %q, want rg", got)
	}
}

func TestModelSignalsLoadsAndRenders(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 180, 60)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'm'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Model Signals")
	assertContains(t, view, "Health:")
	assertContains(t, view, "warning")
	assertContains(t, view, "Metric Explorer")
	assertContains(t, view, "P90 latency")
	assertContains(t, view, "Cache savings")
	assertContains(t, view, "Failure pressure")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("model signal tab switch returned a command")
	}
	view = st.view()
	assertContains(t, view, "Model Breakdown")
	assertContains(t, view, "gpt-5-codex")
	assertContains(t, view, "Top Drift Cohorts")
	assertContains(t, view, "Tool failures above baseline")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Daily Efficiency")
	assertContains(t, view, "Cost/H")
	assertContains(t, view, "P90/P50")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Source Model Matrix")
	assertContains(t, view, "RiskLvl")
	assertContains(t, view, "gpt-5-codex")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Project Hotspots")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Anomaly Sessions")
	assertContains(t, view, "Output")
	assertContains(t, view, "Started")
}

func TestModelSignalTabNavigationWrapsAndResetsScroll(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 24)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'm'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	st.scroll = 5
	cmd, quit = st.update(keyMsg{typ: keyRune, ch: '['})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("tab switch returned a command")
	}
	if st.modelSignalsTab != modelSignalsTabAnomalies {
		t.Fatalf("tab = %s, want Anomalies", st.modelSignalsTab.title())
	}
	if st.scroll != 0 {
		t.Fatalf("scroll = %d, want reset to 0", st.scroll)
	}

	st.update(keyMsg{typ: keyRune, ch: 'h'})
	if st.modelSignalsTab != modelSignalsTabProjects {
		t.Fatalf("tab = %s, want Projects", st.modelSignalsTab.title())
	}
	st.update(keyMsg{typ: keyRune, ch: 'l'})
	if st.modelSignalsTab != modelSignalsTabAnomalies {
		t.Fatalf("tab = %s, want Anomalies", st.modelSignalsTab.title())
	}
	st.update(keyMsg{typ: keyRune, ch: ']'})
	if st.modelSignalsTab != modelSignalsTabCharts {
		t.Fatalf("tab = %s, want Charts", st.modelSignalsTab.title())
	}
}

func TestModelSignalsNarrowWidthViewsRender(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 80, 24)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'm'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	for _, want := range []string{
		"Metric Explorer",
		"Health Overview",
		"Daily Efficiency",
		"Top Drift Cohorts",
		"Source Model Matrix",
		"Project Hotspots",
		"Anomaly Sessions",
	} {
		assertContains(t, st.view(), want)
		st.update(keyMsg{typ: keyRune, ch: ']'})
	}
}

func TestTimePageTabsRender(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 140, 50)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '2'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Time")
	assertContains(t, view, "Composition")
	assertContains(t, view, "Source Time Attribution")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Source Time Comparison")
	assertContains(t, view, "Work Codex")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Tool Duration Leaders")
	assertContains(t, view, "web_fetch")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Slow Sessions")
	assertContains(t, view, "gpt-5-codex")
}

func TestTokensPageTabsAndBreakdownGroupRender(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 150, 60)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '3'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Tokens")
	assertContains(t, view, "Token Mix")
	assertContains(t, view, "Source Cache Hit Rate")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Cache Hit Trend")
	assertContains(t, view, "Latest hit rate")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "Usage Breakdown")
	assertContains(t, view, "Group: Global")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'd'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	view = st.view()
	assertContains(t, view, "Group: Source")

	for i := 0; i < 3; i++ {
		cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'd'})
		if quit {
			t.Fatal("unexpected quit")
		}
		st.update(runCommand(t, cmd))
	}
	view = st.view()
	assertContains(t, view, "Group: Project")
	assertContains(t, view, "AgentMeter")

	st.update(keyMsg{typ: keyRune, ch: ']'})
	view = st.view()
	assertContains(t, view, "High Token Sessions")
	assertContains(t, view, "12,345")
}

func TestModelRiskLoadsAndRenders(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 160, 40)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '5'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Model Risk")
	assertContains(t, view, "Highest risk")
	assertContains(t, view, "Score Drivers")
	assertContains(t, view, "Risk Explanations")
	assertContains(t, view, "gpt-5-codex")
}

func TestModelRiskReasonMatchesWebPriority(t *testing.T) {
	rows := buildModelRiskRows(agentmodel.ModelSignals{
		Matrix: []agentmodel.ModelSignalsMatrixRow{{
			SourceKey:   "source:7",
			SourceLabel: "Work Codex",
			Cells: []agentmodel.ModelSignalsMatrixCell{{
				ModelProvider: "openai",
				Model:         "gpt-5-codex",
				KeyReason:     "key reason should be fallback",
				Current: agentmodel.ModelSignalsMetricSet{
					SessionCount:         2,
					ModelCalls:           3,
					TotalTokens:          1000,
					FailurePressure:      0.5,
					DegradationRiskScore: 0.5,
				},
				Drift: agentmodel.ModelSignalsDrift{
					Reasons: []string{"drift reason first"},
				},
			}},
		}},
	})
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if rows[0].Reason != "drift reason first" {
		t.Fatalf("reason = %q, want drift reason first", rows[0].Reason)
	}
}

func TestAuditSummaryFindingsAndDetailRender(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 150, 55)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '8'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Audit")
	assertContains(t, view, "Recent Findings")
	assertContains(t, view, "shell.suspicious")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'f'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageAuditFindings {
		t.Fatalf("page = %v, want audit findings", st.page)
	}
	view = st.view()
	assertContains(t, view, "Audit Findings")
	assertContains(t, view, "shell.suspicious")

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageAuditDetail {
		t.Fatalf("page = %v, want audit detail", st.page)
	}
	view = st.view()
	assertContains(t, view, "Finding Detail")
	assertContains(t, view, "rg --files")
	assertContains(t, view, "Linked Session")
	assertContains(t, view, "Session Tool Calls")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'b'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageAuditFindings {
		t.Fatalf("page = %v, want audit findings after back", st.page)
	}
}

func TestAuditFiltersAreApplied(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 150, 40)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '8'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'u'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAuditSummaryFilter(t, svc.auditSummaryFilters).Agent; got != "source:7" {
		t.Fatalf("audit summary agent filter = %q, want source:7", got)
	}
	assertContains(t, st.view(), "filters source Work Codex")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'f'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAuditFindingFilter(t, svc.auditFindingFilters).Agent; got != "source:7" {
		t.Fatalf("audit finding agent filter = %q, want source:7", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'c'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAuditFindingFilter(t, svc.auditFindingFilters).Category; got != "command" {
		t.Fatalf("audit category filter = %q, want command", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'v'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAuditFindingFilter(t, svc.auditFindingFilters).Severity; got != "critical" {
		t.Fatalf("audit severity filter = %q, want critical", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'y'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if got := lastAuditFindingFilter(t, svc.auditFindingFilters).ShellFamily; got != "posix" {
		t.Fatalf("audit shell filter = %q, want posix", got)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'U'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	cleared := lastAuditFindingFilter(t, svc.auditFindingFilters)
	if cleared.Agent != "" || cleared.Category != "" || cleared.Severity != "" || cleared.ShellFamily != "" {
		t.Fatalf("cleared audit filters = %+v, want empty", cleared)
	}
}

func TestAuditDetailReloadsWhenSourceFilterChanges(t *testing.T) {
	svc := sampleService()
	svc.overview.AgentUsage = append([]agentmodel.AgentUsage{{
		SourceID:       99,
		SourceKey:      "source:99",
		SourceLabel:    "Archive Agent",
		AgentKind:      "codex",
		AgentName:      "Archive",
		SourceRootPath: `D:\sessions\archive`,
	}}, svc.overview.AgentUsage...)
	st := newState(svc, 150, 45)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '8'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageAuditDetail || st.finding == nil {
		t.Fatalf("page/finding = %v/%v, want audit detail with finding", st.page, st.finding)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'u'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageAuditDetail {
		t.Fatalf("page = %v, want audit detail", st.page)
	}
	if st.err == nil || !strings.Contains(st.err.Error(), "does not match source filter") {
		t.Fatalf("err = %v, want source filter mismatch", st.err)
	}
}

func TestToolsAndToolCallsKeepSelectedRowVisible(t *testing.T) {
	svc := sampleService()
	started := time.Date(2026, 6, 27, 9, 30, 0, 0, time.Local)
	svc.tools = nil
	svc.toolCalls = nil
	for i := 0; i < 8; i++ {
		toolName := fmt.Sprintf("tool-%02d", i)
		svc.tools = append(svc.tools, agentmodel.ToolStat{ToolName: toolName, Calls: 1})
		svc.toolCalls = append(svc.toolCalls, agentmodel.ToolCall{
			ID:           int64(i + 1),
			SessionID:    42,
			ToolName:     toolName,
			Status:       "completed",
			StartedAt:    started.Add(time.Duration(i) * time.Minute),
			DurationMS:   int64(100 + i),
			InputSummary: "call " + toolName,
			SourceLabel:  "Work Codex",
		})
	}
	st := newState(svc, 100, 12)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 't'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	cmd, quit = st.update(keyMsg{typ: keyPageDown})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("page down returned a command on tools")
	}
	selectedTool := st.tools[st.selected].ToolName
	assertSelectedLineContains(t, st.view(), selectedTool)

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'c'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	cmd, quit = st.update(keyMsg{typ: keyPageDown})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("page down returned a command on tool calls")
	}
	selectedCall := st.toolCalls[st.selected].ToolName
	assertSelectedLineContains(t, st.view(), selectedCall)
}

func TestPrivacyPageLoadsAndRenders(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 70)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'p'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	view := st.view()
	assertContains(t, view, "Agent Privacy")
	assertContains(t, view, "Selected: Codex (1/4)")
	assertContains(t, view, "Next: Enter recommended, A strict, u defaults")
	assertContains(t, view, "Codex")
	assertContains(t, view, "Claude Code")
	assertContains(t, view, "CodeBuddy")
	assertContains(t, view, "Target: codex  Config: exists  Safe: 2/3 (66%)")
	assertContains(t, view, "[attention] Web search")
	assertContains(t, view, "default-safe")
	assertContains(t, view, "default=unset")
	assertContains(t, view, "recommended=false")
	assertContains(t, view, "Codex warning")

	cmd, quit = st.update(keyMsg{typ: keyDown})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("target selection returned a command")
	}
	view = st.view()
	assertContains(t, view, "Selected: Gemini CLI (2/4)")
	assertContains(t, view, `strict=["web_search","web_fetch"]`)
	assertContains(t, view, "Broken JSON")
	assertContains(t, view, "read-only")

	cmd, quit = st.update(keyMsg{typ: keyTab})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageSettings {
		t.Fatalf("page = %v, want settings after tab from privacy", st.page)
	}

	cmd, quit = st.update(keyMsg{typ: keyShiftTab})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pagePrivacy {
		t.Fatalf("page = %v, want privacy after shift-tab from settings", st.page)
	}
}

func TestPrivacyProfileRequiresConfirmationBeforeApply(t *testing.T) {
	svc := sampleService()
	svc.privacyApply = agentmodel.PrivacyConfigApplyResult{
		Status: agentmodel.PrivacyConfigStatus{
			Target: "gemini",
			Name:   "Gemini CLI",
			Exists: true,
			Summary: agentmodel.PrivacyConfigSummary{
				Score:    100,
				Total:    1,
				Hardened: 1,
			},
			Settings: []agentmodel.PrivacyConfigSetting{
				{ID: "tools.exclude.web", Title: "Web tools", Key: "tools.exclude", CurrentValue: []string{"web_search", "web_fetch"}, DesiredValue: []string{"web_fetch"}, StrictValue: []string{"web_search", "web_fetch"}, Configured: true, CanApply: true, Status: "hardened"},
			},
		},
		Changed: []agentmodel.PrivacyConfigChange{{
			ID:     "tools.exclude.web",
			Key:    "tools.exclude",
			Before: []string{"web_fetch"},
			After:  []string{"web_search", "web_fetch"},
		}},
		Warnings: []string{"restart Gemini CLI to pick up changes"},
	}
	st := newState(svc, 100, 60)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: 'p'})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: ']'})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("target selection returned a command")
	}
	assertContains(t, st.view(), "Selected: Gemini CLI (2/4)")

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'a'})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("profile selection should not apply immediately")
	}
	if len(svc.privacyApplyCall) != 0 {
		t.Fatalf("profile apply calls = %v, want none before confirmation", svc.privacyApplyCall)
	}
	assertContains(t, st.view(), "Pending: apply recommended profile to Gemini CLI")

	cmd, quit = st.update(keyMsg{typ: keyEsc})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("cancel returned a command")
	}
	if len(svc.privacyApplyCall) != 0 {
		t.Fatalf("profile apply calls = %v, want none after cancel", svc.privacyApplyCall)
	}

	cmd, quit = st.update(keyMsg{typ: keyRune, ch: 'A'})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("strict profile selection should not apply immediately")
	}

	cmd, quit = st.update(keyMsg{typ: keyEnter})
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd == nil {
		t.Fatal("confirm did not return apply command")
	}
	if !st.privacyApplying {
		t.Fatal("expected applying state after confirmation")
	}
	secondCmd, secondQuit := st.update(keyMsg{typ: keyRune, ch: 'a'})
	if secondQuit {
		t.Fatal("unexpected quit while applying")
	}
	if secondCmd != nil {
		t.Fatal("profile key returned a second command while apply was running")
	}
	if st.privacyPending != nil {
		t.Fatal("profile key queued a second pending action while apply was running")
	}
	assertContains(t, st.status, "privacy profile already applying")
	msg := runCommand(t, cmd)
	cmd, quit = st.update(msg)
	if quit {
		t.Fatal("unexpected quit")
	}
	if cmd != nil {
		t.Fatal("apply result returned unexpected command")
	}
	if len(svc.privacyApplyCall) != 1 {
		t.Fatalf("profile apply calls = %v, want one", svc.privacyApplyCall)
	}
	if got := svc.privacyApplyCall[0]; got.target != "gemini" || got.profile != "strict" {
		t.Fatalf("profile apply call = %+v, want gemini strict", got)
	}
	if st.privacyPending != nil {
		t.Fatal("pending profile was not cleared")
	}
	assertContains(t, st.status, "applied strict profile to Gemini CLI")
	assertContains(t, st.status, "1 change")
	assertContains(t, st.status, "1 warning(s): restart Gemini CLI to pick up changes")
	assertContains(t, st.view(), "Selected: Gemini CLI (2/4)")
	assertContains(t, st.view(), "Safe: 1/1 (100%)")
}

func runCommand(t *testing.T, cmd command) message {
	t.Helper()
	if cmd == nil {
		t.Fatal("nil command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ch := make(chan message, 1)
	cmd(ctx, ch)
	select {
	case msg := <-ch:
		return msg
	case <-ctx.Done():
		t.Fatal("command did not send a message")
	}
	return nil
}

func sampleService() *fakeService {
	started := time.Date(2026, 6, 27, 9, 30, 0, 0, time.Local)
	cost := 0.0123
	session := agentmodel.Session{
		ID:                 42,
		SourceID:           7,
		SourceKey:          "source:7",
		SourceLabel:        "Work Codex",
		SourceRootPath:     `D:\sessions\codex-work`,
		SourceSessionsPath: `D:\sessions\codex-work\sessions`,
		AgentKind:          "codex",
		AgentName:          "Codex",
		SessionKey:         "session-42",
		ProjectPath:        `D:\tools\custom\AgentMeter`,
		Model:              "gpt-5-codex",
		StartedAt:          started,
		EndedAt:            started.Add(15 * time.Minute),
		WallDurationMS:     900000,
		ActiveDurationMS:   600000,
		ModelDurationMS:    300000,
		ToolDurationMS:     300000,
		IdleDurationMS:     300000,
		TokenUsage: agentmodel.Usage{
			TotalTokens:           12345,
			InputTokens:           8000,
			CachedInputTokens:     1200,
			OutputTokens:          4345,
			ReasoningOutputTokens: 745,
		},
		EstimatedCostUSD: &cost,
		ToolCallCount:    2,
		RawSourcePath:    `D:\sessions\codex-work\sessions\2026\06\27\session-42.jsonl`,
	}
	toolCall := agentmodel.ToolCall{
		ID:                   99,
		SessionID:            session.ID,
		SourceID:             session.SourceID,
		SourceKey:            session.SourceKey,
		SourceLabel:          session.SourceLabel,
		SourceRootPath:       session.SourceRootPath,
		SourceSessionsPath:   session.SourceSessionsPath,
		StartedAt:            started.Add(2 * time.Minute),
		EndedAt:              started.Add(2*time.Minute + 250*time.Millisecond),
		DurationMS:           250,
		ToolName:             "shell_command",
		Status:               "completed",
		InputSummary:         "rg --files",
		OutputSummary:        "repository file list",
		CallID:               "call-shell-99",
		RawEventID:           1001,
		RawStartEventID:      1001,
		RawEndEventID:        1002,
		RawEventLine:         17,
		RawStartEventLine:    17,
		RawEndEventLine:      18,
		RawStartEventType:    "tool_call",
		RawEndEventType:      "tool_result",
		RawStartEventSummary: "started shell command",
		RawEndEventSummary:   "completed shell command",
		SessionKey:           session.SessionKey,
		CodexSessionID:       session.CodexSessionID,
		ProjectPath:          session.ProjectPath,
		AgentKind:            session.AgentKind,
		AgentName:            session.AgentName,
		RawSourcePath:        session.RawSourcePath,
	}
	webCall := agentmodel.ToolCall{
		ID:                 100,
		SessionID:          session.ID,
		SourceID:           session.SourceID,
		SourceKey:          session.SourceKey,
		SourceLabel:        session.SourceLabel,
		SourceRootPath:     session.SourceRootPath,
		SourceSessionsPath: session.SourceSessionsPath,
		StartedAt:          started.Add(3 * time.Minute),
		EndedAt:            started.Add(3*time.Minute + 500*time.Millisecond),
		DurationMS:         500,
		ToolName:           "web_fetch",
		Status:             "failed",
		InputSummary:       "https://example.test",
		Error:              "network disabled",
		SessionKey:         session.SessionKey,
		ProjectPath:        session.ProjectPath,
		AgentKind:          session.AgentKind,
		AgentName:          session.AgentName,
		RawSourcePath:      session.RawSourcePath,
	}
	index := agentmodel.IndexResult{FilesSeen: 4, Indexed: 3, Skipped: 1, Sessions: 2, DurationMS: 1200}
	auditFinding := agentmodel.AuditFinding{
		ID:                 501,
		SessionID:          session.ID,
		SourceID:           session.SourceID,
		SourceKey:          session.SourceKey,
		SourceLabel:        session.SourceLabel,
		SourceRootPath:     session.SourceRootPath,
		SourceSessionsPath: session.SourceSessionsPath,
		ToolCallID:         toolCall.ID,
		RawEventID:         toolCall.RawEventID,
		SourceLine:         17,
		Timestamp:          started.Add(2 * time.Minute),
		Source:             "offline",
		EventType:          "tool_call",
		Category:           "command",
		Severity:           "high",
		RuleID:             "shell.suspicious",
		Title:              "Suspicious shell command",
		Description:        "Shell command should be reviewed",
		Evidence:           "rg --files listed repository paths",
		Command:            "rg --files",
		ShellFamily:        "powershell",
		Platform:           "windows",
		Decision:           "observed",
		CreatedAt:          started.Add(2 * time.Minute),
		SessionKey:         session.SessionKey,
		ProjectPath:        session.ProjectPath,
		AgentKind:          session.AgentKind,
		AgentName:          session.AgentName,
		RawSourcePath:      session.RawSourcePath,
	}
	return &fakeService{
		overview: agentmodel.Overview{
			TotalSessions:                  2,
			TotalTokens:                    12345,
			TotalInputTokens:               8000,
			TotalCachedInputTokens:         1200,
			TotalOutputTokens:              4345,
			TotalReasoningTokens:           745,
			EstimatedCostUSD:               &cost,
			TotalWallDurationMS:            900000,
			TotalActiveDurationMS:          600000,
			TotalModelDurationMS:           300000,
			TotalToolDurationMS:            300000,
			TotalIdleDurationMS:            300000,
			TotalToolCalls:                 2,
			SuspectedNetworkToolDurationMS: 500,
			SuspectedNetworkToolCalls:      1,
			DailyUsage: []agentmodel.DailyUsage{{
				Date:                 "2026-06-27",
				SessionCount:         2,
				TotalTokens:          12345,
				InputTokens:          8000,
				CachedInputTokens:    1200,
				OutputTokens:         4345,
				ToolCalls:            2,
				CacheUtilizationRate: 0.15,
				EstimatedCostUSD:     &cost,
			}},
			CacheHitTrend: []agentmodel.CacheHitTrendPoint{{
				Date:                        "2026-06-27",
				SessionCount:                2,
				TotalTokens:                 12345,
				InputTokens:                 8000,
				CachedInputTokens:           1200,
				CacheUtilizationRate:        0.15,
				RollingCacheUtilizationRate: 0.15,
				LowInputVolume:              false,
				HasUsage:                    true,
			}},
			ModelUsage: []agentmodel.ModelUsage{
				{Model: "gpt-5-codex", SessionCount: 2, TotalTokens: 12345, InputTokens: 8000, CachedInputTokens: 1200, OutputTokens: 4345, ReasoningOutputTokens: 745, EstimatedCostUSD: &cost},
			},
			AgentUsage: []agentmodel.AgentUsage{
				{SourceID: 7, SourceKey: "source:7", SourceLabel: "Work Codex", SourceRootPath: `D:\sessions\codex-work`, SourceSessionsPath: `D:\sessions\codex-work\sessions`, AgentKind: "codex", AgentName: "Codex", SessionCount: 2, TotalTokens: 12345, InputTokens: 8000, CachedInputTokens: 1200, OutputTokens: 4345, ReasoningOutputTokens: 745, CacheUtilizationRate: 0.15, ToolCalls: 2, EstimatedCostUSD: &cost},
			},
			ToolTimeLeaders: []agentmodel.ToolTimeUsage{
				{ToolName: "web_fetch", Calls: 1, FailedCalls: 1, TotalDurationMS: 500, AvgDurationMS: 500, MaxDurationMS: 500, SuspectedNetwork: true},
				{ToolName: "shell_command", Calls: 2, SuccessCalls: 2, TotalDurationMS: 500, AvgDurationMS: 250, MaxDurationMS: 250},
			},
			AgentTimeUsage: []agentmodel.AgentTimeUsage{
				{SourceID: 7, SourceKey: "source:7", SourceLabel: "Work Codex", SourceRootPath: `D:\sessions\codex-work`, SourceSessionsPath: `D:\sessions\codex-work\sessions`, AgentKind: "codex", AgentName: "Codex", SessionCount: 2, ToolCalls: 2, WallDurationMS: 900000, ActiveDurationMS: 600000, ModelDurationMS: 300000, ToolDurationMS: 300000, IdleDurationMS: 300000, SuspectedNetworkToolDurationMS: 500},
			},
			ModelTimeUsage: []agentmodel.ModelTimeUsage{
				{Model: "gpt-5-codex", SessionCount: 2, TotalTokens: 12345, WallDurationMS: 900000, ActiveDurationMS: 600000, ModelDurationMS: 300000, ToolDurationMS: 300000, IdleDurationMS: 300000},
			},
			RecentSessions: []agentmodel.Session{session},
			SlowSessions:   []agentmodel.Session{session},
		},
		tokens: agentmodel.TokenAnalytics{
			TotalSessions:          2,
			TotalInputTokens:       8000,
			TotalCachedInputTokens: 1200,
			TotalOutputTokens:      4345,
			TotalReasoningTokens:   745,
			TotalTokens:            12345,
			CacheUtilizationRate:   0.15,
			EstimatedCostUSD:       &cost,
			CacheHitTrend: []agentmodel.CacheHitTrendPoint{{
				Date:                        "2026-06-27",
				SessionCount:                2,
				TotalTokens:                 12345,
				InputTokens:                 8000,
				CachedInputTokens:           1200,
				CacheUtilizationRate:        0.15,
				RollingCacheUtilizationRate: 0.15,
				HasUsage:                    true,
			}},
			ModelUsage: []agentmodel.ModelUsage{
				{Model: "gpt-5-codex", SessionCount: 2, TotalTokens: 12345, InputTokens: 8000, CachedInputTokens: 1200, OutputTokens: 4345, ReasoningOutputTokens: 745, EstimatedCostUSD: &cost},
			},
			AgentUsage: []agentmodel.AgentUsage{
				{SourceID: 7, SourceKey: "source:7", SourceLabel: "Work Codex", SourceRootPath: `D:\sessions\codex-work`, SourceSessionsPath: `D:\sessions\codex-work\sessions`, AgentKind: "codex", AgentName: "Codex", SessionCount: 2, TotalTokens: 12345, InputTokens: 8000, CachedInputTokens: 1200, OutputTokens: 4345, ReasoningOutputTokens: 745, CacheUtilizationRate: 0.15, ToolCalls: 2, EstimatedCostUSD: &cost},
			},
			RecentSessions:    []agentmodel.Session{session},
			HighTokenSessions: []agentmodel.Session{session},
		},
		breakdown: map[string]agentmodel.UsageBreakdown{
			"project": {
				GroupBy: "project",
				Buckets: []agentmodel.UsageBreakdownBucket{{
					ProjectPath:           session.ProjectPath,
					SessionCount:          2,
					TotalTokens:           12345,
					InputTokens:           8000,
					CachedInputTokens:     1200,
					OutputTokens:          4345,
					ReasoningOutputTokens: 745,
					CacheUtilizationRate:  0.15,
					EstimatedCostUSD:      &cost,
				}},
			},
		},
		signals: agentmodel.ModelSignals{
			TotalSessions:                        2,
			TotalModelCalls:                      3,
			TotalToolCalls:                       2,
			FailedToolCalls:                      1,
			ToolFailureRate:                      0.5,
			ToolDependencyRate:                   1,
			AvgModelCallsPerSession:              1.5,
			OutputExpansionRate:                  0.54,
			ReasoningTokenShare:                  0.18,
			ReasoningOverheadRate:                0.22,
			VisibleOutputTokens:                  3600,
			BillableOutputTokens:                 4345,
			CacheMissRate:                        0.72,
			ModelThroughputTokensPerSecond:       18.4,
			ModelThroughputOutputTokensPerSecond: 6.5,
			HealthSummary: agentmodel.ModelSignalsHealthSummary{
				CurrentWindow:        agentmodel.ModelSignalsWindow{From: "2026-06-27", To: "2026-06-27", SessionCount: 2, ModelCalls: 3},
				BaselineWindow:       agentmodel.ModelSignalsWindow{From: "2026-05-28", To: "2026-06-26", SessionCount: 12, ModelCalls: 20},
				Severity:             "warning",
				CohortCount:          1,
				WarningCohorts:       1,
				CriticalCohorts:      0,
				LowConfidenceCohorts: 0,
				TopReasons:           []string{"Tool failures above baseline"},
			},
			ModelBreakdown: []agentmodel.ModelSignalsBreakdown{{
				Model:                          "gpt-5-codex",
				SessionCount:                   2,
				ModelCalls:                     3,
				ToolCalls:                      2,
				FailedToolCalls:                1,
				TotalTokens:                    12345,
				OutputTokens:                   4345,
				ReasoningOutputTokens:          745,
				ToolFailureRate:                0.5,
				ReasoningTokenShare:            0.18,
				CacheMissRate:                  0.72,
				ModelThroughputTokensPerSecond: 18.4,
			}},
			Cohorts: []agentmodel.ModelSignalsCohort{{
				SourceID:      7,
				SourceKey:     "source:7",
				SourceLabel:   "Work Codex",
				AgentKind:     "codex",
				AgentName:     "Codex",
				ModelProvider: "openai",
				Model:         "gpt-5-codex",
				ProjectPath:   session.ProjectPath,
				CohortKey:     "openai:gpt-5-codex:codex:AgentMeter",
				ModelSignalsMetricSet: agentmodel.ModelSignalsMetricSet{
					SessionCount:                         2,
					ModelCalls:                           3,
					FailedToolCalls:                      1,
					TotalTokens:                          12345,
					ModelThroughputTokensPerSecond:       18.4,
					ModelLatencyMsPer1kOutputTokens:      2100,
					P90ModelLatencyMsPer1kOutputTokens:   2400,
					P10ModelThroughputTokensPerSecond:    11.2,
					ToolFailureRate:                      0.5,
					FailurePressure:                      0.5,
					DegradationRiskScore:                 0.43,
					ModelThroughputOutputTokensPerSecond: 6.5,
				},
				Current: agentmodel.ModelSignalsMetricSet{
					SessionCount:                       2,
					ModelCalls:                         3,
					FailedToolCalls:                    1,
					TotalTokens:                        12345,
					ModelThroughputTokensPerSecond:     18.4,
					P90ModelLatencyMsPer1kOutputTokens: 2400,
					P10ModelThroughputTokensPerSecond:  11.2,
					ToolFailureRate:                    0.5,
					FailurePressure:                    0.5,
					DegradationRiskScore:               0.43,
				},
				Baseline: agentmodel.ModelSignalsMetricSet{
					SessionCount:                   12,
					ModelCalls:                     20,
					ModelThroughputTokensPerSecond: 30,
				},
				Drift: agentmodel.ModelSignalsDrift{
					Severity:   "warning",
					Confidence: "medium",
					Reasons:    []string{"Tool failures above baseline"},
				},
			}},
			DailyMetrics: []agentmodel.ModelSignalsDailyMetric{{
				Date:                  "2026-06-27",
				ModelSignalsMetricSet: agentmodel.ModelSignalsMetricSet{SessionCount: 2, ModelCalls: 3, EstimatedCostUSD: &cost, CostPerSession: &cost, P90ModelLatencyMsPer1kOutputTokens: 2400, P10ModelThroughputTokensPerSecond: 11.2, DegradationRiskScore: 0.43},
				LowSample:             true,
				KeyReason:             "low sample",
				Drift:                 agentmodel.ModelSignalsDrift{Severity: "watch", Confidence: "low"},
				Baseline:              agentmodel.ModelSignalsMetricSet{SessionCount: 7},
			}},
			ProjectMetrics: []agentmodel.ModelSignalsProjectMetric{{
				ProjectPath:           session.ProjectPath,
				ModelCount:            1,
				SourceCount:           1,
				DominantModelProvider: "openai",
				DominantModel:         "gpt-5-codex",
				DominantModelShare:    1,
				ModelSignalsMetricSet: agentmodel.ModelSignalsMetricSet{SessionCount: 2, TotalTokens: 12345},
				Current:               agentmodel.ModelSignalsMetricSet{SessionCount: 2, EstimatedCostUSD: &cost, P90ModelLatencyMsPer1kOutputTokens: 2400, P10ModelThroughputTokensPerSecond: 11.2, DegradationRiskScore: 0.43},
				Baseline:              agentmodel.ModelSignalsMetricSet{SessionCount: 12},
				Drift:                 agentmodel.ModelSignalsDrift{Severity: "warning", Confidence: "medium", Reasons: []string{"Tool failures above baseline"}},
			}},
			Matrix: []agentmodel.ModelSignalsMatrixRow{{
				SourceID:    7,
				SourceKey:   "source:7",
				SourceLabel: "Work Codex",
				AgentKind:   "codex",
				AgentName:   "Codex",
				Cells: []agentmodel.ModelSignalsMatrixCell{{
					ModelProvider: "openai",
					Model:         "gpt-5-codex",
					CohortCount:   1,
					Severity:      "warning",
					Confidence:    "medium",
					KeyReason:     "Tool failures above baseline",
					SessionCount:  2,
					ModelCalls:    3,
					TotalTokens:   12345,
					Current: agentmodel.ModelSignalsMetricSet{
						SessionCount:                         2,
						ModelCalls:                           3,
						FailedToolCalls:                      1,
						TotalTokens:                          12345,
						P90ModelLatencyMsPer1kOutputTokens:   2400,
						P10ModelThroughputTokensPerSecond:    11.2,
						ModelThroughputOutputTokensPerSecond: 6.5,
						ToolFailureRate:                      0.5,
						CacheMissRate:                        0.72,
						AvgModelCallsPerSession:              1.5,
						OutputExpansionRate:                  0.54,
						ReasoningOverheadRate:                0.22,
						FailurePressure:                      0.5,
						DegradationRiskScore:                 0.43,
					},
					Baseline: agentmodel.ModelSignalsMetricSet{
						SessionCount:                   12,
						ModelCalls:                     20,
						ModelThroughputTokensPerSecond: 30,
					},
					Drift: agentmodel.ModelSignalsDrift{
						Severity:   "warning",
						Confidence: "medium",
						Reasons:    []string{"Tool failures above baseline"},
					},
				}},
			}},
			AnomalySessions: []agentmodel.ModelSignalsAnomalySession{{
				SessionID:                      session.ID,
				SourceID:                       session.SourceID,
				SourceKey:                      session.SourceKey,
				SourceLabel:                    session.SourceLabel,
				AgentKind:                      session.AgentKind,
				AgentName:                      session.AgentName,
				SessionKey:                     session.SessionKey,
				ProjectPath:                    session.ProjectPath,
				Model:                          session.Model,
				StartedAt:                      started,
				TotalTokens:                    12345,
				FailedToolCalls:                1,
				ReasoningTokenShare:            0.18,
				CacheMissRate:                  0.72,
				ModelThroughputTokensPerSecond: 18.4,
				ReasonLabels:                   []string{"tool failures", "cache miss"},
				Score:                          0.43,
			}},
		},
		sessions: []agentmodel.Session{session},
		detail: agentmodel.SessionDetail{
			Session: session,
			Events: []agentmodel.Event{
				{Timestamp: started, Kind: "message", Summary: "loaded repository"},
			},
			ModelCalls: []agentmodel.ModelCall{
				{StartedAt: started, Model: "gpt-5-codex", DurationMS: 3000, TotalTokens: 12345, CostUSD: &cost},
			},
			ToolCalls: []agentmodel.ToolCall{
				toolCall,
			},
		},
		tools: []agentmodel.ToolStat{
			{ToolName: "shell_command", Calls: 2, SuccessCalls: 2, TotalDurationMS: 500, AvgDurationMS: 250},
			{ToolName: "web_fetch", Calls: 1, FailedCalls: 1, TotalDurationMS: 500, AvgDurationMS: 500},
		},
		toolCalls: []agentmodel.ToolCall{toolCall, webCall},
		audit: agentmodel.AuditSummary{
			TotalFindings:        1,
			HighFindings:         1,
			CommandFindings:      1,
			SessionsWithFindings: 1,
			RecentFindings:       []agentmodel.AuditFinding{auditFinding},
		},
		findings: []agentmodel.AuditFinding{auditFinding},
		settings: agentmodel.Settings{
			DatabasePath:    `D:\tools\custom\AgentMeter\agentmeter.sqlite`,
			SourceEntries:   []agentmodel.SourceEntry{{Path: `D:\sessions\codex-work`, Label: "Work Codex", Enabled: true}},
			LastIndexResult: &index,
			PricingModels:   []agentmodel.PricingModel{{Model: "gpt-5-codex", InputPer1M: 1.25, CachedInputPer1M: 0.25, OutputPer1M: 10}},
			LastIndexStartedAt: func() *time.Time {
				value := started
				return &value
			}(),
		},
		privacy: []agentmodel.PrivacyConfigStatus{{
			Target:     "codex",
			Name:       "Codex",
			ConfigPath: `C:\Users\agent\.codex\config.toml`,
			Exists:     true,
			Summary: agentmodel.PrivacyConfigSummary{
				Score:     66,
				Total:     3,
				Hardened:  1,
				Implicit:  1,
				Attention: 1,
			},
			Settings: []agentmodel.PrivacyConfigSetting{
				{ID: "analytics.enabled", Title: "Analytics", Key: "analytics.enabled", DesiredValue: false, StrictValue: false, CurrentValue: false, Configured: true, CanApply: true, Status: "hardened", Impact: "limits telemetry"},
				{ID: "history.persistence", Title: "Conversation history", Key: "history.persistence", DesiredValue: "none", StrictValue: "none", SupportsUnset: true, CanApply: true, Status: "implicit"},
				{ID: "web_search", Title: "Web search", Key: "tools.web_search", DesiredValue: false, StrictValue: false, CurrentValue: true, Configured: true, CanApply: true, Status: "attention"},
			},
			Warnings: []string{"Codex warning"},
		}, {
			Target:     "gemini",
			Name:       "Gemini CLI",
			ConfigPath: `C:\Users\agent\.gemini\settings.json`,
			Exists:     false,
			Summary: agentmodel.PrivacyConfigSummary{
				Score:     33,
				Total:     3,
				Hardened:  1,
				Attention: 2,
			},
			Settings: []agentmodel.PrivacyConfigSetting{
				{ID: "privacy.usageStatisticsEnabled", Title: "Usage statistics", Key: "privacy.usageStatisticsEnabled", DesiredValue: false, StrictValue: false, CurrentValue: false, Configured: true, CanApply: true, Status: "hardened"},
				{ID: "tools.exclude.web", Title: "Web tools", Key: "tools.exclude", DesiredValue: []string{"web_fetch"}, StrictValue: []string{"web_search", "web_fetch"}, SupportsUnset: true, CanApply: true, Status: "attention"},
				{ID: "settings.parse", Title: "Broken JSON", Key: "settings.json", DesiredValue: nil, StrictValue: nil, CanApply: false, Status: "attention"},
			},
		}, {
			Target:     "claude",
			Name:       "Claude Code",
			ConfigPath: `C:\Users\agent\.claude\settings.json`,
			Exists:     true,
			Summary: agentmodel.PrivacyConfigSummary{
				Score:     100,
				Total:     1,
				Hardened:  1,
				Attention: 0,
			},
			Settings: []agentmodel.PrivacyConfigSetting{
				{ID: "env.DISABLE_TELEMETRY", Title: "Telemetry", Key: "env.DISABLE_TELEMETRY", DesiredValue: "1", StrictValue: "1", CurrentValue: "1", Configured: true, CanApply: true, Status: "hardened"},
			},
		}, {
			Target:     "codebuddy",
			Name:       "CodeBuddy",
			ConfigPath: `C:\Users\agent\.codebuddy\settings.json`,
			Exists:     true,
			Summary: agentmodel.PrivacyConfigSummary{
				Score:     100,
				Total:     1,
				Implicit:  1,
				Attention: 0,
			},
			Settings: []agentmodel.PrivacyConfigSetting{
				{ID: "env.OTEL_TRACES_EXPORTER", Title: "OTel exporter", Key: "env.OTEL_TRACES_EXPORTER", DesiredValue: "none", StrictValue: "none", CanApply: true, Status: "implicit"},
			},
		}},
		index: index,
	}
}

func assertContains(t *testing.T, value, want string) {
	t.Helper()
	if !strings.Contains(value, want) {
		t.Fatalf("view does not contain %q:\n%s", want, value)
	}
}

func assertNotContains(t *testing.T, value, want string) {
	t.Helper()
	if strings.Contains(value, want) {
		t.Fatalf("view contains %q:\n%s", want, value)
	}
}

func assertSelectedLineContains(t *testing.T, view, want string) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.HasPrefix(line, "> ") && strings.Contains(line, want) {
			return
		}
	}
	t.Fatalf("selected row does not contain %q:\n%s", want, view)
}

func lastToolCallFilter(t *testing.T, filters []agentmodel.ToolCallFilters) agentmodel.ToolCallFilters {
	t.Helper()
	if len(filters) == 0 {
		t.Fatal("no tool call filters recorded")
	}
	return filters[len(filters)-1]
}

func lastToolFilter(t *testing.T, filters []agentmodel.ToolFilters) agentmodel.ToolFilters {
	t.Helper()
	if len(filters) == 0 {
		t.Fatal("no tool filters recorded")
	}
	return filters[len(filters)-1]
}

func lastAnalyticsFilter(t *testing.T, filters []agentmodel.AnalyticsFilters) agentmodel.AnalyticsFilters {
	t.Helper()
	if len(filters) == 0 {
		t.Fatal("no analytics filters recorded")
	}
	return filters[len(filters)-1]
}

func lastAuditSummaryFilter(t *testing.T, filters []agentmodel.AuditFindingFilters) agentmodel.AuditFindingFilters {
	t.Helper()
	if len(filters) == 0 {
		t.Fatal("no audit summary filters recorded")
	}
	return filters[len(filters)-1]
}

func lastAuditFindingFilter(t *testing.T, filters []agentmodel.AuditFindingFilters) agentmodel.AuditFindingFilters {
	t.Helper()
	if len(filters) == 0 {
		t.Fatal("no audit finding filters recorded")
	}
	return filters[len(filters)-1]
}

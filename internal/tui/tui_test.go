package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	agentmodel "AgentMeter/internal/model"
)

type fakeService struct {
	overview agentmodel.Overview
	sessions []agentmodel.Session
	detail   agentmodel.SessionDetail
	tools    []agentmodel.ToolStat
	settings agentmodel.Settings
	privacy  []agentmodel.PrivacyConfigStatus
	index    agentmodel.IndexResult

	indexCalls       []bool
	privacyApply     agentmodel.PrivacyConfigApplyResult
	privacyApplyErr  error
	privacyApplyCall []privacyApplyCall
}

type privacyApplyCall struct {
	target  string
	profile string
}

func (f *fakeService) GetOverview() (agentmodel.Overview, error) {
	return f.overview, nil
}

func (f *fakeService) ListSessions(_ agentmodel.SessionFilters) ([]agentmodel.Session, error) {
	return f.sessions, nil
}

func (f *fakeService) GetSessionDetail(_ int64) (agentmodel.SessionDetail, error) {
	return f.detail, nil
}

func (f *fakeService) GetTools() ([]agentmodel.ToolStat, error) {
	return f.tools, nil
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
	st := newState(svc, 100, 24)

	cmd := st.init()
	_, quit := st.update(runCommand(t, cmd))
	if quit {
		t.Fatal("unexpected quit")
	}

	view := st.view()
	assertContains(t, view, "Overview")
	assertContains(t, view, "Sessions: 2")
	assertContains(t, view, "gpt-5-codex")
	assertContains(t, view, "Recent Sessions")
}

func TestSessionsOpenDetail(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 40)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '2'})
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
	assertContains(t, view, "Tool Calls")
	assertContains(t, view, "shell_command")
	assertContains(t, view, "loaded repository")
}

func TestIndexNowRefreshesCurrentPage(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 24)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '4'})
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
	assertContains(t, view, "Indexed: 3")
}

func TestPrivacyPageLoadsAndRenders(t *testing.T) {
	svc := sampleService()
	st := newState(svc, 100, 70)

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '5'})
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

	cmd, quit = st.update(keyMsg{typ: keyShiftTab})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pageSettings {
		t.Fatalf("page = %v, want settings after shift-tab from privacy", st.page)
	}

	cmd, quit = st.update(keyMsg{typ: keyTab})
	if quit {
		t.Fatal("unexpected quit")
	}
	st.update(runCommand(t, cmd))
	if st.page != pagePrivacy {
		t.Fatalf("page = %v, want privacy after tab from settings", st.page)
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

	cmd, quit := st.update(keyMsg{typ: keyRune, ch: '5'})
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
		ID:             42,
		AgentKind:      "codex",
		AgentName:      "Codex",
		SessionKey:     "session-42",
		ProjectPath:    `D:\tools\custom\AgentMeter`,
		Model:          "gpt-5-codex",
		StartedAt:      started,
		EndedAt:        started.Add(15 * time.Minute),
		WallDurationMS: 900000,
		TokenUsage: agentmodel.Usage{
			TotalTokens:  12345,
			InputTokens:  8000,
			OutputTokens: 4345,
		},
		EstimatedCostUSD: &cost,
		ToolCallCount:    2,
	}
	index := agentmodel.IndexResult{FilesSeen: 4, Indexed: 3, Skipped: 1, Sessions: 2, DurationMS: 1200}
	return &fakeService{
		overview: agentmodel.Overview{
			TotalSessions:         2,
			TotalTokens:           12345,
			TotalInputTokens:      8000,
			TotalOutputTokens:     4345,
			EstimatedCostUSD:      &cost,
			TotalWallDurationMS:   900000,
			TotalActiveDurationMS: 600000,
			TotalToolCalls:        2,
			ModelUsage: []agentmodel.ModelUsage{
				{Model: "gpt-5-codex", SessionCount: 2, TotalTokens: 12345, EstimatedCostUSD: &cost},
			},
			RecentSessions: []agentmodel.Session{session},
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
				{StartedAt: started, ToolName: "shell_command", Status: "completed", DurationMS: 250, InputSummary: "rg --files"},
			},
		},
		tools: []agentmodel.ToolStat{
			{ToolName: "shell_command", Calls: 2, SuccessCalls: 2, TotalDurationMS: 500, AvgDurationMS: 250},
		},
		settings: agentmodel.Settings{
			DatabasePath:    `D:\tools\custom\AgentMeter\agentmeter.sqlite`,
			SourceEntries:   []agentmodel.SourceEntry{{Path: `D:\sessions`, Enabled: true}},
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

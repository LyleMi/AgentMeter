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
	index    agentmodel.IndexResult

	indexCalls []bool
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

func (f *fakeService) IndexNow(rebuild bool) (agentmodel.IndexResult, error) {
	f.indexCalls = append(f.indexCalls, rebuild)
	return f.index, nil
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
		index: index,
	}
}

func assertContains(t *testing.T, value, want string) {
	t.Helper()
	if !strings.Contains(value, want) {
		t.Fatalf("view does not contain %q:\n%s", want, value)
	}
}

package tui

import (
	"fmt"
	"strconv"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) auditLines() []string {
	lines := []string{
		bold("Audit"),
		fmt.Sprintf("Total: %-8s Critical: %-8s High: %-8s Medium: %-8s Low: %s",
			formatInt(int64(s.audit.TotalFindings)),
			formatInt(int64(s.audit.CriticalFindings)),
			formatInt(int64(s.audit.HighFindings)),
			formatInt(int64(s.audit.MediumFindings)),
			formatInt(int64(s.audit.LowFindings)),
		),
		fmt.Sprintf("Command: %-8s Privacy: %-8s Egress: %-8s File: %-8s Sessions: %s",
			formatInt(int64(s.audit.CommandFindings)),
			formatInt(int64(s.audit.PrivacyFindings)),
			formatInt(int64(s.audit.EgressFindings)),
			formatInt(int64(s.audit.FileFindings)),
			formatInt(int64(s.audit.SessionsWithFindings)),
		),
		dim("Enter opens recent finding detail; f opens the full findings list."),
		"",
		bold("Recent Findings"),
		fmt.Sprintf("%s loaded", formatInt(int64(len(s.audit.RecentFindings)))),
		auditFindingHeader(s.width),
	}
	if len(s.audit.RecentFindings) == 0 {
		if s.loading {
			return lines
		}
		return append(lines, "No audit findings indexed yet.")
	}
	visible := s.contentHeight() - len(lines)
	if visible < 1 {
		visible = 1
	}
	if s.scroll > len(s.audit.RecentFindings)-1 {
		s.scroll = len(s.audit.RecentFindings) - 1
	}
	end := s.scroll + visible
	if end > len(s.audit.RecentFindings) {
		end = len(s.audit.RecentFindings)
	}
	for i := s.scroll; i < end; i++ {
		lines = append(lines, auditFindingRow(s.audit.RecentFindings[i], i == s.selected, s.width))
	}
	return lines
}

func (s *state) auditFindingLines() []string {
	lines := []string{
		bold("Audit Findings"),
		fmt.Sprintf("%s loaded findings", formatInt(int64(len(s.findings)))),
		auditFindingHeader(s.width),
	}
	if len(s.findings) == 0 {
		if s.loading {
			return lines
		}
		return append(lines, "No audit findings indexed yet.")
	}
	visible := s.contentHeight() - len(lines)
	if visible < 1 {
		visible = 1
	}
	if s.scroll > len(s.findings)-1 {
		s.scroll = len(s.findings) - 1
	}
	end := s.scroll + visible
	if end > len(s.findings) {
		end = len(s.findings)
	}
	for i := s.scroll; i < end; i++ {
		lines = append(lines, auditFindingRow(s.findings[i], i == s.selected, s.width))
	}
	return lines
}

func (s *state) auditDetailViewportLines() []string {
	lines := auditDetailLines(s.finding, s.auditSession, s.width)
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

func auditFindingHeader(width int) string {
	return fit("  Time            Severity   Category   Source       Rule / finding                 Command / evidence", width)
}

func auditFindingRow(item agentmodel.AuditFinding, selected bool, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	title := auditFindingTitle(item)
	snippet := firstNonEmpty(item.Command, item.Evidence, item.Description)
	return fit(fmt.Sprintf("%s%-15s %-10s %-10s %-12s %-30s %s",
		prefix,
		formatTime(item.Timestamp),
		truncate(auditSeverityLabel(item.Severity), 10),
		truncate(auditCategoryLabel(item.Category), 10),
		truncate(auditSourceName(item), 12),
		truncate(firstNonEmpty(item.RuleID, title), 30),
		truncate(snippet, 42),
	), width)
}

func auditDetailLines(finding *agentmodel.AuditFinding, detail *agentmodel.SessionDetail, width int) []string {
	if finding == nil {
		return []string{bold("Audit Detail"), "No audit finding selected."}
	}
	item := *finding
	lines := []string{
		bold("Finding Detail"),
		"ID: " + strconv.FormatInt(item.ID, 10) + "  Rule: " + empty(item.RuleID, "unknown") + "  Decision: " + empty(item.Decision, "unknown"),
		"Severity: " + auditSeverityLabel(item.Severity) + "  Category: " + auditCategoryLabel(item.Category) + "  Source: " + auditSourceName(item),
		"Title: " + empty(item.Title, "unknown"),
		"Description: " + empty(item.Description, "-"),
		"Runtime: shell=" + empty(item.ShellFamily, "unknown") + "  platform=" + empty(item.Platform, "unknown") + "  event=" + empty(item.EventType, "unknown"),
		"Time: " + formatFullTime(item.Timestamp) + "  Source line: " + auditIntLabel(item.SourceLine) + "  Raw event: " + auditInt64Label(item.RawEventID) + "  Tool call: " + auditInt64Label(item.ToolCallID),
		"Session: " + auditSessionLabel(item) + "  Project: " + empty(item.ProjectPath, "unknown"),
		"Source root: " + empty(item.SourceRootPath, "unknown"),
		"Source sessions: " + empty(item.SourceSessionsPath, "unknown"),
		"Raw source: " + empty(item.RawSourcePath, "unknown"),
		"",
		bold("Command"),
	}
	lines = appendAuditValue(lines, item.Command, width)
	lines = append(lines, "", bold("Evidence"))
	lines = appendAuditValue(lines, item.Evidence, width)

	lines = append(lines, "", bold("Linked Session"))
	if detail == nil {
		lines = append(lines, "Session detail was not available.")
		return lines
	}
	session := detail.Session
	lines = append(lines,
		"Session: "+sessionLabel(session)+"  Source: "+sessionSourceName(session)+"  Model: "+empty(session.Model, "unknown"),
		"Started: "+formatFullTime(session.StartedAt)+"  Ended: "+formatFullTime(session.EndedAt),
		"Tokens: "+formatInt(session.TokenUsage.TotalTokens)+"  Cost: "+formatCost(session.EstimatedCostUSD)+"  Tools: "+formatInt(int64(session.ToolCallCount)),
		"Wall: "+formatDuration(session.WallDurationMS)+"  Active: "+formatDuration(session.ActiveDurationMS)+"  Model: "+formatDuration(session.ModelDurationMS)+"  Tools: "+formatDuration(session.ToolDurationMS),
		"Project: "+empty(session.ProjectPath, "unknown"),
	)
	if len(detail.ToolCalls) > 0 {
		lines = append(lines, "", bold("Session Tool Calls"))
		lines = append(lines, fit(fmt.Sprintf("  %-16s %-24s %-10s %10s %s", "Started", "Tool", "Status", "Duration", "Input"), width))
		for _, call := range limitSlice(detail.ToolCalls, 8) {
			lines = append(lines, fit(fmt.Sprintf("  %-16s %-24s %-10s %10s %s",
				formatTime(call.StartedAt),
				truncate(empty(call.ToolName, "unknown"), 24),
				truncate(empty(call.Status, "unknown"), 10),
				formatDuration(call.DurationMS),
				truncate(toolCallPrimarySummary(call), 40),
			), width))
		}
	}
	return lines
}

func appendAuditValue(lines []string, value string, width int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return append(lines, "-")
	}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, fit(line, width))
	}
	return lines
}

func auditFindingTitle(item agentmodel.AuditFinding) string {
	return firstNonEmpty(item.Title, item.RuleID, "unknown")
}

func auditSourceName(item agentmodel.AuditFinding) string {
	return sourceDisplayName(item.SourceLabel, item.AgentName, item.AgentKind, item.SourceKey)
}

func auditSessionLabel(item agentmodel.AuditFinding) string {
	return sessionLabel(agentmodel.Session{
		ID:             item.SessionID,
		SessionKey:     item.SessionKey,
		CodexSessionID: item.CodexSessionID,
	})
}

func auditSeverityLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func auditCategoryLabel(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func auditIntLabel(value int) string {
	if value <= 0 {
		return "-"
	}
	return formatInt(int64(value))
}

func auditInt64Label(value int64) string {
	if value <= 0 {
		return "-"
	}
	return "#" + formatInt(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

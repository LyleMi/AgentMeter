package tui

import (
	"fmt"
	"strings"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
)

func (s *state) isAuditPage() bool {
	switch s.page {
	case pageAudit, pageAuditFindings, pageAuditDetail:
		return true
	default:
		return false
	}
}

func (s *state) auditSummaryFilters() agentmodel.AuditFindingFilters {
	return agentmodel.AuditFindingFilters{Agent: strings.TrimSpace(s.auditAgent)}
}

func (s *state) auditFilters() agentmodel.AuditFindingFilters {
	return agentmodel.AuditFindingFilters{
		Agent:       strings.TrimSpace(s.auditAgent),
		Category:    strings.TrimSpace(s.auditCategory),
		Severity:    strings.TrimSpace(s.auditSeverity),
		ShellFamily: strings.TrimSpace(s.auditShell),
		Limit:       500,
	}
}

func (s *state) cycleAuditAgent() command {
	options := usageAgentOptions(s.scopeOverview)
	next, label := cycleStringOption(s.auditAgent, options)
	s.auditAgent = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit source filter: " + label
	return s.reloadAuditPage()
}

func (s *state) cycleAuditCategory() command {
	next, label := cycleStringOption(s.auditCategory, auditCategoryOptions())
	s.auditCategory = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit category filter: " + label
	return s.load(pageAuditFindings)
}

func (s *state) cycleAuditSeverity() command {
	next, label := cycleStringOption(s.auditSeverity, auditSeverityOptions())
	s.auditSeverity = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit severity filter: " + label
	return s.load(pageAuditFindings)
}

func (s *state) cycleAuditShell() command {
	next, label := cycleStringOption(s.auditShell, auditShellOptions())
	s.auditShell = next
	s.selected = 0
	s.scroll = 0
	s.status = "audit shell filter: " + label
	return s.load(pageAuditFindings)
}

func (s *state) clearAuditFilters() command {
	if s.auditAgent == "" && s.auditCategory == "" && s.auditSeverity == "" && s.auditShell == "" {
		s.status = "audit filters already clear"
		return nil
	}
	s.auditAgent = ""
	s.auditCategory = ""
	s.auditSeverity = ""
	s.auditShell = ""
	s.selected = 0
	s.scroll = 0
	s.status = "audit filters cleared"
	return s.reloadAuditPage()
}

func (s *state) reloadAuditPage() command {
	if s.page == pageAuditDetail {
		if s.finding == nil {
			return nil
		}
		return s.loadAuditDetail(s.finding.ID)
	}
	return s.load(auditListRootPage(s.page))
}

func auditListRootPage(current page) page {
	if current == pageAuditDetail {
		return pageAuditFindings
	}
	return current
}

func auditCategoryOptions() []stringOption {
	return []stringOption{
		{value: "command", label: "Command"},
		{value: "privacy", label: "Privacy"},
		{value: "egress", label: "Egress"},
		{value: "file", label: "File"},
	}
}

func auditSeverityOptions() []stringOption {
	return []stringOption{
		{value: "critical", label: "Critical"},
		{value: "high", label: "High"},
		{value: "medium", label: "Medium"},
		{value: "low", label: "Low"},
	}
}

func auditShellOptions() []stringOption {
	return []stringOption{
		{value: "posix", label: "POSIX"},
		{value: "powershell", label: "PowerShell"},
		{value: "cmd", label: "cmd.exe"},
		{value: "unknown", label: "Unknown shell"},
	}
}

func auditFindingMatchesAgent(finding agentmodel.AuditFinding, filter string) bool {
	filter = strings.ToLower(strings.TrimSpace(filter))
	if filter == "" {
		return true
	}
	values := []string{
		finding.SourceKey,
		finding.SourceLabel,
		finding.AgentKind,
		finding.AgentName,
	}
	if finding.SourceID > 0 {
		values = append(values, fmt.Sprintf("source:%d", finding.SourceID))
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == filter {
			return true
		}
	}
	return false
}

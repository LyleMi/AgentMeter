package audit

import (
	"fmt"
	"strings"
	"time"

	"AgentMeter/internal/model"
)

const (
	CategoryCommand = "command"
	CategoryEgress  = "egress"
	CategoryFile    = "file"
	CategoryPrivacy = "privacy"

	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// SessionInput is the compact audit input used by offline callers.
type SessionInput struct {
	Session   model.Session
	ToolCalls []model.ToolCall
	Events    []model.Event
}

type Finding struct {
	ID         string `json:"id"`
	RuleID     string `json:"ruleId"`
	Category   string `json:"category"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Evidence   string `json:"evidence"`
	Source     string `json:"source"`
	Field      string `json:"field,omitempty"`
	SessionID  int64  `json:"sessionId,omitempty"`
	SessionKey string `json:"sessionKey,omitempty"`

	ToolCallID int64  `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`

	EventID    int64 `json:"eventId,omitempty"`
	SourceLine int   `json:"sourceLine,omitempty"`

	ProjectPath string      `json:"projectPath,omitempty"`
	StartedAt   time.Time   `json:"startedAt,omitempty"`
	ShellFamily ShellFamily `json:"shellFamily,omitempty"`
	Command     string      `json:"command,omitempty"`
}

func Audit(input SessionInput) []Finding {
	return AuditSession(input.Session, input.ToolCalls, input.Events)
}

func AuditParsedSession(parsed model.ParsedSession) []Finding {
	return AuditSession(parsed.Session, parsed.ToolCall, parsed.Events)
}

func AuditSession(session model.Session, toolCalls []model.ToolCall, events []model.Event) []Finding {
	var findings []Finding
	toolEventLines := map[int]bool{}
	for i, call := range toolCalls {
		if command, ok := ExtractShellCommand(call); ok {
			findings = append(findings, shellFindingsForToolCall(session, call, i, command)...)
		}
		findings = append(findings, secretFindingsForToolCall(session, call, i)...)
		for _, line := range []int{call.RawEventLine, call.RawStartEventLine, call.RawEndEventLine} {
			if line != 0 {
				toolEventLines[line] = true
			}
		}
	}
	for i, event := range events {
		if toolEventLines[event.SourceLine] {
			continue
		}
		findings = append(findings, secretFindingsForEvent(session, event, i)...)
	}
	return findings
}

func shellFindingsForToolCall(session model.Session, call model.ToolCall, index int, command CommandInfo) []Finding {
	risks := ClassifyCommandRisks(command)
	findings := make([]Finding, 0, len(risks))
	for i, risk := range risks {
		findings = append(findings, Finding{
			ID:          findingID(session, "tool", index, risk.RuleID, i+1),
			RuleID:      risk.RuleID,
			Category:    firstNonEmpty(risk.Category, CategoryCommand),
			Severity:    risk.Severity,
			Title:       risk.Title,
			Evidence:    command.Command,
			Source:      "tool_call",
			SessionID:   session.ID,
			SessionKey:  session.SessionKey,
			ToolCallID:  call.ID,
			ToolName:    call.ToolName,
			SourceLine:  firstNonZero(call.RawStartEventLine, call.RawEventLine),
			ProjectPath: firstNonEmpty(call.ProjectPath, session.ProjectPath),
			StartedAt:   call.StartedAt,
			ShellFamily: command.Family,
			Command:     command.Command,
		})
	}
	return findings
}

func secretFindingsForToolCall(session model.Session, call model.ToolCall, index int) []Finding {
	startLine := firstNonZero(call.RawStartEventLine, call.RawEventLine)
	endLine := firstNonZero(call.RawEndEventLine, startLine)
	endEventID := call.RawEndEventID
	if endEventID == 0 {
		endEventID = call.RawStartEventID
	}
	fields := []scanField{
		{name: "input_summary", text: call.InputSummary, sourceLine: startLine, eventID: call.RawStartEventID},
		{name: "raw_start_event_json", text: call.RawStartEventJSON, sourceLine: startLine, eventID: call.RawStartEventID},
		{name: "output_summary", text: call.OutputSummary, sourceLine: endLine, eventID: endEventID},
		{name: "error", text: call.Error, sourceLine: endLine, eventID: endEventID},
		{name: "raw_end_event_json", text: call.RawEndEventJSON, sourceLine: endLine, eventID: endEventID},
	}
	seen := map[string]bool{}
	var findings []Finding
	occurrence := 0
	for _, field := range fields {
		for _, hit := range FindSecrets(field.text) {
			key := hit.RuleID + "\x00" + hit.Evidence
			if seen[key] {
				continue
			}
			seen[key] = true
			occurrence++
			findings = append(findings, Finding{
				ID:          findingID(session, "tool", index, hit.RuleID, occurrence),
				RuleID:      hit.RuleID,
				Category:    CategoryPrivacy,
				Severity:    hit.Severity,
				Title:       hit.Title,
				Evidence:    hit.Evidence,
				Source:      "tool_call",
				Field:       field.name,
				SessionID:   session.ID,
				SessionKey:  session.SessionKey,
				ToolCallID:  call.ID,
				ToolName:    call.ToolName,
				EventID:     field.eventID,
				SourceLine:  field.sourceLine,
				ProjectPath: firstNonEmpty(call.ProjectPath, session.ProjectPath),
				StartedAt:   call.StartedAt,
			})
		}
	}
	return findings
}

func secretFindingsForEvent(session model.Session, event model.Event, index int) []Finding {
	fields := []scanField{
		{name: "summary", text: event.Summary},
		{name: "raw_json", text: event.RawJSON},
	}
	seen := map[string]bool{}
	var findings []Finding
	occurrence := 0
	for _, field := range fields {
		for _, hit := range FindSecrets(field.text) {
			key := hit.RuleID + "\x00" + hit.Evidence
			if seen[key] {
				continue
			}
			seen[key] = true
			occurrence++
			findings = append(findings, Finding{
				ID:          findingID(session, "event", index, hit.RuleID, occurrence),
				RuleID:      hit.RuleID,
				Category:    CategoryPrivacy,
				Severity:    hit.Severity,
				Title:       hit.Title,
				Evidence:    hit.Evidence,
				Source:      "event",
				Field:       field.name,
				SessionID:   session.ID,
				SessionKey:  session.SessionKey,
				EventID:     event.ID,
				SourceLine:  event.SourceLine,
				ProjectPath: session.ProjectPath,
				StartedAt:   event.Timestamp,
			})
		}
	}
	return findings
}

type scanField struct {
	name       string
	text       string
	sourceLine int
	eventID    int64
}

func findingID(session model.Session, source string, index int, ruleID string, occurrence int) string {
	return fmt.Sprintf("%s/%s/%03d/%s/%02d", stableSessionKey(session), source, index, ruleID, occurrence)
}

func stableSessionKey(session model.Session) string {
	for _, value := range []string{session.SessionKey, session.CodexSessionID} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return sanitizeIDPart(trimmed)
		}
	}
	if session.ID != 0 {
		return fmt.Sprintf("session-%d", session.ID)
	}
	return "session"
}

func sanitizeIDPart(value string) string {
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", " ", "_", "\t", "_", "\n", "_", "\r", "_")
	return replacer.Replace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

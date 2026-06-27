package audit

import (
	"regexp"
	"strings"
	"unicode"
)

type SecretHit struct {
	RuleID   string `json:"ruleId"`
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Evidence string `json:"evidence"`
}

type secretRule struct {
	ruleID   string
	severity string
	title    string
	pattern  *regexp.Regexp
	validate func(string) bool
}

var secretRules = []secretRule{
	{
		ruleID:   "privacy.private-key",
		severity: SeverityCritical,
		title:    "Private key block",
		pattern:  regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?-----END [A-Z0-9 ]*PRIVATE KEY-----`),
	},
	{
		ruleID:   "privacy.aws-access-key-id",
		severity: SeverityHigh,
		title:    "AWS access key id",
		pattern:  regexp.MustCompile(`\b(?:AKIA|ASIA)[A-Z0-9]{16}\b`),
	},
	{
		ruleID:   "privacy.github-token",
		severity: SeverityHigh,
		title:    "GitHub token",
		pattern:  regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9_]{36,}|github_pat_[A-Za-z0-9_]{22,}_[A-Za-z0-9_]{59,})\b`),
	},
	{
		ruleID:   "privacy.openai-key",
		severity: SeverityHigh,
		title:    "OpenAI API key",
		pattern:  regexp.MustCompile(`\bsk-(?:proj-|svcacct-)?[A-Za-z0-9_-]{20,}\b`),
	},
	{
		ruleID:   "privacy.api-key-assignment",
		severity: SeverityHigh,
		title:    "API key-like assignment",
		pattern:  regexp.MustCompile(`(?i)\b[A-Z0-9_.-]*(api[_-]?key|secret[_-]?key|access[_-]?token|auth[_-]?token|bearer[_-]?token|client[_-]?secret|password|passwd|pwd)\b\s*[:=]\s*["']?[A-Za-z0-9][A-Za-z0-9_./+=:@%\-]{7,}["']?`),
	},
	{
		ruleID:   "privacy.email",
		severity: SeverityLow,
		title:    "Email address",
		pattern:  regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`),
	},
	{
		ruleID:   "privacy.ssn",
		severity: SeverityMedium,
		title:    "SSN-like value",
		pattern:  regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	},
	{
		ruleID:   "privacy.credit-card",
		severity: SeverityMedium,
		title:    "Credit-card-like value",
		pattern:  regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
		validate: validLuhnEvidence,
	},
}

func FindSecrets(text string) []SecretHit {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	seen := map[string]bool{}
	var hits []SecretHit
	for _, rule := range secretRules {
		matches := rule.pattern.FindAllString(text, -1)
		for _, match := range matches {
			evidence := strings.TrimSpace(match)
			if evidence == "" {
				continue
			}
			if rule.validate != nil && !rule.validate(evidence) {
				continue
			}
			key := rule.ruleID + "\x00" + evidence
			if seen[key] {
				continue
			}
			seen[key] = true
			hits = append(hits, SecretHit{
				RuleID:   rule.ruleID,
				Severity: rule.severity,
				Title:    rule.title,
				Evidence: evidence,
			})
		}
	}
	return hits
}

func validLuhnEvidence(evidence string) bool {
	digits := digitsOnly(evidence)
	if len(digits) < 13 || len(digits) > 19 {
		return false
	}
	return luhnValid(digits)
}

func digitsOnly(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func luhnValid(digits string) bool {
	sum := 0
	double := false
	for i := len(digits) - 1; i >= 0; i-- {
		n := int(digits[i] - '0')
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}

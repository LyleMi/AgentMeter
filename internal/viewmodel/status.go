package viewmodel

import (
	"strings"
	"unicode"
)

type Tone string

const (
	ToneDefault    Tone = "default"
	ToneSuccess    Tone = "success"
	ToneWarning    Tone = "warning"
	ToneError      Tone = "error"
	ToneProcessing Tone = "processing"
)

type Signal struct {
	Tone  Tone   `json:"tone"`
	Label string `json:"label"`
}

// ClassifyStatus maps parser, scanner, model-call, and tool-call statuses to a
// small display vocabulary shared by viewmodels.
func ClassifyStatus(status string) Signal {
	normalized := normalizeStatus(status)
	if normalized == "" {
		return Signal{Tone: ToneDefault, Label: "unknown"}
	}
	switch normalized {
	case "ok", "indexed", "completed", "complete", "success", "succeeded", "clear":
		return Signal{Tone: ToneSuccess, Label: humanizeStatus(normalized)}
	case "warning", "warn", "partial":
		return Signal{Tone: ToneWarning, Label: humanizeStatus(normalized)}
	case "failed", "failure", "error", "errored", "aborted", "cancelled", "canceled", "timeout", "timed_out":
		return Signal{Tone: ToneError, Label: humanizeStatus(normalized)}
	case "pending", "started", "running", "indexing", "processing", "in_progress":
		return Signal{Tone: ToneProcessing, Label: humanizeStatus(normalized)}
	case "skipped", "unknown":
		return Signal{Tone: ToneDefault, Label: humanizeStatus(normalized)}
	default:
		return Signal{Tone: ToneDefault, Label: humanizeStatus(normalized)}
	}
}

func IsSuccessfulStatus(status string) bool {
	switch normalizeStatus(status) {
	case "ok", "indexed", "completed", "complete", "success", "succeeded":
		return true
	default:
		return false
	}
}

func IsProblemStatus(status string) bool {
	switch ClassifyStatus(status).Tone {
	case ToneWarning, ToneError:
		return true
	default:
		return false
	}
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	status = strings.ReplaceAll(status, "-", "_")
	status = strings.Join(strings.Fields(status), "_")
	return status
}

func humanizeStatus(status string) string {
	status = strings.ReplaceAll(status, "_", " ")
	if status == "" {
		return "unknown"
	}
	runes := []rune(status)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

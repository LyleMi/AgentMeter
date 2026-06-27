package viewmodel

import (
	"testing"
	"time"

	"AgentMeter/internal/model"
)

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		value int64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{123456789, "123,456,789"},
		{-1234567, "-1,234,567"},
	}
	for _, test := range tests {
		if got := FormatNumber(test.value); got != test.want {
			t.Fatalf("FormatNumber(%d) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestFormatCost(t *testing.T) {
	value := 12.34567
	whole := 12.0
	if got := FormatCost(nil); got != "unpriced" {
		t.Fatalf("FormatCost(nil) = %q", got)
	}
	if got := FormatCost(&value); got != "$12.3457" {
		t.Fatalf("FormatCost(value) = %q", got)
	}
	if got := FormatCost(&whole); got != "$12.00" {
		t.Fatalf("FormatCost(whole) = %q", got)
	}
	if got := FormatCostPerThousand(&value, 0); got != "$0" {
		t.Fatalf("FormatCostPerThousand zero tokens = %q", got)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		ms   float64
		want string
	}{
		{-1, "0s"},
		{1499, "1s"},
		{1500, "2s"},
		{65_000, "1m 5s"},
		{3_660_000, "1h 1m"},
	}
	for _, test := range tests {
		if got := FormatDuration(test.ms); got != test.want {
			t.Fatalf("FormatDuration(%v) = %q, want %q", test.ms, got, test.want)
		}
	}
}

func TestFormatRatio(t *testing.T) {
	if got := FormatRatio(1000); got != "1,000" {
		t.Fatalf("FormatRatio(1000) = %q", got)
	}
	if got := FormatRatio(1.25); got != "1.3" {
		t.Fatalf("FormatRatio(1.25) = %q", got)
	}
}

func TestFormatDateTime(t *testing.T) {
	value := time.Date(2026, 6, 27, 9, 8, 7, 0, time.UTC)
	if got := FormatDateTime(time.Time{}); got != "-" {
		t.Fatalf("zero time = %q", got)
	}
	if got := FormatDateTime(value); got != "Jun 27, 09:08" {
		t.Fatalf("time = %q", got)
	}
}

func TestShortPath(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"", "unknown"},
		{`C:\workspace\project`, `C:\workspace\project`},
		{`C:\users\me\workspace\project`, ".../me/workspace/project"},
		{"/home/me/workspace/project", ".../me/workspace/project"},
	}
	for _, test := range tests {
		if got := ShortPath(test.value); got != test.want {
			t.Fatalf("ShortPath(%q) = %q, want %q", test.value, got, test.want)
		}
	}
}

func TestSessionLabel(t *testing.T) {
	tests := []struct {
		session model.Session
		want    string
	}{
		{model.Session{ID: 7, SessionKey: "sess"}, "sess"},
		{model.Session{ID: 7, CodexSessionID: "codex"}, "codex"},
		{model.Session{ID: 7}, "#7"},
	}
	for _, test := range tests {
		if got := SessionLabel(test.session); got != test.want {
			t.Fatalf("SessionLabel(%+v) = %q, want %q", test.session, got, test.want)
		}
	}
}

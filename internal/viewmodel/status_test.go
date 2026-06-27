package viewmodel

import "testing"

func TestClassifyStatus(t *testing.T) {
	tests := []struct {
		status string
		tone   Tone
		label  string
	}{
		{"completed", ToneSuccess, "Completed"},
		{"warning", ToneWarning, "Warning"},
		{"timed-out", ToneError, "Timed out"},
		{"in progress", ToneProcessing, "In progress"},
		{"", ToneDefault, "unknown"},
	}
	for _, test := range tests {
		got := ClassifyStatus(test.status)
		if got.Tone != test.tone || got.Label != test.label {
			t.Fatalf("ClassifyStatus(%q) = %+v, want tone %q label %q", test.status, got, test.tone, test.label)
		}
	}
}

func TestStatusHelpers(t *testing.T) {
	if !IsSuccessfulStatus("success") {
		t.Fatal("success should be successful")
	}
	if IsSuccessfulStatus("warning") {
		t.Fatal("warning should not be successful")
	}
	if !IsProblemStatus("failed") || !IsProblemStatus("warning") {
		t.Fatal("failed and warning should be problem statuses")
	}
	if IsProblemStatus("completed") {
		t.Fatal("completed should not be a problem status")
	}
}

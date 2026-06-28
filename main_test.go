package main

import (
	"reflect"
	"testing"
)

func TestNormalizeCommandArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "empty"},
		{name: "tui command", args: []string{"tui"}, want: []string{"-ui", "tui"}},
		{name: "cli command", args: []string{"cli", "-http", ":34115"}, want: []string{"-ui", "tui", "-http", ":34115"}},
		{name: "web command", args: []string{"web"}, want: []string{"-ui", "web"}},
		{name: "start command", args: []string{"start", "-skip-browser"}, want: []string{"-start", "-skip-browser"}},
		{name: "flags unchanged", args: []string{"-ui", "tui"}, want: []string{"-ui", "tui"}},
		{name: "unknown command unchanged", args: []string{"serve"}, want: []string{"serve"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeCommandArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeCommandArgs(%v) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}

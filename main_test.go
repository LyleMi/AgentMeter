package main

import (
	"reflect"
	"testing"
)

func TestNormalizeRuntimeConfig(t *testing.T) {
	tests := []struct {
		name string
		in   runtimeConfig
		want runtimeConfig
	}{
		{name: "empty UI defaults to web", in: runtimeConfig{uiMode: "  ", httpAddr: "127.0.0.1:1"}, want: runtimeConfig{uiMode: "web", httpAddr: "127.0.0.1:1"}},
		{name: "normalizes UI and local address", in: runtimeConfig{uiMode: " TUI ", httpAddr: ":34115"}, want: runtimeConfig{uiMode: "tui", httpAddr: "127.0.0.1:34115"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeRuntimeConfig(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeRuntimeConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestValidateRuntimeConfig(t *testing.T) {
	valid := []runtimeConfig{
		{uiMode: "web"},
		{uiMode: "tui"},
		{uiMode: "web", start: true, skipBrowser: true, forceBuild: true},
	}
	for _, config := range valid {
		if err := validateRuntimeConfig(config); err != nil {
			t.Errorf("validateRuntimeConfig(%+v) = %v", config, err)
		}
	}

	invalid := []runtimeConfig{
		{uiMode: "web", skipBrowser: true},
		{uiMode: "web", forceBuild: true},
		{uiMode: "tui", start: true},
	}
	for _, config := range invalid {
		if err := validateRuntimeConfig(config); err == nil {
			t.Errorf("validateRuntimeConfig(%+v) unexpectedly succeeded", config)
		}
	}
}

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

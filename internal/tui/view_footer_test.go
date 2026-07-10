package tui

import "testing"

func TestFooterTextForState(t *testing.T) {
	pending := &privacyProfileAction{}
	tests := []struct {
		name  string
		state state
		want  string
	}{
		{name: "fixed page", state: state{page: pageOverview}, want: pageFooterText[pageOverview]},
		{name: "token summary", state: state{page: pageTokens, tokensTab: tokensTabSummary}, want: tokensFooterText},
		{name: "token breakdown", state: state{page: pageTokens, tokensTab: tokensTabBreakdown}, want: tokenBreakdownFooterText},
		{name: "tools overview", state: state{page: pageTools, toolsTab: toolsTabOverview}, want: toolsFooterText},
		{name: "tools shell", state: state{page: pageTools, toolsTab: toolsTabShell}, want: toolShellFooterText},
		{name: "tools calls", state: state{page: pageTools, toolsTab: toolsTabCalls}, want: toolCallsFooterText},
		{name: "privacy", state: state{page: pagePrivacy}, want: privacyFooterText},
		{name: "privacy pending", state: state{page: pagePrivacy, privacyPending: pending}, want: privacyPendingFooterText},
		{name: "unknown page", state: state{page: page(999)}, want: defaultFooterText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.footerText(); got != tt.want {
				t.Fatalf("footerText() = %q, want %q", got, tt.want)
			}
		})
	}
}

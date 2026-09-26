package tui

import (
	"fmt"
	"strings"
	"testing"

	agentmodel "github.com/LyleMi/AgentMeter/internal/model"
	"github.com/charmbracelet/x/ansi"
)

func TestLayoutFitsTerminal(t *testing.T) {
	for _, size := range [][2]int{{120, 40}, {80, 24}, {40, 12}, {20, 8}, {10, 3}} {
		for _, page := range []page{pageAttention, pageOverview, pageSessions, pageSettings, pagePrivacy, pageAudit} {
			st := newState(sampleService(), size[0], size[1])
			st.page = page
			st.update(runCommand(t, st.load(page)))
			for _, help := range []bool{false, true} {
				st.helpOpen = help
				lines := strings.Split(st.view(), "\n")
				if len(lines) > size[1] {
					t.Fatalf("page %v at %v: %d lines", page, size, len(lines))
				}
				for _, line := range lines {
					if ansi.StringWidth(line) > size[0] {
						t.Fatalf("page %v at %v overflows: %q", page, size, line)
					}
				}
			}
		}
	}
}

func TestFitPreservesANSIAndDisplayWidth(t *testing.T) {
	for _, value := range []string{accent("中文路径和字符"), bold("abcdef"), "a👩‍💻你好界"} {
		for width := 1; width < 12; width++ {
			got := fit(value, width)
			if ansi.StringWidth(got) > width {
				t.Fatalf("fit(%q, %d) = %q", value, width, got)
			}
			if strings.Contains(ansi.Strip(got), "\x1b") {
				t.Fatalf("broken escape: %q", got)
			}
		}
	}
	if got := fit(accent("hello"), 5); got != accent("hello") {
		t.Fatalf("color consumes columns: %q", got)
	}
}

func TestHelpIsReadOnlyAndRestoresView(t *testing.T) {
	st := newState(sampleService(), 60, 16)
	st.page = pagePrivacy
	st.privacyPending = &privacyProfileAction{target: "codex", profile: "strict"}
	st.scroll = 3
	st.handleKey(keyMsg{typ: keyRune, ch: '?'})
	for _, key := range []keyMsg{{typ: keyEnter}, {typ: keyRune, ch: 'i'}, {typ: keyRune, ch: 'u'}, {typ: keyRune, ch: '2'}} {
		cmd, quit := st.handleKey(key)
		if cmd != nil || quit || st.privacyPending == nil || st.page != pagePrivacy {
			t.Fatal("help dispatched a page action")
		}
	}
	st.handleKey(keyMsg{typ: keyEnd})
	lines := st.helpLines()
	if len(lines) > st.contentHeight() {
		t.Fatal("help End did not reach bottom")
	}
	st.handleKey(keyMsg{typ: keyEsc})
	if st.helpOpen || st.scroll != 3 || st.privacyPending == nil {
		t.Fatal("help did not preserve underlying state")
	}
}

func TestAttentionKeepsSelectionVisible(t *testing.T) {
	for _, height := range []int{10, 14, 24, 40} {
		st := newState(sampleService(), 80, height)
		for i := 0; i < 30; i++ {
			st.attention.Items = append(st.attention.Items, agentmodel.AttentionItem{Subject: fmt.Sprintf("subject-%02d", i), Severity: "warning", Reason: "Review this session"})
		}
		for i := range st.attention.Items {
			st.moveTo(i)
			if !strings.Contains(ansi.Strip(st.view()), "› WARNING  "+fmt.Sprintf("subject-%02d", i)) {
				t.Fatalf("selection %d hidden at height %d", i, height)
			}
		}
	}
}

func TestUsageScrollChangesVisibleContent(t *testing.T) {
	st := newState(sampleService(), 100, 16)
	st.page = pageOverview
	st.update(runCommand(t, st.load(pageOverview)))
	before := strings.Join(st.content(), "\n")
	st.handleKey(keyMsg{typ: keyEnd})
	if st.scroll == 0 || before == strings.Join(st.content(), "\n") {
		t.Fatal("usage scroll did not change the viewport")
	}
	st.handleKey(keyMsg{typ: keyHome})
	if st.scroll != 0 {
		t.Fatal("Home did not restore the viewport")
	}
}

func TestSafetyPrivacyTabCanSwitchBack(t *testing.T) {
	st := newState(sampleService(), 80, 24)
	st.page = pageAudit
	st.safetyTab = safetyTabPrivacy
	st.handleKey(keyMsg{typ: keyRune, ch: '['})
	if st.safetyTab != safetyTabAudit {
		t.Fatal("privacy intercepted safety tab navigation")
	}
}

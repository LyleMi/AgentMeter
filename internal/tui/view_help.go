package tui

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
)

// Help uses its own scroll position so closing it restores the current view.
func (s *state) allHelpLines() []string {
	lines := []string{bold("Keyboard shortcuts"), dim(s.page.title()), "",
		"1–5               Attention / Analyze / Sessions / Safety / Settings",
		"tab / shift-tab   Next / previous task",
		"[ / ]             Previous / next tab (Analyze, Safety, detail analysis)",
		"↑↓ / j k          Select or scroll",
		"pgup / pgdn       Page through content",
		"home / end        Jump to first / last item",
		"b / esc           Back from detail",
		"r                 Refresh current view",
		"? / esc           Close help",
		"q / ctrl-c        Quit",
		"", bold("This view")}
	for _, entry := range strings.Split(strings.TrimPrefix(s.footerText(), "Keys: "), "  ") {
		if entry != "" {
			lines = append(lines, entry)
		}
	}
	if s.isUsageScopePage() {
		lines = append(lines, "", bold("Scope"), "u source · v model · w project · e time range", "U reset to the last 7 days")
	}
	return lines
}

func (s *state) helpLines() []string {
	lines := s.allHelpLines()
	width := s.width
	if width <= 0 {
		width = defaultWidth
	}
	wrapped := []string{}
	for _, line := range lines {
		// ANSI-aware wrapping preserves key descriptions on narrow screens.
		wrapped = append(wrapped, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	s.helpScroll = max(0, min(s.helpScroll, max(0, len(wrapped)-s.contentHeight())))
	return wrapped[s.helpScroll:]
}

func (s *state) handleHelpKey(k keyMsg) (command, bool) {
	switch k.typ {
	case keyEsc:
		s.helpOpen = false
	case keyUp:
		s.helpScroll--
	case keyDown:
		s.helpScroll++
	case keyPageUp:
		s.helpScroll -= s.contentHeight()
	case keyPageDown:
		s.helpScroll += s.contentHeight()
	case keyHome:
		s.helpScroll = 0
	case keyEnd:
		s.helpScroll = int(^uint(0) >> 1)
	case keyRune:
		switch k.ch {
		case 'q', 'Q':
			return nil, true
		case 'j', 'J':
			s.helpScroll++
		case 'k', 'K':
			s.helpScroll--
		}
	}
	return nil, false
}

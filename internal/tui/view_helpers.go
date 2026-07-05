package tui

func (s *state) viewportLines(lines []string) []string {
	return s.viewportLinesWithHeight(lines, s.contentHeight())
}

func (s *state) viewportLinesWithHeight(lines []string, height int) []string {
	if height < 1 {
		height = 1
	}
	s.scroll = clampViewportScroll(s.scroll, len(lines))
	end := s.scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return lines[s.scroll:end]
}

func (s *state) visibleItemRange(itemCount, headerLines int) (int, int) {
	visible := s.contentHeight() - headerLines
	if visible < 1 {
		visible = 1
	}
	s.scroll = clampViewportScroll(s.scroll, itemCount)
	end := s.scroll + visible
	if end > itemCount {
		end = itemCount
	}
	return s.scroll, end
}

func clampViewportScroll(scroll, itemCount int) int {
	if itemCount <= 0 {
		return 0
	}
	if scroll >= itemCount {
		return itemCount - 1
	}
	if scroll < 0 {
		return 0
	}
	return scroll
}

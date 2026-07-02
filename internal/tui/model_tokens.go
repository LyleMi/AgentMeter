package tui

func (s *state) cycleTokenBreakdownGroup() command {
	index := 0
	for i, group := range tokenBreakdownGroups {
		if group == s.tokenBreakdownGroup {
			index = i
			break
		}
	}
	index = (index + 1) % len(tokenBreakdownGroups)
	s.tokenBreakdownGroup = tokenBreakdownGroups[index]
	s.selected = 0
	s.scroll = 0
	s.status = "token breakdown group: " + tokenBreakdownGroupTitle(s.tokenBreakdownGroup)
	return s.load(pageTokens)
}

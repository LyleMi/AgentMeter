package tui

import (
	"context"
	"fmt"

	agentmodel "AgentMeter/internal/model"
)

type appService interface {
	GetOverview() (agentmodel.Overview, error)
	ListSessions(agentmodel.SessionFilters) ([]agentmodel.Session, error)
	GetSessionDetail(id int64) (agentmodel.SessionDetail, error)
	GetTools() ([]agentmodel.ToolStat, error)
	GetSettings() (agentmodel.Settings, error)
	GetPrivacyConfigs() ([]agentmodel.PrivacyConfigStatus, error)
	IndexNow(rebuild bool) (agentmodel.IndexResult, error)
}

type privacyProfileApplier interface {
	ApplyPrivacyProfile(target, profile string) (agentmodel.PrivacyConfigApplyResult, error)
}

type page int

const (
	pageOverview page = iota
	pageSessions
	pageSessionDetail
	pageTools
	pageSettings
	pagePrivacy
)

func (p page) title() string {
	switch p {
	case pageOverview:
		return "Overview"
	case pageSessions:
		return "Sessions"
	case pageSessionDetail:
		return "Session Detail"
	case pageTools:
		return "Tools"
	case pageSettings:
		return "Settings"
	case pagePrivacy:
		return "Agent Privacy"
	default:
		return "Unknown"
	}
}

type keyType int

const (
	keyUnknown keyType = iota
	keyRune
	keyEnter
	keyEsc
	keyCtrlC
	keyTab
	keyShiftTab
	keyUp
	keyDown
	keyLeft
	keyRight
	keyPageUp
	keyPageDown
	keyHome
	keyEnd
)

type keyMsg struct {
	typ keyType
	ch  rune
}

type loadMsg struct {
	seq      int
	page     page
	overview agentmodel.Overview
	sessions []agentmodel.Session
	detail   agentmodel.SessionDetail
	tools    []agentmodel.ToolStat
	settings agentmodel.Settings
	privacy  []agentmodel.PrivacyConfigStatus
	err      error
}

type indexMsg struct {
	result  agentmodel.IndexResult
	rebuild bool
	err     error
}

type privacyProfileAction struct {
	target     string
	targetName string
	profile    string
}

type privacyProfileMsg struct {
	target     string
	targetName string
	profile    string
	result     agentmodel.PrivacyConfigApplyResult
	err        error
}

type resizeMsg struct {
	width  int
	height int
}

type message interface{}

type command func(context.Context, chan<- message)

type state struct {
	service appService

	page     page
	previous page

	width  int
	height int

	loadSeq int
	loading bool
	err     error
	status  string

	selected int
	scroll   int

	overview agentmodel.Overview
	sessions []agentmodel.Session
	detail   *agentmodel.SessionDetail
	tools    []agentmodel.ToolStat
	settings agentmodel.Settings
	privacy  []agentmodel.PrivacyConfigStatus

	indexing  bool
	lastIndex *agentmodel.IndexResult

	privacyTarget   int
	privacyPending  *privacyProfileAction
	privacyApplying bool
}

func newState(service appService, width, height int) *state {
	if width <= 0 {
		width = defaultWidth
	}
	if height <= 0 {
		height = defaultHeight
	}
	return &state{
		service: service,
		page:    pageOverview,
		width:   width,
		height:  height,
	}
}

func (s *state) init() command {
	return s.load(pageOverview)
}

func (s *state) update(msg message) (command, bool) {
	switch m := msg.(type) {
	case keyMsg:
		return s.handleKey(m)
	case loadMsg:
		if m.seq != s.loadSeq {
			return nil, false
		}
		s.loading = false
		s.err = m.err
		if m.err != nil {
			s.status = "load failed: " + m.err.Error()
			return nil, false
		}
		switch m.page {
		case pageOverview:
			s.overview = m.overview
		case pageSessions:
			s.sessions = m.sessions
			s.clampSelection(len(s.sessions))
		case pageSessionDetail:
			detail := m.detail
			s.detail = &detail
			s.scroll = 0
		case pageTools:
			s.tools = m.tools
			s.clampSelection(len(s.tools))
		case pageSettings:
			s.settings = m.settings
		case pagePrivacy:
			s.privacy = m.privacy
			s.clampPrivacyTarget()
		}
	case indexMsg:
		s.indexing = false
		if m.err != nil {
			s.err = m.err
			s.status = "index failed: " + m.err.Error()
			return nil, false
		}
		s.err = nil
		result := m.result
		s.lastIndex = &result
		mode := "index"
		if m.rebuild {
			mode = "rebuild index"
		}
		s.status = fmt.Sprintf("%s complete: %d indexed, %d skipped, %d failed, %d sessions",
			mode, result.Indexed, result.Skipped, result.Failed, result.Sessions)
		return s.load(s.page), false
	case privacyProfileMsg:
		s.privacyApplying = false
		s.privacyPending = nil
		if m.err != nil {
			s.err = m.err
			s.status = "privacy profile failed: " + m.err.Error()
			return nil, false
		}
		s.err = nil
		s.mergePrivacyStatus(m.result.Status, m.target)
		s.status = privacyApplyStatus(m.profile, m.targetName, m.result)
		return nil, false
	case resizeMsg:
		if m.width > 0 {
			s.width = m.width
		}
		if m.height > 0 {
			s.height = m.height
		}
		s.ensureVisible()
	}
	return nil, false
}

func (s *state) handleKey(k keyMsg) (command, bool) {
	if k.typ == keyCtrlC {
		return nil, true
	}
	if s.page == pagePrivacy {
		if cmd, quit, handled := s.handlePrivacyKey(k); handled {
			return cmd, quit
		}
	}
	if k.typ == keyRune {
		switch k.ch {
		case 'q', 'Q':
			return nil, true
		case '1', 'o', 'O':
			return s.switchPage(pageOverview), false
		case '2', 's', 'S':
			return s.switchPage(pageSessions), false
		case '3', 't', 'T':
			return s.switchPage(pageTools), false
		case '4', 'g', 'G':
			return s.switchPage(pageSettings), false
		case '5', 'p', 'P':
			return s.switchPage(pagePrivacy), false
		case 'r', 'R':
			return s.load(s.page), false
		case 'i':
			return s.index(false), false
		case 'I':
			return s.index(true), false
		case 'j', 'J':
			s.move(1)
		case 'k', 'K':
			s.move(-1)
		case 'b', 'B':
			if s.page == pageSessionDetail {
				return s.switchPage(pageSessions), false
			}
		}
	}

	switch k.typ {
	case keyTab, keyRight:
		return s.switchPage(s.nextPage()), false
	case keyShiftTab, keyLeft:
		return s.switchPage(s.previousPage()), false
	case keyUp:
		s.move(-1)
	case keyDown:
		s.move(1)
	case keyPageUp:
		s.move(-s.pageStep())
	case keyPageDown:
		s.move(s.pageStep())
	case keyHome:
		s.moveTo(0)
	case keyEnd:
		s.moveTo(s.itemCount() - 1)
	case keyEnter:
		if s.page == pageSessions && len(s.sessions) > 0 {
			id := s.sessions[s.selected].ID
			s.previous = pageSessions
			s.page = pageSessionDetail
			s.selected = 0
			s.scroll = 0
			s.detail = nil
			return s.loadDetail(id), false
		}
	case keyEsc:
		if s.page == pageSessionDetail {
			return s.switchPage(s.previous), false
		}
	}
	return nil, false
}

func (s *state) switchPage(target page) command {
	if target == pageSessionDetail {
		target = pageSessions
	}
	if target == s.page && !s.loading {
		return nil
	}
	s.page = target
	s.selected = 0
	s.scroll = 0
	s.detail = nil
	if target != pagePrivacy {
		s.privacyPending = nil
	}
	return s.load(target)
}

func (s *state) nextPage() page {
	switch s.page {
	case pageOverview:
		return pageSessions
	case pageSessions, pageSessionDetail:
		return pageTools
	case pageTools:
		return pageSettings
	case pageSettings:
		return pagePrivacy
	default:
		return pageOverview
	}
}

func (s *state) previousPage() page {
	switch s.page {
	case pageOverview:
		return pagePrivacy
	case pageSessions, pageSessionDetail:
		return pageOverview
	case pageTools:
		return pageSessions
	case pageSettings:
		return pageTools
	default:
		return pageSettings
	}
}

func (s *state) load(target page) command {
	s.loadSeq++
	seq := s.loadSeq
	s.loading = true
	s.err = nil
	return func(ctx context.Context, ch chan<- message) {
		msg := loadMsg{seq: seq, page: target}
		switch target {
		case pageOverview:
			msg.overview, msg.err = s.service.GetOverview()
		case pageSessions:
			msg.sessions, msg.err = s.service.ListSessions(agentmodel.SessionFilters{Limit: 200})
		case pageTools:
			msg.tools, msg.err = s.service.GetTools()
		case pageSettings:
			msg.settings, msg.err = s.service.GetSettings()
		case pagePrivacy:
			msg.privacy, msg.err = s.service.GetPrivacyConfigs()
		default:
			msg.err = fmt.Errorf("unsupported page: %s", target.title())
		}
		sendMessage(ctx, ch, msg)
	}
}

func (s *state) loadDetail(id int64) command {
	s.loadSeq++
	seq := s.loadSeq
	s.loading = true
	s.err = nil
	return func(ctx context.Context, ch chan<- message) {
		detail, err := s.service.GetSessionDetail(id)
		sendMessage(ctx, ch, loadMsg{
			seq:    seq,
			page:   pageSessionDetail,
			detail: detail,
			err:    err,
		})
	}
}

func (s *state) index(rebuild bool) command {
	if s.indexing {
		s.status = "index already running"
		return nil
	}
	s.indexing = true
	if rebuild {
		s.status = "rebuilding index..."
	} else {
		s.status = "updating index..."
	}
	return func(ctx context.Context, ch chan<- message) {
		result, err := s.service.IndexNow(rebuild)
		sendMessage(ctx, ch, indexMsg{result: result, rebuild: rebuild, err: err})
	}
}

func sendMessage(ctx context.Context, ch chan<- message, msg message) {
	select {
	case <-ctx.Done():
	case ch <- msg:
	}
}

func (s *state) itemCount() int {
	switch s.page {
	case pageSessions:
		return len(s.sessions)
	case pageTools:
		return len(s.tools)
	case pageSessionDetail:
		if s.detail == nil {
			return 0
		}
		return len(sessionDetailLines(*s.detail, s.width))
	case pageSettings:
		return len(settingsLines(s.settings, s.width))
	case pagePrivacy:
		if status := s.selectedPrivacyStatus(); status != nil {
			return len(privacyDetailLines(*status, s.width))
		}
		return 0
	default:
		return 0
	}
}

func (s *state) pageStep() int {
	step := s.contentHeight() - 2
	if step < 1 {
		return 1
	}
	return step
}

func (s *state) move(delta int) {
	if delta == 0 {
		return
	}
	if s.page == pageSessionDetail || s.page == pageSettings || s.page == pagePrivacy {
		maxScroll := s.maxScroll()
		s.scroll += delta
		if s.scroll < 0 {
			s.scroll = 0
		}
		if s.scroll > maxScroll {
			s.scroll = maxScroll
		}
		return
	}
	s.moveTo(s.selected + delta)
}

func (s *state) moveTo(index int) {
	count := s.itemCount()
	if count <= 0 {
		s.selected = 0
		s.scroll = 0
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= count {
		index = count - 1
	}
	s.selected = index
	s.ensureVisible()
}

func (s *state) clampSelection(count int) {
	if count <= 0 {
		s.selected = 0
		s.scroll = 0
		return
	}
	if s.selected >= count {
		s.selected = count - 1
	}
	if s.selected < 0 {
		s.selected = 0
	}
	s.ensureVisible()
}

func (s *state) ensureVisible() {
	if s.page == pageSessionDetail || s.page == pageSettings || s.page == pagePrivacy {
		maxScroll := s.maxScroll()
		if s.scroll > maxScroll {
			s.scroll = maxScroll
		}
		if s.scroll < 0 {
			s.scroll = 0
		}
		return
	}
	visible := s.contentHeight() - 2
	if visible < 1 {
		visible = 1
	}
	if s.selected < s.scroll {
		s.scroll = s.selected
	}
	if s.selected >= s.scroll+visible {
		s.scroll = s.selected - visible + 1
	}
}

func (s *state) maxScroll() int {
	if s.page == pagePrivacy {
		return s.privacyMaxScroll()
	}
	max := s.itemCount() - s.contentHeight()
	if max < 0 {
		return 0
	}
	return max
}

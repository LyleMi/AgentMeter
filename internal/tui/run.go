package tui

import (
	"context"
	"errors"

	"AgentMeter/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

type teaModel struct {
	ctx   context.Context
	state *state
}

type contextDoneMsg struct{}

// Run starts AgentMeter's terminal UI against the shared application service.
func Run(ctx context.Context, service *app.App) error {
	if ctx == nil {
		ctx = context.Background()
	}
	model := teaModel{
		ctx:   ctx,
		state: newState(service, defaultWidth, defaultHeight),
	}
	_, err := tea.NewProgram(model, tea.WithContext(ctx), tea.WithAltScreen()).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, tea.ErrInterrupted) {
		return nil
	}
	return err
}

func (m teaModel) Init() tea.Cmd {
	return m.wrap(m.state.init())
}

func (m teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		cmd, quit := m.state.update(mapTeaKey(typed))
		if quit {
			return m, tea.Quit
		}
		return m, m.wrap(cmd)
	case tea.WindowSizeMsg:
		cmd, quit := m.state.update(resizeMsg{width: typed.Width, height: typed.Height})
		if quit {
			return m, tea.Quit
		}
		return m, m.wrap(cmd)
	case message:
		cmd, quit := m.state.update(typed)
		if quit {
			return m, tea.Quit
		}
		return m, m.wrap(cmd)
	case contextDoneMsg:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m teaModel) View() string {
	return m.state.view()
}

func (m teaModel) wrap(cmd command) tea.Cmd {
	if cmd == nil {
		return nil
	}
	return func() tea.Msg {
		ch := make(chan message, 1)
		cmd(m.ctx, ch)
		select {
		case msg := <-ch:
			return msg
		case <-m.ctx.Done():
			return contextDoneMsg{}
		}
	}
}

func mapTeaKey(key tea.KeyMsg) keyMsg {
	switch key.Type {
	case tea.KeyCtrlC:
		return keyMsg{typ: keyCtrlC}
	case tea.KeyEnter:
		return keyMsg{typ: keyEnter}
	case tea.KeyEsc:
		return keyMsg{typ: keyEsc}
	case tea.KeyTab:
		return keyMsg{typ: keyTab}
	case tea.KeyShiftTab:
		return keyMsg{typ: keyShiftTab}
	case tea.KeyUp:
		return keyMsg{typ: keyUp}
	case tea.KeyDown:
		return keyMsg{typ: keyDown}
	case tea.KeyLeft:
		return keyMsg{typ: keyLeft}
	case tea.KeyRight:
		return keyMsg{typ: keyRight}
	case tea.KeyPgUp:
		return keyMsg{typ: keyPageUp}
	case tea.KeyPgDown:
		return keyMsg{typ: keyPageDown}
	case tea.KeyHome:
		return keyMsg{typ: keyHome}
	case tea.KeyEnd:
		return keyMsg{typ: keyEnd}
	case tea.KeyRunes:
		runes := key.Runes
		if len(runes) > 0 {
			return keyMsg{typ: keyRune, ch: runes[0]}
		}
	}
	return keyMsg{typ: keyUnknown}
}

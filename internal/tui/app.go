package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/koo/aigotchi/internal/engine"
	"github.com/koo/aigotchi/internal/pet"
	"github.com/koo/aigotchi/internal/storage"
)

type screen int

const (
	screenMain      screen = iota
	screenEvolution screen = iota
	screenPlay      screen = iota
)

// Model is the main bubbletea model for aigotchi.
type Model struct {
	store   *storage.Store
	pet     pet.Pet
	state   engine.State
	screen  screen
	width   int
	height  int
	message string

	// play sub-model
	playModel *PlayModel
}

// NewModel creates a new Model, applies time delta, checks de/evolution, saves state.
func NewModel(store *storage.Store) (Model, error) {
	var p pet.Pet
	var s engine.State

	if err := store.ReadJSON("pet.json", &p); err != nil {
		return Model{}, fmt.Errorf("read pet: %w", err)
	}
	if err := store.ReadJSON("state.json", &s); err != nil {
		return Model{}, fmt.Errorf("read state: %w", err)
	}

	// Apply time delta
	delta := time.Since(s.LastUpdated)
	s = engine.ApplyTimeDelta(s, delta)
	s.LastUpdated = time.Now()

	// Check de-evolution first
	devolved, p, s := engine.CheckDeevolution(p, s)

	// Check evolution
	evolved, p, s := engine.CheckEvolution(p, s)

	var msg string
	switch {
	case devolved && evolved:
		msg = fmt.Sprintf("%s evolved!", p.Name)
	case evolved:
		msg = fmt.Sprintf("%s evolved to %s!", p.Name, p.Stage)
	case devolved:
		msg = fmt.Sprintf("%s de-evolved...", p.Name)
	}

	// Revive dormant if needed
	if p.Dormant {
		p, s = engine.ReviveDormant(p, s)
		msg = fmt.Sprintf("%s has been revived!", p.Name)
	}

	// Save updated state
	if err := store.WriteJSON("pet.json", p); err != nil {
		return Model{}, fmt.Errorf("write pet: %w", err)
	}
	if err := store.WriteJSON("state.json", s); err != nil {
		return Model{}, fmt.Errorf("write state: %w", err)
	}

	return Model{
		store:   store,
		pet:     p,
		state:   s,
		screen:  screenMain,
		message: msg,
	}, nil
}

// saveState persists the current pet and state.
func (m *Model) saveState() {
	m.state.LastUpdated = time.Now()
	_ = m.store.WriteJSON("pet.json", m.pet)
	_ = m.store.WriteJSON("state.json", m.state)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenPlay:
		return m.updatePlay(msg)
	case screenEvolution:
		return m.updateEvolution(msg)
	default:
		return m.updateMain(msg)
	}
}

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.saveState()
			return m, tea.Quit

		case "f":
			newState, err := engine.Feed(m.state)
			if err != nil {
				m.message = fmt.Sprintf("Cannot feed: %v", err)
			} else {
				m.state = newState
				m.message = fmt.Sprintf("Fed %s! Hunger: %d", m.pet.Name, m.state.Hunger)
				m.saveState()
			}

		case "p":
			// Switch to play screen
			m.screen = screenPlay
			pm := NewPlayModel()
			m.playModel = &pm
			return m, m.playModel.Init()
		}
	}

	return m, nil
}

func (m Model) updateEvolution(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "enter", " ":
			m.screen = screenMain
		}
	}
	return m, nil
}

func (m Model) updatePlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.playModel == nil {
		m.screen = screenMain
		return m, nil
	}

	updatedPlay, cmd := m.playModel.Update(msg)
	pm := updatedPlay.(PlayModel)
	m.playModel = &pm

	if pm.done {
		// Game finished — apply result
		m.state = engine.Play(m.state, pm.success)
		m.screen = screenMain
		m.playModel = nil

		if pm.success {
			m.message = fmt.Sprintf("Great job! Happiness: %d", m.state.Happiness)
		} else {
			m.message = fmt.Sprintf("Keep practicing! Happiness: %d", m.state.Happiness)
		}
		m.saveState()
		return m, nil
	}

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	switch m.screen {
	case screenPlay:
		if m.playModel != nil {
			return m.playModel.View()
		}
	case screenEvolution:
		return renderEvolutionView(m)
	}
	return renderMainView(m)
}

func renderEvolutionView(m Model) string {
	return fmt.Sprintf("\n  ✨ %s\n\n  Press Enter to continue.", m.message)
}

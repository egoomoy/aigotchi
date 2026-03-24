package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/koo/aigotchi/internal/engine"
	"github.com/koo/aigotchi/internal/renderer"
)

var (
	titleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true)
	stageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166"))
	traitStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#88d498"))
	msgStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f4845f")).Italic(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	controlStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
)

func renderMainView(m Model) string {
	var sb strings.Builder

	sb.WriteString("\n")

	// Pet ASCII art
	petArt := renderer.RenderPet(&m.pet)
	sb.WriteString(petArt)

	// Name, stage, personality
	personality := m.pet.Personality
	if personality == "" {
		personality = "mysterious"
	}
	header := fmt.Sprintf("  %s  %s  %s",
		titleStyle.Render(m.pet.Name),
		stageStyle.Render("["+m.pet.Stage.String()+"]"),
		dimStyle.Render("("+personality+")"),
	)
	sb.WriteString(header)
	sb.WriteString("\n\n")

	// Gauges
	gauges := renderer.RenderGauges(m.state.Hunger, m.state.Happiness, m.state.Health)
	// Indent gauges
	for _, line := range strings.Split(gauges, "\n") {
		sb.WriteString("  ")
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// XP
	xp := engine.CurrentXP(m.state)
	xpLine := fmt.Sprintf("  XP: %s  (tokens: %d earned / %d spent)",
		traitStyle.Render(fmt.Sprintf("%d", xp)),
		m.state.TotalTokensEarned,
		m.state.TotalTokensSpent,
	)
	sb.WriteString(xpLine)
	sb.WriteString("\n")

	// Traits
	if len(m.pet.Traits) > 0 {
		traits := fmt.Sprintf("  Traits: %s", traitStyle.Render(strings.Join(m.pet.Traits, ", ")))
		sb.WriteString(traits)
		sb.WriteString("\n")
	}

	// Rare trait
	if m.pet.Rare != nil {
		rare := fmt.Sprintf("  Rare: %s", lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166")).Bold(true).Render(*m.pet.Rare))
		sb.WriteString(rare)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// Message
	if m.message != "" {
		sb.WriteString("  ")
		sb.WriteString(msgStyle.Render(m.message))
		sb.WriteString("\n\n")
	}

	// Controls
	controls := controlStyle.Render("  [f] feed  [p] play  [q] quit")
	sb.WriteString(controls)
	sb.WriteString("\n")

	return sb.String()
}

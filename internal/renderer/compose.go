package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/koo/aigotchi/internal/pet"
)

const barSegments = 10

// bar renders a full 10-segment bar with filled (█) and empty (░) blocks.
func bar(value int) string {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	filled := (value * barSegments) / 100
	return strings.Repeat("█", filled) + strings.Repeat("░", barSegments-filled)
}

// miniBar renders a compact 3-character bar for the status line.
func miniBar(value int) string {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	filled := (value * 3) / 100
	return strings.Repeat("█", filled) + strings.Repeat("░", 3-filled)
}

// formatXP formats an XP value as a compact human-readable string.
// Values >= 100_000 are shown as "0.1M" etc.; values >= 1_000 as "12.4K".
func formatXP(xp int) string {
	if xp >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(xp)/1_000_000)
	}
	if xp >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(xp)/1_000)
	}
	return fmt.Sprintf("%d", xp)
}

// stageAbbrev returns a short bracket label for the pet's stage.
func stageAbbrev(s pet.Stage) string {
	switch s {
	case pet.StageEgg:
		return "Eg"
	case pet.StageBaby:
		return "Bb"
	case pet.StageJunior:
		return "Jr"
	case pet.StageSenior:
		return "Sr"
	case pet.StageLegend:
		return "Lg"
	default:
		return "??"
	}
}

// RenderGauges returns a 3-line string with colored labels and bar charts.
func RenderGauges(hunger, happiness, health int) string {
	hungerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166")).Bold(true)
	happyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#88d498")).Bold(true)
	healthStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f4845f")).Bold(true)
	barStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc"))

	hungerBar := barStyle.Render(bar(hunger))
	happyBar := barStyle.Render(bar(happiness))
	healthBar := barStyle.Render(bar(health))

	line1 := fmt.Sprintf("%s %s", hungerStyle.Render("Hunger"), hungerBar)
	line2 := fmt.Sprintf("%s  %s", happyStyle.Render("Happy"), happyBar)
	line3 := fmt.Sprintf("%s %s", healthStyle.Render("Health"), healthBar)

	return strings.Join([]string{line1, line2, line3}, "\n")
}

// RenderStatusLine returns a compact one-line status string.
// Format: [Sr] Mochi | H:██░ ☺:█░░ ♥:██░ | 12.4K xp
func RenderStatusLine(p *pet.Pet, hunger, happiness, health, xp int) string {
	stageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166")).Bold(true)
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true)
	xpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#88d498"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))

	abbrev := stageAbbrev(p.Stage)
	stageTag := stageStyle.Render(fmt.Sprintf("[%s]", abbrev))
	name := nameStyle.Render(p.Name)
	sep := dimStyle.Render("|")

	hBar := miniBar(hunger)
	smileBar := miniBar(happiness)
	heartBar := miniBar(health)

	gauges := fmt.Sprintf("H:%s ☺:%s ♥:%s", hBar, smileBar, heartBar)
	xpStr := xpStyle.Render(fmt.Sprintf("%s xp", formatXP(xp)))

	return fmt.Sprintf("%s %s %s %s %s %s", stageTag, name, sep, gauges, sep, xpStr)
}

package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	totalRounds    = 5
	totalSeconds   = 30
	successScore   = 3
)

var codeKeywords = []string{
	"func", "return", "import", "struct", "interface",
	"package", "var", "const", "type", "range",
	"defer", "goroutine", "channel", "select", "switch",
	"for", "if", "else", "map", "slice",
	"append", "make", "len", "cap", "nil",
}

// tickMsg fires every second for the timer.
type tickMsg time.Time

// PlayModel is the bubbletea model for the typing minigame.
type PlayModel struct {
	words       []string // all words to type in order
	current     int      // index of current word
	input       string   // current user input
	score       int      // correct words typed
	timeLeft    int      // seconds remaining
	done        bool     // game over
	success     bool     // did the player win?
	rng         *rand.Rand
}

func NewPlayModel() PlayModel {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Pick totalRounds unique words
	perm := rng.Perm(len(codeKeywords))
	words := make([]string, totalRounds)
	for i := 0; i < totalRounds; i++ {
		words[i] = codeKeywords[perm[i]]
	}

	return PlayModel{
		words:    words,
		current:  0,
		timeLeft: totalSeconds,
		rng:      rng,
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (pm PlayModel) Init() tea.Cmd {
	return tick()
}

func (pm PlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if pm.done {
		return pm, nil
	}

	switch msg := msg.(type) {
	case tickMsg:
		pm.timeLeft--
		if pm.timeLeft <= 0 {
			pm.done = true
			pm.success = pm.score >= successScore
			return pm, nil
		}
		return pm, tick()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			pm.done = true
			pm.success = pm.score >= successScore
			return pm, nil

		case tea.KeyEsc:
			pm.done = true
			pm.success = false
			return pm, nil

		case tea.KeyBackspace:
			if len(pm.input) > 0 {
				pm.input = pm.input[:len(pm.input)-1]
			}

		case tea.KeyEnter, tea.KeySpace:
			// Check if input matches current word
			if pm.current < len(pm.words) {
				if strings.TrimSpace(pm.input) == pm.words[pm.current] {
					pm.score++
				}
				pm.current++
				pm.input = ""
			}
			if pm.current >= len(pm.words) {
				pm.done = true
				pm.success = pm.score >= successScore
				return pm, nil
			}

		default:
			if msg.Type == tea.KeyRunes {
				pm.input += string(msg.Runes)
			}
		}
	}

	return pm, nil
}

func (pm PlayModel) View() string {
	var sb strings.Builder

	titleS := lipgloss.NewStyle().Foreground(lipgloss.Color("#c084fc")).Bold(true)
	keywordS := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166")).Bold(true)
	inputS := lipgloss.NewStyle().Foreground(lipgloss.Color("#88d498"))
	dimS := lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))
	successS := lipgloss.NewStyle().Foreground(lipgloss.Color("#88d498")).Bold(true)
	failS := lipgloss.NewStyle().Foreground(lipgloss.Color("#f4845f")).Bold(true)

	if pm.done {
		sb.WriteString("\n\n")
		if pm.success {
			sb.WriteString("  " + successS.Render("You win! Great typing!"))
		} else {
			sb.WriteString("  " + failS.Render("Game over! Better luck next time."))
		}
		sb.WriteString(fmt.Sprintf("\n\n  Score: %d / %d\n", pm.score, totalRounds))
		sb.WriteString("\n  " + dimS.Render("Returning to main screen..."))
		return sb.String()
	}

	sb.WriteString("\n")
	sb.WriteString("  " + titleS.Render("=== Typing Minigame ==="))
	sb.WriteString("\n\n")

	// Timer and score
	sb.WriteString(fmt.Sprintf("  Time: %s   Score: %s   Round: %s\n\n",
		dimS.Render(fmt.Sprintf("%ds", pm.timeLeft)),
		dimS.Render(fmt.Sprintf("%d/%d", pm.score, totalRounds)),
		dimS.Render(fmt.Sprintf("%d/%d", pm.current+1, totalRounds)),
	))

	// Show upcoming words
	for i, w := range pm.words {
		if i < pm.current {
			// Already typed
			sb.WriteString("  " + dimS.Render("✓ "+w) + "\n")
		} else if i == pm.current {
			// Current word
			sb.WriteString("  Type: " + keywordS.Render(w) + "\n\n")
			sb.WriteString("  > " + inputS.Render(pm.input) + dimS.Render("_") + "\n")
		}
	}

	sb.WriteString("\n  " + dimS.Render("[Enter/Space] submit  [Esc] quit"))
	sb.WriteString("\n")

	return sb.String()
}

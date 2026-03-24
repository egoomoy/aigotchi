package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/koo/aigotchi/internal/pet"
)

// ASCII art templates keyed by stage. Lines use %s placeholders where traits slot in.
var stageTemplates = map[pet.Stage][]string{
	pet.StageEgg: {
		"   _____  ",
		"  /     \\ ",
		" |       |",
		" |       |",
		"  \\_____/ ",
	},
	pet.StageBaby: {
		"  (•‿•)  ",
		" /(     )\\ ",
		"  | o o | ",
		"   \\___/  ",
	},
	pet.StageJunior: {
		"  (%s)  ",
		" /(     )\\ ",
		"  | ~ ~ | ",
		"   \\___/  ",
	},
	pet.StageSenior: {
		"   %s   ",
		"  (%s)  ",
		" /(     )\\ ",
		"  | ^ ^ | ",
		"   \\___/  ",
	},
	pet.StageLegend: {
		" ✦  ✧  ✦ ",
		"  (%s)  ",
		" /(     )\\ ",
		"  | ★ ★ | ",
		"   \\___/  ",
		" ✧  ✦  ✧ ",
	},
}

// eyeMap maps eye trait names to ASCII art glyphs.
var eyeMap = map[string]string{
	"neutral":   "°_°",
	"happy":     "◕‿◕",
	"surprised": "⊙_⊙",
	"skeptical": "≖_≖",
	"wide":      "◉‿◉",
	"relaxed":   "￣▽￣",
	// pet trait names from traits.go
	"round":   "°‿°",
	"narrow":  "≖_≖",
	"sparkle": "✦‿✦",
	"sleepy":  "－‿－",
	"fierce":  "◣_◢",
}

// accessoryMap maps accessory trait names to ASCII art glyphs.
var accessoryMap = map[string]string{
	"hat":     "🎩",
	"crown":   "♛",
	"hood":    "╱▔╲",
	"horns":   "∧ ∧",
	"bow":     "🎀",
	"scarf":   "〜〜",
	"glasses": "⌐■-■",
	"cape":    "⌒⌒⌒",
	"bandana": "~ ~ ~",
}

// colorMap maps color trait names to hex color codes.
var colorMap = map[string]string{
	"mint":     "#88d498",
	"coral":    "#f4845f",
	"lavender": "#c084fc",
	"gold":     "#ffd166",
	"crimson":  "#ef4444",
	"ice":      "#5bc0eb",
	"shadow":   "#6b7280",
	"neon":     "#22ff44",
	// pet trait names from traits.go
	"cobalt":   "#4a90d9",
	"emerald":  "#50c878",
	"amber":    "#ffbf00",
	"violet":   "#8b00ff",
	"ivory":    "#fffff0",
	"obsidian": "#1c1c1c",
	"rose":     "#ff007f",
}

// auraColorMap maps aura names to highlight colors.
var auraColorMap = map[string]string{
	"flame":  "#ff6b35",
	"frost":  "#5bc0eb",
	"storm":  "#9b59b6",
	"bloom":  "#ff85a1",
	"shadow": "#6b7280",
}

// defaultEyes is used when no eye trait is present.
const defaultEyes = "•‿•"

// defaultAccessory is used when no accessory trait is present.
const defaultAccessory = "    "

// RenderPet renders ASCII art for the pet with ANSI colors based on its traits.
func RenderPet(p *pet.Pet) string {
	traits := pet.BuildTraits(p.Seed, p.Stage)

	// Determine body color
	bodyColor := "#ffffff"
	if len(traits.All) > 0 {
		if c, ok := colorMap[traits.All[0]]; ok {
			bodyColor = c
		}
	}

	// Determine eyes glyph
	eyeGlyph := defaultEyes
	if len(traits.All) > 1 {
		if e, ok := eyeMap[traits.All[1]]; ok {
			eyeGlyph = e
		}
	}

	// Determine accessory glyph
	accessoryGlyph := defaultAccessory
	if len(traits.All) > 2 {
		if a, ok := accessoryMap[traits.All[2]]; ok {
			accessoryGlyph = a
		}
	}

	// Determine aura color (used for Legend stage highlights)
	auraColor := bodyColor
	if len(traits.All) > 3 {
		if c, ok := auraColorMap[traits.All[3]]; ok {
			auraColor = c
		}
	}

	tmpl, ok := stageTemplates[p.Stage]
	if !ok {
		tmpl = stageTemplates[pet.StageEgg]
	}

	lines := make([]string, len(tmpl))
	for i, line := range tmpl {
		switch p.Stage {
		case pet.StageJunior:
			if i == 0 {
				line = fmt.Sprintf(line, eyeGlyph)
			}
		case pet.StageSenior:
			if i == 0 {
				line = fmt.Sprintf(line, accessoryGlyph)
			} else if i == 1 {
				line = fmt.Sprintf(line, eyeGlyph)
			}
		case pet.StageLegend:
			if i == 1 {
				line = fmt.Sprintf(line, eyeGlyph)
			}
		}
		lines[i] = line
	}

	// Apply colors via lipgloss
	bodyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(bodyColor))
	auraStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(auraColor))

	var sb strings.Builder
	for i, line := range lines {
		var styled string
		// Legend stage: first and last lines use aura color
		if p.Stage == pet.StageLegend && (i == 0 || i == len(lines)-1) {
			styled = auraStyle.Render(line)
		} else {
			styled = bodyStyle.Render(line)
		}
		sb.WriteString(styled)
		sb.WriteString("\n")
	}

	return sb.String()
}

# aigotchi

Terminal Tamagotchi that grows with your Claude Code token usage.

```
     /     \
    |  °‿°  |
    |       |
     \_____/
     Mochi  [Baby]  (playful)

     Hunger ████████░░
     Happy  ██████░░░░
     Health ██████████

     XP: 15,200  (tokens: 15.2M earned / 40K spent)
```

## Install

```bash
brew tap egoomoy/tap
brew install aigotchi
```

Or build from source:

```bash
go build -o bin/aigotchi ./cmd/aigotchi
```

## Quick Start

```bash
aigotchi init          # Create pet + register Claude Code hook
aigotchi name Mochi    # Name your pet
aigotchi               # Launch interactive TUI
```

`aigotchi init` registers a [Stop hook](https://docs.anthropic.com/en/docs/claude-code/hooks) in `~/.claude/settings.json` that automatically collects tokens after each Claude Code session.

## Commands

| Command | Description |
|---------|-------------|
| `aigotchi` | Interactive TUI |
| `aigotchi init` | First-time setup + hook registration |
| `aigotchi status` | One-line status (for status bars) |
| `aigotchi feed` | Feed your pet (costs 10K tokens) |
| `aigotchi play` | Typing minigame for happiness |
| `aigotchi name <name>` | Rename your pet |
| `aigotchi collect` | Parse transcript & update tokens (called by hook) |
| `aigotchi reset` | Reset pet to Egg with zero tokens |
| `aigotchi version` | Print version |

## How It Works

### Token Economy

Claude Code sessions generate tokens. When a session ends, the Stop hook calls `aigotchi collect`, which parses the transcript and accumulates tokens.

```
XP = TotalTokensEarned / 1000
```

Feeding costs tokens. Play is free.

### Evolution Stages

| Stage | XP Required | Tokens | Traits Unlocked |
|-------|-------------|--------|-----------------|
| Egg | 0 | 0 | — |
| Baby | 10,000 | 10M | Body Color, Personality |
| Junior | 50,000 | 50M | Eyes |
| Senior | 150,000 | 150M | Accessory |
| Legend | 500,000 | 500M | Aura, Rare |

Evolution requires XP threshold **and** Health >= 50.

### Gauges

Three gauges decay over time when you're away:

- **Hunger**: -10 every 6 hours. Restore with `feed` (+30).
- **Happiness**: -10 every 8 hours. Restore with `play` (+30 on success, +10 on failure).
- **Health**: Decays (-10 / 3h) when Hunger hits 0. Recovers (+5 / 12h) when Hunger >= 30 and Happiness >= 30.

If Health reaches 0, the pet **de-evolves** one stage (Health resets to 50). An Egg at 0 Health goes **dormant** instead.

### Trait System

Each pet has a unique seed (set at creation) that deterministically generates traits via FNV-1a hashing. Same seed = same evolution path, every time.

**Trait pools:**

| Category | Options |
|----------|---------|
| Body Color | crimson, cobalt, emerald, amber, violet, ivory, obsidian, rose |
| Eyes | round, narrow, wide, sparkle, sleepy, fierce |
| Accessory | bow, scarf, hat, glasses, cape, crown, bandana |
| Aura | flame, frost, storm, bloom, shadow |
| Personality | bold, gentle, quirky, stoic, playful, mysterious |
| Rare | starborn, voidwalker, sunforged, moonbound, crystalheart |

**50,400 unique combinations** at Legend stage.

## Architecture

```
aigotchi/
├── cmd/aigotchi/
│   └── main.go              # CLI entry (Cobra), hook registration, collect logic
├── internal/
│   ├── pet/
│   │   ├── pet.go           # Pet struct, stages, XP thresholds
│   │   └── traits.go        # FNV-1a trait generation, trait pools
│   ├── engine/
│   │   ├── state.go         # Gauges, time-based decay/recovery
│   │   ├── interaction.go   # Feed & Play mechanics
│   │   └── evolution.go     # Evolution, de-evolution, dormancy
│   ├── collector/
│   │   └── collector.go     # Claude Code transcript JSONL parser
│   ├── renderer/
│   │   ├── ascii.go         # Stage-specific ASCII art + ANSI colors
│   │   └── compose.go       # Gauge bars, status line formatting
│   ├── storage/
│   │   └── storage.go       # ~/.aigotchi/ file I/O (atomic writes)
│   └── tui/
│       ├── app.go           # Bubbletea app model, screen routing
│       ├── main_view.go     # Main pet display screen
│       └── play_view.go     # Typing minigame (30s, 5 rounds)
└── .github/workflows/
    └── release.yml          # GoReleaser + Homebrew tap publishing
```

### Data Flow

```
Claude Code session ends
  → Stop hook fires
  → aigotchi collect --session-id <id>
  → Parse ~/.claude/projects/*/<id>.jsonl
  → Sum input/output/cache tokens from assistant messages
  → Append event to ~/.aigotchi/events.jsonl
  → Update state.json (TotalTokensEarned += tokens)
  → Check evolution
```

### Storage

All state lives in `~/.aigotchi/`:

| File | Format | Purpose |
|------|--------|---------|
| `pet.json` | JSON | Name, stage, seed, traits, personality, rare, dormant |
| `state.json` | JSON | Hunger, happiness, health, tokens earned/spent, last_updated |
| `events.jsonl` | JSONL | Append-only log of token collection events |

JSON writes use write-to-temp + rename (atomic). JSONL uses `O_APPEND`.

## Dependencies

- [Bubbletea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Cobra](https://github.com/spf13/cobra) — CLI framework

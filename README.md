# aigotchi

Terminal Tamagotchi that grows based on your Claude Code token usage.

## Install

go build -o bin/aigotchi ./cmd/aigotchi

## Quick Start

aigotchi init
aigotchi name Mochi
aigotchi

## Commands

- aigotchi — interactive TUI
- aigotchi status — one-line status (for agent-deck)
- aigotchi feed — feed your pet (costs XP)
- aigotchi play — typing minigame for happiness
- aigotchi name <name> — name your pet
- aigotchi stats — token usage statistics
- aigotchi init — first-time setup + hook registration

# Aigotchi Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a terminal Tamagotchi (aigotchi) that grows based on Claude Code token usage, with NFT-style random trait combinations, state management, and interactive TUI.

**Architecture:** File-based, daemon-less. Claude Code `Stop` hook feeds token events into `~/.aigotchi/events.jsonl`. On-demand TUI reads events, computes elapsed-time state changes, renders ANSI-colored ASCII pet. Go + Bubbletea/Lipgloss.

**Tech Stack:** Go 1.22+, Cobra (CLI), Bubbletea + Lipgloss + Bubbles (TUI), FNV-1a (trait hashing)

**Spec:** `docs/superpowers/specs/2026-03-24-aigotchi-design.md`

---

## File Structure

```
aigotchi/
├── cmd/aigotchi/
│   └── main.go                     # Cobra root command + subcommands
├── internal/
│   ├── storage/
│   │   └── storage.go              # Read/write ~/.aigotchi/ files (atomic writes)
│   ├── pet/
│   │   ├── pet.go                  # Pet struct, Stage enum, serialization
│   │   └── traits.go               # Trait definitions, seed-based selection
│   ├── engine/
│   │   ├── state.go                # Gauge decay/recovery, time-elapsed calculation
│   │   ├── evolution.go            # Evolution/de-evolution logic
│   │   └── interaction.go          # Feed, play commands
│   ├── collector/
│   │   └── collector.go            # Transcript JSONL parser, offset tracking
│   ├── renderer/
│   │   ├── ascii.go                # Stage-specific ASCII art templates
│   │   └── compose.go              # Trait layer composition + ANSI coloring
│   └── tui/
│       ├── app.go                  # Bubbletea main model
│       ├── main_view.go            # Main screen (pet + gauges + traits)
│       ├── evolution_view.go       # Evolution animation screen
│       ├── play_view.go            # Typing minigame screen
│       └── statusline.go           # Quick one-line status output
├── go.mod
├── go.sum
└── Makefile
```

---

## Task 1: Project Scaffolding & Storage Layer

**Files:**
- Create: `go.mod`, `Makefile`
- Create: `internal/storage/storage.go`
- Create: `internal/storage/storage_test.go`

- [ ] **Step 1: Initialize Go module and install dependencies**

```bash
cd /Users/koo/codecode/aigotchi
go mod init github.com/koo/aigotchi
go get github.com/spf13/cobra@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

- [ ] **Step 2: Create Makefile**

```makefile
.PHONY: build test run clean

build:
	go build -o bin/aigotchi ./cmd/aigotchi

test:
	go test ./... -v

run: build
	./bin/aigotchi

clean:
	rm -rf bin/
```

- [ ] **Step 3: Write failing test for storage**

Create `internal/storage/storage_test.go`:

```go
package storage_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/koo/aigotchi/internal/storage"
)

func TestNewStore_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".aigotchi")

	_, err := storage.NewStore(path)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("expected directory to be created")
	}
}

func TestStore_WriteAndReadJSON(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	type testData struct {
		Version int    `json:"version"`
		Name    string `json:"name"`
	}

	input := testData{Version: 1, Name: "Mochi"}
	err := store.WriteJSON("test.json", input)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output testData
	err = store.ReadJSON("test.json", &output)
	if err != nil {
		t.Fatalf("ReadJSON failed: %v", err)
	}

	if output.Name != "Mochi" || output.Version != 1 {
		t.Fatalf("expected {1, Mochi}, got {%d, %s}", output.Version, output.Name)
	}
}

func TestStore_AppendLine(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	err := store.AppendLine("events.jsonl", []byte(`{"tokens":100}`))
	if err != nil {
		t.Fatalf("AppendLine failed: %v", err)
	}
	err = store.AppendLine("events.jsonl", []byte(`{"tokens":200}`))
	if err != nil {
		t.Fatalf("AppendLine failed: %v", err)
	}

	lines, err := store.ReadLines("events.jsonl")
	if err != nil {
		t.Fatalf("ReadLines failed: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestStore_Exists(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	if store.Exists("nonexistent.json") {
		t.Fatal("expected file to not exist")
	}

	store.WriteJSON("exists.json", map[string]int{"version": 1})
	if !store.Exists("exists.json") {
		t.Fatal("expected file to exist")
	}
}
```

- [ ] **Step 4: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/storage/ -v`
Expected: FAIL — package does not exist

- [ ] **Step 5: Implement storage**

Create `internal/storage/storage.go`:

```go
package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) Path(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.Path(name))
	return err == nil
}

func (s *Store) WriteJSON(name string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	tmp := s.Path(name + ".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	return os.Rename(tmp, s.Path(name))
}

func (s *Store) ReadJSON(name string, v any) error {
	data, err := os.ReadFile(s.Path(name))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (s *Store) AppendLine(name string, line []byte) error {
	f, err := os.OpenFile(s.Path(name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	line = append(line, '\n')
	_, err = f.Write(line)
	return err
}

func (s *Store) ReadLines(name string) ([][]byte, error) {
	f, err := os.Open(s.Path(name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, append([]byte{}, scanner.Bytes()...))
	}
	return lines, scanner.Err()
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/storage/ -v`
Expected: All 4 tests PASS

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum Makefile internal/storage/
git commit -m "feat: add project scaffolding and storage layer"
```

---

## Task 2: Pet Model & Trait System

**Files:**
- Create: `internal/pet/pet.go`
- Create: `internal/pet/traits.go`
- Create: `internal/pet/pet_test.go`
- Create: `internal/pet/traits_test.go`

- [ ] **Step 1: Write failing test for Pet struct**

Create `internal/pet/pet_test.go`:

```go
package pet_test

import (
	"encoding/json"
	"testing"

	"github.com/koo/aigotchi/internal/pet"
)

func TestNewPet_HasSeedAndEggStage(t *testing.T) {
	p := pet.NewPet()
	if p.Stage != pet.StageEgg {
		t.Fatalf("expected StageEgg, got %d", p.Stage)
	}
	if p.Seed == 0 {
		t.Fatal("expected non-zero seed")
	}
	if p.Version != 1 {
		t.Fatal("expected version 1")
	}
}

func TestPet_JSONRoundtrip(t *testing.T) {
	p := pet.NewPet()
	p.Name = "Mochi"

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var p2 pet.Pet
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if p2.Name != "Mochi" || p2.Seed != p.Seed {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestStage_String(t *testing.T) {
	tests := []struct {
		stage pet.Stage
		want  string
		short string
	}{
		{pet.StageEgg, "Egg", "Eg"},
		{pet.StageBaby, "Baby", "Ba"},
		{pet.StageJunior, "Junior", "Jr"},
		{pet.StageSenior, "Senior", "Sr"},
		{pet.StageLegend, "Legend", "Lg"},
	}
	for _, tt := range tests {
		if got := tt.stage.String(); got != tt.want {
			t.Errorf("Stage(%d).String() = %s, want %s", tt.stage, got, tt.want)
		}
		if got := tt.stage.Short(); got != tt.short {
			t.Errorf("Stage(%d).Short() = %s, want %s", tt.stage, got, tt.short)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/pet/ -v`
Expected: FAIL

- [ ] **Step 3: Implement Pet model**

Create `internal/pet/pet.go`:

```go
package pet

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

type Stage int

const (
	StageEgg    Stage = 1
	StageBaby   Stage = 2
	StageJunior Stage = 3
	StageSenior Stage = 4
	StageLegend Stage = 5
)

func (s Stage) String() string {
	switch s {
	case StageEgg:
		return "Egg"
	case StageBaby:
		return "Baby"
	case StageJunior:
		return "Junior"
	case StageSenior:
		return "Senior"
	case StageLegend:
		return "Legend"
	default:
		return "Unknown"
	}
}

func (s Stage) Short() string {
	switch s {
	case StageEgg:
		return "Eg"
	case StageBaby:
		return "Ba"
	case StageJunior:
		return "Jr"
	case StageSenior:
		return "Sr"
	case StageLegend:
		return "Lg"
	default:
		return "??"
	}
}

// XP threshold to reach this stage (from spec)
func (s Stage) XPThreshold() int {
	switch s {
	case StageBaby:
		return 100
	case StageJunior:
		return 1_000
	case StageSenior:
		return 10_000
	case StageLegend:
		return 100_000
	default:
		return 0
	}
}

type Pet struct {
	Version     int       `json:"version"`
	Seed        int64     `json:"seed"`
	Name        string    `json:"name"`
	Stage       Stage     `json:"stage"`
	Traits      []string  `json:"traits"`
	Personality string    `json:"personality"`
	Rare        *string   `json:"rare"`
	Dormant     bool      `json:"dormant"`
	CreatedAt   time.Time `json:"created_at"`
}

func NewPet() *Pet {
	var seed int64
	binary.Read(rand.Reader, binary.LittleEndian, &seed)
	if seed < 0 {
		seed = -seed
	}
	return &Pet{
		Version:   1,
		Seed:      seed,
		Stage:     StageEgg,
		Traits:    []string{},
		CreatedAt: time.Now().UTC(),
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/pet/ -v -run TestNewPet -run TestPet_JSON -run TestStage`
Expected: PASS

- [ ] **Step 5: Write failing test for traits**

Create `internal/pet/traits_test.go`:

```go
package pet_test

import (
	"testing"

	"github.com/koo/aigotchi/internal/pet"
)

func TestTraitForStage_Deterministic(t *testing.T) {
	seed := int64(847291)

	// Same seed + stage should always produce the same trait
	t1 := pet.TraitForStage(seed, pet.StageBaby)
	t2 := pet.TraitForStage(seed, pet.StageBaby)

	if t1.BodyColor != t2.BodyColor {
		t.Fatalf("expected deterministic body color, got %s and %s", t1.BodyColor, t2.BodyColor)
	}
	if t1.Personality != t2.Personality {
		t.Fatalf("expected deterministic personality, got %s and %s", t1.Personality, t2.Personality)
	}
}

func TestTraitForStage_DifferentStagesProduceDifferent(t *testing.T) {
	seed := int64(847291)

	baby := pet.TraitForStage(seed, pet.StageBaby)
	junior := pet.TraitForStage(seed, pet.StageJunior)

	// Baby produces body color, Junior produces eyes — they come from different layers
	if baby.BodyColor == "" {
		t.Fatal("baby should have body color")
	}
	if junior.Eyes == "" {
		t.Fatal("junior should have eyes")
	}
}

func TestTraitForStage_AllOptionsValid(t *testing.T) {
	// Run through many seeds and verify all traits are from valid sets
	for seed := int64(0); seed < 1000; seed++ {
		st := pet.TraitForStage(seed, pet.StageBaby)
		if !pet.IsValidBodyColor(st.BodyColor) {
			t.Fatalf("invalid body color: %s (seed=%d)", st.BodyColor, seed)
		}
		if !pet.IsValidPersonality(st.Personality) {
			t.Fatalf("invalid personality: %s (seed=%d)", st.Personality, seed)
		}
	}
}

func TestBuildTraits_AccumulatesPerStage(t *testing.T) {
	seed := int64(12345)

	egg := pet.BuildTraits(seed, pet.StageEgg)
	if len(egg.All) != 0 {
		t.Fatalf("egg should have 0 traits, got %d", len(egg.All))
	}

	baby := pet.BuildTraits(seed, pet.StageBaby)
	if len(baby.All) != 1 {
		t.Fatalf("baby should have 1 trait, got %d", len(baby.All))
	}

	junior := pet.BuildTraits(seed, pet.StageJunior)
	if len(junior.All) != 2 {
		t.Fatalf("junior should have 2 traits, got %d", len(junior.All))
	}

	senior := pet.BuildTraits(seed, pet.StageSenior)
	if len(senior.All) != 3 {
		t.Fatalf("senior should have 3 traits, got %d", len(senior.All))
	}

	legend := pet.BuildTraits(seed, pet.StageLegend)
	if len(legend.All) != 4 {
		t.Fatalf("legend should have 4 traits, got %d", len(legend.All))
	}
	if legend.Rare == nil {
		t.Fatal("legend should have rare trait")
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/pet/ -v -run TestTrait`
Expected: FAIL

- [ ] **Step 7: Implement traits**

Create `internal/pet/traits.go`:

```go
package pet

import (
	"encoding/binary"
	"hash/fnv"
)

var BodyColors = []string{"mint", "coral", "lavender", "gold", "crimson", "ice", "shadow", "neon"}
var Eyes = []string{"neutral", "happy", "surprised", "skeptical", "wide", "relaxed"}
var Accessories = []string{"hat", "glasses", "cape", "crown", "hood", "wings", "horns"}
var Auras = []string{"sparkle", "electric", "fire", "ice", "crystal"}
var Personalities = []string{"chill", "hyper", "grumpy", "nerdy", "sleepy", "chaotic"}
var Rares = []string{"holographic", "glitch", "rainbow", "cosmic", "void"}

func IsValidBodyColor(s string) bool { return contains(BodyColors, s) }
func IsValidPersonality(s string) bool { return contains(Personalities, s) }

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// StageTrait holds the trait(s) gained at a specific evolution stage
type StageTrait struct {
	BodyColor   string
	Eyes        string
	Accessory   string
	Aura        string
	Personality string
	Rare        *string
}

// FullTraits holds accumulated traits up to a given stage
type FullTraits struct {
	BodyColor   string
	Eyes        string
	Accessory   string
	Aura        string
	Personality string
	Rare        *string
	All         []string // trait identifiers for serialization
}

func hashPick(seed int64, stage Stage, salt byte, options []string) string {
	h := fnv.New64a()
	buf := make([]byte, 9)
	binary.LittleEndian.PutUint64(buf, uint64(seed)+uint64(stage))
	buf[8] = salt
	h.Write(buf)
	idx := int(h.Sum64() % uint64(len(options)))
	return options[idx]
}

func TraitForStage(seed int64, stage Stage) StageTrait {
	var st StageTrait
	switch stage {
	case StageBaby:
		st.BodyColor = hashPick(seed, stage, 0, BodyColors)
		st.Personality = hashPick(seed, stage, 1, Personalities)
	case StageJunior:
		st.Eyes = hashPick(seed, stage, 0, Eyes)
	case StageSenior:
		st.Accessory = hashPick(seed, stage, 0, Accessories)
	case StageLegend:
		st.Aura = hashPick(seed, stage, 0, Auras)
		r := hashPick(seed, stage, 1, Rares)
		st.Rare = &r
	}
	return st
}

func BuildTraits(seed int64, upToStage Stage) FullTraits {
	var ft FullTraits
	if upToStage >= StageBaby {
		st := TraitForStage(seed, StageBaby)
		ft.BodyColor = st.BodyColor
		ft.Personality = st.Personality
		ft.All = append(ft.All, st.BodyColor)
	}
	if upToStage >= StageJunior {
		st := TraitForStage(seed, StageJunior)
		ft.Eyes = st.Eyes
		ft.All = append(ft.All, st.Eyes)
	}
	if upToStage >= StageSenior {
		st := TraitForStage(seed, StageSenior)
		ft.Accessory = st.Accessory
		ft.All = append(ft.All, st.Accessory)
	}
	if upToStage >= StageLegend {
		st := TraitForStage(seed, StageLegend)
		ft.Aura = st.Aura
		ft.Rare = st.Rare
		ft.All = append(ft.All, st.Aura)
	}
	return ft
}
```

Note: Both `pet.go` and `traits.go` are in the `pet` package. The `Stage` type and constants from `pet.go` are directly accessible in `traits.go`.

- [ ] **Step 8: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/pet/ -v`
Expected: All tests PASS

- [ ] **Step 9: Commit**

```bash
git add internal/pet/
git commit -m "feat: add pet model and trait system with FNV-1a seed hashing"
```

---

## Task 3: State Engine (Gauges, Decay, Recovery)

**Files:**
- Create: `internal/engine/state.go`
- Create: `internal/engine/state_test.go`

- [ ] **Step 1: Write failing test for gauge decay**

Create `internal/engine/state_test.go`:

```go
package engine_test

import (
	"testing"
	"time"

	"github.com/koo/aigotchi/internal/engine"
)

func TestApplyTimeDelta_HungerDecay(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 100
	s.Happiness = 100
	s.Health = 100

	// 6 hours should cause -10 hunger
	delta := 6 * time.Hour
	s = engine.ApplyTimeDelta(s, delta)

	if s.Hunger != 90 {
		t.Fatalf("expected hunger 90, got %d", s.Hunger)
	}
}

func TestApplyTimeDelta_HappinessDecay(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 100
	s.Happiness = 100
	s.Health = 100

	// 8 hours should cause -10 happiness
	delta := 8 * time.Hour
	s = engine.ApplyTimeDelta(s, delta)

	if s.Happiness != 90 {
		t.Fatalf("expected happiness 90, got %d", s.Happiness)
	}
}

func TestApplyTimeDelta_HealthDecayWhenHungerZero(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 0
	s.Happiness = 50
	s.Health = 100

	// 3 hours with hunger=0 should cause -10 health
	delta := 3 * time.Hour
	s = engine.ApplyTimeDelta(s, delta)

	if s.Health != 90 {
		t.Fatalf("expected health 90, got %d", s.Health)
	}
}

func TestApplyTimeDelta_HealthRecovery(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 50
	s.Happiness = 50
	s.Health = 80

	// 12 hours with hunger>=30 and happiness>=30 → +5 health
	// Also hunger decays: 12h = 2×6h = -20, happiness: 12h = 1×8h + 4h leftover = -10
	delta := 12 * time.Hour
	s = engine.ApplyTimeDelta(s, delta)

	// hunger: 50-20=30, happiness: 50-10=40, health: 80+5=85
	// But check health recovery condition per tick (hunger and happiness must be >=30 at time of recovery)
	if s.Health != 85 {
		t.Fatalf("expected health 85, got %d", s.Health)
	}
}

func TestApplyTimeDelta_GaugesClampAtZero(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 5
	s.Happiness = 5
	s.Health = 100

	delta := 24 * time.Hour
	s = engine.ApplyTimeDelta(s, delta)

	if s.Hunger < 0 {
		t.Fatalf("hunger should not go below 0, got %d", s.Hunger)
	}
	if s.Happiness < 0 {
		t.Fatalf("happiness should not go below 0, got %d", s.Happiness)
	}
}

func TestApplyTimeDelta_GaugesClampAt100(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 50
	s.Happiness = 50
	s.Health = 99

	// Even with recovery, health shouldn't exceed 100
	delta := 48 * time.Hour
	s = engine.ApplyTimeDelta(s, delta)

	if s.Health > 100 {
		t.Fatalf("health should not exceed 100, got %d", s.Health)
	}
}

func TestNewState_Defaults(t *testing.T) {
	s := engine.NewState()
	if s.Version != 1 {
		t.Fatalf("expected version 1, got %d", s.Version)
	}
	if s.Hunger != 100 || s.Happiness != 100 || s.Health != 100 {
		t.Fatal("new state should start at 100/100/100")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/engine/ -v`
Expected: FAIL

- [ ] **Step 3: Implement state engine**

Create `internal/engine/state.go`:

```go
package engine

import "time"

const (
	HungerDecayInterval   = 6 * time.Hour  // -10 per interval
	HungerDecayAmount     = 10
	HappyDecayInterval    = 8 * time.Hour  // -10 per interval
	HappyDecayAmount      = 10
	HealthDecayInterval   = 3 * time.Hour  // -10 when hunger=0
	HealthDecayAmount     = 10
	HealthRecoverInterval = 12 * time.Hour // +5 when hunger>=30 & happiness>=30
	HealthRecoverAmount   = 5
	FeedCost              = 10             // XP cost to feed
	FeedRestore           = 30             // Hunger points restored
	PlayRestore           = 30             // Happiness on success
	PlayRestoreMin        = 10             // Happiness on failure
)

type State struct {
	Version          int       `json:"version"`
	Hunger           int       `json:"hunger"`
	Happiness        int       `json:"happiness"`
	Health           int       `json:"health"`
	XP               int       `json:"xp"`
	TotalTokensEarned int64   `json:"total_tokens_earned"`
	TotalTokensSpent  int64   `json:"total_tokens_spent"`
	LastUpdated      time.Time `json:"last_updated"`
}

func NewState() *State {
	return &State{
		Version:     1,
		Hunger:      100,
		Happiness:   100,
		Health:      100,
		LastUpdated: time.Now().UTC(),
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ApplyTimeDelta simulates time passing using bulk interval computation.
// Decay is applied first (hunger, happiness), then health effects are computed
// based on the post-decay gauge values. This is an intentional simplification —
// exact per-hour tick simulation is unnecessary for the game feel.
func ApplyTimeDelta(s *State, delta time.Duration) *State {
	result := *s
	hours := int(delta.Hours())
	if hours <= 0 {
		return &result
	}

	// Apply hunger decay
	hungerTicks := int(delta / HungerDecayInterval)
	result.Hunger = clamp(result.Hunger-(hungerTicks*HungerDecayAmount), 0, 100)

	// Apply happiness decay
	happyTicks := int(delta / HappyDecayInterval)
	result.Happiness = clamp(result.Happiness-(happyTicks*HappyDecayAmount), 0, 100)

	// Health: decay if hunger=0 (after decay), recover if hunger>=30 & happiness>=30
	if result.Hunger == 0 {
		healthDecayTicks := int(delta / HealthDecayInterval)
		result.Health = clamp(result.Health-(healthDecayTicks*HealthDecayAmount), 0, 100)
	} else if result.Hunger >= 30 && result.Happiness >= 30 {
		healthRecoverTicks := int(delta / HealthRecoverInterval)
		result.Health = clamp(result.Health+(healthRecoverTicks*HealthRecoverAmount), 0, 100)
	}

	result.LastUpdated = s.LastUpdated.Add(delta)
	return &result
}

// AvailableTokens returns the token balance available for spending (feed).
func AvailableTokens(s *State) int64 {
	return s.TotalTokensEarned - s.TotalTokensSpent
}

// CurrentXP returns XP based on total tokens earned (1K tokens = 1 XP).
func (s *State) CurrentXP() int {
	return int(s.TotalTokensEarned / 1000)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/engine/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/state.go internal/engine/state_test.go
git commit -m "feat: add state engine with gauge decay and recovery"
```

---

## Task 4: Evolution & De-evolution Logic

**Files:**
- Create: `internal/engine/evolution.go`
- Create: `internal/engine/evolution_test.go`

- [ ] **Step 1: Write failing test for evolution**

Create `internal/engine/evolution_test.go`:

```go
package engine_test

import (
	"testing"

	"github.com/koo/aigotchi/internal/engine"
	"github.com/koo/aigotchi/internal/pet"
)

func TestCheckEvolution_EggToBaby(t *testing.T) {
	p := pet.NewPet()
	s := engine.NewState()
	s.TotalTokensEarned = 100_000 // 100 XP
	s.Health = 50

	evolved, newPet, newState := engine.CheckEvolution(p, s)
	if !evolved {
		t.Fatal("expected evolution to Baby")
	}
	if newPet.Stage != pet.StageBaby {
		t.Fatalf("expected StageBaby, got %d", newPet.Stage)
	}
	if len(newPet.Traits) != 1 {
		t.Fatalf("expected 1 trait, got %d", len(newPet.Traits))
	}
	if newPet.Personality == "" {
		t.Fatal("expected personality to be set")
	}
	_ = newState
}

func TestCheckEvolution_NotEnoughXP(t *testing.T) {
	p := pet.NewPet()
	s := engine.NewState()
	s.TotalTokensEarned = 50_000 // 50 XP, need 100

	evolved, _, _ := engine.CheckEvolution(p, s)
	if evolved {
		t.Fatal("should not evolve with insufficient XP")
	}
}

func TestCheckEvolution_TooLowHealth(t *testing.T) {
	p := pet.NewPet()
	s := engine.NewState()
	s.TotalTokensEarned = 100_000
	s.Health = 49 // need >= 50

	evolved, _, _ := engine.CheckEvolution(p, s)
	if evolved {
		t.Fatal("should not evolve with health < 50")
	}
}

func TestCheckDeevolution_HealthZero(t *testing.T) {
	p := pet.NewPet()
	p.Stage = pet.StageSenior
	p.Traits = []string{"coral", "happy", "crown"}

	s := engine.NewState()
	s.Health = 0

	devolved, newPet, newState := engine.CheckDeevolution(p, s)
	if !devolved {
		t.Fatal("expected de-evolution")
	}
	if newPet.Stage != pet.StageJunior {
		t.Fatalf("expected StageJunior, got %d", newPet.Stage)
	}
	if len(newPet.Traits) != 2 {
		t.Fatalf("expected 2 traits after de-evolution, got %d", len(newPet.Traits))
	}
	if newState.Health != 50 {
		t.Fatalf("expected health reset to 50, got %d", newState.Health)
	}
}

func TestCheckDeevolution_EggBecomesDormant(t *testing.T) {
	p := pet.NewPet()
	p.Stage = pet.StageEgg

	s := engine.NewState()
	s.Health = 0

	devolved, newPet, newState := engine.CheckDeevolution(p, s)
	if !devolved {
		t.Fatal("expected dormant")
	}
	if !newPet.Dormant {
		t.Fatal("expected egg to become dormant")
	}
	_ = newState
}

func TestReviveDormant_PreservesXP(t *testing.T) {
	p := pet.NewPet()
	p.Dormant = true

	s := engine.NewState()
	s.Health = 0
	s.TotalTokensEarned = 500_000 // should be preserved

	newPet, newState := engine.ReviveDormant(p, s)
	if newPet.Dormant {
		t.Fatal("expected dormant to be false")
	}
	if newState.Health != 50 {
		t.Fatalf("expected health 50, got %d", newState.Health)
	}
	if newState.TotalTokensEarned != 500_000 {
		t.Fatalf("expected XP preserved, got %d", newState.TotalTokensEarned)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/engine/ -v -run TestCheck`
Expected: FAIL

- [ ] **Step 3: Implement evolution logic**

Create `internal/engine/evolution.go`:

```go
package engine

import "github.com/koo/aigotchi/internal/pet"

// CheckEvolution returns true if the pet should evolve, along with updated pet and state.
func CheckEvolution(p *pet.Pet, s *State) (bool, *pet.Pet, *State) {
	if p.Stage >= pet.StageLegend {
		return false, p, s
	}
	if p.Dormant {
		return false, p, s
	}

	nextStage := p.Stage + 1
	threshold := nextStage.XPThreshold()
	currentXP := int(s.TotalTokensEarned / 1000)

	if currentXP < threshold || s.Health < 50 {
		return false, p, s
	}

	newPet := *p
	newPet.Stage = nextStage

	// Build traits up to new stage
	traits := pet.BuildTraits(p.Seed, nextStage)
	newPet.Traits = traits.All
	newPet.Personality = traits.Personality
	newPet.Rare = traits.Rare

	return true, &newPet, s
}

// CheckDeevolution returns true if the pet should de-evolve due to health=0.
func CheckDeevolution(p *pet.Pet, s *State) (bool, *pet.Pet, *State) {
	if s.Health > 0 {
		return false, p, s
	}

	newPet := *p
	newState := *s

	if p.Stage == pet.StageEgg {
		newPet.Dormant = true
		return true, &newPet, &newState
	}

	// De-evolve one stage
	newPet.Stage = p.Stage - 1

	// Rebuild traits for lower stage
	traits := pet.BuildTraits(p.Seed, newPet.Stage)
	newPet.Traits = traits.All
	newPet.Personality = traits.Personality
	newPet.Rare = traits.Rare

	// Reset health to 50
	newState.Health = 50

	return true, &newPet, &newState
}

// ReviveDormant revives a dormant egg when new token events arrive.
// Preserves existing state (XP, tokens) and only resets health.
func ReviveDormant(p *pet.Pet, s *State) (*pet.Pet, *State) {
	newPet := *p
	newPet.Dormant = false

	newState := *s
	newState.Health = 50
	newState.Hunger = 100
	newState.Happiness = 100

	return &newPet, &newState
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/engine/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/evolution.go internal/engine/evolution_test.go
git commit -m "feat: add evolution and de-evolution logic"
```

---

## Task 5: Interaction Commands (Feed, Play)

**Files:**
- Create: `internal/engine/interaction.go`
- Create: `internal/engine/interaction_test.go`

- [ ] **Step 1: Write failing test for feed**

Create `internal/engine/interaction_test.go`:

```go
package engine_test

import (
	"testing"

	"github.com/koo/aigotchi/internal/engine"
)

func TestFeed_Success(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 50
	s.TotalTokensEarned = 100_000 // 100 XP available

	newState, err := engine.Feed(s)
	if err != nil {
		t.Fatalf("feed failed: %v", err)
	}
	if newState.Hunger != 80 { // 50 + 30 = 80
		t.Fatalf("expected hunger 80, got %d", newState.Hunger)
	}
	if newState.TotalTokensSpent != 10_000 { // 10 XP = 10K tokens
		t.Fatalf("expected 10000 spent, got %d", newState.TotalTokensSpent)
	}
}

func TestFeed_InsufficientXP(t *testing.T) {
	s := engine.NewState()
	s.TotalTokensEarned = 5_000 // 5 XP, need 10

	_, err := engine.Feed(s)
	if err == nil {
		t.Fatal("expected error for insufficient XP")
	}
}

func TestFeed_HungerClampAt100(t *testing.T) {
	s := engine.NewState()
	s.Hunger = 90
	s.TotalTokensEarned = 100_000

	newState, _ := engine.Feed(s)
	if newState.Hunger > 100 {
		t.Fatalf("hunger should clamp at 100, got %d", newState.Hunger)
	}
}

func TestPlay_Success(t *testing.T) {
	s := engine.NewState()
	s.Happiness = 50

	newState := engine.Play(s, true)
	if newState.Happiness != 80 { // 50 + 30 = 80
		t.Fatalf("expected happiness 80, got %d", newState.Happiness)
	}
}

func TestPlay_Failure(t *testing.T) {
	s := engine.NewState()
	s.Happiness = 50

	newState := engine.Play(s, false)
	if newState.Happiness != 60 { // 50 + 10 = 60
		t.Fatalf("expected happiness 60, got %d", newState.Happiness)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/engine/ -v -run TestFeed -run TestPlay`
Expected: FAIL

- [ ] **Step 3: Implement interaction commands**

Create `internal/engine/interaction.go`:

```go
package engine

import "fmt"

func Feed(s *State) (*State, error) {
	available := s.TotalTokensEarned - s.TotalTokensSpent
	costInTokens := int64(FeedCost) * 1000

	if available < costInTokens {
		return nil, fmt.Errorf("insufficient XP: need %d, have %d", FeedCost, available/1000)
	}

	result := *s
	result.Hunger = clamp(result.Hunger+FeedRestore, 0, 100)
	result.TotalTokensSpent += costInTokens
	return &result, nil
}

func Play(s *State, success bool) *State {
	result := *s
	if success {
		result.Happiness = clamp(result.Happiness+PlayRestore, 0, 100)
	} else {
		result.Happiness = clamp(result.Happiness+PlayRestoreMin, 0, 100)
	}
	return &result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/engine/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/engine/interaction.go internal/engine/interaction_test.go
git commit -m "feat: add feed and play interaction commands"
```

---

## Task 6: Collector (Transcript Parser)

**Files:**
- Create: `internal/collector/collector.go`
- Create: `internal/collector/collector_test.go`
- Create: `internal/collector/testdata/sample.jsonl`

- [ ] **Step 1: Create test data**

Create `internal/collector/testdata/sample.jsonl` — a minimal Claude Code transcript with known token counts:

```jsonl
{"type":"human","message":{"content":"hello"},"timestamp":"2026-03-24T12:00:00Z"}
{"type":"assistant","message":{"model":"claude-opus-4-6[1m]","usage":{"input_tokens":100,"cache_creation_input_tokens":500,"cache_read_input_tokens":200,"output_tokens":50},"content":[{"type":"text","text":"hi"}]},"timestamp":"2026-03-24T12:00:01Z"}
{"type":"human","message":{"content":"write code"},"timestamp":"2026-03-24T12:01:00Z"}
{"type":"assistant","message":{"model":"claude-opus-4-6[1m]","usage":{"input_tokens":200,"cache_creation_input_tokens":0,"cache_read_input_tokens":800,"output_tokens":150},"content":[{"type":"text","text":"done"}]},"timestamp":"2026-03-24T12:01:01Z"}
```

Total tokens: (100+500+200+50) + (200+0+800+150) = 850 + 1150 = 2000

- [ ] **Step 2: Write failing test for collector**

Create `internal/collector/collector_test.go`:

```go
package collector_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/koo/aigotchi/internal/collector"
)

func testdataPath(name string) string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata", name)
}

func TestParseTranscript_TotalTokens(t *testing.T) {
	path := testdataPath("sample.jsonl")
	result, err := collector.ParseTranscript(path, 0)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if result.TotalTokens != 2000 {
		t.Fatalf("expected 2000 tokens, got %d", result.TotalTokens)
	}
	if result.MessageCount != 2 {
		t.Fatalf("expected 2 messages, got %d", result.MessageCount)
	}
	if result.NewOffset <= 0 {
		t.Fatal("expected positive new offset")
	}
}

func TestParseTranscript_WithOffset(t *testing.T) {
	path := testdataPath("sample.jsonl")

	// First pass — get offset after first message
	result1, _ := collector.ParseTranscript(path, 0)

	// Parse again from offset — should get 0 new tokens (already read everything)
	result2, err := collector.ParseTranscript(path, result1.NewOffset)
	if err != nil {
		t.Fatalf("parse with offset failed: %v", err)
	}
	if result2.TotalTokens != 0 {
		t.Fatalf("expected 0 new tokens, got %d", result2.TotalTokens)
	}
}

func TestParseTranscript_SkipsNonAssistant(t *testing.T) {
	path := testdataPath("sample.jsonl")
	result, _ := collector.ParseTranscript(path, 0)

	// Only 2 assistant messages, not the 2 human messages
	if result.MessageCount != 2 {
		t.Fatalf("expected 2 assistant messages, got %d", result.MessageCount)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/collector/ -v`
Expected: FAIL

- [ ] **Step 4: Implement collector**

Create `internal/collector/collector.go`:

```go
package collector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type ParseResult struct {
	TotalTokens  int64
	MessageCount int
	Model        string
	NewOffset    int64
}

type transcriptLine struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message"`
}

type assistantMessage struct {
	Model string `json:"model"`
	Usage struct {
		InputTokens              int64 `json:"input_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
	} `json:"usage"`
}

func ParseTranscript(path string, fromOffset int64) (*ParseResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	if fromOffset > 0 {
		if _, err := f.Seek(fromOffset, 0); err != nil {
			return nil, fmt.Errorf("seek: %w", err)
		}
	}

	result := &ParseResult{}
	var bytesRead int64
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for large lines

	for scanner.Scan() {
		line := scanner.Bytes()
		bytesRead += int64(len(line)) + 1 // +1 for newline

		var tl transcriptLine
		if err := json.Unmarshal(line, &tl); err != nil {
			continue // skip malformed lines
		}

		if tl.Type != "assistant" {
			continue
		}

		var msg assistantMessage
		if err := json.Unmarshal(tl.Message, &msg); err != nil {
			continue
		}

		tokens := msg.Usage.InputTokens +
			msg.Usage.CacheCreationInputTokens +
			msg.Usage.CacheReadInputTokens +
			msg.Usage.OutputTokens

		result.TotalTokens += tokens
		result.MessageCount++
		result.Model = msg.Model
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan: %w", err)
	}

	// Track offset manually since bufio.Scanner buffers ahead
	result.NewOffset = fromOffset + bytesRead

	return result, nil
}

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/collector/ -v`
Expected: All tests PASS (may need offset tracking fix — see note above)

- [ ] **Step 6: Commit**

```bash
git add internal/collector/
git commit -m "feat: add transcript JSONL parser for token collection"
```

---

## Task 7: ASCII Art Renderer

**Files:**
- Create: `internal/renderer/ascii.go`
- Create: `internal/renderer/compose.go`
- Create: `internal/renderer/ascii_test.go`

- [ ] **Step 1: Write failing test for renderer**

Create `internal/renderer/ascii_test.go`:

```go
package renderer_test

import (
	"strings"
	"testing"

	"github.com/koo/aigotchi/internal/pet"
	"github.com/koo/aigotchi/internal/renderer"
)

func TestRenderPet_Egg(t *testing.T) {
	p := &pet.Pet{Stage: pet.StageEgg}
	art := renderer.RenderPet(p)
	if art == "" {
		t.Fatal("expected non-empty art for egg")
	}
	// Egg should contain egg-like characters
	if !strings.Contains(art, "(") {
		t.Fatal("expected egg art to contain parentheses")
	}
}

func TestRenderPet_AllStages(t *testing.T) {
	stages := []pet.Stage{pet.StageEgg, pet.StageBaby, pet.StageJunior, pet.StageSenior, pet.StageLegend}
	for _, stage := range stages {
		p := &pet.Pet{
			Stage: stage,
			Seed:  12345,
		}
		if stage >= pet.StageBaby {
			traits := pet.BuildTraits(p.Seed, stage)
			p.Traits = traits.All
			p.Personality = traits.Personality
			p.Rare = traits.Rare
		}
		art := renderer.RenderPet(p)
		if art == "" {
			t.Fatalf("expected non-empty art for stage %s", stage)
		}
	}
}

func TestRenderGauges(t *testing.T) {
	output := renderer.RenderGauges(80, 60, 90)
	if !strings.Contains(output, "Hunger") {
		t.Fatal("expected Hunger label")
	}
	if !strings.Contains(output, "Happy") {
		t.Fatal("expected Happy label")
	}
	if !strings.Contains(output, "Health") {
		t.Fatal("expected Health label")
	}
}

func TestRenderStatusLine(t *testing.T) {
	p := &pet.Pet{Stage: pet.StageSenior, Name: "Mochi"}
	line := renderer.RenderStatusLine(p, 80, 60, 90, 12400)
	if !strings.Contains(line, "Sr") {
		t.Fatal("expected stage abbreviation")
	}
	if !strings.Contains(line, "Mochi") {
		t.Fatal("expected pet name")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/renderer/ -v`
Expected: FAIL

- [ ] **Step 3: Implement ASCII renderer**

Create `internal/renderer/ascii.go`:

```go
package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/koo/aigotchi/internal/pet"
)

var stageArt = map[pet.Stage][]string{
	pet.StageEgg: {
		"    ___   ",
		"   /   \\  ",
		"  | ··· | ",
		"   \\___/  ",
	},
	pet.StageBaby: {
		"   (•‿•)  ",
		"    /|\\   ",
		"    / \\   ",
	},
	pet.StageJunior: {
		"  ╭(%s)╮ ",
		"   /|██|\\  ",
		"    / \\    ",
	},
	pet.StageSenior: {
		"  %s       ",
		"  ╔(%s)╗  ",
		"  ║|██|║   ",
		"  ╚═/\\═╝  ",
	},
	pet.StageLegend: {
		" ★%s★     ",
		" ╔(%s)╗   ",
		" ║|████|║  ",
		" ╚══/\\══╝ ",
		"  ✦    ✦   ",
	},
}

var eyeMap = map[string]string{
	"neutral":   "°_°",
	"happy":     "◕‿◕",
	"surprised": "⊙_⊙",
	"skeptical": "≖_≖",
	"wide":      "◉‿◉",
	"relaxed":   "￣▽￣",
}

var accessoryTop = map[string]string{
	"hat":     "  🎩  ",
	"glasses": "",
	"cape":    "",
	"crown":   "  ♛   ",
	"hood":    " ╱▔╲  ",
	"wings":   "",
	"horns":   " ∧  ∧ ",
}

func RenderPet(p *pet.Pet) string {
	lines, ok := stageArt[p.Stage]
	if !ok {
		return "(???)"
	}

	// Clone lines
	result := make([]string, len(lines))
	copy(result, lines)

	if p.Stage >= pet.StageBaby && p.Stage != pet.StageEgg {
		traits := pet.BuildTraits(p.Seed, p.Stage)

		eyes := "°_°"
		if e, ok := eyeMap[traits.Eyes]; ok && traits.Eyes != "" {
			eyes = e
		}

		acc := ""
		if a, ok := accessoryTop[traits.Accessory]; ok && traits.Accessory != "" {
			acc = a
		}

		// Apply color
		color := traitColor(traits.BodyColor)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))

		for i, line := range result {
			line = strings.ReplaceAll(line, "%s", eyes)
			if i == 0 && acc != "" {
				line = strings.ReplaceAll(line, "%s", acc)
			}
			result[i] = style.Render(line)
		}
	}

	return strings.Join(result, "\n")
}

func traitColor(bodyColor string) string {
	colors := map[string]string{
		"mint":     "#88d498",
		"coral":    "#f4845f",
		"lavender": "#c084fc",
		"gold":     "#ffd166",
		"crimson":  "#ef4444",
		"ice":      "#5bc0eb",
		"shadow":   "#6b7280",
		"neon":     "#22ff44",
	}
	if c, ok := colors[bodyColor]; ok {
		return c
	}
	return "#c9d1d9"
}
```

Create `internal/renderer/compose.go`:

```go
package renderer

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/koo/aigotchi/internal/pet"
)

var (
	filledChar   = "█"
	emptyChar    = "░"
	labelHunger  = lipgloss.NewStyle().Foreground(lipgloss.Color("#f97583")).Render("Hunger")
	labelHappy   = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166")).Render("Happy ")
	labelHealth  = lipgloss.NewStyle().Foreground(lipgloss.Color("#58a6ff")).Render("Health")
)

func renderBar(value int) string {
	filled := value / 10
	empty := 10 - filled
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("#238636"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("#30363d"))
	return green.Render(strings.Repeat(filledChar, filled)) + dim.Render(strings.Repeat(emptyChar, empty))
}

func RenderGauges(hunger, happiness, health int) string {
	return fmt.Sprintf(
		"  %s  %s  %d%%\n  %s  %s  %d%%\n  %s  %s  %d%%",
		labelHunger, renderBar(hunger), hunger,
		labelHappy, renderBar(happiness), happiness,
		labelHealth, renderBar(health), health,
	)
}

func formatXP(xp int) string {
	if xp >= 100_000 {
		return fmt.Sprintf("%.1fM", float64(xp)/1_000_000)
	}
	if xp >= 1000 {
		return fmt.Sprintf("%.1fK", float64(xp)/1000)
	}
	return fmt.Sprintf("%d", xp)
}

func RenderStatusLine(p *pet.Pet, hunger, happiness, health, xp int) string {
	hBar := miniBar(hunger)
	sBar := miniBar(happiness)
	hpBar := miniBar(health)
	return fmt.Sprintf("[%s] %s | H:%s ☺:%s ♥:%s | %s xp",
		p.Stage.Short(), p.Name, hBar, sBar, hpBar, formatXP(xp))
}

func miniBar(value int) string {
	filled := value / 34 // 0-2 filled out of 3
	if value > 66 {
		filled = 3
	} else if value > 33 {
		filled = 2
	} else if value > 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", 3-filled)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/koo/codecode/aigotchi && go test ./internal/renderer/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/renderer/
git commit -m "feat: add ANSI ASCII art renderer with trait composition"
```

---

## Task 8: CLI Scaffolding (Cobra)

**Files:**
- Create: `cmd/aigotchi/main.go`

- [ ] **Step 1: Implement CLI with all subcommands**

Create `cmd/aigotchi/main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/koo/aigotchi/internal/collector"
	"github.com/koo/aigotchi/internal/engine"
	"github.com/koo/aigotchi/internal/pet"
	"github.com/koo/aigotchi/internal/renderer"
	"github.com/koo/aigotchi/internal/storage"
)

func aigotchiDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".aigotchi")
}

func loadStore() *storage.Store {
	s, err := storage.NewStore(aigotchiDir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return s
}

func loadPetAndState(store *storage.Store) (*pet.Pet, *engine.State) {
	var p pet.Pet
	if err := store.ReadJSON("pet.json", &p); err != nil {
		fmt.Fprintf(os.Stderr, "No pet found. Run 'aigotchi init' first.\n")
		os.Exit(1)
	}
	var s engine.State
	if err := store.ReadJSON("state.json", &s); err != nil {
		fmt.Fprintf(os.Stderr, "No state found. Run 'aigotchi init' first.\n")
		os.Exit(1)
	}
	return &p, &s
}

func savePetAndState(store *storage.Store, p *pet.Pet, s *engine.State) {
	store.WriteJSON("pet.json", p)
	store.WriteJSON("state.json", s)
}

func main() {
	root := &cobra.Command{
		Use:   "aigotchi",
		Short: "Your AI coding companion — a terminal Tamagotchi",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO: Task 9 — launch TUI
			store := loadStore()
			p, s := loadPetAndState(store)
			fmt.Println(renderer.RenderPet(p))
			fmt.Println()
			fmt.Println(renderer.RenderGauges(s.Hunger, s.Happiness, s.Health))
		},
	}

	root.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize aigotchi and register Claude Code hook",
		Run:   cmdInit,
	})

	root.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "One-line status (for agent-deck)",
		Run:   cmdStatus,
	})

	root.AddCommand(&cobra.Command{
		Use:   "feed",
		Short: "Feed your pet (costs XP)",
		Run:   cmdFeed,
	})

	collectCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect tokens from Claude Code transcript (called by hook)",
		Run:   cmdCollect,
	}
	collectCmd.Flags().String("session-id", "", "Claude Code session ID")
	collectCmd.Flags().String("cwd", "", "Working directory")
	root.AddCommand(collectCmd)

	root.AddCommand(&cobra.Command{
		Use:   "play",
		Short: "Play a minigame to boost happiness",
		Run:   cmdPlay,
	})

	root.AddCommand(&cobra.Command{
		Use:   "name [name]",
		Short: "Name your pet",
		Args:  cobra.ExactArgs(1),
		Run:   cmdName,
	})

	root.Execute()
}

func cmdInit(cmd *cobra.Command, args []string) {
	store := loadStore()

	if !store.Exists("pet.json") {
		p := pet.NewPet()
		store.WriteJSON("pet.json", p)
		s := engine.NewState()
		store.WriteJSON("state.json", s)
		fmt.Println("🥚 A new egg has appeared! Name it with: aigotchi name <name>")
	} else {
		fmt.Println("Pet already exists. Hook re-registered.")
	}

	// Register hook in Claude Code settings
	registerHook()
}

func registerHook() {
	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	var settings map[string]any
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		settings = make(map[string]any)
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]any)
		}
	}

	hooks, _ := settings["hooks"].(map[string]any)
	if hooks == nil {
		hooks = make(map[string]any)
	}

	// Preserve existing Stop hooks — only add aigotchi if not already present
	aigotchiCmd := "aigotchi collect --session-id $SESSION_ID --cwd $CWD"
	stopHooks, _ := hooks["Stop"].([]any)
	alreadyRegistered := false
	for _, h := range stopHooks {
		if hm, ok := h.(map[string]any); ok {
			if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "aigotchi collect") {
				alreadyRegistered = true
				break
			}
		}
	}
	if !alreadyRegistered {
		stopHooks = append(stopHooks, map[string]string{"command": aigotchiCmd})
		hooks["Stop"] = stopHooks
	}
	settings["hooks"] = hooks

	data, _ = json.MarshalIndent(settings, "", "  ")
	os.WriteFile(settingsPath, data, 0644)
	fmt.Println("✓ Claude Code hook registered")
}

func cmdStatus(cmd *cobra.Command, args []string) {
	store := loadStore()
	p, s := loadPetAndState(store)
	xp := int(s.TotalTokensEarned / 1000)
	fmt.Println(renderer.RenderStatusLine(p, s.Hunger, s.Happiness, s.Health, xp))
}

func cmdFeed(cmd *cobra.Command, args []string) {
	store := loadStore()
	p, s := loadPetAndState(store)

	newState, err := engine.Feed(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't feed: %v\n", err)
		os.Exit(1)
	}

	savePetAndState(store, p, newState)
	fmt.Printf("🍖 Fed %s! Hunger: %d → %d\n", p.Name, s.Hunger, newState.Hunger)
}

func cmdCollect(cmd *cobra.Command, args []string) {
	store, err := storage.NewStore(aigotchiDir())
	if err != nil {
		os.Exit(0) // silent exit if not initialized
	}
	if !store.Exists("pet.json") {
		os.Exit(0)
	}

	sessionID, _ := cmd.Flags().GetString("session-id")
	cwd, _ := cmd.Flags().GetString("cwd")
	_ = cwd

	// Find transcript file
	home, _ := os.UserHomeDir()
	// Claude Code stores transcripts in project-specific dirs
	// For now, search for the session JSONL
	transcriptPath := findTranscript(home, sessionID)
	if transcriptPath == "" {
		return
	}

	// Load collect state
	var collectState struct {
		Version int              `json:"version"`
		Offsets map[string]int64 `json:"offsets"`
	}
	collectState.Version = 1
	collectState.Offsets = make(map[string]int64)
	store.ReadJSON("collect.json", &collectState)

	offset := collectState.Offsets[transcriptPath]

	result, err := collector.ParseTranscript(transcriptPath, offset)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect error: %v\n", err)
		return
	}

	if result.TotalTokens > 0 {
		event := fmt.Sprintf(`{"ts":"%s","tokens":%d,"model":"%s","session":"%s"}`,
			time.Now().UTC().Format(time.RFC3339), result.TotalTokens, result.Model, sessionID)
		store.AppendLine("events.jsonl", []byte(event))

		// Update state.json with new tokens — this is what drives XP and evolution
		var s engine.State
		if err := store.ReadJSON("state.json", &s); err == nil {
			s.TotalTokensEarned += result.TotalTokens
			s.LastUpdated = time.Now().UTC()
			store.WriteJSON("state.json", &s)
		}
	}

	collectState.Offsets[transcriptPath] = result.NewOffset
	store.WriteJSON("collect.json", collectState)
}

func findTranscript(home, sessionID string) string {
	projectsDir := filepath.Join(home, ".claude", "projects")
	var found string
	filepath.Walk(projectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.Contains(info.Name(), sessionID) && strings.HasSuffix(info.Name(), ".jsonl") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	return found
}

func cmdPlay(cmd *cobra.Command, args []string) {
	// TODO: Task 10 — launch play TUI screen
	store := loadStore()
	_, s := loadPetAndState(store)
	fmt.Println("⌨ Minigame coming in TUI mode! For now, +10 happiness.")
	s = engine.Play(s, false)
	store.WriteJSON("state.json", s)
}

func cmdName(cmd *cobra.Command, args []string) {
	store := loadStore()
	p, s := loadPetAndState(store)
	p.Name = args[0]
	savePetAndState(store, p, s)
	fmt.Printf("Named your pet: %s\n", p.Name)
}

- [ ] **Step 2: Build and verify compilation**

Run: `cd /Users/koo/codecode/aigotchi && go build ./cmd/aigotchi/`
Expected: Compiles (after fixing imports)

- [ ] **Step 3: Test init command**

Run: `cd /Users/koo/codecode/aigotchi && go run ./cmd/aigotchi/ init`
Expected: Creates `~/.aigotchi/` with `pet.json` and `state.json`

- [ ] **Step 4: Test status command**

Run: `cd /Users/koo/codecode/aigotchi && go run ./cmd/aigotchi/ status`
Expected: One-line status output

- [ ] **Step 5: Commit**

```bash
git add cmd/
git commit -m "feat: add CLI scaffolding with init, status, feed, collect, name commands"
```

---

## Task 9: TUI (Bubbletea Main App)

**Files:**
- Create: `internal/tui/app.go`
- Create: `internal/tui/main_view.go`
- Modify: `cmd/aigotchi/main.go` — wire up root command to TUI

- [ ] **Step 1: Implement TUI app model**

Create `internal/tui/app.go`:

```go
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/koo/aigotchi/internal/engine"
	"github.com/koo/aigotchi/internal/pet"
	"github.com/koo/aigotchi/internal/storage"
)

type screen int

const (
	screenMain screen = iota
	screenEvolution
	screenPlay
)

type Model struct {
	store   *storage.Store
	pet     *pet.Pet
	state   *engine.State
	screen  screen
	width   int
	height  int
	message string
}

func NewModel(store *storage.Store, p *pet.Pet, s *engine.State) Model {
	// Apply time delta since last update
	now := time.Now().UTC()
	delta := now.Sub(s.LastUpdated)
	s = engine.ApplyTimeDelta(s, delta)

	// Check de-evolution
	if devolved, newPet, newState := engine.CheckDeevolution(p, s); devolved {
		p = newPet
		s = newState
	}

	// Check evolution
	if evolved, newPet, newState := engine.CheckEvolution(p, s); evolved {
		p = newPet
		s = newState
	}

	// Process pending events
	s = processEvents(store, s)

	// Save updated state
	store.WriteJSON("pet.json", p)
	store.WriteJSON("state.json", s)

	return Model{
		store: store,
		pet:   p,
		state: s,
		screen: screenMain,
	}
}

// processEvents is a no-op for now — the collect command already updates
// TotalTokensEarned in state.json when tokens arrive. The events.jsonl file
// serves as an audit log. Future enhancement: reconcile events.jsonl with
// state.json on startup to recover from partial writes.

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "f":
			newState, err := engine.Feed(m.state)
			if err != nil {
				m.message = "Not enough XP to feed!"
			} else {
				m.message = "Fed your pet! 🍖"
				m.state = newState
				m.store.WriteJSON("state.json", m.state)
			}
			return m, nil
		case "s":
			m.message = "Stats coming soon..."
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenMain:
		return renderMainView(m)
	default:
		return renderMainView(m)
	}
}
```

- [ ] **Step 2: Implement main view**

Create `internal/tui/main_view.go`:

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/koo/aigotchi/internal/renderer"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffd166"))
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#484f58")).
			Padding(1, 2)
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#484f58"))
	msgStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8b949e")).
			Italic(true)
)

func renderMainView(m Model) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Aigotchi"))
	b.WriteString("\n\n")

	// Pet art
	b.WriteString(renderer.RenderPet(m.pet))
	b.WriteString("\n\n")

	// Name + stage + personality
	name := m.pet.Name
	if name == "" {
		name = "???"
	}
	b.WriteString(fmt.Sprintf("  %s · %s", name, m.pet.Stage))
	if m.pet.Personality != "" {
		b.WriteString(fmt.Sprintf(" · %s", m.pet.Personality))
	}
	b.WriteString("\n\n")

	// Gauges
	b.WriteString(renderer.RenderGauges(m.state.Hunger, m.state.Happiness, m.state.Health))
	b.WriteString("\n\n")

	// XP
	xp := int(m.state.TotalTokensEarned / 1000)
	b.WriteString(fmt.Sprintf("  XP %d", xp))
	if m.pet.Stage.XPThreshold() > 0 {
		b.WriteString(fmt.Sprintf(" / %d", m.pet.Stage.XPThreshold()))
	}
	b.WriteString("\n")

	// Traits
	if len(m.pet.Traits) > 0 {
		b.WriteString(fmt.Sprintf("  Traits: %s\n", strings.Join(m.pet.Traits, " ")))
	}

	b.WriteString("\n")

	// Message
	if m.message != "" {
		b.WriteString(msgStyle.Render("  "+m.message))
		b.WriteString("\n\n")
	}

	// Controls
	b.WriteString(dimStyle.Render("  [f]eed  [p]lay  [s]tats  [q]uit"))
	b.WriteString("\n")

	return borderStyle.Render(b.String())
}
```

- [ ] **Step 3: Wire TUI into root command**

Modify `cmd/aigotchi/main.go` root command `Run` to:

```go
Run: func(cmd *cobra.Command, args []string) {
    store := loadStore()
    p, s := loadPetAndState(store)
    m := tui.NewModel(store, p, s)
    prog := tea.NewProgram(m, tea.WithAltScreen())
    if _, err := prog.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
        os.Exit(1)
    }
},
```

- [ ] **Step 4: Build and manually test**

Run: `cd /Users/koo/codecode/aigotchi && go build -o bin/aigotchi ./cmd/aigotchi/ && ./bin/aigotchi`
Expected: TUI opens with pet display, gauges, and keyboard controls. Press `q` to quit.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/ cmd/aigotchi/main.go
git commit -m "feat: add bubbletea TUI with main view"
```

---

## Task 10: Typing Minigame

**Files:**
- Create: `internal/tui/play_view.go`
- Modify: `internal/tui/app.go` — add play screen and transitions

- [ ] **Step 1: Implement typing game view**

Create `internal/tui/play_view.go`:

```go
package tui

import (
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var codeKeywords = []string{
	"func", "return", "import", "struct", "interface",
	"channel", "goroutine", "defer", "select", "switch",
	"package", "const", "type", "range", "append",
	"context", "error", "string", "slice", "mutex",
}

type playModel struct {
	target    string
	input     string
	timeLeft  time.Duration
	startTime time.Time
	done      bool
	success   bool
	round     int
	maxRounds int
	score     int
}

type tickMsg time.Time

func newPlayModel() playModel {
	return playModel{
		target:    codeKeywords[rand.Intn(len(codeKeywords))],
		timeLeft:  30 * time.Second,
		startTime: time.Now(),
		maxRounds: 5,
		round:     1,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (p playModel) Update(msg tea.Msg) (playModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		elapsed := time.Since(p.startTime)
		p.timeLeft = 30*time.Second - elapsed
		if p.timeLeft <= 0 {
			p.done = true
			p.success = p.score >= 3
		}
		return p, tickCmd()
	case tea.KeyMsg:
		if p.done {
			return p, nil
		}
		switch msg.Type {
		case tea.KeyBackspace:
			if len(p.input) > 0 {
				p.input = p.input[:len(p.input)-1]
			}
		case tea.KeyEnter:
			if p.input == p.target {
				p.score++
			}
			p.round++
			if p.round > p.maxRounds {
				p.done = true
				p.success = p.score >= 3
			} else {
				p.target = codeKeywords[rand.Intn(len(codeKeywords))]
				p.input = ""
			}
		default:
			if msg.Type == tea.KeyRunes {
				p.input += string(msg.Runes)
			}
		}
	}
	return p, nil
}

func (p playModel) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffd166"))
	targetStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#58a6ff")).Width(20).Align(lipgloss.Center)
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#c9d1d9"))

	b.WriteString(titleStyle.Render("⌨ Typing Game"))
	b.WriteString("\n\n")

	if p.done {
		if p.success {
			b.WriteString("  🎉 Great job! Score: " + fmt.Sprintf("%d/%d", p.score, p.maxRounds))
		} else {
			b.WriteString("  😅 Nice try! Score: " + fmt.Sprintf("%d/%d", p.score, p.maxRounds))
		}
		b.WriteString("\n\n  Press any key to return...")
	} else {
		b.WriteString(fmt.Sprintf("  Round %d/%d  |  Time: %.0fs\n\n", p.round, p.maxRounds, p.timeLeft.Seconds()))
		b.WriteString("  Type: ")
		b.WriteString(targetStyle.Render(p.target))
		b.WriteString("\n\n")
		b.WriteString("  > ")
		b.WriteString(inputStyle.Render(p.input))
		b.WriteString("█")
	}

	return b.String()
}
```

Note: Needs `fmt` import. The implementing agent should add it.

- [ ] **Step 2: Add play screen to app.go**

Add `screenPlay` to the `screen` enum. In `Update`, handle `"p"` key to switch to play screen. Add play model to `Model` struct. Handle play-done transition back to main screen, applying happiness change via `engine.Play()`.

- [ ] **Step 3: Build and test minigame**

Run: `cd /Users/koo/codecode/aigotchi && go run ./cmd/aigotchi/`
Then press `p` to start the minigame.
Expected: Typing game starts, shows keywords, accepts input, returns to main screen.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/
git commit -m "feat: add typing minigame for happiness recovery"
```

---

## Task 11: Integration Test & Polish

**Files:**
- Create: `internal/integration_test.go`
- Modify: various — fix any compilation/runtime issues

- [ ] **Step 1: Write integration test for full lifecycle**

Create `internal/integration_test.go`:

```go
package internal_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/koo/aigotchi/internal/engine"
	"github.com/koo/aigotchi/internal/pet"
	"github.com/koo/aigotchi/internal/renderer"
	"github.com/koo/aigotchi/internal/storage"
)

func TestFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	// 1. Create pet
	p := pet.NewPet()
	p.Name = "TestPet"
	s := engine.NewState()

	// 2. Simulate token earning (enough for Baby evolution)
	s.TotalTokensEarned = 150_000 // 150 XP > 100 threshold

	// 3. Check evolution
	evolved, p, s := engine.CheckEvolution(p, s)
	if !evolved {
		t.Fatal("expected evolution to Baby")
	}
	if p.Stage != pet.StageBaby {
		t.Fatalf("expected Baby, got %s", p.Stage)
	}

	// 4. Render pet (should not panic)
	art := renderer.RenderPet(p)
	if art == "" {
		t.Fatal("expected non-empty art")
	}

	// 5. Feed
	s2, err := engine.Feed(s)
	if err != nil {
		t.Fatalf("feed failed: %v", err)
	}
	if s2.TotalTokensSpent != 10_000 {
		t.Fatalf("expected 10000 spent, got %d", s2.TotalTokensSpent)
	}

	// 6. Time decay
	s3 := engine.ApplyTimeDelta(s2, 12*time.Hour)
	if s3.Hunger >= s2.Hunger {
		t.Fatal("expected hunger to decrease over time")
	}

	// 7. Save and reload
	store.WriteJSON("pet.json", p)
	store.WriteJSON("state.json", s3)

	var loadedPet pet.Pet
	store.ReadJSON("pet.json", &loadedPet)
	if loadedPet.Name != "TestPet" || loadedPet.Stage != pet.StageBaby {
		t.Fatal("reload mismatch")
	}

	// 8. Status line
	status := renderer.RenderStatusLine(p, s3.Hunger, s3.Happiness, s3.Health, int(s3.TotalTokensEarned/1000))
	if status == "" {
		t.Fatal("expected non-empty status line")
	}
}
```

- [ ] **Step 2: Run all tests**

Run: `cd /Users/koo/codecode/aigotchi && go test ./... -v`
Expected: All tests PASS

- [ ] **Step 3: Build final binary**

Run: `cd /Users/koo/codecode/aigotchi && make build && ./bin/aigotchi --help`
Expected: Help output showing all commands

- [ ] **Step 4: Commit**

```bash
git add internal/integration_test.go
git commit -m "feat: add integration test for full pet lifecycle"
```

---

## Task 12: README & .gitignore

**Files:**
- Create: `README.md`
- Create: `.gitignore`

- [ ] **Step 1: Create .gitignore**

```
bin/
.superpowers/
*.exe
```

- [ ] **Step 2: Create README**

Minimal README with:
- Project name and one-line description
- `go install` / `go build` instructions
- `aigotchi init` quickstart
- List of commands

- [ ] **Step 3: Commit**

```bash
git add .gitignore README.md
git commit -m "docs: add README and gitignore"
```

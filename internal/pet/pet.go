package pet

import (
	"crypto/rand"
	"encoding/binary"
	"time"
)

// Stage represents the evolution stage of a pet.
type Stage int

const (
	StageEgg    Stage = 1
	StageBaby   Stage = 2
	StageJunior Stage = 3
	StageSenior Stage = 4
	StageLegend Stage = 5
)

// String returns the full name of the stage.
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

// Short returns an emoji representation of the stage.
func (s Stage) Short() string {
	switch s {
	case StageEgg:
		return "🥚"
	case StageBaby:
		return "🐣"
	case StageJunior:
		return "🐥"
	case StageSenior:
		return "🐦"
	case StageLegend:
		return "🦅"
	default:
		return "?"
	}
}

// XPThreshold returns the minimum XP required to reach this stage.
func (s Stage) XPThreshold() int {
	switch s {
	case StageEgg:
		return 0
	case StageBaby:
		return 10000
	case StageJunior:
		return 50000
	case StageSenior:
		return 150000
	case StageLegend:
		return 500000
	default:
		return 0
	}
}

// Pet represents a virtual pet with its state and traits.
type Pet struct {
	Version     int       `json:"version"`
	Seed        int64     `json:"seed"`
	Name        string    `json:"name"`
	Stage       Stage     `json:"stage"`
	Traits      []string  `json:"traits"`
	Personality string    `json:"personality"`
	Rare        *string   `json:"rare,omitempty"`
	Dormant     bool      `json:"dormant"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewPet creates a new Pet at StageEgg with a random seed.
func NewPet(name string) Pet {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// Fallback: use time-based seed if crypto/rand fails
		binary.LittleEndian.PutUint64(buf[:], uint64(time.Now().UnixNano()))
	}
	seed := int64(binary.LittleEndian.Uint64(buf[:]))
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	return Pet{
		Version:   1,
		Seed:      seed,
		Name:      name,
		Stage:     StageEgg,
		Traits:    []string{},
		CreatedAt: time.Now(),
	}
}

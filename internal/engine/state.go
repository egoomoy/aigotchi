package engine

import "time"

// Decay/recovery intervals and amounts.
const (
	HungerDecayInterval = 6 * time.Hour
	HungerDecayAmount   = 10

	HappyDecayInterval = 8 * time.Hour
	HappyDecayAmount   = 10

	// HealthDecayInterval applies when Hunger == 0.
	HealthDecayInterval = 3 * time.Hour
	HealthDecayAmount   = 10

	// HealthRecoverInterval applies when Hunger >= 30 && Happiness >= 30.
	HealthRecoverInterval = 12 * time.Hour
	HealthRecoverAmount   = 5

	FeedCost        = 10
	FeedRestore     = 30
	PlayRestore     = 30
	PlayRestoreMin  = 10
)

// State holds all runtime gauges for a pet.
type State struct {
	Version           int       `json:"version"`
	Hunger            int       `json:"hunger"`
	Happiness         int       `json:"happiness"`
	Health            int       `json:"health"`
	XP                int       `json:"xp"`
	TotalTokensEarned int64     `json:"total_tokens_earned"`
	TotalTokensSpent  int64     `json:"total_tokens_spent"`
	LastUpdated       time.Time `json:"last_updated"`
}

// NewState returns a State with all gauges at maximum defaults.
func NewState() State {
	return State{
		Version:     1,
		Hunger:      100,
		Happiness:   100,
		Health:      100,
		LastUpdated: time.Now(),
	}
}

// clamp constrains v to [0, 100].
func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// ApplyTimeDelta applies bulk decay/recovery over a duration delta.
// Hunger and happiness decay first; health effects are derived from the
// post-decay values.
func ApplyTimeDelta(s State, delta time.Duration) State {
	if delta <= 0 {
		return s
	}

	// Hunger decay
	hungerTicks := int(delta / HungerDecayInterval)
	s.Hunger = clamp(s.Hunger - hungerTicks*HungerDecayAmount)

	// Happiness decay
	happyTicks := int(delta / HappyDecayInterval)
	s.Happiness = clamp(s.Happiness - happyTicks*HappyDecayAmount)

	// Health effects based on post-decay hunger/happiness
	if s.Hunger == 0 {
		healthDecayTicks := int(delta / HealthDecayInterval)
		s.Health = clamp(s.Health - healthDecayTicks*HealthDecayAmount)
	} else if s.Hunger >= 30 && s.Happiness >= 30 {
		healthRecoverTicks := int(delta / HealthRecoverInterval)
		s.Health = clamp(s.Health + healthRecoverTicks*HealthRecoverAmount)
	}

	return s
}

// AvailableTokens returns tokens available to spend.
func AvailableTokens(s State) int64 {
	return s.TotalTokensEarned - s.TotalTokensSpent
}

// CurrentXP returns the pet's XP derived from total tokens earned.
func CurrentXP(s State) int {
	return int(s.TotalTokensEarned / 1000)
}

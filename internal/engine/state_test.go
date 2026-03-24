package engine

import (
	"testing"
	"time"
)

func TestNewStateDefaults(t *testing.T) {
	s := NewState()
	if s.Hunger != 100 {
		t.Errorf("expected Hunger=100, got %d", s.Hunger)
	}
	if s.Happiness != 100 {
		t.Errorf("expected Happiness=100, got %d", s.Happiness)
	}
	if s.Health != 100 {
		t.Errorf("expected Health=100, got %d", s.Health)
	}
}

func TestHungerDecay(t *testing.T) {
	s := NewState()
	s.Hunger = 100
	s = ApplyTimeDelta(s, 6*time.Hour)
	if s.Hunger != 90 {
		t.Errorf("expected Hunger=90 after 6h, got %d", s.Hunger)
	}
}

func TestHappinessDecay(t *testing.T) {
	s := NewState()
	s.Happiness = 100
	s = ApplyTimeDelta(s, 8*time.Hour)
	if s.Happiness != 90 {
		t.Errorf("expected Happiness=90 after 8h, got %d", s.Happiness)
	}
}

func TestHealthDecayWhenHungerZero(t *testing.T) {
	s := NewState()
	s.Hunger = 0
	s.Happiness = 100
	s.Health = 100
	s = ApplyTimeDelta(s, 3*time.Hour)
	if s.Health != 90 {
		t.Errorf("expected Health=90 after 3h with hunger=0, got %d", s.Health)
	}
}

func TestHealthRecovery(t *testing.T) {
	s := NewState()
	s.Hunger = 80
	s.Happiness = 80
	s.Health = 80
	s = ApplyTimeDelta(s, 12*time.Hour)
	// health recover: +5 after 12h
	if s.Health != 85 {
		t.Errorf("expected Health=85 after 12h recovery, got %d", s.Health)
	}
}

func TestClampAtZero(t *testing.T) {
	s := NewState()
	s.Hunger = 5
	// 6h per tick, 2 ticks = -20; 5 - 20 = -15 → clamped to 0
	s = ApplyTimeDelta(s, 12*time.Hour)
	if s.Hunger != 0 {
		t.Errorf("expected Hunger clamped to 0, got %d", s.Hunger)
	}
}

func TestClampAtOneHundred(t *testing.T) {
	if got := clamp(110); got != 100 {
		t.Errorf("expected clamp(110)=100, got %d", got)
	}
	if got := clamp(-5); got != 0 {
		t.Errorf("expected clamp(-5)=0, got %d", got)
	}
}

func TestAvailableTokens(t *testing.T) {
	s := NewState()
	s.TotalTokensEarned = 50000
	s.TotalTokensSpent = 10000
	if got := AvailableTokens(s); got != 40000 {
		t.Errorf("expected AvailableTokens=40000, got %d", got)
	}
}

func TestCurrentXP(t *testing.T) {
	s := NewState()
	s.TotalTokensEarned = 150000
	if got := CurrentXP(s); got != 150 {
		t.Errorf("expected CurrentXP=150, got %d", got)
	}
}

package engine

import (
	"testing"
)

func TestFeedSuccess(t *testing.T) {
	s := NewState()
	s.Hunger = 50
	s.TotalTokensEarned = 100 * 1000 // 100 XP, 100000 tokens
	s.TotalTokensSpent = 0

	newState, err := Feed(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newState.Hunger != 80 {
		t.Errorf("expected Hunger=80 after feed, got %d", newState.Hunger)
	}
	// FeedCost=10 XP = 10000 tokens spent
	if newState.TotalTokensSpent != 10000 {
		t.Errorf("expected TotalTokensSpent=10000, got %d", newState.TotalTokensSpent)
	}
}

func TestFeedInsufficientTokens(t *testing.T) {
	s := NewState()
	s.Hunger = 50
	s.TotalTokensEarned = 5000 // only 5 XP worth
	s.TotalTokensSpent = 0

	_, err := Feed(s)
	if err == nil {
		t.Error("expected error for insufficient tokens, got nil")
	}
}

func TestFeedHungerClampAt100(t *testing.T) {
	s := NewState()
	s.Hunger = 90 // 90 + 30 = 120, should clamp to 100
	s.TotalTokensEarned = 100 * 1000
	s.TotalTokensSpent = 0

	newState, err := Feed(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newState.Hunger != 100 {
		t.Errorf("expected Hunger clamped to 100, got %d", newState.Hunger)
	}
}

func TestPlaySuccess(t *testing.T) {
	s := NewState()
	s.Happiness = 50

	newState := Play(s, true)
	if newState.Happiness != 80 {
		t.Errorf("expected Happiness=80 after successful play, got %d", newState.Happiness)
	}
}

func TestPlayFailure(t *testing.T) {
	s := NewState()
	s.Happiness = 50

	newState := Play(s, false)
	if newState.Happiness != 60 {
		t.Errorf("expected Happiness=60 after failed play, got %d", newState.Happiness)
	}
}

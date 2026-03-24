package engine

import "errors"

// ErrInsufficientTokens is returned when a pet action costs more tokens than available.
var ErrInsufficientTokens = errors.New("insufficient tokens")

// Feed attempts to feed the pet, consuming FeedCost*1000 tokens and restoring
// FeedRestore hunger points. Returns an error if the pet cannot afford the cost.
func Feed(s State) (State, error) {
	cost := int64(FeedCost * 1000)
	if AvailableTokens(s) < cost {
		return s, ErrInsufficientTokens
	}
	s.TotalTokensSpent += cost
	s.Hunger = clamp(s.Hunger + FeedRestore)
	return s, nil
}

// Play applies a happiness boost to the pet. On success the full PlayRestore is
// applied; on failure only PlayRestoreMin is added.
func Play(s State, success bool) State {
	if success {
		s.Happiness = clamp(s.Happiness + PlayRestore)
	} else {
		s.Happiness = clamp(s.Happiness + PlayRestoreMin)
	}
	return s
}

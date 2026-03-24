package engine

import (
	"testing"

	"github.com/egoomoy/aigotchi/internal/pet"
)

func TestEggToBabyEvolution(t *testing.T) {
	p := pet.NewPet("Tester")
	s := NewState()
	// Need XP >= Baby threshold (10000)
	s.TotalTokensEarned = 10000 * 1000
	s.Health = 80

	evolved, newPet, _ := CheckEvolution(p, s)
	if !evolved {
		t.Fatal("expected evolution from Egg to Baby")
	}
	if newPet.Stage != pet.StageBaby {
		t.Errorf("expected StageBaby, got %v", newPet.Stage)
	}
	if len(newPet.Traits) == 0 {
		t.Error("expected traits to be set after evolution")
	}
}

func TestInsufficientXPBlocksEvolution(t *testing.T) {
	p := pet.NewPet("Tester")
	s := NewState()
	s.TotalTokensEarned = 5000 * 1000 // XP = 5000, need 10000
	s.Health = 80

	evolved, _, _ := CheckEvolution(p, s)
	if evolved {
		t.Error("should not evolve with insufficient XP")
	}
}

func TestLowHealthBlocksEvolution(t *testing.T) {
	p := pet.NewPet("Tester")
	s := NewState()
	s.TotalTokensEarned = 20000 * 1000 // XP = 20000, enough for Baby
	s.Health = 40                     // below 50

	evolved, _, _ := CheckEvolution(p, s)
	if evolved {
		t.Error("should not evolve with health < 50")
	}
}

func TestLegendDoesNotEvolve(t *testing.T) {
	p := pet.NewPet("Tester")
	p.Stage = pet.StageLegend
	s := NewState()
	s.TotalTokensEarned = 600000 * 1000
	s.Health = 100

	evolved, _, _ := CheckEvolution(p, s)
	if evolved {
		t.Error("Legend should not evolve further")
	}
}

func TestDormantDoesNotEvolve(t *testing.T) {
	p := pet.NewPet("Tester")
	p.Dormant = true
	s := NewState()
	s.TotalTokensEarned = 20000 * 1000
	s.Health = 80

	evolved, _, _ := CheckEvolution(p, s)
	if evolved {
		t.Error("Dormant pet should not evolve")
	}
}

func TestDeevolutionRemovesLastTrait(t *testing.T) {
	p := pet.NewPet("Tester")
	p.Stage = pet.StageJunior
	traits := pet.BuildTraits(p.Seed, pet.StageJunior)
	p.Traits = traits.All
	p.Personality = traits.Personality

	s := NewState()
	s.Health = 0
	s.TotalTokensEarned = 60000 * 1000

	devolved, newPet, newState := CheckDeevolution(p, s)
	if !devolved {
		t.Fatal("expected de-evolution from Junior to Baby")
	}
	if newPet.Stage != pet.StageBaby {
		t.Errorf("expected StageBaby after de-evolution, got %v", newPet.Stage)
	}
	// Junior had 2 traits (BodyColor + Eyes); after de-evolution to Baby should have 1
	if len(newPet.Traits) != 1 {
		t.Errorf("expected 1 trait after de-evolution, got %d", len(newPet.Traits))
	}
	if newState.Health != 50 {
		t.Errorf("expected Health reset to 50, got %d", newState.Health)
	}
}

func TestEggBecomesDormantOnHealthZero(t *testing.T) {
	p := pet.NewPet("Tester")
	p.Stage = pet.StageEgg
	s := NewState()
	s.Health = 0

	devolved, newPet, _ := CheckDeevolution(p, s)
	if !devolved {
		t.Fatal("expected de-evolution result for egg at health=0")
	}
	if !newPet.Dormant {
		t.Error("expected Egg to become Dormant when health=0")
	}
}

func TestReviveDormantPreservesXP(t *testing.T) {
	p := pet.NewPet("Tester")
	p.Dormant = true
	s := NewState()
	s.TotalTokensEarned = 60000 * 1000
	s.TotalTokensSpent = 50 * 1000
	s.Health = 0
	s.Hunger = 10
	s.Happiness = 10

	newPet, newState := ReviveDormant(p, s)
	if newPet.Dormant {
		t.Error("pet should no longer be dormant after revive")
	}
	if newState.Health != 50 {
		t.Errorf("expected Health=50 after revive, got %d", newState.Health)
	}
	if newState.Hunger != 100 {
		t.Errorf("expected Hunger=100 after revive, got %d", newState.Hunger)
	}
	if newState.Happiness != 100 {
		t.Errorf("expected Happiness=100 after revive, got %d", newState.Happiness)
	}
	if newState.TotalTokensEarned != 60000*1000 {
		t.Errorf("expected TotalTokensEarned preserved at 60000000, got %d", newState.TotalTokensEarned)
	}
}

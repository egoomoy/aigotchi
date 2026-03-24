package pet_test

import (
	"testing"

	"github.com/koo/aigotchi/internal/pet"
)

func TestTraitForStageDeterministic(t *testing.T) {
	seed := int64(42)

	// Same seed + stage should always yield the same result
	for i := 0; i < 10; i++ {
		r1 := pet.TraitForStage(seed, pet.StageBaby)
		r2 := pet.TraitForStage(seed, pet.StageBaby)
		if r1.BodyColor != r2.BodyColor {
			t.Errorf("non-deterministic body color: %s vs %s", r1.BodyColor, r2.BodyColor)
		}
		if r1.Personality != r2.Personality {
			t.Errorf("non-deterministic personality: %s vs %s", r1.Personality, r2.Personality)
		}
	}
}

func TestTraitForStageDifferentStagesDifferentLayers(t *testing.T) {
	seed := int64(99)

	baby := pet.TraitForStage(seed, pet.StageBaby)
	junior := pet.TraitForStage(seed, pet.StageJunior)
	senior := pet.TraitForStage(seed, pet.StageSenior)
	legend := pet.TraitForStage(seed, pet.StageLegend)

	// Each stage should provide different fields
	// Baby has BodyColor + Personality
	if baby.BodyColor == "" {
		t.Error("Baby should have a BodyColor")
	}
	if baby.Personality == "" {
		t.Error("Baby should have a Personality")
	}

	// Junior has Eyes
	if junior.Eyes == "" {
		t.Error("Junior should have Eyes")
	}

	// Senior has Accessory
	if senior.Accessory == "" {
		t.Error("Senior should have an Accessory")
	}

	// Legend has Aura + Rare
	if legend.Aura == "" {
		t.Error("Legend should have an Aura")
	}
	if legend.Rare == nil {
		t.Error("Legend should have a Rare trait")
	}
}

func TestBuildTraitsAccumulation(t *testing.T) {
	seed := int64(123)

	egg := pet.BuildTraits(seed, pet.StageEgg)
	baby := pet.BuildTraits(seed, pet.StageBaby)
	junior := pet.BuildTraits(seed, pet.StageJunior)
	senior := pet.BuildTraits(seed, pet.StageSenior)
	legend := pet.BuildTraits(seed, pet.StageLegend)

	if len(egg.All) != 0 {
		t.Errorf("Egg should have 0 traits, got %d", len(egg.All))
	}
	if len(baby.All) != 1 {
		t.Errorf("Baby should have 1 trait, got %d", len(baby.All))
	}
	if len(junior.All) != 2 {
		t.Errorf("Junior should have 2 traits, got %d", len(junior.All))
	}
	if len(senior.All) != 3 {
		t.Errorf("Senior should have 3 traits, got %d", len(senior.All))
	}
	if len(legend.All) != 4 {
		t.Errorf("Legend should have 4 traits, got %d", len(legend.All))
	}
}

func TestBuildTraitsConsistencyAcrossStages(t *testing.T) {
	seed := int64(777)

	junior := pet.BuildTraits(seed, pet.StageJunior)
	legend := pet.BuildTraits(seed, pet.StageLegend)

	// Traits accumulated at earlier stages should remain the same
	if junior.All[0] != legend.All[0] {
		t.Errorf("BodyColor should be consistent: %s vs %s", junior.All[0], legend.All[0])
	}
	if junior.All[1] != legend.All[1] {
		t.Errorf("Eyes should be consistent: %s vs %s", junior.All[1], legend.All[1])
	}
}

func TestAllSeedsProduceValidTraits(t *testing.T) {
	validBodyColors := map[string]bool{}
	validPersonalities := map[string]bool{}

	for _, bc := range pet.BodyColors {
		validBodyColors[bc] = true
	}
	for _, p := range pet.Personalities {
		validPersonalities[p] = true
	}

	for i := int64(0); i < 1000; i++ {
		result := pet.TraitForStage(i, pet.StageBaby)
		if !validBodyColors[result.BodyColor] {
			t.Errorf("seed %d produced invalid BodyColor: %q", i, result.BodyColor)
		}
		if !validPersonalities[result.Personality] {
			t.Errorf("seed %d produced invalid Personality: %q", i, result.Personality)
		}
	}
}

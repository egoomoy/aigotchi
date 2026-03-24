package engine

import "github.com/koo/aigotchi/internal/pet"

// nextStage returns the stage after p.Stage, or p.Stage if already at max.
func nextStage(stage pet.Stage) pet.Stage {
	switch stage {
	case pet.StageEgg:
		return pet.StageBaby
	case pet.StageBaby:
		return pet.StageJunior
	case pet.StageJunior:
		return pet.StageSenior
	case pet.StageSenior:
		return pet.StageLegend
	default:
		return stage
	}
}

// prevStage returns the stage before p.Stage, or p.Stage if already at min.
func prevStage(stage pet.Stage) pet.Stage {
	switch stage {
	case pet.StageBaby:
		return pet.StageEgg
	case pet.StageJunior:
		return pet.StageBaby
	case pet.StageSenior:
		return pet.StageJunior
	case pet.StageLegend:
		return pet.StageSenior
	default:
		return stage
	}
}

// applyTraits writes the accumulated trait data from BuildTraits into the pet.
func applyTraits(p pet.Pet, stage pet.Stage) pet.Pet {
	t := pet.BuildTraits(p.Seed, stage)
	p.Traits = t.All
	p.Personality = t.Personality
	p.Rare = t.Rare
	return p
}

// CheckEvolution checks whether the pet should evolve given its current state.
// Returns (evolved, newPet, newState). If not eligible, the originals are returned unchanged.
func CheckEvolution(p pet.Pet, s State) (bool, pet.Pet, State) {
	// Legend and Dormant pets cannot evolve.
	if p.Stage == pet.StageLegend || p.Dormant {
		return false, p, s
	}

	next := nextStage(p.Stage)
	xp := CurrentXP(s)

	if xp < next.XPThreshold() {
		return false, p, s
	}
	if s.Health < 50 {
		return false, p, s
	}

	p.Stage = next
	p = applyTraits(p, next)

	return true, p, s
}

// CheckDeevolution checks whether the pet should de-evolve (health == 0).
// Returns (devolved, newPet, newState). If not eligible, originals returned unchanged.
func CheckDeevolution(p pet.Pet, s State) (bool, pet.Pet, State) {
	if s.Health != 0 {
		return false, p, s
	}

	if p.Stage == pet.StageEgg {
		// Egg becomes dormant
		p.Dormant = true
		return true, p, s
	}

	prev := prevStage(p.Stage)
	p.Stage = prev
	p = applyTraits(p, prev)
	s.Health = 50

	return true, p, s
}

// ReviveDormant wakes a dormant pet, resetting its gauges while preserving XP.
func ReviveDormant(p pet.Pet, s State) (pet.Pet, State) {
	p.Dormant = false
	s.Health = 50
	s.Hunger = 100
	s.Happiness = 100
	// TotalTokensEarned is preserved by not modifying it.
	return p, s
}

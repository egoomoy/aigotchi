package pet

// Trait option slices — exported for test validation.
var BodyColors = []string{
	"crimson", "cobalt", "emerald", "amber",
	"violet", "ivory", "obsidian", "rose",
}

var Eyes = []string{
	"round", "narrow", "wide", "sparkle", "sleepy", "fierce",
}

var Accessories = []string{
	"bow", "scarf", "hat", "glasses", "cape", "crown", "bandana",
}

var Auras = []string{
	"flame", "frost", "storm", "bloom", "shadow",
}

var Personalities = []string{
	"bold", "gentle", "quirky", "stoic", "playful", "mysterious",
}

var Rares = []string{
	"starborn", "voidwalker", "sunforged", "moonbound", "crystalheart",
}

// hashPick computes an FNV-1a hash over (seed, stage, salt) and picks an index.
func hashPick(seed int64, stage Stage, salt uint64, options int) int {
	const fnvOffset uint64 = 14695981039346656037
	const fnvPrime uint64 = 1099511628211

	h := fnvOffset

	// Mix in seed bytes
	s := uint64(seed)
	for i := 0; i < 8; i++ {
		h ^= s & 0xff
		h *= fnvPrime
		s >>= 8
	}

	// Mix in stage
	h ^= uint64(stage)
	h *= fnvPrime

	// Mix in salt
	salt64 := salt
	for i := 0; i < 8; i++ {
		h ^= salt64 & 0xff
		h *= fnvPrime
		salt64 >>= 8
	}

	return int(h % uint64(options))
}

// StageTrait holds the traits unlocked at a particular stage.
type StageTrait struct {
	BodyColor   string
	Eyes        string
	Accessory   string
	Aura        string
	Personality string
	Rare        *string
}

// TraitForStage computes the traits assigned at a specific stage using the pet's seed.
// It is deterministic: same seed + stage always yields the same result.
func TraitForStage(seed int64, stage Stage) StageTrait {
	var t StageTrait
	switch stage {
	case StageBaby:
		t.BodyColor = BodyColors[hashPick(seed, stage, 1, len(BodyColors))]
		t.Personality = Personalities[hashPick(seed, stage, 2, len(Personalities))]
	case StageJunior:
		t.Eyes = Eyes[hashPick(seed, stage, 3, len(Eyes))]
	case StageSenior:
		t.Accessory = Accessories[hashPick(seed, stage, 4, len(Accessories))]
	case StageLegend:
		t.Aura = Auras[hashPick(seed, stage, 5, len(Auras))]
		rare := Rares[hashPick(seed, stage, 6, len(Rares))]
		t.Rare = &rare
	}
	return t
}

// Traits holds the accumulated trait list for a pet at a given stage.
type Traits struct {
	// All contains one trait per evolution step in order:
	// index 0 = BodyColor (Baby), 1 = Eyes (Junior), 2 = Accessory (Senior), 3 = Aura (Legend)
	All         []string
	Personality string
	Rare        *string
}

// BuildTraits accumulates traits from Baby up to (and including) the given stage.
// Egg stage returns zero traits; each subsequent stage adds one entry to All.
func BuildTraits(seed int64, stage Stage) Traits {
	var t Traits
	t.All = []string{}

	if stage < StageBaby {
		return t
	}

	// Baby: BodyColor + Personality
	baby := TraitForStage(seed, StageBaby)
	t.All = append(t.All, baby.BodyColor)
	t.Personality = baby.Personality

	if stage < StageJunior {
		return t
	}

	// Junior: Eyes
	junior := TraitForStage(seed, StageJunior)
	t.All = append(t.All, junior.Eyes)

	if stage < StageSenior {
		return t
	}

	// Senior: Accessory
	senior := TraitForStage(seed, StageSenior)
	t.All = append(t.All, senior.Accessory)

	if stage < StageLegend {
		return t
	}

	// Legend: Aura + Rare
	legend := TraitForStage(seed, StageLegend)
	t.All = append(t.All, legend.Aura)
	t.Rare = legend.Rare

	return t
}

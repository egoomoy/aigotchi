package pet_test

import (
	"encoding/json"
	"testing"

	"github.com/koo/aigotchi/internal/pet"
)

func TestNewPet(t *testing.T) {
	p := pet.NewPet("Taro")

	if p.Stage != pet.StageEgg {
		t.Errorf("expected StageEgg, got %v", p.Stage)
	}
	if p.Seed == 0 {
		t.Error("expected non-zero seed")
	}
	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}
	if p.Name != "Taro" {
		t.Errorf("expected name Taro, got %s", p.Name)
	}
}

func TestPetJSONRoundtrip(t *testing.T) {
	p := pet.NewPet("Mochi")

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var p2 pet.Pet
	if err := json.Unmarshal(data, &p2); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if p.Name != p2.Name {
		t.Errorf("name mismatch: %s vs %s", p.Name, p2.Name)
	}
	if p.Seed != p2.Seed {
		t.Errorf("seed mismatch: %d vs %d", p.Seed, p2.Seed)
	}
	if p.Stage != p2.Stage {
		t.Errorf("stage mismatch: %v vs %v", p.Stage, p2.Stage)
	}
	if p.Version != p2.Version {
		t.Errorf("version mismatch: %d vs %d", p.Version, p2.Version)
	}
}

func TestStageString(t *testing.T) {
	cases := []struct {
		stage    pet.Stage
		str      string
		short    string
	}{
		{pet.StageEgg, "Egg", "🥚"},
		{pet.StageBaby, "Baby", "🐣"},
		{pet.StageJunior, "Junior", "🐥"},
		{pet.StageSenior, "Senior", "🐦"},
		{pet.StageLegend, "Legend", "🦅"},
	}

	for _, c := range cases {
		if got := c.stage.String(); got != c.str {
			t.Errorf("Stage(%d).String() = %q, want %q", c.stage, got, c.str)
		}
		if got := c.stage.Short(); got != c.short {
			t.Errorf("Stage(%d).Short() = %q, want %q", c.stage, got, c.short)
		}
	}
}

func TestStageXPThreshold(t *testing.T) {
	cases := []struct {
		stage     pet.Stage
		threshold int
	}{
		{pet.StageEgg, 0},
		{pet.StageBaby, 100},
		{pet.StageJunior, 300},
		{pet.StageSenior, 600},
		{pet.StageLegend, 1000},
	}

	for _, c := range cases {
		if got := c.stage.XPThreshold(); got != c.threshold {
			t.Errorf("Stage(%d).XPThreshold() = %d, want %d", c.stage, got, c.threshold)
		}
	}
}

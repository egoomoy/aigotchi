package renderer_test

import (
	"strings"
	"testing"

	"github.com/koo/aigotchi/internal/pet"
	"github.com/koo/aigotchi/internal/renderer"
)

func TestRenderPetEggNonEmpty(t *testing.T) {
	p := &pet.Pet{
		Name:  "Taro",
		Stage: pet.StageEgg,
		Seed:  12345,
	}
	got := renderer.RenderPet(p)
	if strings.TrimSpace(got) == "" {
		t.Error("RenderPet for Egg returned empty string")
	}
}

func TestRenderPetAllStagesNoPanic(t *testing.T) {
	stages := []pet.Stage{
		pet.StageEgg,
		pet.StageBaby,
		pet.StageJunior,
		pet.StageSenior,
		pet.StageLegend,
	}
	for _, s := range stages {
		s := s
		t.Run(s.String(), func(t *testing.T) {
			p := &pet.Pet{
				Name:  "Mochi",
				Stage: s,
				Seed:  99999,
			}
			// Must not panic
			got := renderer.RenderPet(p)
			if got == "" {
				t.Errorf("RenderPet(%s) returned empty string", s)
			}
		})
	}
}

func TestRenderGaugesContainsLabels(t *testing.T) {
	got := renderer.RenderGauges(80, 60, 40)
	for _, label := range []string{"Hunger", "Happy", "Health"} {
		if !strings.Contains(got, label) {
			t.Errorf("RenderGauges output missing label %q", label)
		}
	}
}

func TestRenderStatusLineContainsStageAndName(t *testing.T) {
	p := &pet.Pet{
		Name:  "Mochi",
		Stage: pet.StageSenior,
		Seed:  42,
	}
	got := renderer.RenderStatusLine(p, 70, 80, 90, 1234)

	// Should contain stage abbreviation
	if !strings.Contains(got, "Sr") {
		t.Errorf("RenderStatusLine missing stage abbreviation 'Sr', got: %q", got)
	}
	// Should contain pet name
	if !strings.Contains(got, "Mochi") {
		t.Errorf("RenderStatusLine missing pet name 'Mochi', got: %q", got)
	}
}

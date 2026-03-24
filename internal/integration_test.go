package internal_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/egoomoy/aigotchi/internal/engine"
	"github.com/egoomoy/aigotchi/internal/pet"
	"github.com/egoomoy/aigotchi/internal/renderer"
	"github.com/egoomoy/aigotchi/internal/storage"
)

func TestFullLifecycle(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(filepath.Join(dir, ".aigotchi"))

	// 1. Create pet
	p := pet.NewPet("TestPet")
	s := engine.NewState()

	// 2. Simulate token earning (enough for Baby evolution)
	s.TotalTokensEarned = 15_000_000 // 15000 XP > 10000 threshold

	// 3. Check evolution
	evolved, p, s := engine.CheckEvolution(p, s)
	if !evolved {
		t.Fatal("expected evolution to Baby")
	}
	if p.Stage != pet.StageBaby {
		t.Fatalf("expected Baby, got %s", p.Stage)
	}

	// 4. Render pet (should not panic)
	art := renderer.RenderPet(&p)
	if art == "" {
		t.Fatal("expected non-empty art")
	}

	// 5. Feed
	s2, err := engine.Feed(s)
	if err != nil {
		t.Fatalf("feed failed: %v", err)
	}
	if s2.TotalTokensSpent != 10_000 {
		t.Fatalf("expected 10000 spent, got %d", s2.TotalTokensSpent)
	}

	// 6. Time decay
	s3 := engine.ApplyTimeDelta(s2, 12*time.Hour)
	if s3.Hunger >= s2.Hunger {
		t.Fatal("expected hunger to decrease over time")
	}

	// 7. Save and reload
	store.WriteJSON("pet.json", p)
	store.WriteJSON("state.json", s3)

	var loadedPet pet.Pet
	store.ReadJSON("pet.json", &loadedPet)
	if loadedPet.Name != "TestPet" || loadedPet.Stage != pet.StageBaby {
		t.Fatal("reload mismatch")
	}

	// 8. Status line
	status := renderer.RenderStatusLine(&p, s3.Hunger, s3.Happiness, s3.Health, int(s3.TotalTokensEarned/1000))
	if status == "" {
		t.Fatal("expected non-empty status line")
	}
}

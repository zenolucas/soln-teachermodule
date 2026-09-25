package types

import "testing"

func TestWorldsEveryMinigameIDOnce(t *testing.T) {
	seen := map[int]bool{}
	for _, world := range Worlds {
		for _, scene := range world.Scenes {
			if seen[scene.MinigameID] {
				t.Errorf("minigame ID %d appears more than once", scene.MinigameID)
			}
			seen[scene.MinigameID] = true
		}
	}
	for id := 1; id <= 12; id++ {
		if !seen[id] {
			t.Errorf("minigame ID %d is missing from Worlds", id)
		}
	}
	if len(seen) != 12 {
		t.Errorf("got %d distinct minigame IDs, want 12", len(seen))
	}
}

func TestSceneOp(t *testing.T) {
	got := map[int]string{}
	for _, world := range Worlds {
		for _, scene := range world.Scenes {
			got[scene.MinigameID] = scene.Op
		}
	}
	for id := 1; id <= 5; id++ {
		if got[id] != "+" {
			t.Errorf("scene %d: Op = %q, want \"+\"", id, got[id])
		}
	}
	for id := 6; id <= 11; id++ {
		if got[id] != "\u2212" {
			t.Errorf("scene %d: Op = %q, want \"\\u2212\"", id, got[id])
		}
	}
	if got[12] != "" {
		t.Errorf("scene 12: Op = %q, want \"\"", got[12])
	}
}

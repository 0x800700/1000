package engine_test

import (
	"testing"

	"thousand/internal/engine/sim"
)

func TestSelfPlayRoundsManySeeds(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		if err := sim.RunSelfPlayRounds(seed, 10, 500); err != nil {
			t.Fatalf("self-play failed: %v", err)
		}
	}
}

func FuzzSelfPlayRounds(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(20250211))
	f.Fuzz(func(t *testing.T, seed int64) {
		if err := sim.RunSelfPlayRounds(seed, 3, 500); err != nil {
			t.Fatalf("self-play failed: %v", err)
		}
	})
}

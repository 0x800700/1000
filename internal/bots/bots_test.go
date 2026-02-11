package bots

import (
	"fmt"
	"testing"

	"thousand/internal/engine"
)

type actionRecord struct {
	round  int
	step   int
	phase  engine.Phase
	player int
	action engine.Action
}

func TestBotSelfPlayManySeeds(t *testing.T) {
	for seed := int64(1); seed <= 200; seed++ {
		if err := runBotSelfPlay(seed, 6, 800); err != nil {
			t.Fatalf("bot self-play failed: %v", err)
		}
	}
}

func FuzzBotSelfPlay(f *testing.F) {
	f.Add(int64(1))
	f.Add(int64(42))
	f.Add(int64(20260211))
	f.Fuzz(func(t *testing.T, seed int64) {
		if err := runBotSelfPlay(seed, 3, 800); err != nil {
			t.Fatalf("bot self-play failed: %v", err)
		}
	})
}

func runBotSelfPlay(seed int64, rounds int, maxSteps int) error {
	rules := engine.TisyachaPreset()
	state := engine.NewGame(rules, seed)

	bots := map[int]Bot{
		0: NewNormal(seed + 10),
		1: NewEasy(seed + 20),
		2: NewNormal(seed + 30),
	}

	for r := 0; r < rounds; r++ {
		state.Seed = seed + int64(r)
		engine.DealRound(&state)
		records := []actionRecord{}
		for step := 0; step < maxSteps; step++ {
			if state.Round.Phase == engine.PhaseDeal && !state.Round.HandsDealt {
				break
			}
			player, ok := engine.CurrentPlayer(state)
			if !ok {
				return failure(seed, r, step, state.Round.Phase, -1, records, "no current player")
			}
			legal := engine.LegalActions(state, player)
			if len(legal) == 0 {
				return failure(seed, r, step, state.Round.Phase, player, records, "no legal actions")
			}
			bot := bots[player]
			action := bot.ChooseAction(state, player)
			if err := engine.ApplyAction(&state, player, action); err != nil {
				return failure(seed, r, step, state.Round.Phase, player, records, fmt.Sprintf("apply error: %v", err))
			}
			records = append(records, actionRecord{round: r, step: step, phase: state.Round.Phase, player: player, action: action})
			if state.Round.Phase == engine.PhaseDeal && !state.Round.HandsDealt {
				break
			}
		}
	}
	return nil
}

func failure(seed int64, round int, step int, phase engine.Phase, player int, records []actionRecord, reason string) error {
	start := 0
	if len(records) > 20 {
		start = len(records) - 20
	}
	log := ""
	for _, r := range records[start:] {
		log += fmt.Sprintf("[r%d s%d p%d %v] %v\n", r.round, r.step, r.player, r.phase, r.action)
	}
	return fmt.Errorf("seed=%d round=%d step=%d phase=%v player=%d reason=%s\nlast actions:\n%s",
		seed, round, step, phase, player, reason, log)
}

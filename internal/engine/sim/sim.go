package sim

import (
	"fmt"
	"sort"

	"thousand/internal/engine"
)

type ActionRecord struct {
	Round int
	Step  int
	Phase engine.Phase
	P     int
	A     engine.Action
}

func RunSelfPlayRounds(seed int64, rounds int, maxStepsPerRound int) error {
	rules := engine.TisyachaPreset()
	state := engine.NewGame(rules, seed)

	for r := 0; r < rounds; r++ {
		state.Seed = seed + int64(r)
		engine.DealRound(&state)

		records := []ActionRecord{}
		for step := 0; step < maxStepsPerRound; step++ {
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
			action := chooseAction(state, player, legal)
			if err := engine.ApplyAction(&state, player, action); err != nil {
				return failure(seed, r, step, state.Round.Phase, player, records, fmt.Sprintf("apply error: %v", err))
			}
			records = append(records, ActionRecord{
				Round: r,
				Step:  step,
				Phase: state.Round.Phase,
				P:     player,
				A:     action,
			})
			if err := checkInvariants(state); err != nil {
				return failure(seed, r, step, state.Round.Phase, player, records, err.Error())
			}
			if state.Round.Phase == engine.PhaseDeal && !state.Round.HandsDealt {
				break
			}
		}
	}
	return nil
}

func chooseAction(state engine.GameState, player int, legal []engine.Action) engine.Action {
	switch state.Round.Phase {
	case engine.PhaseSnos:
		return discardLowest(state, player)
	case engine.PhasePlayTricks:
		return lowestLegalPlay(legal)
	default:
		sort.Slice(legal, func(i, j int) bool {
			return actionKey(legal[i]) < actionKey(legal[j])
		})
		return legal[0]
	}
}

func discardLowest(state engine.GameState, player int) engine.Action {
	hand := append([]engine.Card(nil), state.Players[player].Hand...)
	sort.Slice(hand, func(i, j int) bool {
		pi := engine.CardPoints(hand[i].Rank)
		pj := engine.CardPoints(hand[j].Rank)
		if pi == pj {
			return engine.RankStrength(hand[i].Rank) < engine.RankStrength(hand[j].Rank)
		}
		return pi < pj
	})
	count := state.Rules.SnosCards
	if count > len(hand) {
		count = len(hand)
	}
	return engine.Action{Type: engine.ActionSnos, Cards: hand[:count]}
}

func lowestLegalPlay(legal []engine.Action) engine.Action {
	best := legal[0]
	bestScore := 1<<31 - 1
	for _, a := range legal {
		if a.Type != engine.ActionPlayCard || a.Card == nil {
			continue
		}
		score := engine.CardPoints(a.Card.Rank)*10 + engine.RankStrength(a.Card.Rank)
		if score < bestScore {
			bestScore = score
			best = a
		}
	}
	return best
}

func actionKey(a engine.Action) string {
	switch a.Type {
	case engine.ActionBid:
		return fmt.Sprintf("1_bid_%04d", a.Bid)
	case engine.ActionPass:
		return "0_pass"
	case engine.ActionTakeKitty:
		return "3_take"
	case engine.ActionSnos:
		return "4_snos"
	case engine.ActionPlayCard:
		if a.Card == nil {
			return "5_play_?"
		}
		return fmt.Sprintf("5_play_%d_%d", a.Card.Suit, a.Card.Rank)
	case engine.ActionRospis:
		return "6_rospis"
	default:
		return "9_unknown"
	}
}

func checkInvariants(state engine.GameState) error {
	if state.Round.Phase == engine.PhaseDeal && !state.Round.HandsDealt {
		return nil
	}
	total, dup := countCards(state)
	expected := len(engine.BuildDeck(state.Rules))
	if total != expected {
		return fmt.Errorf("card count mismatch: %d", total)
	}
	if dup {
		return fmt.Errorf("duplicate card detected")
	}
	if len(state.Round.TrickCards) > 3 {
		return fmt.Errorf("invalid trick size: %d", len(state.Round.TrickCards))
	}
	if state.Round.Phase == engine.PhaseBidding || state.Round.Phase == engine.PhaseKittyTake {
		for _, p := range state.Players {
			if len(p.Hand) != state.Rules.DealHandSize {
				return fmt.Errorf("hand size mismatch in bidding: %d", len(p.Hand))
			}
		}
		if len(state.Round.Kitty) != state.Rules.KittySize {
			return fmt.Errorf("kitty size mismatch in bidding: %d", len(state.Round.Kitty))
		}
	}
	if state.Round.Phase == engine.PhaseSnos {
		if len(state.Players[state.Round.BidWinner].Hand) != state.Rules.DealHandSize+state.Rules.KittySize {
			return fmt.Errorf("bidder hand not expanded after kitty")
		}
		if len(state.Round.Kitty) != 0 {
			return fmt.Errorf("kitty should be empty after take")
		}
		for i, p := range state.Players {
			if i == state.Round.BidWinner {
				continue
			}
			if len(p.Hand) != state.Rules.DealHandSize {
				return fmt.Errorf("opponent hand size mismatch in snos: %d", len(p.Hand))
			}
		}
	}
	if state.Round.Phase == engine.PhasePlayTricks {
		for _, p := range state.Players {
			if len(p.Hand) > state.Rules.PlayHandSize {
				return fmt.Errorf("hand size too large: %d", len(p.Hand))
			}
		}
		if totalTricks(state) == 0 && len(state.Round.TrickCards) == 0 {
			for _, p := range state.Players {
				if len(p.Hand) != state.Rules.PlayHandSize {
					return fmt.Errorf("hand size mismatch at start of play: %d", len(p.Hand))
				}
			}
		}
	}
	return nil
}

func totalTricks(state engine.GameState) int {
	total := 0
	for _, p := range state.Players {
		total += len(p.Tricks)
	}
	return total
}

func countCards(state engine.GameState) (int, bool) {
	seen := map[engine.Card]int{}
	total := 0
	dup := false
	add := func(c engine.Card) {
		total++
		if seen[c] > 0 {
			dup = true
		}
		seen[c]++
	}
	for _, p := range state.Players {
		for _, c := range p.Hand {
			add(c)
		}
		for _, trick := range p.Tricks {
			for _, c := range trick {
				add(c)
			}
		}
	}
	for _, c := range state.Round.Kitty {
		add(c)
	}
	for _, c := range state.Round.TrickCards {
		add(c)
	}
	return total, dup
}

func failure(seed int64, round int, step int, phase engine.Phase, player int, records []ActionRecord, reason string) error {
	start := 0
	if len(records) > 20 {
		start = len(records) - 20
	}
	log := ""
	for _, r := range records[start:] {
		log += fmt.Sprintf("[r%d s%d p%d %v] %v\n", r.Round, r.Step, r.P, r.Phase, r.A)
	}
	return fmt.Errorf("seed=%d round=%d step=%d phase=%v player=%d reason=%s\nlast actions:\n%s",
		seed, round, step, phase, player, reason, log)
}

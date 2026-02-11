package bots

import (
	"math/rand"
	"sort"

	"thousand/internal/engine"
)

type Bot interface {
	ChooseAction(state engine.GameState, player int) engine.Action
}

type EasyBot struct {
	RNG *rand.Rand
}

func NewEasy(seed int64) *EasyBot {
	return &EasyBot{RNG: rand.New(rand.NewSource(seed))}
}

func (b *EasyBot) ChooseAction(state engine.GameState, player int) engine.Action {
	legal := engine.LegalActions(state, player)
	if len(legal) == 0 {
		return engine.Action{Type: engine.ActionPass}
	}
	switch state.Round.Phase {
	case engine.PhaseSnos:
		return discardLowestPoints(state, player, state.Rules.SnosCards)
	case engine.PhaseBidding:
		return legal[b.RNG.Intn(len(legal))]
	case engine.PhasePlayTricks:
		return legal[b.RNG.Intn(len(legal))]
	default:
		return legal[0]
	}
}

type NormalBot struct {
	RNG *rand.Rand
}

func NewNormal(seed int64) *NormalBot {
	return &NormalBot{RNG: rand.New(rand.NewSource(seed))}
}

func (b *NormalBot) ChooseAction(state engine.GameState, player int) engine.Action {
	switch state.Round.Phase {
	case engine.PhaseBidding:
		return bidByHeuristic(state, player)
	case engine.PhaseSnos:
		return discardLowestPoints(state, player, state.Rules.SnosCards)
	case engine.PhasePlayTricks:
		return playHeuristic(state, player)
	default:
		legal := engine.LegalActions(state, player)
		if len(legal) == 0 {
			return engine.Action{Type: engine.ActionPass}
		}
		return legal[0]
	}
}

func discardLowestPoints(state engine.GameState, player int, count int) engine.Action {
	hand := append([]engine.Card(nil), state.Players[player].Hand...)
	pairSuit := marriagePairs(hand)
	sort.Slice(hand, func(i, j int) bool {
		pi := engine.CardPoints(hand[i].Rank)
		pj := engine.CardPoints(hand[j].Rank)
		if pairSuit[hand[i].Suit] && (hand[i].Rank == engine.RankQ || hand[i].Rank == engine.RankK) {
			pi += 20
		}
		if pairSuit[hand[j].Suit] && (hand[j].Rank == engine.RankQ || hand[j].Rank == engine.RankK) {
			pj += 20
		}
		if pi == pj {
			return engine.RankStrength(hand[i].Rank) < engine.RankStrength(hand[j].Rank)
		}
		return pi < pj
	})
	if count > len(hand) {
		count = len(hand)
	}
	return engine.Action{Type: engine.ActionSnos, Cards: hand[:count]}
}

func bidByHeuristic(state engine.GameState, player int) engine.Action {
	hand := state.Players[player].Hand
	points := 0
	suitCounts := map[engine.Suit]int{}
	for _, c := range hand {
		points += engine.CardPoints(c.Rank)
		suitCounts[c.Suit]++
	}
	for suit := range marriagePairs(hand) {
		points += marriageValue(suit)
	}
	bonus := 0
	for _, c := range suitCounts {
		if c >= 3 {
			bonus += (c - 2) * 4
		}
	}
	estimate := points + bonus
	maxBid := (estimate / state.Rules.BidStep) * state.Rules.BidStep
	rulesMax := state.Rules.MaxBid
	if rulesMax <= 0 {
		rulesMax = state.Rules.WinScore
	}
	if maxBid > rulesMax {
		maxBid = rulesMax
	}
	if maxBid < state.Rules.BidMin {
		return engine.Action{Type: engine.ActionPass}
	}
	if maxBid <= state.Round.BidValue {
		return engine.Action{Type: engine.ActionPass}
	}
	return engine.Action{Type: engine.ActionBid, Bid: maxBid}
}

func playHeuristic(state engine.GameState, player int) engine.Action {
	legal := engine.LegalActions(state, player)
	if len(legal) == 0 {
		return engine.Action{Type: engine.ActionPass}
	}
	if m, ok := bestMarriageAction(legal); ok {
		return m
	}
	hand := state.Players[player].Hand
	_ = hand
	if len(state.Round.TrickCards) == 0 {
		// Lead with highest point card
		best := legal[0]
		bestScore := -1
		for _, a := range legal {
			if a.Card == nil {
				continue
			}
			score := engine.CardPoints(a.Card.Rank)*10 + engine.RankStrength(a.Card.Rank)
			if score > bestScore {
				bestScore = score
				best = a
			}
		}
		return best
	}
	// Try to win trick with lowest winning card if possible
	trump := state.Round.Trump
	bestWinning := (engine.Action{})
	bestRank := 999
	for _, a := range legal {
		if a.Card == nil {
			continue
		}
		if winsIfPlayed(state, player, *a.Card, trump) {
			r := engine.RankStrength(a.Card.Rank)
			if r < bestRank {
				bestRank = r
				bestWinning = a
			}
		}
	}
	if bestRank != 999 {
		return bestWinning
	}
	// Otherwise shed lowest point card
	lowest := legal[0]
	lowestScore := 999
	for _, a := range legal {
		if a.Card == nil {
			continue
		}
		score := engine.CardPoints(a.Card.Rank)*10 + engine.RankStrength(a.Card.Rank)
		if score < lowestScore {
			lowestScore = score
			lowest = a
		}
	}
	return lowest
}

func marriagePairs(hand []engine.Card) map[engine.Suit]bool {
	pairs := map[engine.Suit]bool{}
	hasQ := map[engine.Suit]bool{}
	hasK := map[engine.Suit]bool{}
	for _, c := range hand {
		if c.Rank == engine.RankQ {
			hasQ[c.Suit] = true
		}
		if c.Rank == engine.RankK {
			hasK[c.Suit] = true
		}
	}
	for s := range hasQ {
		if hasK[s] {
			pairs[s] = true
		}
	}
	return pairs
}

func marriageValue(s engine.Suit) int {
	switch s {
	case engine.SuitHearts:
		return 100
	case engine.SuitDiamonds:
		return 80
	case engine.SuitClubs:
		return 60
	case engine.SuitSpades:
		return 40
	default:
		return 0
	}
}

func bestMarriageAction(legal []engine.Action) (engine.Action, bool) {
	best := engine.Action{}
	bestValue := -1
	for _, a := range legal {
		if a.MarriageSuit == nil {
			continue
		}
		val := marriageValue(*a.MarriageSuit)
		if val > bestValue {
			bestValue = val
			best = a
		}
	}
	if bestValue >= 0 {
		return best, true
	}
	return engine.Action{}, false
}

func winsIfPlayed(state engine.GameState, player int, card engine.Card, trump *engine.Suit) bool {
	cards := append([]engine.Card(nil), state.Round.TrickCards...)
	orderPlayers := append([]int(nil), state.Round.TrickOrder...)
	if len(orderPlayers) == 0 {
		orderPlayers = make([]int, 0, state.Rules.Players)
		for i := 0; i < state.Rules.Players; i++ {
			orderPlayers = append(orderPlayers, (state.Round.Leader+i)%state.Rules.Players)
		}
	}
	cards = append(cards, card)
	orderPlayers = append(orderPlayers, player)
	winnerID := trickWinnerLocal(orderPlayers, cards, trump)
	return winnerID == player
}

// local copy to avoid exposing engine internal function
func trickWinnerLocal(order []int, cards []engine.Card, trump *engine.Suit) int {
	if len(order) == 0 || len(cards) == 0 {
		return -1
	}
	leadSuit := cards[0].Suit
	bestIdx := 0
	for i := 1; i < len(cards); i++ {
		c := cards[i]
		best := cards[bestIdx]

		if trump != nil {
			if c.Suit == *trump && best.Suit != *trump {
				bestIdx = i
				continue
			}
			if c.Suit != *trump && best.Suit == *trump {
				continue
			}
		}

		if c.Suit == best.Suit {
			if engine.RankStrength(c.Rank) > engine.RankStrength(best.Rank) {
				bestIdx = i
			}
			continue
		}

		if best.Suit != leadSuit && c.Suit == leadSuit {
			bestIdx = i
		}
	}
	return order[bestIdx]
}

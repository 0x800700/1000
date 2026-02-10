package engine

import "math/rand"

func BuildDeck(r Rules) []Card {
	deck := make([]Card, 0, 24)
	suits := []Suit{SuitClubs, SuitDiamonds, SuitHearts, SuitSpades}
	for _, s := range suits {
		for _, rank := range r.DeckRanks {
			deck = append(deck, Card{Suit: s, Rank: rank})
		}
	}
	return deck
}

func Shuffle(deck []Card, seed int64) []Card {
	shuffled := make([]Card, len(deck))
	copy(shuffled, deck)
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

// DealRound deals cards into player hands and kitty based on rules.
// It mutates game state deterministically based on seed.
func DealRound(g *GameState) {
	deck := Shuffle(BuildDeck(g.Rules), g.Seed)
	players := g.Rules.Players
	handSize := g.Rules.HandSize
	kittySize := g.Rules.KittySize

	if handSize*players+kittySize != len(deck) {
		panic("invalid deal configuration: does not exhaust deck")
	}

	idx := 0
	for p := 0; p < players; p++ {
		g.Players[p].Hand = append([]Card(nil), deck[idx:idx+handSize]...)
		idx += handSize
	}
	g.Round.Kitty = append([]Card(nil), deck[idx:idx+kittySize]...)
	g.Round.HandsDealt = true
	g.Round.Phase = PhaseBidding
	g.Round.Bids = make(map[int]int)
	g.Round.Passed = make(map[int]bool)
	g.Round.BidTurn = (g.Round.Dealer + 1) % players
	g.Round.BidWinner = -1
	g.Round.BidValue = 0
}

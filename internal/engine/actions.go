package engine

import "errors"

type ActionType int

const (
	ActionBid ActionType = iota
	ActionPass
	ActionChooseTrump
	ActionTakeKitty
	ActionDiscard
	ActionPlayCard
)

type Action struct {
	Type  ActionType
	Bid   int
	Suit  *Suit
	Card  *Card
	Cards []Card
}

func LegalActions(g GameState, player int) []Action {
	switch g.Round.Phase {
	case PhaseBidding:
		return legalBids(g, player)
	case PhaseTrumpSelect:
		if player != g.Round.BidWinner {
			return nil
		}
		return []Action{
			{Type: ActionChooseTrump, Suit: suitPtr(SuitClubs)},
			{Type: ActionChooseTrump, Suit: suitPtr(SuitDiamonds)},
			{Type: ActionChooseTrump, Suit: suitPtr(SuitHearts)},
			{Type: ActionChooseTrump, Suit: suitPtr(SuitSpades)},
		}
	case PhaseKittyTake:
		if player != g.Round.BidWinner {
			return nil
		}
		return []Action{{Type: ActionTakeKitty}}
	case PhaseDiscard:
		if player != g.Round.BidWinner {
			return nil
		}
		// Too many combinations; client/bot should choose.
		return []Action{{Type: ActionDiscard}}
	case PhasePlayTricks:
		return legalPlays(g, player)
	default:
		return nil
	}
}

func ApplyAction(g *GameState, player int, a Action) error {
	switch g.Round.Phase {
	case PhaseBidding:
		return applyBid(g, player, a)
	case PhaseTrumpSelect:
		return applyTrump(g, player, a)
	case PhaseKittyTake:
		return applyKittyTake(g, player, a)
	case PhaseDiscard:
		return applyDiscard(g, player, a)
	case PhasePlayTricks:
		return applyPlay(g, player, a)
	case PhaseScoreRound:
		scoreRound(g)
		return nil
	default:
		return errors.New("invalid phase")
	}
}

func applyBid(g *GameState, player int, a Action) error {
	if player != g.Round.BidTurn {
		return errors.New("not your turn")
	}
	if g.Round.Passed[player] {
		return errors.New("player already passed")
	}

	switch a.Type {
	case ActionPass:
		g.Round.Passed[player] = true
	case ActionBid:
		if a.Bid < g.Rules.BidMin {
			return errors.New("bid below minimum")
		}
		if (a.Bid-g.Rules.BidMin)%g.Rules.BidStep != 0 {
			return errors.New("invalid bid step")
		}
		if a.Bid <= g.Round.BidValue {
			return errors.New("bid not high enough")
		}
		g.Round.BidValue = a.Bid
		g.Round.BidWinner = player
		g.Round.Bids[player] = a.Bid
	default:
		return errors.New("invalid action for bidding")
	}

	// Advance bidding
	active := 0
	for p := 0; p < g.Rules.Players; p++ {
		if !g.Round.Passed[p] {
			active++
		}
	}
	if active == 1 && g.Round.BidWinner >= 0 {
		g.Round.Phase = PhaseTrumpSelect
		return nil
	}
	if active == 0 {
		// All passed, redeal next round
		g.Round.Dealer = (g.Round.Dealer + 1) % g.Rules.Players
		g.ResetRound()
		return nil
	}

	g.Round.BidTurn = nextBidTurn(g)
	return nil
}

func applyTrump(g *GameState, player int, a Action) error {
	if player != g.Round.BidWinner {
		return errors.New("only bidder chooses trump")
	}
	if a.Type != ActionChooseTrump || a.Suit == nil {
		return errors.New("invalid trump action")
	}
	g.Round.Trump = a.Suit
	g.Round.Phase = PhaseKittyTake
	return nil
}

func applyKittyTake(g *GameState, player int, a Action) error {
	if player != g.Round.BidWinner {
		return errors.New("only bidder takes kitty")
	}
	if a.Type != ActionTakeKitty {
		return errors.New("invalid action for kitty")
	}
	g.Players[player].Hand = append(g.Players[player].Hand, g.Round.Kitty...)
	g.Round.Kitty = nil
	g.Round.Phase = PhaseDiscard
	return nil
}

func applyDiscard(g *GameState, player int, a Action) error {
	if player != g.Round.BidWinner {
		return errors.New("only bidder discards")
	}
	if a.Type != ActionDiscard || len(a.Cards) != g.Rules.KittySize {
		return errors.New("discard requires exact kitty size")
	}

	for _, c := range a.Cards {
		if !removeCard(&g.Players[player].Hand, c) {
			return errors.New("discard card not in hand")
		}
	}
	if len(g.Players[player].Hand) != g.Rules.HandSize {
		return errors.New("invalid hand size after discard")
	}
	g.Round.Phase = PhasePlayTricks
	g.Round.Leader = g.Round.BidWinner
	g.Round.TrickCards = nil
	g.Round.TrickOrder = nil
	return nil
}

func applyPlay(g *GameState, player int, a Action) error {
	if a.Type != ActionPlayCard || a.Card == nil {
		return errors.New("invalid play action")
	}
	if len(g.Round.TrickOrder) == 0 {
		g.Round.TrickOrder = buildTrickOrder(g.Round.Leader, g.Rules.Players)
	}
	expected := g.Round.TrickOrder[len(g.Round.TrickCards)]
	if player != expected {
		return errors.New("not your turn to play")
	}
	if !removeCard(&g.Players[player].Hand, *a.Card) {
		return errors.New("card not in hand")
	}
	// Validate legal play (follow suit)
	legal := legalPlays(*g, player)
	if !actionInList(Action{Type: ActionPlayCard, Card: a.Card}, legal) {
		return errors.New("illegal card play")
	}

	g.Round.TrickCards = append(g.Round.TrickCards, *a.Card)
	if len(g.Round.TrickCards) == g.Rules.Players {
		winner := trickWinner(g.Round.TrickOrder, g.Round.TrickCards, g.Round.Trump)
		g.Players[winner].Tricks = append(g.Players[winner].Tricks, append([]Card(nil), g.Round.TrickCards...))
		g.Round.Leader = winner
		g.Round.TrickCards = nil
		g.Round.TrickOrder = nil

		if len(g.Players[winner].Hand) == 0 {
			g.Round.Phase = PhaseScoreRound
			scoreRound(g)
		}
	}
	return nil
}

func legalBids(g GameState, player int) []Action {
	if player != g.Round.BidTurn || g.Round.Passed[player] {
		return nil
	}
	out := []Action{{Type: ActionPass}}
	for bid := g.Rules.BidMin; bid <= g.Rules.WinScore; bid += g.Rules.BidStep {
		if bid > g.Round.BidValue {
			out = append(out, Action{Type: ActionBid, Bid: bid})
		}
	}
	return out
}

func legalPlays(g GameState, player int) []Action {
	if g.Round.Phase != PhasePlayTricks {
		return nil
	}
	if len(g.Round.TrickOrder) == 0 {
		g.Round.TrickOrder = buildTrickOrder(g.Round.Leader, g.Rules.Players)
	}
	expected := g.Round.TrickOrder[len(g.Round.TrickCards)]
	if player != expected {
		return nil
	}
	hand := g.Players[player].Hand
	if len(hand) == 0 {
		return nil
	}
	// If must follow suit and trick has led suit, restrict.
	if g.Rules.MustFollowSuit && len(g.Round.TrickCards) > 0 {
		led := g.Round.TrickCards[0].Suit
		if hasSuit(hand, led) {
			return cardsToActions(filterBySuit(hand, led))
		}
	}
	return cardsToActions(hand)
}

func cardsToActions(cards []Card) []Action {
	out := make([]Action, 0, len(cards))
	for i := range cards {
		c := cards[i]
		out = append(out, Action{Type: ActionPlayCard, Card: &c})
	}
	return out
}

func buildTrickOrder(leader, players int) []int {
	order := make([]int, 0, players)
	for i := 0; i < players; i++ {
		order = append(order, (leader+i)%players)
	}
	return order
}

func nextBidTurn(g *GameState) int {
	for i := 1; i <= g.Rules.Players; i++ {
		n := (g.Round.BidTurn + i) % g.Rules.Players
		if !g.Round.Passed[n] {
			return n
		}
	}
	return g.Round.BidTurn
}

func hasSuit(cards []Card, suit Suit) bool {
	for _, c := range cards {
		if c.Suit == suit {
			return true
		}
	}
	return false
}

func filterBySuit(cards []Card, suit Suit) []Card {
	out := []Card{}
	for _, c := range cards {
		if c.Suit == suit {
			out = append(out, c)
		}
	}
	return out
}

func removeCard(hand *[]Card, card Card) bool {
	for i, c := range *hand {
		if c == card {
			*hand = append((*hand)[:i], (*hand)[i+1:]...)
			return true
		}
	}
	return false
}

func actionInList(a Action, list []Action) bool {
	for _, l := range list {
		if a.Type != l.Type {
			continue
		}
		if a.Type == ActionPlayCard && l.Card != nil && a.Card != nil && *a.Card == *l.Card {
			return true
		}
	}
	return false
}

func suitPtr(s Suit) *Suit {
	return &s
}

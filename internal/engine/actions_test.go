package engine

import "testing"

func TestBidValidation(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	DealRound(&g)

	player := g.Round.BidTurn
	if err := ApplyAction(&g, player, Action{Type: ActionBid, Bid: r.BidMin - 10}); err == nil {
		t.Fatalf("expected error for bid below minimum")
	}
	if err := ApplyAction(&g, player, Action{Type: ActionBid, Bid: r.BidMin}); err != nil {
		t.Fatalf("valid bid rejected: %v", err)
	}
}

func TestLegalPlaysFollowSuit(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.Phase = PhasePlayTricks
	g.Round.Leader = 0
	g.Round.TrickOrder = []int{0, 1, 2}
	g.Round.TrickCards = []Card{{Suit: SuitHearts, Rank: RankA}}

	g.Players[1].Hand = []Card{
		{Suit: SuitHearts, Rank: Rank9},
		{Suit: SuitSpades, Rank: RankA},
	}

	actions := LegalActions(g, 1)
	if len(actions) != 1 {
		t.Fatalf("expected 1 legal action, got %d", len(actions))
	}
	if actions[0].Card == nil || actions[0].Card.Suit != SuitHearts {
		t.Fatalf("expected only hearts to be legal")
	}
}

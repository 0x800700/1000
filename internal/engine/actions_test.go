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

func TestLegalBidsRespectsMaxBid(t *testing.T) {
	r := ClassicPreset()
	r.MaxBid = 200
	g := NewGame(r, 1)
	DealRound(&g)

	acts := LegalActions(g, g.Round.BidTurn)
	for _, a := range acts {
		if a.Type == ActionBid && a.Bid > r.MaxBid {
			t.Fatalf("bid exceeds max: %d", a.Bid)
		}
	}
}

func TestApplyActionRejectsIllegal(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	DealRound(&g)
	// Not the bid turn should fail
	illegalPlayer := (g.Round.BidTurn + 1) % r.Players
	err := ApplyAction(&g, illegalPlayer, Action{Type: ActionPass})
	if err == nil {
		t.Fatalf("expected error for illegal turn")
	}
}

func TestSnosDistributesToOpponents(t *testing.T) {
	r := ClassicPreset()
	r.DealHandSize = 7
	r.PlayHandSize = 8
	r.KittySize = 3
	r.SnosCards = 2
	g := NewGame(r, 1)
	g.Round.Phase = PhaseSnos
	g.Round.BidWinner = 0
	g.Players[0].Hand = []Card{
		{Suit: SuitHearts, Rank: RankA},
		{Suit: SuitSpades, Rank: Rank10},
		{Suit: SuitClubs, Rank: RankK},
		{Suit: SuitDiamonds, Rank: RankQ},
		{Suit: SuitHearts, Rank: RankJ},
		{Suit: SuitSpades, Rank: Rank9},
		{Suit: SuitClubs, Rank: Rank9},
		{Suit: SuitDiamonds, Rank: Rank9},
		{Suit: SuitHearts, Rank: Rank10},
		{Suit: SuitSpades, Rank: RankA},
	}
	g.Players[1].Hand = make([]Card, 7)
	g.Players[2].Hand = make([]Card, 7)
	c1 := g.Players[0].Hand[0]
	c2 := g.Players[0].Hand[1]

	if err := ApplyAction(&g, 0, Action{Type: ActionSnos, Cards: []Card{c1, c2}}); err != nil {
		t.Fatalf("snos failed: %v", err)
	}
	if len(g.Players[0].Hand) != r.PlayHandSize {
		t.Fatalf("bidder hand size expected %d got %d", r.PlayHandSize, len(g.Players[0].Hand))
	}
	if len(g.Players[1].Hand) != r.PlayHandSize || len(g.Players[2].Hand) != r.PlayHandSize {
		t.Fatalf("opponent hand sizes expected %d got %d/%d", r.PlayHandSize, len(g.Players[1].Hand), len(g.Players[2].Hand))
	}
	if !containsCard(g.Players[1].Hand, c1) || !containsCard(g.Players[2].Hand, c2) {
		t.Fatalf("snos cards not distributed to opponents")
	}
	if g.Round.Phase != PhasePlayTricks {
		t.Fatalf("expected phase to move to play tricks")
	}
}

func containsCard(hand []Card, card Card) bool {
	for _, c := range hand {
		if c == card {
			return true
		}
	}
	return false
}

func TestMarriageDeclarationSetsTrump(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.Phase = PhasePlayTricks
	g.Round.Leader = 0
	g.Round.DeclaredMarriages = make(map[int]map[Suit]bool)

	g.Players[0].Hand = []Card{
		{Suit: SuitHearts, Rank: RankQ},
		{Suit: SuitHearts, Rank: RankK},
		{Suit: SuitClubs, Rank: Rank9},
	}
	g.Players[0].Tricks = [][]Card{{{Suit: SuitClubs, Rank: RankA}}}

	card := g.Players[0].Hand[0]
	suit := SuitHearts
	if err := ApplyAction(&g, 0, Action{Type: ActionPlayCard, Card: &card, MarriageSuit: &suit}); err != nil {
		t.Fatalf("marriage play failed: %v", err)
	}
	if g.Round.Trump == nil || *g.Round.Trump != SuitHearts {
		t.Fatalf("expected trump to be hearts")
	}
	if g.Players[0].MarriagePts != 100 {
		t.Fatalf("expected marriage points 100, got %d", g.Players[0].MarriagePts)
	}
}

func TestMarriageRequiresTrick(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.Phase = PhasePlayTricks
	g.Round.Leader = 0
	g.Round.DeclaredMarriages = make(map[int]map[Suit]bool)

	g.Players[0].Hand = []Card{
		{Suit: SuitHearts, Rank: RankQ},
		{Suit: SuitHearts, Rank: RankK},
	}

	card := g.Players[0].Hand[0]
	suit := SuitHearts
	if err := ApplyAction(&g, 0, Action{Type: ActionPlayCard, Card: &card, MarriageSuit: &suit}); err == nil {
		t.Fatalf("expected error when declaring marriage without trick")
	}
}

func TestRospisAdjustsScoresAndResetsRound(t *testing.T) {
	r := ClassicPreset()
	g := NewGame(r, 1)
	g.Round.Phase = PhasePlayTricks
	g.Round.BidWinner = 0
	g.Round.BidValue = 120
	g.Round.Bids = map[int]int{0: 120}

	if err := ApplyAction(&g, 0, Action{Type: ActionRospis}); err != nil {
		t.Fatalf("rospis failed: %v", err)
	}
	if g.Players[0].GameScore != -120 {
		t.Fatalf("expected bidder score -120, got %d", g.Players[0].GameScore)
	}
	if g.Players[1].GameScore != 60 || g.Players[2].GameScore != 60 {
		t.Fatalf("expected opponents +60 each, got %d/%d", g.Players[1].GameScore, g.Players[2].GameScore)
	}
	if g.Round.Phase != PhaseDeal {
		t.Fatalf("expected round reset to deal")
	}
}
